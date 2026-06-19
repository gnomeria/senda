// senda-cli runs a collection (or one folder of it) headlessly — same pipeline
// as the desktop app: scripts, variables, secrets, asserts, cookie jar.
// Exit code 0 = every request passed; 1 = at least one failure.
//
//	senda-cli -collection ./my-api [-folder auth] [-env dev] [-q]
//	senda-cli -collection ./my-api --docs [-o docs/api.md] [--docs-format html|md]
//	senda-cli mock [-collection ./my-api] [-addr :8787] [-scenario error]
//	senda-cli mock init oauth [-collection ./my-api]   # scaffold a mock preset
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"senda/internal/docgen"
	"senda/internal/model"
	"senda/internal/pipeline"
	"senda/internal/runner"
	"senda/internal/store"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "mock" {
		runMock(os.Args[2:])
		return
	}

	collPath := flag.String("collection", ".", "collection root directory")
	folder := flag.String("folder", "", "subfolder to run (default: whole collection)")
	env := flag.String("env", "", "environment name")
	quiet := flag.Bool("q", false, "only print the summary line")
	docs := flag.Bool("docs", false, "generate API documentation instead of running requests")
	docsOutput := flag.String("o", "", "output file for docs (default: stdout)")
	docsFormat := flag.String("docs-format", "md", "docs output format: md or html")
	dataFile := flag.String("data", "", "CSV or JSON data file for data-driven runs")
	flag.Parse()

	root, err := filepath.Abs(*collPath)
	if err != nil {
		fatal(err)
	}

	if *docs {
		runDocs(root, *folder, *docsOutput, *docsFormat)
		return
	}

	target := root
	if *folder != "" {
		target = filepath.Join(root, *folder)
	}
	paths, err := store.ListRequests(target)
	if err != nil {
		fatal(err)
	}
	if len(paths) == 0 {
		fatal(fmt.Errorf("no requests under %s", target))
	}

	// Data-driven: load rows if --data is set.
	var dataRows []map[string]string
	if *dataFile != "" {
		var err error
		dataRows, err = runner.LoadDataFile(*dataFile)
		if err != nil {
			fatal(err)
		}
	}

	session := pipeline.NewSession()
	makeSend := func(extra map[string]string) runner.Send {
		return func(ctx context.Context, path string) (model.Request, model.Response, error) {
			req, err := store.ReadRequest(path)
			if err != nil {
				return req, model.Response{}, err
			}
			resp, _ := session.SendWithExtra(ctx, req, root, path, *env, extra)
			return req, resp, nil
		}
	}
	send := makeSend(nil)

	onResult := func(r model.RunResult) {
		if *quiet {
			return
		}
		fmt.Println(formatResult(r))
	}

	var results []model.RunResult
	if len(dataRows) > 0 {
		ri := 0
		results = runner.RunFolderWithData(context.Background(), paths, dataRows,
			makeSend,
			func(rowIdx int, r model.RunResult) {
				ri++
				if !*quiet {
					fmt.Printf("[row %d] ", rowIdx+1)
				}
				onResult(r)
			})
		_ = ri
	} else {
		results = runner.RunFolder(context.Background(), paths, send, onResult)
	}

	passed := 0
	for _, r := range results {
		if r.OK {
			passed++
		}
	}
	fmt.Printf("\n%d/%d passed\n", passed, len(results))
	if passed != len(results) {
		os.Exit(1)
	}
}

// formatResult renders one run result as the single status line senda-cli
// prints per request: a pass/fail mark, the name, method, status, duration, and
// (when present) the assertion tally and any error.
func formatResult(r model.RunResult) string {
	mark := "✓"
	if !r.OK {
		mark = "✗"
	}
	line := fmt.Sprintf("%s %-40s %s %d (%dms)", mark, r.Name, r.Method, r.Status, r.DurationMs)
	if r.AssertPass+r.AssertFail > 0 {
		line += fmt.Sprintf("  asserts %d/%d", r.AssertPass, r.AssertPass+r.AssertFail)
	}
	if r.Error != "" {
		line += "  " + r.Error
	}
	return line
}

func runDocs(collPath, subFolder, outFile, format string) {
	subPath := ""
	if subFolder != "" {
		subPath = filepath.Join(collPath, subFolder)
	}

	var (
		content string
		err     error
	)
	switch docgen.Format(format) {
	case docgen.FormatHTML:
		content, err = docgen.GenerateHTML(collPath, subPath)
	default:
		content, err = docgen.GenerateMarkdown(collPath, subPath)
	}
	if err != nil {
		fatal(err)
	}

	if outFile == "" {
		fmt.Print(content)
		return
	}
	if err := os.WriteFile(outFile, []byte(content), 0o644); err != nil {
		fatal(err)
	}
	fmt.Fprintf(os.Stderr, "docs written to %s\n", outFile)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "senda-cli:", err)
	os.Exit(1)
}
