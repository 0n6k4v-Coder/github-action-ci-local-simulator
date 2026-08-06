# Pre-Release Checklist

This checklist MUST be completed before tagging any release.
Skipping any item requires explicit justification documented in the release notes.

## 1. Build & Static Analysis

- [ ] `go build ./...` passes with no errors
- [ ] `go vet ./...` passes with no warnings
- [ ] No compiler warnings in output

## 2. Unit Tests

- [ ] `go test -race -count=1 ./internal/...` passes
- [ ] No skipped tests without documented justification
- [ ] Test coverage ≥ previous release
- [ ] All new code has corresponding tests

## 3. Integration Tests (requires Docker)

- [ ] `go test -v ./test/integration/ -timeout 10m` passes
- [ ] No orphaned Docker resources after tests
- [ ] Verify: `docker ps` shows no gacils containers
- [ ] Verify: `docker network ls` shows no gacils networks

> **Known flakiness:** `TestAutoInstall_PythonCommands` may time out if the Docker image
> pull is slow (image: `python:3.11-slim`). Re-run once before blocking a release.
> Use `--timeout 10m` to avoid premature failures.

## 4. E2E Tests (requires Docker)

- [ ] `go test -v ./test/e2e/ -timeout 10m` passes
- [ ] All E2E scenarios covered
- [ ] No orphaned Docker resources after tests

## 5. CLI Smoke Test

- [ ] `gacils --version` outputs correct version
- [ ] `gacils --help` shows all commands
- [ ] `gacils run --help` shows all flags
- [ ] `gacils run -W basic-workflow.yml` works
- [ ] `gacils run -W multi-job.yml --job <job>` filters correctly
- [ ] `gacils run -W multi-job.yml --job nonexistent` shows error
- [ ] Every flag in `flags.go` tested manually (see Flag Audit below)

## 6. Flag Audit

- [ ] Every flag in `internal/cli/flags.go` is **defined**
- [ ] Every flag is **used** somewhere in `internal/cli/run.go` (no dead code)
- [ ] Every flag has **at least one test** in `internal/cli/`
- [ ] Run the audit command below and verify each flag shows `used in >= 1 places`

```bash
grep -oP 'flags\.\K\w+' internal/cli/flags.go | sort -u | while read var; do
  count=$(grep -rn "flags\.$var" internal/ --include="*.go" | grep -v flags.go | wc -l)
  status="OK"
  [ "$count" -eq 0 ] && status="DEAD CODE WARNING"
  echo "flags.$var: used in $count place(s) -- $status"
done
```

**Current flag status (as of last audit):**

| Flag | Used in run.go | Has Test |
|------|:--------------:|:--------:|
| `Workflow` | YES | YES |
| `Job` | YES | YES |
| `DryRun` | YES | YES |
| `Parallel` | YES (warns) | YES |
| `CRLF` | YES (warns) | YES |
| `Platform` | YES (warns) | YES |
| `Offline` | YES (warns) | YES |

> **Note:** Parallel/CRLF/Platform/Offline now warn users when used. Full implementation is a future task.
> (`Parallel`, `CRLF`, `Platform`, `Offline`). See [flags.go](internal/cli/flags.go).

## 7. Documentation

- [ ] `CHANGELOG.md` updated with new release section
- [ ] `README.md` updated (if applicable)
- [ ] `RELEASE_NOTES.md` drafted
- [ ] Breaking changes documented (if any)
- [ ] Flag descriptions in `--help` are accurate and up to date

## 8. Version & Tag

- [ ] Correct semantic version (`major.minor.patch`) per semver.org
- [ ] Tag message is descriptive
- [ ] Tag points to the correct commit
- [ ] Version string in code matches the tag

## 9. Push & Verify

- [ ] `git push origin main` succeeds
- [ ] `git push origin vX.Y.Z` succeeds
- [ ] Verify tag exists: `git ls-remote --tags origin | grep vX.Y.Z`
- [ ] Verify on GitHub: tag visible in the Releases page

## 10. Post-Release

- [ ] Notify users / stakeholders
- [ ] Monitor for immediate issues
- [ ] Update project status documentation

---

## Release Decision

**DO NOT RELEASE if any of the following fail:**

| Condition | Blocks Release |
|-----------|:--------------:|
| Build fails | YES |
| Unit tests fail | YES |
| Integration tests fail (with Docker) | YES |
| E2E tests fail (with Docker) | YES |
| CLI smoke test fails | YES |
| Dead code detected and unacknowledged | YES |
| Untested flags detected | YES |
| CHANGELOG not updated | YES |

**RELEASE only when ALL checklist items are checked.**

---

## Quick Commands

```bash
# Build & Vet
go build ./... && go vet ./...

# Unit Tests
go test -race -count=1 ./internal/... -timeout 60s

# CLI Package Tests (unit + integration)
go test -v ./internal/cli/ -timeout 120s

# Integration Tests (requires Docker)
go test -v ./test/integration/ -timeout 10m

# E2E Tests (requires Docker)
go test -v ./test/e2e/ -timeout 10m

# CLI Smoke Test
go build -o /tmp/gacils ./cmd/gacils
/tmp/gacils --version
/tmp/gacils --help
/tmp/gacils run --help

# Flag Audit
grep -oP 'flags\.\K\w+' internal/cli/flags.go | sort -u | while read var; do
  count=$(grep -rn "flags\.$var" internal/ --include="*.go" | grep -v flags.go | wc -l)
  status="OK"
  [ "$count" -eq 0 ] && status="DEAD CODE WARNING"
  echo "flags.$var: used in $count place(s) -- $status"
done

# Tag & Push
VERSION=vX.Y.Z
git tag -a "$VERSION" -m "Release $VERSION"
git push origin main
git push origin "$VERSION"
git ls-remote --tags origin | grep "$VERSION"
```
