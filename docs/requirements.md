# Requirements — github-action-ci-local-simulator

Document: `docs/requirements.md`

Project:

```text
github-action-ci-local-simulator
```

Binary:

```text
gacils
```

License:

```text
MIT
```

Status:

```text
Locked requirements for MVP and v2 planning
```

---

## 1. Overview

`gacils` is a personal open-source tool for running GitHub Actions workflows locally using Docker.

The tool reads GitHub Actions workflow files from:

```text
.github/workflows/*.yml
.github/workflows/*.yaml
```

and executes supported workflow features locally inside Docker containers.

The main purpose is:

```text
Run CI locally → fix errors → push → GitHub CI passes
```

This tool is not affiliated with GitHub.

---

## 2. Goals

### 2.1 Primary Goals

The tool must:

1. Run GitHub Actions workflow files locally.
2. Reduce GitHub Actions minute usage.
3. Allow fast debugging of CI failures.
4. Work on Windows, macOS, and Linux.
5. Use Docker containers to simulate GitHub-hosted Ubuntu runners.
6. Fail clearly when encountering unsupported features.
7. Require no modification to workflow files for supported features.
8. Be distributable as a single binary.

### 2.2 Learning Goals

The project is also a personal learning project for:

```text
Go
Docker Engine API
CLI tooling
CI systems
GitHub Actions semantics
Cross-platform developer tools
```

---

## 3. Non-Goals

The tool is not intended to:

1. Fully replace GitHub Actions.
2. Call GitHub APIs.
3. Perform deployments.
4. Create releases.
5. Manage pull requests.
6. Register self-hosted runners.
7. Enforce GitHub permissions.
8. Support reusable workflows.
9. Support OIDC token issuance.
10. Guarantee 100% GitHub Actions compatibility.

---

## 4. Target Platforms

### 4.1 Operating Systems

The tool must support:

| OS | Requirement |
|---|---|
| Windows 10/11 | Primary platform |
| macOS Apple Silicon | Supported |
| macOS Intel | Supported |
| Ubuntu Linux | Supported |
| Debian Linux | Supported |

### 4.2 Docker Runtime

The tool requires Docker.

Supported runtimes:

| Runtime | Requirement |
|---|---|
| Docker Desktop on Windows | Required |
| Docker Desktop on macOS | Required |
| Docker Engine on Linux | Required |
| Rootless Docker | Supported if Docker API is reachable |

Default Docker endpoints:

| OS | Endpoint |
|---|---|
| Linux | `unix:///var/run/docker.sock` |
| macOS | `unix:///var/run/docker.sock` |
| Windows | `npipe:////./pipe/docker_engine` |

The tool must respect:

```text
DOCKER_HOST
DOCKER_CONTEXT
DOCKER_TLS_VERIFY
DOCKER_CERT_PATH
```

---

## 5. Technology Requirements

| Item | Requirement |
|---|---|
| Language | Go |
| Go version | Go 1.26 target; Go 1.22+ acceptable during development |
| CLI framework | Cobra |
| YAML parser | `gopkg.in/yaml.v3` |
| Docker client | Official Docker Engine API Go SDK |
| Expression evaluator | `expr-lang/expr` or equivalent sandboxed evaluator |
| Colored output | Supported |
| Release tool | GoReleaser |
| License | MIT |
| Distribution | Single binary |

---

## 6. Installation Requirements

### 6.1 Single Binary

The tool must be buildable as a single binary.

Example:

```sh
go build ./cmd/gacils
```

### 6.2 Cross Compilation

The build system must support:

```text
GOOS=linux
GOOS=darwin
GOOS=windows
```

and:

```text
GOARCH=amd64
GOARCH=arm64
```

### 6.3 Installation Time

Installation should take less than 1 minute for a user with Go installed.

Example:

```sh
go install github.com/0n6k4v-Coder/github-action-ci-local-simulator/cmd/gacils@latest
```

---

## 7. Workflow Discovery Requirements

### 7.1 Default Workflow Directory

If no workflow path is specified, the tool must default to:

```text
.github/workflows
```

### 7.2 Explicit Workflow Path

The tool must allow:

```sh
gacils run -W <path>
```

where `<path>` can be:

