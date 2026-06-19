package security

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"senda/internal/store"
)

// makeRepo builds a throwaway git repo containing one valid template and a
// junk file, and returns its path. Uses go-git so it needs no external git.
func makeRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := "id: remote-check\n" +
		"info:\n  name: Remote Check\n  severity: low\n" +
		"http:\n  - method: GET\n    path: [\"{{BaseURL}}\"]\n" +
		"    matchers:\n      - type: status\n        status: [200]\n"
	if err := os.WriteFile(filepath.Join(dir, "check.yaml"), []byte(tmpl), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("junk"), 0o644); err != nil {
		t.Fatal(err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Add("."); err != nil {
		t.Fatal(err)
	}
	_, err = wt.Commit("init", &git.CommitOptions{
		Author: &object.Signature{Name: "t", Email: "t@t.t", When: time.Now()},
	})
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestSyncTemplatesClonePullState(t *testing.T) {
	src := makeRepo(t)
	coll := t.TempDir()

	st, err := SyncTemplates(context.Background(), coll, src, "")
	if err != nil {
		t.Fatalf("clone: %v", err)
	}
	if st.URL != src || st.Commit == "" || st.SyncedAt == "" {
		t.Fatalf("state not populated: %+v", st)
	}
	if st.Templates < 1 {
		t.Fatalf("expected >=1 supported template, got %d", st.Templates)
	}

	// Checkout landed under .senda/security/templates and is loadable.
	dir := filepath.Join(store.SecurityDir(coll), TemplatesSubdir)
	if len(LoadDir(os.DirFS(dir), ".")) < 1 {
		t.Fatal("synced templates not loadable from checkout")
	}

	// State round-trips.
	got, err := ReadSyncState(coll)
	if err != nil || got.Commit != st.Commit {
		t.Fatalf("ReadSyncState mismatch: %+v err=%v", got, err)
	}

	// Second sync (pull path) succeeds and stays consistent.
	st2, err := SyncTemplates(context.Background(), coll, src, "")
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if st2.Commit != st.Commit {
		t.Fatalf("commit changed without upstream change: %s -> %s", st.Commit, st2.Commit)
	}
}

func TestSyncTemplatesEmptyURL(t *testing.T) {
	if _, err := SyncTemplates(context.Background(), t.TempDir(), "  ", ""); err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestReadSyncStateMissing(t *testing.T) {
	st, err := ReadSyncState(t.TempDir())
	if err != nil || st.URL != "" {
		t.Fatalf("expected zero state, no error; got %+v err=%v", st, err)
	}
}
