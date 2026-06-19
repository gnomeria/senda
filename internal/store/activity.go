// Activity derives per-request "last run" info from a collection's history
// log so the sidebar can show recency pills on requests and folders.
package store

import (
	"io/fs"
	"path/filepath"
	"strings"

	"senda/internal/history"
)

// Activity is the most recent run of a request, matched out of the history
// log. Folders are not represented here — the UI rolls folder recency up from
// the activity of their descendant requests.
type Activity struct {
	At     string `json:"at"`     // RFC3339 timestamp of the most recent run
	Status int    `json:"status"` // HTTP status of that run (0 when it errored)
	Error  bool   `json:"error"`  // true when the last run failed to complete
}

// histKey normalises a method+URL pair into the key used to match history
// entries against requests. History stores the resolved URL, so requests with
// literal URLs match; environment-templated URLs ({{HOST}}/...) will not.
func histKey(method, url string) string {
	return strings.ToUpper(strings.TrimSpace(method)) + " " + strings.TrimSpace(url)
}

// CollectionActivity maps each request file path to its last-run Activity.
// It reads the history log once (newest-first), indexes it by method+URL, then
// walks the request files and attaches the newest matching entry. A request
// that has never run is simply absent from the map.
func CollectionActivity(collPath string) (map[string]Activity, error) {
	if collPath == "" {
		return map[string]Activity{}, nil
	}

	entries, err := history.List(collPath, 0)
	if err != nil {
		return nil, err
	}
	// First occurrence per key wins: List returns newest-first.
	byKey := make(map[string]Activity, len(entries))
	for _, e := range entries {
		k := histKey(e.Method, e.URL)
		if _, seen := byKey[k]; seen {
			continue
		}
		byKey[k] = Activity{At: e.At, Status: e.Status, Error: e.Error != ""}
	}

	out := make(map[string]Activity)
	walkErr := filepath.WalkDir(collPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries, best-effort
		}
		name := d.Name()
		if d.IsDir() {
			if path != collPath && isReserved(name) {
				return filepath.SkipDir
			}
			return nil
		}
		if isReserved(name) || !isYAMLFile(name) {
			return nil
		}
		req, err := ReadRequest(path)
		if err != nil {
			return nil
		}
		if a, ok := byKey[histKey(req.Method, req.URL)]; ok {
			out[path] = a
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return out, nil
}