```text
a workflow file
a directory containing workflow files
```

### 7.3 File Extensions

The tool must load:

```text
*.yml
*.yaml
```

when a directory is specified.

### 7.4 No Workflows Found

If no workflow files are found, the tool must return a clear error.

---

## 8. Workflow Parsing Requirements

The tool must parse GitHub Actions workflow YAML.

Minimum supported top-level fields:

```yaml
name
on
env
defaults
jobs
```

Minimum supported job fields:

```yaml
jobs.<id>.name
jobs.<id>.runs-on
jobs.<id>.needs
jobs.<id>.if
jobs.<id>.env
jobs.<id>.steps
jobs.<id>.strategy
jobs.<id>.outputs
jobs.<id>.timeout-minutes
jobs.<id>.defaults
```

Minimum supported step fields:

```yaml
steps[].id
steps[].name
steps[].if
steps[].run
steps[].uses
steps[].with
steps[].env
steps[].shell
steps[].working-directory
steps[].continue-on-error
steps[].timeout-minutes
```

---

## 9. Workflow Validation Requirements

The tool must reject invalid workflows with clear errors.

Required validation rules:

1. A workflow must have at least one job.
2. A job must have at least one step.
3. A step must have either `run` or `uses`.
4. A step must not have both `run` and `uses`.
5. Job IDs must not be empty.
6. Duplicate job IDs must be rejected.
7. `needs` dependencies must exist.
8. Dependency cycles must be rejected.
9. Unsupported `runs-on` labels must be rejected.
10. Unsupported shells must be rejected.
11. Unsupported actions must be rejected.
12. Unsupported expression functions must be rejected.

---

## 10. Runner Label Requirements

The tool must support Ubuntu runner labels.

| GitHub runner label | Local image |
|---|---|
| `ubuntu-latest` | `ubuntu:24.04` |
| `ubuntu-24.04` | `ubuntu:24.04` |
| `ubuntu-22.04` | `ubuntu:22.04` |

Unsupported labels:

```text
windows-*
macos-*
self-hosted
```

Unsupported labels must fail clearly.

---

## 11. Execution Model Requirements

The tool must use:

```text
1 job instance = 1 Docker container
1 step = 1 docker exec inside that container
```

This is required because steps must share:

```text
filesystem state
environment changes
PATH changes
step outputs
```

---

## 12. Shell Execution Requirements

The default shell must be:

```bash
bash -e {0}
```

The tool must execute step scripts using:

```bash
bash -e /tmp/gacils/steps/<step-id>/script.sh
```

The tool must not use:

```bash
bash --noprofile --norc -e {0}
```

Supported shells:

| Shell | Support |
|---|---:|
| `bash` | Required |
| default | Required, equivalent to bash |
| `sh` | Required |
| `pwsh` | Unsupported |
| `powershell` | Unsupported |
| `cmd` | Unsupported |

---

## 13. Environment Variable Requirements

### 13.1 Precedence

Environment variable precedence must be:

```text
step.env
  > job.env
  > workflow.env
  > gacils-generated GitHub/runner env
  > container base / OS env
```

### 13.2 Host Environment Isolation

Host OS environment variables must not be injected into job containers automatically.

Only explicitly provided secrets or workflow-defined environment variables may be injected.

### 13.3 Protected Variables

The following variables are managed by the tool:

```text
CI
GITHUB_ACTIONS
GITHUB_ENV
GITHUB_PATH
GITHUB_OUTPUT
GITHUB_STATE
GITHUB_STEP_SUMMARY
GITHUB_WORKSPACE
RUNNER_OS
RUNNER_ARCH
RUNNER_NAME
RUNNER_TEMP
RUNNER_TOOL_CACHE
```

If a workflow attempts to override protected variables, the tool should ignore the override and warn.

---

## 14. GitHub Context Requirements

The tool must provide a `github` context.

Minimum required values:

```text
github.action
github.action_path
github.actor
github.triggering_actor
github.api_url
github.base_ref
github.event
github.event_name
github.event_path
github.graphql_url
github.head_ref
github.job
github.ref
github.ref_name
github.ref_type
github.repository
github.repository_owner
github.run_attempt
github.run_id
github.run_number
github.server_url
github.sha
github.token
github.workflow
github.workflow_ref
github.workflow_sha
github.workspace
```

