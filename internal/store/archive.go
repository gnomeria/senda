// Archive support: a collection may live as a single .zip file instead of a
// directory tree. Zip is used because DEFLATE (RFC 1951) is the most portable
// compression on the planet and Go ships archive/zip in stdlib; .zip also opens
// natively on Windows/macOS/Linux without extra tools.
//
// Opening an archive is transparent: it is extracted to a per-archive cache dir
// and the rest of store/ operates on that real directory. Edits are written to
// the cache dir and only folded back into the .zip when PackArchive is called
// (on explicit save or app close), so the committed archive stays a single,
// reproducible binary blob — small git diffs instead of hundreds of files.
package store

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ArchiveExt is the canonical suffix for a packed Senda collection. The doubled
// extension namespaces the file as ours (.senda) while keeping the .zip tail so
// it still opens with every OS's native unzip/preview tooling.
const ArchiveExt = ".senda.zip"

// archiveExts are the suffixes treated as a packed collection, longest first so
// the canonical .senda.zip is stripped before the bare .zip when deriving names.
var archiveExts = []string{ArchiveExt, ".zip"}

// zipEpoch is a fixed timestamp written into every zip entry so identical
// content always produces byte-identical archives (no embedded mtimes ->
// reproducible commits). Zip cannot encode dates before 1980-01-01.
var zipEpoch = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

// markerFile records which archive a cache dir was extracted from, so PackArchive
// knows the destination and OpenArchive can detect a stale extraction.
const markerFile = ".senda-archive.json"

type archiveMarker struct {
	Source  string `json:"source"`   // absolute path of the .zip
	ModUnix int64  `json:"mod_unix"` // source mtime at extraction time
	Size    int64  `json:"size"`     // source size at extraction time
}

var (
	archiveMu  sync.Mutex
	archiveSrc = map[string]string{} // liveDir(abs) -> archive(abs)
)

// IsArchive reports whether path points at a packed collection (by extension).
func IsArchive(path string) bool {
	lower := strings.ToLower(path)
	for _, ext := range archiveExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// archiveCacheDir returns the stable extraction directory for an archive,
// keyed by its absolute path so the same archive always maps to the same dir.
func archiveCacheDir(absArchive string) (string, error) {
	base := filepath.Base(absArchive)
	for _, ext := range archiveExts {
		base = strings.TrimSuffix(base, ext)
		base = strings.TrimSuffix(base, strings.ToUpper(ext))
	}
	sum := sha256.Sum256([]byte(absArchive))
	cache, err := os.UserCacheDir()
	if err != nil {
		cache = os.TempDir()
	}
	return filepath.Join(cache, "senda", "collections", base+"-"+hex.EncodeToString(sum[:])[:8]), nil
}

// OpenArchive extracts archivePath into its cache dir (only when missing or
// stale) and returns that live directory. The mapping is remembered so a later
// PackArchive(liveDir) folds edits back into the archive.
func OpenArchive(archivePath string) (string, error) {
	abs, err := filepath.Abs(archivePath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	live, err := archiveCacheDir(abs)
	if err != nil {
		return "", err
	}

	if !cacheFresh(live, abs, info) {
		if err := os.RemoveAll(live); err != nil {
			return "", err
		}
		if err := unzip(abs, live); err != nil {
			return "", err
		}
		if err := writeMarker(live, abs, info); err != nil {
			return "", err
		}
	}

	archiveMu.Lock()
	archiveSrc[live] = abs
	archiveMu.Unlock()
	return live, nil
}

// cacheFresh reports whether an existing extraction matches the current archive
// (same source path, mtime and size). A stale or missing cache returns false so
// callers re-extract; a fresh one is reused, preserving unsaved edits.
func cacheFresh(live, absArchive string, info os.FileInfo) bool {
	data, err := os.ReadFile(filepath.Join(live, markerFile))
	if err != nil {
		return false
	}
	var m archiveMarker
	if err := json.Unmarshal(data, &m); err != nil {
		return false
	}
	return m.Source == absArchive && m.ModUnix == info.ModTime().Unix() && m.Size == info.Size()
}

func writeMarker(live, absArchive string, info os.FileInfo) error {
	data, err := json.Marshal(archiveMarker{Source: absArchive, ModUnix: info.ModTime().Unix(), Size: info.Size()})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(live, markerFile), data, 0o644)
}

// PackArchive repacks an archive-backed live directory into its source .zip.
// It is a no-op (nil) for directories that were not opened from an archive, so
// callers can invoke it unconditionally. The archive's mtime/size marker is
// refreshed so the cache stays fresh across the rewrite.
func PackArchive(liveDir string) error {
	abs, err := filepath.Abs(liveDir)
	if err != nil {
		return err
	}
	archiveMu.Lock()
	dest, ok := archiveSrc[abs]
	archiveMu.Unlock()
	if !ok {
		return nil
	}
	if err := PackDir(abs, dest); err != nil {
		return err
	}
	if info, err := os.Stat(dest); err == nil {
		_ = writeMarker(abs, dest, info)
	}
	return nil
}

// PackOpen repacks every archive opened in this process. Intended for app
// shutdown so edits made during the session are persisted.
func PackOpen() error {
	archiveMu.Lock()
	dirs := make([]string, 0, len(archiveSrc))
	for d := range archiveSrc {
		dirs = append(dirs, d)
	}
	archiveMu.Unlock()
	sort.Strings(dirs)
	for _, d := range dirs {
		if err := PackArchive(d); err != nil {
			return err
		}
	}
	return nil
}

// PackDir writes srcDir into a reproducible .zip at zipPath. Entries are stored
// sorted with a fixed timestamp, so unchanged content yields byte-identical
// archives. The write is atomic (temp file + rename) and the internal marker
// file is excluded.
func PackDir(srcDir, zipPath string) error {
	var files []string
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if info.Name() == markerFile {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(files)

	if err := os.MkdirAll(filepath.Dir(zipPath), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(zipPath), ".senda-zip-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after successful rename

	zw := zip.NewWriter(tmp)
	for _, rel := range files {
		hdr := &zip.FileHeader{Name: rel, Method: zip.Deflate, Modified: zipEpoch}
		hdr.SetMode(0o644)
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			_ = tmp.Close()
			return err
		}
		data, err := os.ReadFile(filepath.Join(srcDir, filepath.FromSlash(rel)))
		if err != nil {
			_ = tmp.Close()
			return err
		}
		if _, err := w.Write(data); err != nil {
			_ = tmp.Close()
			return err
		}
	}
	if err := zw.Close(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, zipPath); err != nil {
		return err
	}
	return os.Chmod(zipPath, 0o644)
}

// unzip extracts a .zip into dest, guarding against path traversal (Zip Slip).
func unzip(zipPath, dest string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	destAbs, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	for _, f := range r.File {
		target := filepath.Join(dest, filepath.FromSlash(f.Name))
		targetAbs, err := filepath.Abs(target)
		if err != nil {
			return err
		}
		if targetAbs != destAbs && !strings.HasPrefix(targetAbs, destAbs+string(os.PathSeparator)) {
			return fmt.Errorf("unsafe path in archive: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := writeZipEntry(f, target); err != nil {
			return err
		}
	}
	return nil
}

func writeZipEntry(f *zip.File, target string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, rc) //nolint:gosec // entries validated against Zip Slip above
	return err
}
