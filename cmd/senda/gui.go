package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// runGUI launches the desktop app (senda-desktop), forwarding any args — the
// `code`-style launcher. The desktop binary is a separate artifact because it
// links GTK4/WebKit via CGO and cannot run on a headless host; this pure-Go
// binary just finds and execs it.
//
// Resolution order: the directory of this executable first (installers drop the
// two binaries side by side), then $PATH.
func runGUI(args []string) {
	name := "senda-desktop"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	bin := ""
	if self, err := os.Executable(); err == nil {
		// Resolve symlinks so an installer-symlinked `senda` still finds the
		// `senda-desktop` sitting next to the real binary (e.g. Homebrew).
		if resolved, e := filepath.EvalSymlinks(self); e == nil {
			self = resolved
		}
		cand := filepath.Join(filepath.Dir(self), name)
		if st, err := os.Stat(cand); err == nil && !st.IsDir() {
			bin = cand
		}
	}
	if bin == "" {
		if p, err := exec.LookPath(name); err == nil {
			bin = p
		}
	}
	if bin == "" {
		fatal(fmt.Errorf("%s not found next to this binary or on PATH — install the desktop build (it needs a WebKitGTK/webview runtime)", name))
	}

	// Launch detached and return the shell prompt (code-style): don't Wait, so
	// the GUI outlives this launcher. stderr/stdout stay wired so an early
	// startup failure (e.g. missing webview runtime) is still visible.
	cmd := exec.Command(bin, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		fatal(err)
	}
	_ = cmd.Process.Release()
}