Default values:

| Key | Default |
|---|---|
| `github.actor` | `gacils-local` |
| `github.event` | `{}` |
| `github.event_name` | `push` |
| `github.ref` | Git-derived or `refs/heads/main` |
| `github.ref_name` | Git-derived or `main` |
| `github.ref_type` | `branch` or `tag` |
| `github.repository` | Git remote-derived or `local/<dirname>` |
| `github.sha` | Git SHA or `0000000000000000000000000000000000000000` |
| `github.token` | empty string |
| `github.workspace` | `/github/workspace` |

The tool must allow overriding context values through CLI flags.

---

## 15. Runner Context Requirements

The tool must provide a `runner` context.

Required values:

```text
runner.os
runner.arch
runner.name
runner.temp
runner.tool_cache
runner.debug
```

Default values:

| Key | Default |
|---|---|
| `runner.os` | `Linux` |
| `runner.arch` | `X64` or `ARM64` |
| `runner.name` | `gacils-local` |
| `runner.temp` | `/tmp/gacils` |
| `runner.tool_cache` | `/opt/hostedtoolcache` |
| `runner.debug` | `0` or `1` |

---

## 16. Environment File Requirements

The tool must support GitHub Actions environment files.

Required behavior:

```text
GITHUB_ENV    = one file per job
GITHUB_PATH   = one file per job
GITHUB_OUTPUT = one file per step
```

Paths:

```text
/tmp/gacils/jobs/<job-instance-id>/github_env
/tmp/gacils/jobs/<job-instance-id>/github_path
/tmp/gacils/steps/<step-id>/github_output
```

After each step:

1. Parse `GITHUB_OUTPUT` and store step outputs.
2. Parse `GITHUB_ENV` and merge into job environment.
3. Truncate `GITHUB_ENV` to avoid reapplying values.
4. Parse `GITHUB_PATH` and prepend to job PATH.
5. Truncate `GITHUB_PATH` to avoid reapplying values.

---

## 17. Step Output Requirements

The tool must support:

```yaml
- id: build
  run: echo "version=1.2.3" >> "$GITHUB_OUTPUT"
```

and later:

```yaml
${{ steps.build.outputs.version }}
```

Step outputs must be available to subsequent steps in the same job.

---

## 18. Job Output Requirements

The tool must support:

```yaml
jobs:
  build:
    outputs:
      version: ${{ steps.meta.outputs.version }}
```

Job outputs must be available to dependent jobs through:

```yaml
${{ needs.build.outputs.version }}
```

---

## 19. Dependency Requirements

The tool must support:

```yaml
jobs:
  test:
    needs: build
```

and:

```yaml
jobs:
  deploy:
    needs: [build, test]
```

Rules:

1. A job runs only after all dependencies are satisfied.
2. If a dependency fails, dependent jobs are skipped.
3. `if: always()` may allow a job to run despite failed dependencies.
4. `if: failure()` may allow a job to run only when dependencies failed.
5. Dependency cycles are invalid.

---

## 20. Matrix Requirements

The tool must support matrix expansion.

Example:

```yaml
strategy:
  matrix:
    python-version: ["3.11", "3.12", "3.13"]
```

The tool must create one job instance per matrix combination.

The tool must support:

```yaml
matrix:
  include:
  exclude:
```

Matrix values must be available through:

```yaml
${{ matrix.<key> }}
```

---

## 21. Conditional Execution Requirements

The tool must support step-level and job-level conditions:

```yaml
if: success()
if: failure()
if: always()
if: cancelled()
```

The tool must support expressions in `if` conditions.

---

## 22. continue-on-error Requirements

The tool must support:

```yaml
continue-on-error: true
```

Behavior:

```text
outcome    = actual result before continue-on-error
conclusion = final result used for job status and failure()
```

If a step fails but has:

```yaml
continue-on-error: true
```

then:

```text
outcome = failure
conclusion = success
```

The `failure()` function must use `conclusion`, not raw outcome.

---

## 23. Timeout Requirements

The tool must support:

```yaml
timeout-minutes: 10
```

at both job and step level.

Default timeout:

```text
360 minutes
```

When a timeout occurs:

