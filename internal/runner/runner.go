// Package runner executes a sequence of requests (a folder run) and collects a
// RunResult per request. It is transport-agnostic: the caller injects a Send
// func, keeping the package unit-testable without a network.
package runner

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"senda/internal/model"
)

// Send sends one request and returns its response. Implemented in production by
// the app's HTTP client + variable scope; faked in tests.
type Send func(ctx context.Context, path string) (model.Request, model.Response, error)

// RunFolder runs each path in order via send, returning one RunResult each.
// A read/transport error becomes a failed result rather than aborting the run.
// onResult, when non-nil, is invoked with each result as it lands so callers
// can stream progress to the UI.
//
// Branching via req.OnFail:
//   - "stop"           — abort the run on any failure
//   - "continue"       — (default) keep going regardless
//   - "jump:<folder>"  — skip ahead to requests under the named folder prefix
func RunFolder(ctx context.Context, paths []string, send Send, onResult func(model.RunResult)) []model.RunResult {
	results := make([]model.RunResult, 0, len(paths))
	i := 0
	for i < len(paths) {
		p := paths[i]
		select {
		case <-ctx.Done():
			cancelled := model.RunResult{Path: p, Error: ctx.Err().Error()}
			if onResult != nil {
				onResult(cancelled)
			}
			results = append(results, cancelled)
			return results
		default:
		}
		req, resp, err := send(ctx, p)
		res := model.RunResult{
			Path:       p,
			Name:       req.Name,
			Method:     req.Method,
			URL:        req.URL,
			Status:     resp.Status,
			DurationMs: resp.DurationMs,
			SizeBytes:  resp.SizeBytes,
		}
		if err == nil {
			r := resp
			res.Response = &r
		}
		for _, ar := range resp.Asserts {
			if ar.Pass {
				res.AssertPass++
			} else {
				res.AssertFail++
			}
		}
		switch {
		case err != nil:
			res.Error = err.Error()
		case resp.Error != "":
			res.Error = resp.Error
		default:
			res.OK = resp.Status >= 200 && resp.Status < 400 && res.AssertFail == 0
		}
		if onResult != nil {
			onResult(res)
		}
		results = append(results, res)

		// Branching on failure.
		if !res.OK && req.OnFail != "" {
			switch {
			case req.OnFail == "stop":
				return results
			case strings.HasPrefix(req.OnFail, "jump:"):
				target := strings.TrimPrefix(req.OnFail, "jump:")
				next := findFolderStart(paths, target, i+1)
				if next >= 0 {
					i = next
					continue
				}
				// target not found — stop
				return results
			}
			// "continue" or unknown — fall through
		}
		i++
	}
	return results
}

// findFolderStart returns the index of the first path whose containing
// directory name matches folderName, starting at offset.
func findFolderStart(paths []string, folderName string, offset int) int {
	for j := offset; j < len(paths); j++ {
		dir := filepath.Base(filepath.Dir(paths[j]))
		if dir == folderName {
			return j
		}
	}
	return -1
}

// RunFolderWithData runs the folder once per data row, injecting each row's
// key/value pairs as extra runtime variables. Each row produces a separate
// batch of RunResults tagged with the row index.
func RunFolderWithData(
	ctx context.Context,
	paths []string,
	rows []map[string]string,
	makeSend func(extraVars map[string]string) Send,
	onResult func(rowIdx int, r model.RunResult),
) []model.RunResult {
	var all []model.RunResult
	for rowIdx, row := range rows {
		send := makeSend(row)
		ri := rowIdx
		results := RunFolder(ctx, paths, send, func(r model.RunResult) {
			if onResult != nil {
				onResult(ri, r)
			}
		})
		all = append(all, results...)
	}
	return all
}

// LoadDataFile reads a CSV or JSON array file and returns rows as
// []map[string]string. CSV: first row is headers. JSON: array of objects.
func LoadDataFile(path string) ([]map[string]string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("data file: %w", err)
	}
	switch ext {
	case ".json":
		var rows []map[string]any
		if err := json.Unmarshal(data, &rows); err != nil {
			return nil, fmt.Errorf("data file JSON: %w", err)
		}
		out := make([]map[string]string, len(rows))
		for i, row := range rows {
			m := make(map[string]string, len(row))
			for k, v := range row {
				m[k] = fmt.Sprintf("%v", v)
			}
			out[i] = m
		}
		return out, nil
	default: // .csv or anything else
		r := csv.NewReader(strings.NewReader(string(data)))
		records, err := r.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("data file CSV: %w", err)
		}
		if len(records) < 2 {
			return nil, fmt.Errorf("data file CSV: need header row + at least one data row")
		}
		headers := records[0]
		out := make([]map[string]string, 0, len(records)-1)
		for _, rec := range records[1:] {
			m := make(map[string]string, len(headers))
			for j, h := range headers {
				if j < len(rec) {
					m[h] = rec[j]
				}
			}
			out = append(out, m)
		}
		return out, nil
	}
}
