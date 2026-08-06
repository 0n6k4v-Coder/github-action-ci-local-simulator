# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.3.1] - 2026-08-06

### Fixed
- WSL2 + Docker Desktop compatibility
  - Fixed: "failed to connect to the docker API at unix:///var/run/docker.sock"
  - Now uses `client.FromEnv` which auto-detects socket paths
  - Supports WSL2 sockets: `~/.docker/desktop/docker.sock`, `/mnt/wsl/docker-desktop/docker.sock`
  - Respects `DOCKER_HOST` environment variable
  - Includes API version negotiation

### Changed
- Refactored Docker client creation to use `dockerx.CreateDockerClient()`
- Simplified Docker client initialization across codebase

## [1.3.0] - 2026-08-06

### Added
- Complete end-to-end (E2E) test coverage
  - 18 E2E test cases across 6 test suites
  - Basic workflow tests (single job, multi-step, env vars)
  - Matrix expansion tests (basic, include, exclude)
  - Job dependency tests (linear, diamond, failure propagation)
  - Actions ecosystem tests (checkout, setup-python, cache, artifacts)
  - Service container tests (Redis, Postgres, multiple services)
  - Production-like pipeline tests (Python CI, full multi-job pipeline)
- E2E test helpers (`test/e2e/helpers_test.go`)

### Changed
- Test pyramid now complete: 84 total test cases
  - Unit tests: 42 (v1.1.0)
  - Integration tests: 24 (v1.2.0)
  - E2E tests: 18 (v1.3.0)

## [1.2.0] - 2026-08-06

### Added
- Complete integration test coverage
  - 24 integration test cases across 7 test suites
  - Docker test infrastructure (`test/integration/helpers_test.go`)
  - Workspace copy tests
  - Python setup tests
  - Cache tests (save, restore, key isolation)
  - Artifact tests (upload, download, cross-job)
  - Service container tests (Redis, health checks, cleanup)
  - Parallel execution tests
  - Secrets masking tests

### Changed
- All integration tests require Docker daemon
- Tests automatically skip in short mode (`go test -short`)

## [1.1.0] - 2026-08-06

### Added
- Complete unit test coverage
  - 42 unit test cases across 5 test suites
  - Step execution tests (shell detection, env handling, conditions)
  - Workflow execution tests (job ordering, parallel execution, matrix)
  - setup-python action tests (input validation, version formats)
  - Expanded workspace and image selection tests

### Changed
- Minimal testability refactor in `steprunner.go` (execCommand mockability)
- Test coverage >80% on modified files

## [1.0.3] - 2026-08-05

### Fixed
- **Bug 1: Workspace root detection**
  - Fixed: Workspace was set to workflow file's directory instead of repo root
  - Now correctly detects git repo root (`.git/`) or `.github/` directory
  - Falls back to workflow directory only if no repo markers found
  - Fixes: Docker build fails, pip install fails, pytest fails
  
- **Bug 2: setup-python hybrid approach**
  - Fixed: `actions/setup-python` simulation only set env vars, didn't install Python
  - Now uses hybrid approach:
    * PRIMARY: Switch image to `python:<version>-slim`
    * FALLBACK: Auto-install `python3 + pip` in ubuntu container
  - Fixes: `pip: command not found` after setup-python step

### Added
- `internal/runner/workspace.go` with `FindRepoRoot()` function
- `findSetupPythonStep()` and `extractPythonVersion()` helpers
- Test fixtures for workspace and setup-python scenarios

## [1.0.2] - 2026-08-05

### Fixed
- Docker CLI not found in workflow steps
  - Auto-installs `docker-ce-cli` when docker commands detected
  - Fallback to `docker.io` package if `docker-ce-cli` unavailable
  - Skips installation for `docker:*` images

### Added
- `internal/runner/dockercli.go` with Docker CLI detection
- Auto-install logic similar to Python auto-install
- Test fixtures for Docker CLI scenarios

## [1.0.1] - 2026-08-04

### Fixed
- Python/pip not found on ubuntu images
  - Auto-installs `python3 + python3-pip + python3-venv` when pip/python commands detected
  - Removes PEP 668 `EXTERNALLY-MANAGED` marker for system-wide pip installs
  - Idempotent: checks if Python already installed before installing

### Added
- `internal/runner/python.go` with Python command detection
- Auto-install logic for Python environments
- Unit tests for Python detection and installation

## [1.0.0] - 2026-08-04

### Added
- Initial release of gacils (GitHub Actions CI Local Simulator)
- Core workflow execution engine
- Phase 0-11 implementation:
  - Workflow parsing and validation
  - Job execution with Docker containers
  - Step execution with shell detection
  - Expression interpolation (Phase 6A-6D)
  - Job dependencies (needs)
  - Matrix strategy expansion
  - Actions simulation (checkout, setup-python, cache, artifacts)
  - Service containers
  - Secrets masking
  - Parallel job execution
- Command-line interface with `run` command
- Basic test suite

[1.3.1]: https://github.com/0n6k4v-Coder/github-action-ci-local-simulator/releases/tag/v1.3.1
[1.3.0]: https://github.com/0n6k4v-Coder/github-action-ci-local-simulator/releases/tag/v1.3.0
[1.2.0]: https://github.com/0n6k4v-Coder/github-action-ci-local-simulator/releases/tag/v1.2.0
[1.1.0]: https://github.com/0n6k4v-Coder/github-action-ci-local-simulator/releases/tag/v1.1.0
[1.0.3]: https://github.com/0n6k4v-Coder/github-action-ci-local-simulator/releases/tag/v1.0.3
[1.0.2]: https://github.com/0n6k4v-Coder/github-action-ci-local-simulator/releases/tag/v1.0.2
[1.0.1]: https://github.com/0n6k4v-Coder/github-action-ci-local-simulator/releases/tag/v1.0.1
[1.0.0]: https://github.com/0n6k4v-Coder/github-action-ci-local-simulator/releases/tag/v1.0.0