1. Kill the running step.
2. Mark the step or job as failed.
3. Clean up containers.
4. Return exit code `5`.

---

## 24. Expression Requirements

### 24.1 Supported Contexts

The tool must support:

```text
matrix
env
github
runner
secrets
steps
needs
job
```

### 24.2 Supported Status Functions

The tool must support:

```text
success()
failure()
always()
cancelled()
```

### 24.3 Planned Common Functions

The tool should support:

```text
contains()
startsWith()
endsWith()
format()
```

### 24.4 Unsupported Functions

The tool must fail clearly for unsupported functions, including:

```text
hashFiles()
fromJson()
toJson()
join()
```

Star expressions such as:

```text
needs.*.result
github.event.labels.*.name
```

must fail clearly until implemented.

---

## 25. Action Simulation Requirements

### 25.1 Required Actions

The tool must support:

```text
actions/checkout@v3
actions/checkout@v4
actions/setup-python@v4
actions/setup-python@v5
```

### 25.2 Unsupported Actions

Unsupported actions must fail clearly.

Example:

```text
⚠️ workflow "Deploy" job "deploy" step 2 uses "actions/deploy-pages@v4": requires GitHub API — not supported locally
```

Exit code:

```text
3
```

---

## 26. actions/checkout Requirements

The tool must simulate local checkout.

Behavior:

1. Use the local workspace copied into `/github/workspace`.
2. If `.git` exists, mark repository safe.
3. If `.git` does not exist, optionally initialize a temporary Git repository.
4. Set GitHub environment variables.

Unsupported checkout inputs must fail clearly:

```text
repository other than current repository
ssh-key
submodules: true
remote fetch requiring network
LFS
```

---

## 27. actions/setup-python Requirements

The tool must simulate Python setup.

Default mode:

```text
--python-mode=image
```

Behavior:

1. Resolve requested Python version.
2. If exactly one version is requested, use:

```text
python:<version>-slim
```

3. If multiple Python versions are requested in one job, fail clearly.
4. Make `python` and `pip` available in PATH.

Future toolcache mode may use:

```text
/opt/hostedtoolcache/Python/<version>/x64
```

---

## 28. CRLF Handling Requirements

The tool must support CRLF handling modes.

Required flag:

```text
--crlf
```

Modes:

| Mode | Behavior |
|---|---|
| `convert` | Convert CRLF to LF for likely script files and warn |
| `preserve` | Do not convert; closest to GitHub behavior |
| `error` | Fail if CRLF is detected in likely script files |

Default:

```text
convert
```

Warning example:

```text
⚠️ Converted CRLF → LF for scripts/test.sh
   GitHub Actions would fail here with /bin/bash^M: bad interpreter
```

---

## 29. Workspace Copy Requirements

The tool must copy the local workspace into the container.

Container workspace path:

```text
/github/workspace
```

The tool should prefer tar stream copy over bind mounts.

Reasons:

```text
Windows path compatibility
macOS file sharing issues
Linux UID/GID issues
CRLF normalization control
closer GitHub Actions behavior
```

---

## 30. Secrets Requirements

The tool must support secrets.

Required flags:

```text
--secret-file
--secret
```

Example:

```sh
gacils run --secret-file .env.secrets
```

Secrets must be available through:

```text
${{ secrets.KEY }}
```

Secrets must be masked in logs.

### 30.1 GITHUB_TOKEN

The tool must simulate:

```text
secrets.GITHUB_TOKEN
```

as an empty string by default.

If referenced, warn:

```text
⚠️ GitHub token is simulated as an empty string locally.
   GitHub API calls will fail.
```

---

## 31. Cache Requirements

Cache support is planned for v2.

The tool should simulate:

```yaml
actions/cache@v4
```

using a local cache directory.

Possible location:

```text
~/.gacils/cache/
```

Unsupported cache behavior must fail clearly until implemented.

---

## 32. Artifact Requirements

Artifact support is planned for v2.

The tool should simulate:

```text
actions/upload-artifact@v4
actions/download-artifact@v4
```

Possible local artifact directory:

```text
.gacils/artifacts/<run-id>/
```

---

## 33. Service Container Requirements

Service container support is planned for v2.

The tool should support:

