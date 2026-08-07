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

**Current flag status (as of v1.4.0):**

| Flag | Used in run.go | Has Test |
|------|:--------------:|:--------:|
| `Workflow` | YES | YES |
| `Job` | YES | YES |
| `DryRun` | YES | YES |
| `Parallel` | YES | YES |
| `CRLF` | YES | YES |
| `Platform` | YES | YES |
| `Offline` | YES | YES |

> All 7 flags are now fully implemented and tested. ✅

## 7. Documentation

- [ ] `CHANGELOG.md` updated with new release section
- [ ] `README.md` updated (if applicable)
- [ ] `RELEASE_NOTES.md` drafted
- [ ] Breaking changes documented (if any)
- [ ] Flag descriptions in `--help` are accurate and up to date

## 8. Error Handling

### Pre-Release Validation

Run the validation script before creating any release tag:

```bash
./scripts/validate-release.sh
```

This script checks:
- No stubs remaining in CLI code
- All tests pass
- Build succeeds
- Binary works
- CHANGELOG.md updated
- README.md updated
- No race conditions

### Error Handling Guidelines

1. **All stubs must return errors, not nil**
   ```go
   // WRONG:
   RunE: func(cmd *cobra.Command, args []string) error {
       fmt.Println("not implemented")
       return nil  // User sees exit code 0
   }

   // CORRECT:
   RunE: func(cmd *cobra.Command, args []string) error {
       return fmt.Errorf("not implemented\nHint: ...")
   }
   ```

2. **All errors must have hints**
   ```go
   return fmt.Errorf("operation failed\n" +
       "Hint: Try running 'gacils clean --dry-run' first")
   ```

3. **Use appropriate exit codes**
   - 0: Success
   - 1: General error
   - 2: User input error
   - 3: System error
   - 4: External dependency error

### Release Process

1. **Implement features**
2. **Run validation script**
   ```bash
   ./scripts/validate-release.sh
   ```
3. **If validation passes, create tag**
   ```bash
   git tag -a vX.Y.Z -m "Release vX.Y.Z: ..."
   ```
4. **Push tag**
   ```bash
   git push origin vX.Y.Z
   ```
5. **Verify on GitHub**
   ```bash
   git ls-remote origin refs/tags/vX.Y.Z
   ```

### Lessons Learned

**v1.4.0 Issue:**
- v1.4.0 was tagged before `gacils clean` was implemented
- Users installing v1.4.0 saw "not implemented yet" errors
- **Solution:** Created v1.4.1 with the complete feature set
- **Prevention:** Added `scripts/validate-release.sh` to catch stubs before tagging

**Key Takeaway:**
- Always run `./scripts/validate-release.sh` before creating a release tag
- Never tag a release until validation passes
- Separate "implement" and "release" phases

## 9. Version & Tag

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
