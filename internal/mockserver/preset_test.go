package mockserver

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestPresetsIncludesOAuth(t *testing.T) {
	if !slices.Contains(Presets(), "oauth") {
		t.Fatalf("expected oauth in presets, got %v", Presets())
	}
}

func TestWritePresetOAuth(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "mocks")

	written, skipped, err := WritePreset("oauth", dir)
	if err != nil {
		t.Fatalf("WritePreset: %v", err)
	}
	if len(skipped) != 0 {
		t.Fatalf("expected nothing skipped on fresh dir, got %v", skipped)
	}
	want := []string{"authorize.yaml", "openid-config.yaml", "token.yaml", "userinfo.yaml"}
	for _, f := range want {
		if !slices.Contains(written, f) {
			t.Errorf("missing %s in written %v", f, written)
		}
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("file not on disk: %s", f)
		}
	}

	// The written files must form a loadable, valid mock set.
	l, err := loadDir(dir)
	if err != nil {
		t.Fatalf("loadDir on preset: %v", err)
	}
	if len(l.rules) != len(want) {
		t.Fatalf("expected %d rule routes, got %d", len(want), len(l.rules))
	}

	// A second write leaves existing files alone.
	written2, skipped2, err := WritePreset("oauth", dir)
	if err != nil {
		t.Fatalf("WritePreset (rerun): %v", err)
	}
	if len(written2) != 0 {
		t.Errorf("expected no new writes on rerun, got %v", written2)
	}
	if len(skipped2) != len(want) {
		t.Errorf("expected all files skipped on rerun, got %v", skipped2)
	}
}

func TestWritePresetUnknown(t *testing.T) {
	if _, _, err := WritePreset("nope", t.TempDir()); err == nil {
		t.Fatal("expected error for unknown preset")
	}
}