```yaml
services:
  postgres:
    image: postgres:16
```

Possible behavior:

1. Create Docker network.
2. Start service containers.
3. Use service ID as hostname.
4. Tear down network after job.

---

## 34. Parallel Execution Requirements

The tool must support:

```text
--parallel
```

Values:

| Value | Meaning |
|---:|---|
| `0` | Default. Run independent jobs concurrently. |
| `1` | Sequential execution. |
| `N` | Maximum N concurrent jobs. |

The tool must respect:

```yaml
strategy:
  max-parallel: N
```

---

## 35. CLI Requirements

The tool must provide:

```text
gacils run
gacils list
gacils init
gacils doctor
gacils clean
gacils setup python <version>
gacils version
```

### 35.1 `gacils run`

Required flags:

```text
-W, --workflow
--job
--dry-run
--secret-file
--secret
--platform
--parallel
--crlf
--python-mode
--offline
--log-dir
--no-timeout
--keep-containers
--verbose
--no-color
--strict
```

### 35.2 `gacils doctor`

The tool must check:

```text
Docker availability
Docker socket access
Docker API negotiation
platform support
~/.gacils writable
disk space
required images
optional Python caches
Git availability
```

### 35.3 `gacils clean`

The tool must support:

```text
--logs
--cache
--containers
--volumes
--all
--older-than
--force
--include-config
```

The tool must not remove config by default.

---

## 36. Logging Requirements

The tool must store logs.

Default log directory:

```text
~/.gacils/logs/<run-id>/
```

Recommended log files:

```text
gacils.log
plan.json
result.json
jobs/<job-id>/job.log
jobs/<job-id>/step-<n>-<id>.log
```

The tool must support:

```text
--log-dir
```

---

## 37. Output Requirements

The tool must provide colored console output.

Required status indicators:

```text
success
failure
skipped
warning
cancelled
```

Color must be disabled when:

```text
--no-color is used
NO_COLOR=1
stdout is not a TTY
```

---

## 38. Exit Code Requirements

The tool must use clear exit codes.

| Code | Meaning |
|---:|---|
| `0` | Success |
| `1` | Workflow/job/step failure |
| `2` | Invalid usage or invalid workflow file |
| `3` | Unsupported feature detected |
| `4` | Docker unavailable |
| `5` | Timeout |
| `6` | Cancelled by user |

---

## 39. Error Handling Requirements

Unsupported features must fail clearly.

The tool must not:

```text
silently skip unsupported features
crash without a clear message
continue with ambiguous behavior
```

Error format:

```text
⚠️ <location> uses "<feature>": <reason> — not supported locally
```

Example:

```text
⚠️ workflow "Deploy" job "deploy" step 2 uses "actions/deploy-pages@v4": requires GitHub API — not supported locally
```

---

## 40. Compatibility Requirements

The tool targets high compatibility for the supported feature subset.

The tool must:

```text
run .github/workflows/*.yml without modification for supported features
fail clearly for unsupported features
provide a GitHub-compatible context where practical
```

The tool does not target full GitHub Actions compatibility.

---

## 41. Cross-Platform Requirements

### 41.1 Windows

The tool must:

```text
handle C:\ paths
support Docker Desktop named pipe
convert CRLF to LF by default with warning
enable ANSI color support
work in PowerShell, CMD, and Git Bash
```

### 41.2 macOS

The tool must:

```text
support Apple Silicon
support Intel
support Docker Desktop socket
allow linux/amd64 emulation for compatibility
allow linux/arm64 for speed
```

### 41.3 Linux

The tool must:

```text
support Ubuntu
support Debian
support Docker Engine
support rootless Docker when reachable
support unix Docker socket
```

---

## 42. Signal Handling Requirements

The tool must handle:

```text
Ctrl+C
SIGINT
SIGTERM
```

On cancellation:

1. Cancel running context.
2. Stop running exec processes.
3. Kill job containers.
4. Remove containers unless `--keep-containers` is used.
5. Return exit code `6`.

---

## 43. Security Requirements

The tool must:

1. Not leak host environment variables into job containers.
2. Mask secrets in logs.
3. Not call GitHub APIs.
4. Not require GitHub credentials for local execution.
5. Warn when `GITHUB