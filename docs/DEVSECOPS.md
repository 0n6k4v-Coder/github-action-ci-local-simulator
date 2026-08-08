# DevSecOps Baseline

> **Phase 1 — DevSecOps Baseline** for `gacils`.
>
> This document describes the CI/CD and security controls implemented as part
> of the DevSecOps baseline. It is the single source of truth for how the
> project is built, tested, and secured.

---

## 1. Pipeline Overview

```text
Pull Request
     |
     +-----------------------------+
     |                             |
     v                             v
 Code Quality                 Security Analysis (CodeQL)
     |                             |
     +---------+-------------------+
               |
               v
          Security Gate
               |
               v
             Merge
               |
               v
             main
               |
               v
        Release Validation (scripts/validate-release.sh)
```

CI runs automatically on every pull request and push to `main`.
Release-time validation (`scripts/validate-release.sh`) is **not** part of
CI — it runs as a manual pre-tag gate to avoid expensive operations during
routine development.

---

## 2. CI Workflow

**File:** `.github/workflows/ci.yml`

**Triggers:**
- `pull_request` — all PRs targeting `main`
- `push` — pushes to `main`

**Permissions (least privilege):**
```yaml
permissions:
  contents: read
  actions: read
```
No write scope is granted. The workflow cannot modify the repository, open
PRs, or alter branch protection.

**Jobs:**

| Step             | Command                                | Purpose                                  |
| ---------------- | -------------------------------------- | ---------------------------------------- |
| gofmt            | `gofmt -l .`                           | Enforce Go source formatting             |
| go vet           | `go vet ./...`                         | Static analysis of Go code               |
| go build         | `go build ./...`                       | Verify all packages compile              |
| Build binary     | `go build -o /tmp/gacils-ci ./cmd/gacils` + `--help` | Verify release binary builds and runs |
| Unit tests       | `go test -short ./... -timeout 120s`   | Run unit tests (skip Docker-dependent)   |
| Race detector    | `go test -race ./internal/cli/`        | Detect data races in CLI layer           |

The `-short` flag follows the project's convention (matching
`scripts/validate-release.sh`) to skip Docker-dependent integration/e2e
tests, ensuring CI is fast and deterministic.

---

## 3. CodeQL (SAST)

**File:** `.github/workflows/codeql.yml`

CodeQL performs static application security testing for the Go codebase.

**Triggers:**
- `push` to `main`
- `pull_request`
- `schedule` (weekly, Mondays at 07:00 UTC)

**Permissions:**
```yaml
permissions:
  contents: read
  actions: read
  security-events: write
```
The `security-events: write` scope is **required** by GitHub's CodeQL
action to upload results to the code-scanning dashboard. This is the
minimum scope necessary and is limited to security-event upload only —
it does not grant any repository or workflow modification rights.

**Analysis:**
- Language: Go
- Go version: 1.26 (matches `go.mod`)
- Autobuild is used to automatically detect and build the Go module.
- Results are viewable in the repository's **Security** → **Code scanning**
  tab.

---

## 4. Dependabot

**File:** `.github/dependabot.yml`

Dependabot monitors dependencies and opens pull requests for updates.

| Package Ecosystem | Path  | Interval | Auto-merge |
| ----------------- | ----- | -------- | ---------- |
| Go modules        | `/`   | Weekly   | No         |
| GitHub Actions    | `/`   | Weekly   | No         |

- Updates are grouped (minor + patch into a single PR) to reduce noise.
- `open-pull-requests-limit: 5` caps concurrent update PRs.
- **No automated merging** — all updates require manual review.
- Security updates from Dependabot will appear as prioritized PRs that
  should be reviewed promptly.

---

## 5. Secret Security

| Control                     | Status                    | Notes                                                                 |
| --------------------------- | ------------------------- | --------------------------------------------------------------------- |
| Secret scanning             | **Manual configuration**  | Repository-level setting — see Section 7.                             |
| Push protection             | **Manual configuration**  | Repository-level setting — see Section 7.                             |
| Secret scanning in PRs      | **Manual configuration**  | Requires GitHub Advanced Security license (public repos: free).       |
| Hardcoded secrets           | Verified clean            | No secrets are committed in any files.                                |
| Secrets in workflows        | Verified clean            | No `GITHUB_TOKEN` write scopes beyond what's strictly needed.        |
| Fork PR isolation           | Verified                  | `pull_request` trigger uses read-only default token; no `secrets` passed to untrusted code. |

No additional secret-scanning workflow (e.g., truffleHog, gitleaks) is
added to avoid duplicate/misleading security controls. GitHub's native
secret scanning is the authoritative mechanism and is documented in
Section 7.

---

## 6. SECURITY.md

**File:** [`SECURITY.md`](../SECURITY.md)

Defines:
- **Supported versions** — only the latest minor series receives fixes.
- **Vulnerability reporting** — via GitHub "Report a vulnerability" or
  private discussion.
