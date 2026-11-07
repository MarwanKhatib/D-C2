// Package version carries build metadata injected with -ldflags.
package version

// These values are overridden at build time, for example:
//
//	-X discord-c2/internal/version.Version=v1.2.3
//	-X discord-c2/internal/version.Commit=<sha>
//	-X discord-c2/internal/version.Date=<rfc3339>
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// String returns a compact human-readable version.
func String() string {
	if Commit == "unknown" || Commit == "" {
		return Version
	}
	if len(Commit) > 12 {
		return Version + " (" + Commit[:12] + ")"
	}
	return Version + " (" + Commit + ")"
}
