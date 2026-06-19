package store

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"senda/internal/model"
)

// writeRawZip writes a zip containing a single entry with an arbitrary (possibly
// malicious) name, bypassing PackDir's sanitization, for traversal testing.
func writeRawZip(path, name, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(content)); err != nil {
		return err
	}
	return zw.Close()
}

// writeTree lays down a small collection in dir.
func writeTree(t *testing.T, dir string) {
	t.Helper()
	if err := SaveCollection(model.Collection{Path: dir, Name: "demo", Auth: model.Auth{Type: model.AuthNone}}); err != nil {
		t.Fatal(err)
	}
	if err := SaveRequest(filepath.Join(dir, "Folder", "Get.yaml"), model.Request{Method: "GET", URL: "https://example.com"}); err != nil {
		t.Fatal(err)
	}
	if err := SaveEnvironment(dir, model.Environment{Name: "dev", Vars: []model.KV{{Key: "BASE", Value: "x", Enabled: true}}}); err != nil {
		t.Fatal(err)
	}
}

func TestPackDirReproducible(t *testing.T) {
	src := t.TempDir()
	writeTree(t, src)

	z1 := filepath.Join(t.TempDir(), "a.zip")
	z2 := filepath.Join(t.TempDir(), "b.zip")
	if err := PackDir(src, z1); err != nil {
		t.Fatal(err)
	}
	if err := PackDir(src, z2); err != nil {
		t.Fatal(err)
	}
	b1, _ := os.ReadFile(z1)
	b2, _ := os.ReadFile(z2)
	if string(b1) != string(b2) {
		t.Fatalf("PackDir not reproducible: %d vs %d bytes differ", len(b1), len(b2))
	}
}

func TestOpenArchiveRoundTrip(t *testing.T) {
	src := t.TempDir()
	writeTree(t, src)
	zipPath := filepath.Join(t.TempDir(), "coll.zip")
	if err := PackDir(src, zipPath); err != nil {
		t.Fatal(err)
	}

	// OpenCollection on the .zip should transparently extract and parse.
	c, err := OpenCollection(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != "demo" {
		t.Fatalf("name = %q, want demo", c.Name)
	}
	if c.Tree == nil || len(c.Tree.Children) == 0 {
		t.Fatal("expected a non-empty tree from extracted archive")
	}

	// Edit in the live dir, repack, reopen -> edit must survive in the .zip.
	reqPath := filepath.Join(c.Path, "Folder", "Edited.yaml")
	if err := SaveRequest(reqPath, model.Request{Method: "POST", URL: "https://edited"}); err != nil {
		t.Fatal(err)
	}
	if err := PackArchive(c.Path); err != nil {
		t.Fatal(err)
	}

	// Force a fresh extraction by bumping the archive mtime past the marker.
	future := zipEpoch.AddDate(50, 0, 0)
	if err := os.Chtimes(zipPath, future, future); err != nil {
		t.Fatal(err)
	}
	c2, err := OpenCollection(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(c2.Path, "Folder", "Edited.yaml")); err != nil {
		t.Fatalf("edit did not survive repack/reopen: %v", err)
	}
}

func TestUnzipRejectsTraversal(t *testing.T) {
	// A crafted archive with a ../ entry must be refused.
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.zip")
	if err := writeRawZip(bad, "../escape.txt", "pwned"); err != nil {
		t.Fatal(err)
	}
	if err := unzip(bad, filepath.Join(dir, "out")); err == nil {
		t.Fatal("expected unzip to reject path traversal entry")
	}
}
