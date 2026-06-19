package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"senda/internal/model"
)

func req(url string) model.Request { return model.Request{Method: "GET", URL: url} }

func TestTargetsDedupesAndResolves(t *testing.T) {
	reqs := []model.Request{
		req("{{base}}/users"),
		req("{{base}}/users"),          // duplicate after resolve
		req("https://api.example.com"), // literal
		req(""),                        // empty -> skipped
		req("{{missing}}/orders"),      // unresolved -> skipped
		req("api.other.com/health"),    // no scheme -> https assumed
	}
	resolve := func(s string) string {
		return strings.ReplaceAll(s, "{{base}}", "https://api.example.com")
	}
	got := Targets(reqs, resolve)
	want := []string{
		"https://api.example.com/users",
		"https://api.example.com",
		"https://api.other.com/health",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Targets = %v, want %v", got, want)
	}
}

func TestBuiltinTemplatesParse(t *testing.T) {
	ts := Builtin()
	if len(ts) < 10 {
		t.Fatalf("expected the full builtin pack, got %d templates", len(ts))
	}
	for _, tpl := range ts {
		if tpl.Info.Name == "" || tpl.Info.Severity == "" {
			t.Errorf("%s: missing name or severity", tpl.ID)
		}
		if tpl.Info.Metadata["owasp"] == "" {
			t.Errorf("%s: missing OWASP id in metadata", tpl.ID)
		}
		if len(tpl.Info.Classification.CWEID) == 0 {
			t.Errorf("%s: missing CWE id", tpl.ID)
		}
	}
}

func TestParseTemplateRejectsUnsupported(t *testing.T) {
	cases := map[string]string{
		"dsl matcher": `
id: t
http:
  - method: GET
    path: ["{{BaseURL}}"]
    matchers:
      - type: dsl
        dsl: ["len(body) > 0"]`,
		"payloads": `
id: t
http:
  - method: GET
    path: ["{{BaseURL}}/{{p}}"]
    payloads:
      p: [a, b]
    matchers:
      - type: status
        status: [200]`,
		"non-http": `
id: t
dns:
  - name: "{{FQDN}}"`,
	}
	for name, src := range cases {
		if _, err := ParseTemplate([]byte(src)); err == nil {
			t.Errorf("%s: expected parse error", name)
		}
	}
}

func TestFilterSeverityTagsAndOverride(t *testing.T) {
	mk := func(id, sev string, tags ...string) *Template {
		tpl := &Template{ID: id}
		tpl.Info.Severity = sev
		tpl.Info.Tags = tags
		return tpl
	}
	base := mk("a", "high", "owasp")
	override := mk("a", "high", "owasp", "custom")
	other := mk("b", "info", "headers")

	got := Filter([]*Template{base, other, override}, "high", nil)
	if len(got) != 1 || got[0] != override {
		t.Fatalf("expected the overriding high template only, got %d", len(got))
	}
	got = Filter([]*Template{base, other}, "", []string{"headers"})
	if len(got) != 1 || got[0].ID != "b" {
		t.Fatalf("tag filter failed: %+v", got)
	}
}

// vulnerable test server: leaks .env, echoes permissive CORS, no security
// headers, discloses X-Powered-By.
func vulnServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/.env", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("APP_KEY=secret\nDB_PASSWORD=hunter2\n"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("X-Powered-By", "Express")
		w.Write([]byte("ok"))
	})
	return httptest.NewServer(mux)
}

func TestRunFindsKnownIssues(t *testing.T) {
	srv := vulnServer()
	defer srv.Close()

	var matched, passed []string
	sum, err := Run(context.Background(), []string{srv.URL}, "", model.SecurityOptions{RateLimit: 1000}, func(c model.SecurityCheck) {
		if c.Matched {
			matched = append(matched, c.TemplateID)
		} else if c.Error == "" {
			passed = append(passed, c.TemplateID)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(matched)
	for _, want := range []string{"exposed-env-file", "missing-security-headers", "permissive-cors", "x-powered-by-disclosure"} {
		if !contains(matched, want) {
			t.Errorf("missing expected finding %s; got %v", want, matched)
		}
	}
	// clean endpoints must surface as passed checks, not findings
	for _, want := range []string{"git-config-exposure", "exposed-sql-dump"} {
		if contains(matched, want) {
			t.Errorf("false positive %s in %v", want, matched)
		}
		if !contains(passed, want) {
			t.Errorf("expected %s in passed checks, got %v", want, passed)
		}
	}
	if sum.Findings != len(matched) || sum.Passed != len(passed) {
		t.Errorf("summary findings/passed %d/%d != events %d/%d", sum.Findings, sum.Passed, len(matched), len(passed))
	}
	if sum.Checks != sum.Findings+sum.Passed+sum.Errors {
		t.Errorf("checks %d != findings+passed+errors %d+%d+%d", sum.Checks, sum.Findings, sum.Passed, sum.Errors)
	}
	if sum.BySeverity["high"] == 0 {
		t.Errorf("expected a high severity finding (.env), got %v", sum.BySeverity)
	}
}

func TestRunReportsUnreachableAsError(t *testing.T) {
	var got []model.SecurityCheck
	sum, err := Run(context.Background(), []string{"http://127.0.0.1:1"}, "", model.SecurityOptions{RateLimit: 1000, Timeout: 1}, func(c model.SecurityCheck) {
		got = append(got, c)
	})
	if err != nil {
		t.Fatal(err)
	}
	if sum.Errors == 0 || sum.Errors != len(got) || sum.Passed != 0 || sum.Findings != 0 {
		t.Fatalf("unreachable target should error every check: %+v", sum)
	}
	for _, c := range got {
		if c.Error == "" || c.Matched {
			t.Fatalf("expected errored check, got %+v", c)
		}
	}
}

func TestRunSeverityFilter(t *testing.T) {
	srv := vulnServer()
	defer srv.Close()

	var got []model.SecurityCheck
	_, err := Run(context.Background(), []string{srv.URL}, "", model.SecurityOptions{Severity: "high,critical", RateLimit: 1000}, func(c model.SecurityCheck) {
		got = append(got, c)
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range got {
		if c.Severity != "high" && c.Severity != "critical" {
			t.Errorf("severity filter leaked %s (%s)", c.TemplateID, c.Severity)
		}
		found = found || c.Matched
	}
	if !found {
		t.Fatal("expected the .env high finding")
	}
}

func TestExtraDirOverridesBuiltin(t *testing.T) {
	srv := vulnServer()
	defer srv.Close()

	dir := t.TempDir()
	custom := `
id: x-powered-by-disclosure
info:
  name: Custom Override
  severity: low
http:
  - method: GET
    path: ["{{BaseURL}}"]
    matchers:
      - type: word
        part: header
        case-insensitive: true
        words: ["X-Powered-By:"]
`
	if err := writeFile(dir+"/custom.yaml", custom); err != nil {
		t.Fatal(err)
	}
	var got []model.SecurityCheck
	_, err := Run(context.Background(), []string{srv.URL}, dir, model.SecurityOptions{Tags: nil, RateLimit: 1000}, func(c model.SecurityCheck) {
		if c.TemplateID == "x-powered-by-disclosure" && c.Matched {
			got = append(got, c)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "Custom Override" || got[0].Severity != "low" {
		t.Fatalf("override not applied: %+v", got)
	}
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
