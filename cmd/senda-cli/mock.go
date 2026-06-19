package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"senda/internal/mockserver"
	"senda/internal/store"
)

// runMock starts the local mock server against a collection's .senda/mocks/ directory
// and blocks until interrupted, logging each request to stdout. The "init"
// subcommand instead scaffolds a bundled preset into that directory.
func runMock(args []string) {
	if len(args) > 0 && args[0] == "init" {
		runMockInit(args[1:])
		return
	}

	fs := flag.NewFlagSet("mock", flag.ExitOnError)
	collPath := fs.String("collection", ".", "collection root directory")
	addr := fs.String("addr", ":8787", "listen address (\":0\" picks a free port)")
	scenario := fs.String("scenario", "", "active scenario")
	_ = fs.Parse(args)

	root, err := filepath.Abs(*collPath)
	if err != nil {
		fatal(err)
	}
	mocksDir := store.MocksDir(root)

	srv, err := mockserver.New(mocksDir, func(e mockserver.LogEntry) {
		fmt.Printf("%s  %-6s %-30s %d  %s\n", e.At, e.Method, e.Path, e.Status, e.Source)
	}, nil)
	if err != nil {
		fatal(err)
	}
	if *scenario != "" {
		srv.SetScenario(*scenario)
	}

	bound, err := srv.Start(*addr)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("senda mock server on http://%s  (mocks: %s)\n", bound, mocksDir)
	routes := srv.Routes()
	if len(routes) == 0 {
		fmt.Println("  no routes loaded — add *.yaml files to the .senda/mocks/ directory")
	}
	for _, r := range routes {
		fmt.Printf("  %-6s %s\n", r.Method, r.Path)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	fmt.Println("\nshutting down")
	_ = srv.Stop()
}

// runMockInit scaffolds a bundled mock preset (e.g. "oauth") into a
// collection's .senda/mocks/ directory.
//
//	senda-cli mock init oauth [-collection ./my-api]
func runMockInit(args []string) {
	fs := flag.NewFlagSet("mock init", flag.ExitOnError)
	collPath := fs.String("collection", ".", "collection root directory")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: senda-cli mock init <preset> [-collection dir]\n\npresets: %s\n", strings.Join(mockserver.Presets(), ", "))
	}
	if len(args) == 0 {
		fs.Usage()
		os.Exit(2)
	}
	preset := args[0]
	_ = fs.Parse(args[1:])

	root, err := filepath.Abs(*collPath)
	if err != nil {
		fatal(err)
	}
	mocksDir := store.MocksDir(root)

	written, skipped, err := mockserver.WritePreset(preset, mocksDir)
	if err != nil {
		fatal(err)
	}
	for _, f := range written {
		fmt.Printf("  created  .senda/mocks/%s\n", f)
	}
	for _, f := range skipped {
		fmt.Printf("  skipped  .senda/mocks/%s (already exists)\n", f)
	}
	fmt.Printf("\npreset %q ready. Start it with:\n  senda-cli mock -collection %s\n", preset, *collPath)
}
