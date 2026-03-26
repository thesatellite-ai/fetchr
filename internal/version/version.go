package version

import "fmt"

// Set via ldflags at build time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String returns the full version string.
func String() string {
	return fmt.Sprintf("fetchr %s (commit: %s, built: %s)", Version, Commit, Date)
}
