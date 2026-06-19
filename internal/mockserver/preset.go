package mockserver

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed presets
var presetFS embed.FS

// Presets lists the names of the bundled mock presets (subdirectories of
// presets/), sorted.
func Presets() []string {
	entries, err := presetFS.ReadDir("presets")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

// WritePreset copies the YAML files of the named preset into mocksDir, creating
// the directory if needed. It returns the relative names of the files written.
// Existing files are left untouched and reported as skipped via skipped; an
// error is returned only for I/O failures or an unknown preset.
func WritePreset(name, mocksDir string) (written, skipped []string, err error) {
	root := "presets/" + name
	if _, statErr := fs.Stat(presetFS, root); statErr != nil {
		return nil, nil, fmt.Errorf("unknown preset %q (have: %s)", name, strings.Join(Presets(), ", "))
	}
	if err := os.MkdirAll(mocksDir, 0o755); err != nil {
		return nil, nil, err
	}
	entries, err := presetFS.ReadDir(root)
	if err != nil {
		return nil, nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !isYAML(e.Name()) {
			continue
		}
		dst := filepath.Join(mocksDir, e.Name())
		if _, statErr := os.Stat(dst); statErr == nil {
			skipped = append(skipped, e.Name())
			continue
		}
		data, readErr := presetFS.ReadFile(root + "/" + e.Name())
		if readErr != nil {
			return written, skipped, readErr
		}
		if writeErr := os.WriteFile(dst, data, 0o644); writeErr != nil {
			return written, skipped, writeErr
		}
		written = append(written, e.Name())
	}
	return written, skipped, nil
}
