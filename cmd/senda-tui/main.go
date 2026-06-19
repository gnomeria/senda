// senda-tui is an interactive terminal client for senda collections — the same
// pipeline as the desktop app and senda-cli (scripts, vars, secrets, asserts,
// cookie jar), driven from a three-pane TUI: collection tree | request | response.
// The request and response panes carry ATAC-style tab bars (Params/Headers/
// Body/Auth/Asserts/Scripts/Docs and Body/Headers/Asserts/Timing/Logs).
//
//	senda-tui [-collection ./my-api] [-env dev]
//
// Keys: tab/shift+tab cycle pane focus · ctrl+\ cycle layout (stacked/3-pane/
// focus) · ctrl+k command palette · j/k or ↓/↑ move/scroll · h/l or ←/→
// collapse-expand (tree) or switch tab (pane) · 1–7 jump tab · enter expand
// folder / load request · s send · e edit in $EDITOR · x export as code ·
// ctrl+w close request tab · [ ] cycle env · E env picker · ? help · q quit.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"senda/internal/store"
)

func main() {
	collPath := flag.String("collection", ".", "collection root directory")
	env := flag.String("env", "", "initial environment name")
	flag.Parse()

	root, err := filepath.Abs(*collPath)
	if err != nil {
		fatal(err)
	}

	coll, err := store.OpenCollection(root)
	if err != nil {
		fatal(fmt.Errorf("open collection %s: %w", root, err))
	}

	envs, err := store.ListEnvironments(root)
	if err != nil {
		fatal(fmt.Errorf("list environments: %w", err))
	}

	m := newModel(coll, root, envs, *env)

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "senda-tui:", err)
	os.Exit(1)
}
