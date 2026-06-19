package store

import (
	"path/filepath"
	"testing"

	"senda/internal/history"
	"senda/internal/model"
)

func TestCollectionActivity(t *testing.T) {
	dir := t.TempDir()

	// Two requests in a subfolder, one untouched request, plus a reserved
	// .secret file that must be ignored.
	mustSave(t, filepath.Join(dir, "api", "get-user.yaml"),
		model.Request{Name: "Get user", Method: "GET", URL: "https://example.com/users/1"})
	mustSave(t, filepath.Join(dir, "api", "create-user.yaml"),
		model.Request{Name: "Create user", Method: "POST", URL: "https://example.com/users"})
	mustSave(t, filepath.Join(dir, "ping.yaml"),
		model.Request{Name: "Ping", Method: "GET", URL: "https://example.com/ping"})

	// History: the GET ran twice (newest must win), the POST errored, ping never ran.
	for _, e := range []model.HistoryEntry{
		{At: "2026-01-01T00:00:00Z", Method: "GET", URL: "https://example.com/users/1", Status: 500},
		{At: "2026-01-02T00:00:00Z", Method: "GET", URL: "https://example.com/users/1", Status: 200},
		{At: "2026-01-02T01:00:00Z", Method: "POST", URL: "https://example.com/users", Status: 0, Error: "boom"},
	} {
		if err := history.Append(dir, e); err != nil {
			t.Fatal(err)
		}
	}

	act, err := CollectionActivity(dir)
	if err != nil {
		t.Fatal(err)
	}

	getPath := filepath.Join(dir, "api", "get-user.yaml")
	if a := act[getPath]; a.At != "2026-01-02T00:00:00Z" || a.Status != 200 || a.Error {
		t.Fatalf("get-user: want newest 200, got %+v", a)
	}
	postPath := filepath.Join(dir, "api", "create-user.yaml")
	if a := act[postPath]; !a.Error || a.Status != 0 {
		t.Fatalf("create-user: want errored entry, got %+v", a)
	}
	if _, ok := act[filepath.Join(dir, "ping.yaml")]; ok {
		t.Fatalf("ping never ran but appeared in activity")
	}
}

func mustSave(t *testing.T, path string, req model.Request) {
	t.Helper()
	if err := SaveRequest(path, req); err != nil {
		t.Fatal(err)
	}
}
