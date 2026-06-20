package main

import (
	"flag"
	"path/filepath"
)

// runDocsCmd is the top-level `senda docs` subcommand: it renders a collection
// (or one folder) to Markdown or HTML. Equivalent to `senda run --docs`, kept as
// its own verb for discoverability.
func runDocsCmd(args []string) {
	fs := flag.NewFlagSet("docs", flag.ExitOnError)
	collPath := fs.String("collection", ".", "collection root directory")
	folder := fs.String("folder", "", "subfolder to render (default: whole collection)")
	out := fs.String("o", "", "output file (default: stdout)")
	format := fs.String("docs-format", "md", "output format: md or html")
	_ = fs.Parse(args)

	root, err := filepath.Abs(*collPath)
	if err != nil {
		fatal(err)
	}
	runDocs(root, *folder, *out, *format)
}
