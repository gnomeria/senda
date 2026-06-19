package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"senda/internal/model"
	"senda/internal/store"
)

// writeMeta writes a senda.meta.yaml into dir from the given Collection.
func writeMeta(t *testing.T, dir string, c model.Collection) {
	t.Helper()
	os.MkdirAll(dir, 0o755)
	c.Path = dir
	if err := store.SaveCollection(c); err != nil {
		t.Fatal(err)
	}
}

// TestScopeFolderVarsOverrideCollection verifies the resolution chain:
// collection vars are overridden by folder vars, deeper folders win over
// shallower ones, and a request directly in the root sees only collection vars.
func TestScopeFolderVarsOverrideCollection(t *testing.T) {
	root := t.TempDir()
	writeMeta(t, root, model.Collection{Vars: []model.KV{
		{Key: "host", Value: "root", Enabled: true},
		{Key: "shared", Value: "from-root", Enabled: true},
	}})
	a := filepath.Join(root, "a")
	writeMeta(t, a, model.Collection{Vars: []model.KV{{Key: "host", Value: "a", Enabled: true}}})
	b := filepath.Join(a, "b")
	writeMeta(t, b, model.Collection{Vars: []model.KV{{Key: "host", Value: "b", Enabled: true}}})

	s := NewSession()
	scope := s.Scope(root, filepath.Join(b, "req.yaml"), "")
	if got, _ := scope.Get("host"); got != "b" {
		t.Errorf("deepest folder var should win, got %q", got)
	}
	if got, _ := scope.Get("shared"); got != "from-root" {
		t.Errorf("collection var should remain when not overridden, got %q", got)
	}

	// request in folder a sees a's override, not b's.
	scope = s.Scope(root, filepath.Join(a, "req.yaml"), "")
	if got, _ := scope.Get("host"); got != "a" {
		t.Errorf("folder a var should win, got %q", got)
	}

	// request in root sees only collection vars.
	scope = s.Scope(root, filepath.Join(root, "req.yaml"), "")
	if got, _ := scope.Get("host"); got != "root" {
		t.Errorf("root request should see collection var, got %q", got)
	}
}

// TestEffectiveAuthFolderInheritance verifies inherit walks request -> nearest
// folder with concrete auth -> collection root.
func TestEffectiveAuthFolderInheritance(t *testing.T) {
	root := t.TempDir()
	writeMeta(t, root, model.Collection{Auth: model.Auth{Type: model.AuthBearer, Token: "root-tok"}})
	a := filepath.Join(root, "a")
	writeMeta(t, a, model.Collection{Auth: model.Auth{Type: model.AuthBasic, Username: "u", Password: "p"}})
	b := filepath.Join(a, "b") // no auth -> inherits from a
	os.MkdirAll(b, 0o755)

	// explicit request auth always wins.
	got := effectiveAuth(root, filepath.Join(b, "req.yaml"), model.Auth{Type: model.AuthAPIKey, Key: "X"})
	if got.Type != model.AuthAPIKey {
		t.Errorf("explicit request auth should win, got %+v", got)
	}

	// inherit from nearest folder (a) since b has none.
	got = effectiveAuth(root, filepath.Join(b, "req.yaml"), model.Auth{Type: model.AuthInherit})
	if got.Type != model.AuthBasic || got.Username != "u" {
		t.Errorf("should inherit folder a's auth, got %+v", got)
	}

	// inherit straight to collection root for a root-level request.
	got = effectiveAuth(root, filepath.Join(root, "req.yaml"), model.Auth{Type: model.AuthInherit})
	if got.Type != model.AuthBearer || got.Token != "root-tok" {
		t.Errorf("should inherit collection auth, got %+v", got)
	}

	// no collection and no folders -> none.
	if got = effectiveAuth("", "", model.Auth{Type: model.AuthInherit}); got.Type != model.AuthNone {
		t.Errorf("want AuthNone, got %+v", got)
	}
}
