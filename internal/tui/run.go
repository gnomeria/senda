// Package tui is the interactive terminal client for senda collections — the
// same pipeline as the desktop app and the headless runner (scripts, vars,
// secrets, asserts, cookie jar), driven from a three-pane TUI: collection tree |
// request | response. The request and response panes carry ATAC-style tab bars
// (Params/Headers/Body/Auth/Asserts/Scripts/Docs and
// Body/Headers/Asserts/Timing/Logs).
//
//	senda [-collection ./my-api] [-env dev]
//
// Keys: tab/shift+tab cycle pane focus · ctrl+\ cycle layout (stacked/3-pane/
// focus) · ctrl+k command palette · j/k or ↓/↑ move/scroll · h/l or ←/→
// collapse-expand (tree) or switch tab (pane) · 1–7 jump tab · enter expand
// folder / load request · s send · e edit in $EDITOR · x export as code ·
// ctrl+w close request tab · [ ] cycle env · E env picker · ? help · q quit.
package tui

import (
	"flag"
	"fmt"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"senda/internal/store"
)

// Run parses the TUI flags from args and launches the interactive program. It
// returns an error rather than exiting so the cmd/senda dispatcher owns the
// process exit code.
func Run(args []string) error {
	fs := flag.NewFlagSet("senda", flag.ContinueOnError)
	collPath := fs.String("collection", ".", "collection root directory")
	env := fs.String("env", "", "initial environment name")
	if err := fs.Parse(args); err != nil {
		return err
	}

	root, err := filepath.Abs(*collPath)
	if err != nil {
		return err
	}

	coll, err := store.OpenCollection(root)
	if err != nil {
		return fmt.Errorf("open collection %s: %w", root, err)
	}

	envs, err := store.ListEnvironments(root)
	if err != nil {
		return fmt.Errorf("list environments: %w", err)
	}

	m := newModel(coll, root, envs, *env)

	if _, err := tea.NewProgram(m).Run(); err != nil {
		return err
	}
	return nil
}
