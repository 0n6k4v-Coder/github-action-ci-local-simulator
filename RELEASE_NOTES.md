# 🎉 gacils v1.4.2 — Critical Error Handling Fixes and Release Verification

We are pleased to announce **v1.4.2** of `gacils` (GitHub Actions CI Local Simulator). This release fixes a critical error handling issue where several CLI stub commands would silently exit with code 0 (success) instead of reporting a proper error, and introduces a complete release verification system to prevent similar issues in future releases.

---

## ✨ Highlights

### 🔧 Fixed: Silent Exit Code 0 on Unimplemented Commands

Previously, five CLI stub commands (`list`, `init`, `doctor`, `setup`, `setup-python`)
printed "not implemented yet" and exited with code 0 (success). This was misleading —
users could not distinguish a successful operation from an unimplemented one, and
automation pipelines would silently treat unimplemented commands as successful.

These commands now return a proper error (exit code 1) with an actionable hint
and a link to the issue tracker:

```text
gacils list is not implemented yet
Hint: This feature is planned for a future release.
Check https://github.com/0n6k4v-Coder/github-action-ci-local-simulator/issues for updates.
```

**Commands fixed:**
- `gacils list` — List workflow files in `.github/workflows`
- `gacils init` — Initialize a new workflow file
- `gacils doctor` — Check local environment readiness
- `gacils setup` — Setup local caches
- `gacils setup-python` — Setup Python tool cache for a specified version

### 📚 Documentation Fix

- Fixed CLI command listing in `README.md` — `gacils clean` was missing from the
  Commands section of the usage block and is now listed alongside `gacils run`.

### 🛡️ Release Verification System

This release introduces a comprehensive release verification system to ensure
future releases are correctly tagged and contain all claimed features:

**Pre-release validation** (`scripts/validate-release.sh`):
- Checks for nil-returning stub commands (prevents silent exit code 0 traps)
- Runs unit tests with race detector
- Builds binary and verifies `--help` output
- Validates CHANGELOG and README are updated
- Verifies `clean --dry-run --force` works

**Install-from-tag testing** (`scripts/test-install-from-tag.sh`):
- Runs against a released tag to verify from a user's perspective
- Checks clean command implementation, stub behavior, README, and CHANGELOG
- Usage: `./scripts/test-install-from-tag.sh v1.4.2`

**Automated Go release tests** (`test/release/release_test.go`):
- Runs on tagged commits to verify tag integrity
- Tests: clean command exists, no nil-returning stubs, README mentions
  `gacils clean`, CHANGELOG has release section
- Usage: `go test ./test/release/ -v` (on tagged commits)

### 🧹 Maintenance

- Generated test binaries and artifacts are now ignored by Git
  - `test/e2e/gacils`, `test/integration/gacils`, and
    `test/integration/.gacils-artifacts/` added to `.gitignore`

---

## 📦 Installation

Install via Go toolchain:
```bash
go install github.com/0n6k4v-Coder/github-action-ci-local-simulator/cmd/gacils@v1.4.2
```

**System requirements:**
- [Docker Engine](https://docs.docker.com/get-docker/) running locally
- [Go 1.22+](https://golang.org/dl/) (only required for building from source)

---

## 🔍 Verification

### Run the pre-release validation:
```bash
./scripts/validate-release.sh
```

This script verifies:
1. No stubs return nil (all unimplemented commands return errors)
2. All unit tests pass (Docker-dependent tests skipped with `-short`)
3. Binary builds successfully
4. `--help` output is valid
5. CHANGELOG.md has a release section
6. README.md is updated
7. `clean --dry-run --force` works
8. Race detector passes on CLI package

### Verify a released tag:
```bash
./scripts/test-install-from-tag.sh v1.4.2
```

---

## 📝 Full Changelog

**Changes since v1.4.1**:
- `c55790b` — `fix(errors): improve error handling and add release validation`
- `331192d` — `feat(release): add install-from-tag testing and Go release tests`
- `f7b992e` — `feat(release): complete release verification system with install-from-tag and Go tests`
- `00e6383` — `docs: fix CLI command listing`
- `618533a` — `chore: ignore generated test artifacts`

---

## Previous Release: v1.0.0

We are thrilled to announce the official **v1.0.0** release of `gacils` (GitHub Actions CI Local Simulator)! `gacils` enables developers to run, debug, and test GitHub Actions workflows locally inside Docker containers with zero CI minute costs and instant feedback loops.

---

## ✨ Highlights

### 🚀 Parallel Job Execution
Built-in Goroutine-based Directed Acyclic Graph (DAG) scheduler capable of resolving complex inter-job dependencies (`needs`) and executing independent workflow jobs concurrently.

### 🐳 Service Containers
Full support for background service containers (PostgreSQL, Redis, MySQL, etc.) linked seamlessly over isolated custom Docker bridge networks with container health checking before step execution.

### 🗄️ Cache & Artifacts
Host-directory simulation for `actions/cache` and `actions/upload-artifact` / `actions/download-artifact`, maintaining persistent caches and build outputs across local workflow runs.

### 🔐 Automatic Secrets Masking
Advanced secret masking engine sorted by length in descending order, guaranteeing sensitive tokens and secrets passed via CLI or environment files are never leaked in log output.

### 🧮 Expression Engine
Robust expression parser handling contexts (`github`, `matrix`, `env`, `secrets`, `steps`, `needs`, `job`, `vars`), built-in evaluation functions (`contains`, `startsWith`, `endsWith`, `format`, `join`, `toJSON`, `fromJSON`, `hashFiles`), and operator precedence.

### 🎯 Matrix Expansion
Full support for GitHub Actions `strategy.matrix` matrix expansion rules, including explicit `include` and `exclude` combination rules.

---

## 📦 Installation

Install via Go toolchain:
```bash
go install github.com/0n6k4v-Coder/github-action-ci-local-simulator/cmd/gacils@latest
```

Or download pre-built release binaries from [GitHub Releases](https://github.com/0n6k4v-Coder/github-action-ci-local-simulator/releases/tag/v1.0.0).

---

## 🎯 Quick Start

```bash
# Navigate to your GitHub repository
cd /path/to/your/project

# Run default workflow (.github/workflows)
gacils run
```

---

## 🙏 Thanks

A huge thank you to all early adopters, contributors, and the open-source community for testing, providing feedback, and helping bring `gacils` to production readiness!

---

## 📋 What's Next

- **v1.1**: Enhanced event trigger simulation and synthetic event payload generator CLI.
- **v1.2**: Terminal UI (TUI) dashboard for live job and container monitoring.
- **v1.3**: Native support for reusable workflows (`jobs.<job_id>.uses`).

---

## Full Changelog

**Full Changelog**: https://github.com/0n6k4v-Coder/github-action-ci-local-simulator/commits/v1.0.0
