package main

import (
	"strings"
	"testing"

	"senda/internal/model"
)

func TestRenderReport(t *testing.T) {
	results := []model.RunResult{
		{Name: "login", Path: "auth/login.yaml", Method: "POST", URL: "http://x/login", Status: 200, DurationMs: 120, OK: true, AssertPass: 2},
		{Name: "me", Path: "auth/me.yaml", Method: "GET", URL: "http://x/me", Status: 500, DurationMs: 50, OK: false, AssertFail: 1, Error: "boom"},
	}

	js, err := renderReport("json", results)
	if err != nil || !strings.Contains(string(js), `"name": "login"`) {
		t.Fatalf("json report bad: %v\n%s", err, js)
	}

	xml, err := renderReport("junit", results)
	if err != nil {
		t.Fatal(err)
	}
	s := string(xml)
	if !strings.Contains(s, `tests="2"`) || !strings.Contains(s, `failures="1"`) {
		t.Fatalf("junit counts wrong:\n%s", s)
	}
	if !strings.Contains(s, `<failure message="boom"`) {
		t.Fatalf("junit missing failure:\n%s", s)
	}
	if strings.Count(s, "<failure") != 1 {
		t.Fatalf("expected 1 failure element:\n%s", s)
	}

	if _, err := renderReport("pdf", results); err == nil {
		t.Fatal("expected error for unknown format")
	}
}
