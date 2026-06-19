package security

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"gopkg.in/yaml.v3"

	"senda/internal/store"
)

// TemplatesSubdir is the folder under a collection's .senda/security directory
// where synced template repos are checked out. User-authored templates live
// directly in .senda/security/ and are never touched by syncing.
const TemplatesSubdir = "templates"

// syncStateFile records the last sync, next to the checkout.
const syncStateFile = ".senda-sync.yaml"

// SyncState is the persisted record of a template-repo sync, surfaced to the
// UI so the scan modal can show the source and freshness.
type SyncState struct {
	URL       string `json:"url" yaml:"url"`
	Ref       string `json:"ref,omitempty" yaml:"ref,omitempty"` // branch/tag; "" = default
	Commit    string `json:"commit,omitempty" yaml:"commit,omitempty"`
	SyncedAt  string `json:"syncedAt,omitempty" yaml:"syncedAt,omitempty"` // RFC3339
	Templates int    `json:"templates" yaml:"templates"`                   // supported templates found
}

// templatesDir is <collection>/.senda/security/templates.
func templatesDir(collPath string) string {
	return filepath.Join(store.SecurityDir(collPath), TemplatesSubdir)
}

// ReadSyncState loads the last sync record for a collection, or a zero value
// (with no error) when nothing has been synced yet.
func ReadSyncState(collPath string) (SyncState, error) {
	var st SyncState
	data, err := os.ReadFile(filepath.Join(templatesDir(collPath), syncStateFile))
	if err != nil {
		if os.IsNotExist(err) {
			return st, nil
		}
		return st, err
	}
	err = yaml.Unmarshal(data, &st)
	return st, err
}

// SyncTemplates clones (or, if already present for the same URL, pulls) the
// template repo at url into <collection>/.senda/security/templates and records the
// result. ref optionally pins a branch or tag. The checkout is shallow
// (depth 1) since history is not needed. Returns the new state.
//
// Only the supported nuclei http-template subset is counted/loaded at scan
// time; the repo may contain anything.
func SyncTemplates(ctx context.Context, collPath, url, ref string) (SyncState, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return SyncState{}, fmt.Errorf("template repo URL is empty")
	}
	dir := templatesDir(collPath)

	// If a checkout exists for a different URL, wipe it so we don't merge
	// unrelated repos.
	if prev, _ := ReadSyncState(collPath); prev.URL != "" && prev.URL != url {
		_ = os.RemoveAll(dir)
	}

	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return SyncState{}, err
	}

	commit, err := cloneOrPull(ctx, dir, url, ref)
	if err != nil {
		return SyncState{}, err
	}

	st := SyncState{
		URL:       url,
		Ref:       ref,
		Commit:    commit,
		SyncedAt:  time.Now().UTC().Format(time.RFC3339),
		Templates: len(LoadDir(os.DirFS(dir), ".")),
	}
	if err := writeSyncState(dir, st); err != nil {
		return st, err
	}
	return st, nil
}

// cloneOrPull clones url into dir, or fast-forwards an existing checkout.
// Returns the resulting HEAD commit hash.
func cloneOrPull(ctx context.Context, dir, url, ref string) (string, error) {
	repo, err := git.PlainOpen(dir)
	if err == git.ErrRepositoryNotExists {
		opts := &git.CloneOptions{URL: url, Depth: 1, SingleBranch: true}
		if ref != "" {
			opts.ReferenceName = plumbing.NewBranchReferenceName(ref)
		}
		repo, err = git.PlainCloneContext(ctx, dir, false, opts)
		if err != nil {
			// retry treating ref as a tag rather than a branch
			if ref != "" {
				opts.ReferenceName = plumbing.NewTagReferenceName(ref)
				repo, err = git.PlainCloneContext(ctx, dir, false, opts)
			}
			if err != nil {
				return "", fmt.Errorf("clone failed: %w", err)
			}
		}
	} else if err != nil {
		return "", err
	} else {
		wt, err := repo.Worktree()
		if err != nil {
			return "", err
		}
		err = wt.PullContext(ctx, &git.PullOptions{Depth: 1, SingleBranch: true, Force: true})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return "", fmt.Errorf("pull failed: %w", err)
		}
	}
	head, err := repo.Head()
	if err != nil {
		return "", err
	}
	return head.Hash().String(), nil
}

func writeSyncState(dir string, st SyncState) error {
	data, err := yaml.Marshal(st)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, syncStateFile), data, 0o644)
}
