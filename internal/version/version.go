package version

var (
	// Version is set at build time via -ldflags.
	// Defaults to "dev" for local builds.
	Version = "dev"

	// Commit is set at build time via -ldflags.
	Commit = "unknown"

	// Date is set at build time via -ldflags.
	Date = "unknown"
)