- **Disclosure policy** — private report → triage → fix → security release
  → public disclosure.
- **Security scope** — what is and is not in-scope for vulnerability
  reports.

---

## 7. Repository-Level Settings (Manual Configuration Required)

Some security controls **cannot** be enabled from repository files and
must be configured manually by a repository administrator in GitHub.

### 7.1 Branch Protection (`main`)

These settings are configured in **Settings → Code and automation →
Branches → Branch protection rules** → `main`:

- [ ] **Require a merge request** before merging commits
- [ ] **Require status checks to pass** before merging:
  - `CI` (the `ci.yml` workflow)
  - `Analyze (CodeQL)` (the `codeql.yml` workflow)
- [ ] **Require branches to be up to date before merging**
- [ ] **Do not allow bypassing the above settings** (or restrict bypass
      to repository admins only)
- [ ] **Restrict who can push to matching branches** (allow only admins or required reviewers)
- [ ] Disable **Allow force pushes**
- [ ] Disable **Allow deletions**
- [ ] (Recommended) Require **1 approving review** from a repository
      collaborator before merging

> For a solo-maintained project, requiring multiple approvals may be
> overly restrictive. The maintainer should decide based on their
> collaboration model.

### 7.2 Secret Scanning

Configured in **Settings → Code security and analysis**:

- [ ] Enable **Secret scanning**
- [ ] Enable **Secret scanning push protection**
- [ ] Enable **Secret scanning for non-provider patterns** (optional,
      detects custom secret formats)

> Secret scanning is available on public repositories at no additional cost
> and does not require a GitHub Advanced Security license. Push protection
> requires GitHub Advanced Security for private repositories but is free
> for public repositories.

### 7.3 Code Security and Analysis

In **Settings → Code security and analysis**:

- [ ] Enable **GitHub Advanced Security** features (free for public repos)
- [ ] **Code scanning** — CodeQL will work automatically once
      `codeql.yml` is committed; the analysis results will appear in the
      **Security** tab.
- [ ] **Dependency graph** — ensure this is enabled so Dependabot can
      function.
- [ ] **Dependabot alerts** — ensure this is enabled (free for public
      repos).
- [ ] **Dependabot security updates** — enable if you want automatic
      security-fix PRs for vulnerable dependencies.

### 7.4 Actions Permissions

In **Settings → Code and automation → Actions → General**:

- [ ] Allow **Read and write permissions** (required for workflows that
      upload CodeQL results):
  - Actually, `ci.yml` only needs **Read repository contents and packages
    permissions**.
  - `codeql.yml` needs **Read and write permissions** because it uploads
    to the security-events store.
  - Alternatively, set the default to **Read repository contents and
    packages permissions** and rely on each workflow's `permissions:`
    block to escalate as needed. The `codeql.yml` workflow declares its
    own `permissions:` with `security-events: write`.
- [ ] **Allow GitHub Actions to create and approve pull requests** —
      recommended to enable so Dependabot and CodeQL can function.
- [ ] Consider **restricting Actions to only those from GitHub.com** or
      allowing trusted publishers.

---

## 8. Release Process

The release mechanism is **tag-based** and **not modified** by this
baseline. The existing process remains:

1. Merge to `main` with updated `CHANGELOG.md`.
2. Run `bash scripts/validate-release.sh` (manual pre-tag validation).
3. Test install from tag: `bash scripts/test-install-from-tag.sh vX.Y.Z`
4. Create and push a tag: `git tag -a vX.Y.Z -m 'Release vX.Y.Z'` &&
   `git push origin vX.Y.Z`
5. Create a GitHub Release.

CI does **not** automate any part of the release process.

---

## 9. DevSecOps Checklist

### Implemented

- [x] CI workflow with gofmt, vet, build, test, race detection
- [x] CodeQL SAST workflow with scheduled analysis
- [x] Dependabot for Go modules and GitHub Actions
- [x] SECURITY.md with supported versions and reporting process
- [x] Least-privilege permissions on all workflows
- [x] `pull_request` trigger (not `pull_request_target`)
- [x] No hardcoded secrets
- [x] No fork PR credential exposure
- [x] Documentation of DevSecOps workflow (`DEVSECOPS.md`)

### Requires GitHub Repository Configuration

- [ ] Branch protection on `main`
- [ ] Require CI + CodeQL status checks before merge
- [ ] Secret scanning enabled
- [ ] Push protection enabled
- [ ] Dependency graph + Dependabot alerts enabled
- [ ] Code scanning configured (will work automatically once `codeql.yml`
      is on `main`)

### Future Phase

- Consider adding `golangci-lint` as a dedicated linting step (currently
  falls back to `go vet`)
- Consider adding `govulncheck` for Go vulnerability detection in CI
- Consider adding a dependency vulnerability scan action
- Consider adding SBOM (Software Bill of Materials) generation
