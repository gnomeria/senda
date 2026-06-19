package store

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("name: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestListRequestsSkipsReserved(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.yaml"))
	writeFile(t, filepath.Join(root, "sub", "b.yaml"))
	writeFile(t, filepath.Join(root, "senda.meta.yaml"))
	writeFile(t, filepath.Join(root, "environments", "dev.yaml"))
	writeFile(t, filepath.Join(root, ".senda", "history.jsonl"))

	got, err := ListRequests(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d requests, want 2: %v", len(got), got)
	}
}

func TestRenameNodeFile(t *testing.T) {
	root := t.TempDir()
	old := filepath.Join(root, "old.yaml")
	writeFile(t, old)
	dest, err := RenameNode(old, "new")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dest) != "new.yaml" {
		t.Errorf("dest = %q, want new.yaml", dest)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("renamed file missing: %v", err)
	}
}

func TestRenameNodeRejectsExisting(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.yaml"))
	writeFile(t, filepath.Join(root, "b.yaml"))
	if _, err := RenameNode(filepath.Join(root, "a.yaml"), "b"); err == nil {
		t.Error("want error renaming onto existing file")
	}
}

func TestRenameNodeRejectsSeparators(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.yaml"))
	if _, err := RenameNode(filepath.Join(root, "a.yaml"), "x/y"); err == nil {
		t.Error("want error for name with separator")
	}
}

func TestMoveNode(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "a.yaml")
	writeFile(t, src)
	destDir := filepath.Join(root, "folder")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dest, err := MoveNode(src, destDir)
	if err != nil {
		t.Fatal(err)
	}
	if dest != filepath.Join(destDir, "a.yaml") {
		t.Errorf("dest = %q", dest)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("moved file missing: %v", err)
	}
}

func TestMoveNodeRejectsIntoSelf(t *testing.T) {
	root := t.TempDir()
	folder := filepath.Join(root, "f")
	sub := filepath.Join(folder, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := MoveNode(folder, sub); err == nil {
		t.Error("want error moving folder into its own subtree")
	}
}
