// Package version holds build-time version information injected via ldflags.
package version

// These variables are set at build time via:
//
//	go build -ldflags "-X github.com/Jules-Solutions/jules-installer/pkg/version.Version=1.0.0 \
//	  -X github.com/Jules-Solutions/jules-installer/pkg/version.Commit=abc1234 \
//	  -X github.com/Jules-Solutions/jules-installer/pkg/version.BuildDate=2026-04-10T00:00:00Z"
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// String returns a human-readable version string.
func String() string {
	return Version + " (" + Commit + ") built " + BuildDate
}
