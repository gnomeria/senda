// Package security runs template-based security checks against the URLs of a
// collection's requests. The engine executes the HTTP subset of the nuclei
// template format (see template.go) with zero scanner dependencies: a curated
// builtin check pack ships embedded, and extra nuclei-compatible http
// templates load from a directory (e.g. <collection>/.security).
//
// Scans send real probe traffic to the targets. Callers must only scan APIs
// they own or are authorized to test.
package security

import (
	"context"
	"embed"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"senda/internal/model"
)

//go:embed templates/*.yaml
var builtinFS embed.FS

const (
	defaultRateLimit   = 50 // requests per second
	defaultTimeout     = 10 // seconds per request
	templateConcurrent = 10 // (target, template) pairs in flight
)

// Builtin returns the embedded check pack.
func Builtin() []*Template {
	return LoadDir(builtinFS, "templates")
}

// A synced nuclei-templates checkout can hold 10k+ yaml files, and parsing them
// all takes seconds — far too slow to repeat on every scan-plan preview as the
// user tweaks scope/severity/tags. Cache the parsed set per directory, keyed by
// a cheap fingerprint (mod time + supported-template count from the sync record)
// so a fresh sync invalidates it but repeated reads are instant.
var (
	dirCacheMu sync.Mutex
	dirCache   = map[string]dirCacheEntry{}
)

type dirCacheEntry struct {
	fingerprint string
	templates   []*Template
}

func dirFingerprint(dir string) string {
	var fp strings.Builder
	if fi, err := os.Stat(dir); err == nil {
		fp.WriteString(fi.ModTime().UTC().Format(time.RFC3339Nano))
	}
	// A synced checkout lives in dir/templates; git rewrites it on pull and the
	// sync record's commit/time change with it, so fold both in. The checkout
	// dir's own mod time alone can miss a pull that only touches nested files.
	sub := filepath.Join(dir, TemplatesSubdir)
	if fi, err := os.Stat(sub); err == nil {
		fp.WriteString("|" + fi.ModTime().UTC().Format(time.RFC3339Nano))
	}
	if data, err := os.ReadFile(filepath.Join(sub, syncStateFile)); err == nil {
		fp.Write(data)
	}
	return fp.String()
}

// loadDirCached parses every supported template under dir, reusing the last
// parse when the directory is unchanged since.
func loadDirCached(dir string) []*Template {
	fp := dirFingerprint(dir)
	dirCacheMu.Lock()
	defer dirCacheMu.Unlock()
	if e, ok := dirCache[dir]; ok && e.fingerprint == fp {
		return e.templates
	}
	t := LoadDir(os.DirFS(dir), ".")
	dirCache[dir] = dirCacheEntry{fingerprint: fp, templates: t}
	return t
}

// selectTemplates is the template set a scan would run: the builtin pack plus
// any templates under extraDir (skipped when empty or missing), narrowed by the
// severity/tags filter. Pass extraDir == "" to scan with the builtin pack only.
func selectTemplates(extraDir string, opts model.SecurityOptions) []*Template {
	templates := Builtin()
	if extraDir != "" {
		if _, err := os.Stat(extraDir); err == nil {
			templates = append(templates, loadDirCached(extraDir)...)
		}
	}
	return Filter(templates, opts.Severity, opts.Tags)
}

// CountTemplates returns how many templates a scan would execute per target for
// the given options — used to preview the run size without sending traffic.
func CountTemplates(extraDir string, opts model.SecurityOptions) int {
	return len(selectTemplates(extraDir, opts))
}

// Targets derives the unique scan targets from a set of requests. Each URL is
// interpolated with resolve (the collection/env variable scope); URLs that are
// empty or still contain unresolved {{vars}} after interpolation are skipped.
// Order follows first appearance.
func Targets(reqs []model.Request, resolve func(string) string) []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range reqs {
		u := strings.TrimSpace(resolve(r.URL))
		if u == "" || strings.Contains(u, "{{") {
			continue
		}
		if !strings.Contains(u, "://") {
			u = "https://" + u
		}
		if seen[u] {
			continue
		}
		seen[u] = true
		out = append(out, u)
	}
	return out
}

// Run scans targets with the builtin check pack plus any templates under
// extraDir (ignored when empty or missing; user templates with a builtin's id
// override it). opts filters by severity/tags and tunes rate limit and
// timeout. onCheck fires for every executed template×target pair — matched,
// passed or errored — as it completes; the returned summary aggregates the
// run.
func Run(ctx context.Context, targets []string, extraDir string, opts model.SecurityOptions, onCheck func(model.SecurityCheck)) (model.SecuritySummary, error) {
	start := time.Now()
	sum := model.SecuritySummary{
		Targets:    len(targets),
		BySeverity: map[string]int{},
	}

	templates := selectTemplates(extraDir, opts)
	if len(targets) == 0 || len(templates) == 0 {
		sum.Duration = time.Since(start).Seconds()
		return sum, nil
	}

	rate := opts.RateLimit
	if rate <= 0 {
		rate = defaultRateLimit
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	eng := newEngine(time.Duration(timeout)*time.Second, rate)

	type job struct {
		t      *Template
		target string
	}
	jobs := make(chan job)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range templateConcurrent {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				matched, err := eng.run(ctx, j.t, j.target)
				if ctx.Err() != nil {
					continue // canceled mid-run: don't report partial checks
				}
				c := check(j.t, j.target)
				mu.Lock()
				sum.Checks++
				switch {
				case err != nil:
					c.Error = err.Error()
					sum.Errors++
				case matched != "":
					c.Matched = true
					c.MatchedAt = matched
					sum.Findings++
					sum.BySeverity[c.Severity]++
				default:
					sum.Passed++
				}
				mu.Unlock()
				if onCheck != nil {
					onCheck(c)
				}
			}
		}()
	}

feed:
	for _, target := range targets {
		for _, t := range templates {
			select {
			case <-ctx.Done():
				break feed
			case jobs <- job{t: t, target: target}:
			}
		}
	}
	close(jobs)
	wg.Wait()

	sum.Duration = time.Since(start).Seconds()
	return sum, ctx.Err()
}
