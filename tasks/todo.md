# Patch Todo

- [x] Review codebase for redundancy, vulnerabilities, and feature gaps
- [x] Refactor AviationStack requests to use HTTPS and query escaping
- [x] Centralize repeated command setup and JSON output handling
- [x] Fix `track` shutdown flow to avoid cross-goroutine spinner races
- [x] Verify behavior and update review notes

## Review

- Patched request construction to use HTTPS and `net/url` query encoding.
- Centralized API-key lookup, cache-backed service creation, JSON rendering, and airport-code normalization in `cmd/helpers.go`.
- Reworked `track` to use a signal-aware loop without sharing a spinner across goroutines.
- Added provider tests covering HTTPS usage, query encoding, and flight-number normalization.
- Verification: `go test ./...` passed with `GOCACHE` set to a workspace-local directory.