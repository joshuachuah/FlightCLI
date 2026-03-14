# Patch Todo

- [x] Continue from the current main-branch state after the interrupted review branch diverged
- [x] Refactor duplicated cache read-through logic in `internal/service/flight_service.go`
- [x] Add service-layer tests for cache hits on pointer and slice responses
- [x] Tighten cache file permissions and add cache edge-case tests
- [x] Verify behavior and update review notes

## Review

- Refactored the service layer around one generic cache read-through helper.
- Added service tests covering cached pointer responses, cached slice responses, and nil-cache behavior.
- Tightened cache directory/file modes and added cache tests for round-trip, expiry cleanup, and corrupt files.
- Verification: `go test ./...` passed with `GOCACHE` set to a workspace-local directory.
