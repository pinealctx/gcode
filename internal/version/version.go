// Package version holds build-time metadata.
//
// Version information is resolved in priority order:
//  1. -ldflags overrides (if set during custom builds)
//  2. Go module version + VCS metadata from runtime/debug.ReadBuildInfo
//     (automatically embedded by Go 1.18+ toolchain for go install @version)
//
// This ensures `go install github.com/pinealctx/gcode/cmd/gcode@latest`
// produces meaningful version output without any build script.
package version

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// Version is the semantic version. Overridden via -ldflags for custom builds.
// Defaults to the module version from go.sum, or "dev" for local builds.
var Version = ""

// Commit is the git commit SHA. Overridden via -ldflags.
// Defaults to VCS revision from embedded build info.
var Commit = ""

// BuildTime is the build timestamp. Overridden via -ldflags.
// Defaults to VCS commit time from embedded build info.
var BuildTime = ""

func init() {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		fillDefaults()
		return
	}

	// Prefer ldflags overrides; fall back to build info.
	if Version == "" {
		Version = bi.Main.Version
		// "(devel)" is Go's placeholder for local or non-versioned builds.
		if Version == "" || Version == "(devel)" {
			Version = "dev"
		}
	}
	if Commit == "" {
		Commit = vcsSetting(bi, "vcs.revision")
		if len(Commit) > 12 {
			Commit = Commit[:12]
		}
	}
	if BuildTime == "" {
		BuildTime = vcsSetting(bi, "vcs.time")
	}

	// Append "+dirty" to commit if working tree had uncommitted changes.
	if vcsSetting(bi, "vcs.modified") == "true" && !strings.HasSuffix(Commit, "+dirty") {
		Commit += "+dirty"
	}

	fillDefaults()
}

func fillDefaults() {
	if Version == "" {
		Version = "dev"
	}
	if Commit == "" {
		Commit = "none"
	}
	if BuildTime == "" {
		BuildTime = "unknown"
	}
}

// vcsSetting returns a VCS-related build setting value, or "" if not found.
func vcsSetting(bi *debug.BuildInfo, key string) string {
	for _, s := range bi.Settings {
		if s.Key == key {
			return s.Value
		}
	}
	return ""
}

// String returns a human-readable version summary.
func String() string {
	return fmt.Sprintf("gcode %s (%s, %s)", Version, Commit, BuildTime)
}
