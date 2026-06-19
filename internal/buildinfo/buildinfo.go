// Package buildinfo holds build-time metadata injected via -ldflags -X.
package buildinfo

// Version is the release version, set at build time
// (-X senda/internal/buildinfo.Version=<tag>). Defaults to "dev".
var Version = "dev"
