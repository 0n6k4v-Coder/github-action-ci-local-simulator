# github-action-ci-local-simulator — Locked Design v3.2

Document: `docs/DESIGN.md`

Status: Locked for implementation

Repo name:

```text
github-action-ci-local-simulator
```

Binary name:

```text
gacils
```

Go module:

```text
github.com/yourname/github-action-ci-local-simulator
```

License:

```text
MIT
```

---

## 1. Project Goal

`gacils` is a personal open-source tool for running GitHub Actions workflows locally using Docker.

Primary goals:

- Run `.github/workflows/*.yml` locally before pushing.
- Reduce GitHub Actions minute usage.
- Debug CI failures locally.
- Work on Windows 10/11, macOS, and Linux.
- Use Docker Desktop or Docker Engine.
- Fail clearly on unsupported features.
- Do not silently skip unsupported behavior.
- Do not crash on unsupported workflow syntax when possible.

This is not intended to replace GitHub Actions or compete with `act`. It is a learning and personal productivity tool.

---

## 2. Locked Technology Stack

| Item | Decision |
|---|---|
| Language | Go |
| Go version | Go 1.26.x |
| CLI framework | Cobra |
| YAML parser | `gopkg.in/yaml.v3` |
| Docker client | Official Docker Engine API Go SDK |
| Expression evaluator | `expr-lang/expr` |
| Colored output | `fatih/color` |
| Secrets file parsing | `godotenv` |
| Release tool | GoReleaser |
| Testing | Go testing + testify |
| License | MIT |

Docker client initialization must use:

```go
client.FromEnv
client.WithAPIVersionNegotiation()
```

---

## 3. Supported Platforms

| Platform | Support |
|---|---:|
| Windows 10/11 | Primary |
| macOS Apple Silicon | Supported |
| macOS Intel | Supported |
| Linux Ubuntu | Supported |
| Linux Debian | Supported |

Docker support:

| Docker runtime | Support |
|---|---:|
| Docker Desktop on Windows | Supported |
| Docker Desktop on macOS | Supported |
| Docker Engine on Linux | Supported |
| Rootless Docker | Supported if Docker API is reachable |

Default Docker endpoints:

| OS | Endpoint |
|---|---|
| Linux | `unix:///var/run/docker.sock` |
| macOS | `unix:///var/run/docker.sock` |
| Windows | `npipe:////./pipe/docker_engine` |

Environment variables respected:

```text
DOCKER_HOST
DOCKER_CONTEXT
DOCKER_TLS_VERIFY
DOCKER_CERT_PATH
```

---

## 4. High-Level Architecture

```text
.github/workflows/*.yml
        ↓
Workflow Loader
        ↓
Validator
        ↓
Expression Resolver
        ↓
Matrix Expander
        ↓
Job DAG Builder
        ↓
Execution Planner
        ↓
Job Runner
        ↓
Docker Client
        ↓
Job Container
        ↓
Step Execution
        ↓
Result Collection
        ↓
Console / JSON / Logs / Exit Code
```

---

## 5. Core Execution Model

Locked model:

```text
1 job instance = 1 Docker container
1 step = 1 docker exec inside that container
```

Reason:

- Steps must share filesystem state.
- Steps must share environment changes.
- `GITHUB_ENV` and `GITHUB_PATH` must affect later steps in the same job.
- `GITHUB_OUTPUT` must be available to later steps through `steps.<id>.outputs`.

---

## 6. MVP Scope

### Supported workflow syntax

```yaml
name
on
env
defaults.run.shell
defaults.run.working-directory
jobs
jobs.<id>.name
jobs.<id>.runs-on
jobs.<id>.needs
jobs.<id>.if
jobs.<id>.env
jobs.<id>.steps
jobs.<id>.strategy.matrix
jobs.<id>.strategy.fail-fast
jobs.<id>.strategy.max-parallel
jobs.<id>.outputs
jobs.<id>.timeout-minutes
jobs.<id>.defaults.run.shell
jobs.<id>.defaults.run.working-directory
```

### Supported step syntax

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

### Supported runner labels

| GitHub runner label | Local image |
|---|---|
| `ubuntu-latest` | `ubuntu:24.04` |
| `ubuntu-24.04` | `ubuntu:24.04` |
| `ubuntu-22.04` | `ubuntu:22.04` |
| `windows-*` | Unsupported |
| `macos-*` | Unsupported |
| `self-hosted` | Unsupported |

### Supported actions in MVP

```text
actions/checkout@v3
actions/checkout@v4
actions/setup-python@v4
actions/setup-python@v5
```

Other actions must fail clearly.

---

## 7. Out of Scope

The following are out of scope and must fail clearly when used:

```text
GitHub API calls
GitHub deployments
GitHub releases
GitHub Pages
OIDC token issuance
Reusable workflows
Self-hosted runner registration
GitHub permissions enforcement
Composite actions
Local actions unless explicitly supported later
Windows containers
macOS containers
GitHub step summaries
Advanced expression functions not listed as supported
```

Unsupported feature error format:

```text
⚠️ <location> uses "<feature>": <reason> — not supported locally
```

Example:

```text
⚠️ workflow "Deploy" job "deploy" step 2 uses "actions/deploy-pages@v4": requires GitHub API — not supported locally
```

Exit code for unsupported feature:

```text
3
```

---

## 8. Directory Structure

```text
github-action-ci-local-simulator/
├── cmd/
│   └── gacils/
│       └── main.go
│
├── internal/
│   ├── app/
│   │   ├── app.go
│   │   └── version.go
│   │
│   ├── cli/
│   │   ├── root.go
│   │   ├── run.go
│   │   ├── list.go
│   │   ├── initcmd.go
│   │   ├── doctor.go
│   │   ├── clean.go
│   │   ├── setup.go
│   │   └── flags.go
│   │
│   ├── workflow/
│   │   ├── model.go
│   │   ├── loader.go
│   │   ├── validator.go
│   │   ├── normalize.go
│   │   └── unsupported.go
│   │
│   ├── expression/
│   │   ├── evaluator.go
│   │   ├── context.go
│   │   ├── functions.go
│   │   ├── interpolate.go
│   │   └── unsupported.go
│   │
│   ├── plan/
│   │   ├── matrix.go
│   │   ├── dag.go
│   │   ├── planner.go
│   │   └── jobinstance.go
│   │
│   ├── runner/
│   │   ├── runner.go
│   │   ├── jobrunner.go
│   │   ├── steprunner.go
│   │   ├── shell.go
│   │   ├── envfiles.go
│   │   ├── outputs.go
│   │   ├── timeout.go
│   │   └── result.go
│   │
│   ├── dockerx/
│   │   ├── client.go
│   │   ├── container.go
│   │   ├── image.go
│   │   ├── copy.go
│   │   ├── exec.go
│   │   ├── platform.go
│   │   ├── socket.go
│   │   └── volume.go
│   │
│   ├── actions/
│   │   ├── registry.go
│   │   ├── actionref.go
│   │   ├── checkout.go
│   │   ├── setuppython.go
│   │   └── unsupported.go
│   │
│   ├── fsx/
│   │   ├── workspace.go
│   │   ├── tar.go
│   │   ├── crlf.go
│   │   ├── gitfiles.go
│   │   └── permissions.go
│   │
│   ├── contextx/
│   │   ├── github.go
│   │   ├── runner.go
│   │   ├── git.go
│   │   └── event.go
│   │
│   ├── secrets/
│   │   ├── envfile.go
│   │   ├── provider.go
│   │   └── masker.go
│   │
│   ├── report/
│   │   ├── console.go
│   │   ├── color.go
│   │   ├── json.go
│   │   └── summary.go
│   │
│   ├── logs/
│   │   ├── logger.go
│   │   └── runlog.go
│   │
│   └── errorsx/
│       ├── codes.go
│       └── unsupported.go
│
├── testdata/
│   ├── workflows/
│   │   ├── simple-run.yml
│   │   ├── env-precedence.yml
│   │   ├── matrix-python.yml
│   │   ├── needs.yml
│   │   ├── needs-outputs.yml
│   │   ├── continue-on-error.yml
│   │   ├── timeout.yml
│   │   ├── github-context.yml
│   │   ├── runner-context.yml
│   │   ├── contains.yml
│   │   ├── unsupported-action.yml
│   │   ├── unsupported-expression.yml
│   │   └── step-summary.yml
│   └── events/
│       └── pull_request.json
│
├── docs/
│   ├── DESIGN.md
│   ├── architecture.md
│   ├── compatibility.md
│   ├── windows.md
│   ├── macos.md
│   ├── linux.md
│   └── troubleshooting.md
│
├── .gitignore
├── .dockerignore
├── .goreleaser.yaml
├── Makefile
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

---
## Repository Ignore Files

### `.gitignore`

Prevents committing build output, secrets, logs, IDE files, and OS-generated files to the repository.

### `.dockerignore`

Prevents unnecessary files from being included in the Docker build context, reducing build time and image size.

---

---

## 9. Host Data Layout

```text
~/.gacils/
├── config.yaml
├── cache/
│   └── python/
│       ├── 3.11/
│       ├── 3.12/
│       ├── 3.13/
│       ├── 3.11.json
│       ├── 3.12.json
│       └── 3.13.json
└── logs/
    └── <run-id>/
        ├── gacils.log
        ├── plan.json
        ├── result.json
        └── jobs/
```

Docker named volume:

```text
gacils-python-cache
```

Mount path for Python tool cache:

```text
/opt/hostedtoolcache
```

---

## 10. Container Directory Layout

```text
/github/workspace/
/tmp/gacils/
├── event.json
├── jobs/
│   └── <job-instance-id>/
│       ├── github_env
│       └── github_path
└── steps/
    └── <step-id>/
        ├── script.sh
        └── github_output
```

If Python toolcache mode is used:

```text
/opt/hostedtoolcache/
└── Python/
    ├── 3.11.10/
    │   └── x64/
    ├── 3.12.8/
    │   └── x64/
    └── 3.13.1/
        └── x64/
```

---

## 11. Workflow Parsing Rules

### File discovery

Default:

```text
.github/workflows/*.yml
.github/workflows/*.yaml
```

User can override:

```sh
gacils run -W .github/workflows/ci.yml
```

or:

```sh
gacils run -W .github/workflows
```

### YAML quirks

The loader must handle:

```text
on as a YAML key
on possibly parsed as boolean true in some YAML modes
runs-on as string or list
needs as string or list
env values as string, number, or boolean
matrix values as string, number, or boolean
```

### Validation

The validator must reject or fail clearly on:

```text
duplicate job IDs
missing job ID
step with both run and uses
step with neither run nor uses
unsupported runs-on
unsupported shell
unsupported actions
unsupported expression functions
unsupported top-level features
cycle in needs
missing needs dependency
```

---

## 12. Execution Algorithm

```text
1. Load workflow files.
2. Parse YAML.
3. Normalize syntax quirks.
4. Validate supported features.
5. Build GitHub context.
6. Build runner context.
7. Expand matrix jobs.
8. Resolve expressions needed for planning.
9. Build job dependency graph.
10. Topologically sort jobs.
11. Create execution plan.
12. If --dry-run, print plan and stop.
13. Run jobs according to dependency order and parallel settings.
14. For each job instance:
    a. Select job image.
    b. Pull image if needed and allowed.
    c. Create container.
    d. Start container.
    e. Copy workspace into /github/workspace.
    f. Create job-level GITHUB_ENV and GITHUB_PATH files.
    g. Run steps sequentially.
    h. Collect step outputs.
    i. Resolve job outputs.
    j. Stop and remove container.
15. Aggregate job results.
16. Write logs and JSON report.
17. Print console summary.
18. Exit with correct exit code.
```

---

## 13. Matrix Expansion

Given:

```yaml
strategy:
  matrix:
    os: [ubuntu-latest]
    python-version: ["3.11", "3.12", "3.13"]
```

Create one job instance per combination.

Support:

```yaml
matrix:
  include:
  exclude:
```

Algorithm:

```text
1. Compute Cartesian product of matrix arrays.
2. Apply exclude rules.
3. Apply include rules.
4. Create one job instance per final combination.
5. Expose values through matrix context.
```

Matrix values must be available as:

```text
${{ matrix.<key> }}
```

Example:

```text
${{ matrix.python-version }}
```

---

## 14. Job Dependency Rules

```yaml
jobs:
  build:
  test:
    needs: build
  deploy:
    needs: [build, test]
```

Rules:

```text
A job runs only after all needs dependencies are satisfied.
If a dependency fails, dependent jobs are skipped.
If a job has if: always(), it may run even if dependencies failed.
If a job has if: failure(), it runs only if dependencies failed.
If a job has if: cancelled(), it runs only if cancelled.
Cycles are errors.
```

For matrix jobs:

```text
A dependent job waits for all matrix legs of the needed job.
```

---

## 15. `github.*` Context Defaults

Locked defaults:

| Key | Default | Source / Fallback |
|---|---|---|
| `github.action` | `""` | Empty |
| `github.action_path` | `""` | Empty |
| `github.actor` | `gacils-local` | Override with `--actor` |
| `github.triggering_actor` | Same as `github.actor` | Override with `--actor` |
| `github.api_url` | `https://api.github.com` | Constant |
| `github.base_ref` | `""` | From event payload if pull_request |
| `github.event` | `{}` | Empty object or `--event-payload` |
| `github.event_name` | `push` | Override with `--event` |
| `github.event_path` | `/tmp/gacils/event.json` | Generated file |
| `github.graphql_url` | `https://api.github.com/graphql` | Constant |
| `github.head_ref` | `""` | From event payload if pull_request |
| `github.job` | Current job ID | Runtime |
| `github.ref` | `refs/heads/main` or Git-derived | See Git detection |
| `github.ref_name` | `main` or Git-derived | See Git detection |
| `github.ref_type` | `branch` or `tag` | See Git detection |
| `github.repository` | `owner/repo` or `local/<dirname>` | See Git detection |
| `github.repository_owner` | `owner` or `local` | Parsed from repository |
| `github.run_attempt` | `1` | Override with `--run-attempt` |
| `github.run_id` | `1` | Override with `--run-id` |
| `github.run_number` | `1` | Override with `--run-number` |
| `github.server_url` | `https://github.com` | Constant |
| `github.sha` | Git SHA or `0000000000000000000000000000000000000000` | See Git detection |
| `github.token` | `""` | Empty + warning if referenced |
| `github.workflow` | Workflow name or file name | Runtime |
| `github.workflow_ref` | `<repository>/.github/workflows/<file>@<ref>` | Best-effort |
| `github.workflow_sha` | `""` | Not computed |
| `github.workspace` | `/github/workspace` | Constant |

---

## 16. Git Detection Rules

### `github.ref`

Order:

```text
1. If inside a Git repository:
   a. If current branch is normal:
      refs/heads/<branch>
   b. If detached HEAD:
      Try:
        git describe --tags --exact-match
      If tag found:
        refs/tags/<tag>
      Else:
        refs/heads/main
2. If not inside a Git repository:
   refs/heads/main
3. If user provides --ref:
   Use user value.
```

### `github.ref_name`

Derived from `github.ref`:

```text
refs/heads/main -> main
refs/tags/v1.0.0 -> v1.0.0
```

### `github.ref_type`

```text
refs/heads/* -> branch
refs/tags/* -> tag
fallback -> branch
```

### `github.sha`

Order:

```text
1. git rev-parse HEAD
2. If unavailable:
   0000000000000000000000000000000000000000
3. If user provides --sha:
   Use user value.
```

### `github.repository`

Order:

```text
1. Read:
   git remote get-url origin

2. Parse:
   git@github.com:owner/repo.git -> owner/repo
   https://github.com/owner/repo.git -> owner/repo

3. If no remote:
   local/<current-directory-name>

4. If directory name is unusable:
   local/local

5. If user provides --repository:
   Use user value.
```

---

## 17. `runner.*` Context Defaults

Locked defaults:

| Key | Default | Notes |
|---|---|---|
| `runner.os` | `Linux` | Constant |
| `runner.arch` | `X64` or `ARM64` | Depends on platform |
| `runner.name` | `gacils-local` | Override with `--runner-name` |
| `runner.temp` | `/tmp/gacils` | Constant inside container |
| `runner.tool_cache` | `/opt/hostedtoolcache` | GitHub-compatible |
| `runner.debug` | `0` or `1` | `1` when verbose |

Platform mapping:

| Docker platform | `runner.arch` |
|---|---|
| `linux/amd64` | `X64` |
| `linux/arm64` | `ARM64` |
| other | Unsupported |

Environment variables:

```bash
RUNNER_OS=Linux
RUNNER_ARCH=X64
RUNNER_NAME=gacils-local
RUNNER_TEMP=/tmp/gacils
RUNNER_TOOL_CACHE=/opt/hostedtoolcache
RUNNER_DEBUG=0
```

---

## 18. Environment Variable Precedence

Locked precedence from highest to lowest:

```text
step.env
  > job.env
  > workflow.env
  > gacils-generated GitHub/runner env
  > container base / OS env
```

Host OS environment variables are not injected into job containers automatically.

Only these host-side values may be used by the `gacils` process itself:

```text
DOCKER_HOST
DOCKER_CONTEXT
DOCKER_TLS_VERIFY
DOCKER_CERT_PATH
NO_COLOR
GACILS_*
```

Secrets may be provided explicitly:

```sh
--secret-file .env
--secret KEY=value
```

---

## 19. Protected Environment Variables

These variables are managed by `gacils`:

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

If a workflow attempts to override them, `gacils` should ignore the workflow value and warn:

```text
⚠️ Environment variable "GITHUB_OUTPUT" is managed by gacils and cannot be overridden.
   The workflow-provided value will be ignored.
```

---

## 20. Shell Execution

Default shell:

```bash
bash -e {0}
```

Locked execution command:

```bash
bash -e /tmp/gacils/steps/<step-id>/script.sh
```

Do not use:

```bash
bash --noprofile --norc -e /tmp/gacils/steps/<step-id>/script.sh
```

Supported shells:

| `shell` value | Supported | Execution |
|---|---:|---|
| `bash` | Yes | `bash -e {0}` |
| default | Yes | `bash -e {0}` |
| `sh` | Yes | `sh -e {0}` |
| `pwsh` | No | Fail |
| `powershell` | No | Fail |
| `cmd` | No | Fail |

Unsupported shell error:

```text
⚠️ job "test" step 3 uses shell "pwsh" — only bash/sh are supported locally
```

---

## 21. `defaults.run` Support

Support:

```yaml
defaults:
  run:
    shell: bash
    working-directory: ./app
```

Support at job level:

```yaml
jobs:
  test:
    defaults:
      run:
        shell: sh
        working-directory: ./backend
```

Precedence:

```text
step.shell
  > job.defaults.run.shell
  > workflow.defaults.run.shell
  > bash
```

```text
step.working-directory
  > job.defaults.run.working-directory
  > workflow.defaults.run.working-directory
  > /github/workspace
```

---

## 22. Environment Files

Locked behavior:

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

Lifecycle:

```text
1. At job start, create job-level github_env and github_path files.
2. For each step, create a step-specific github_output file.
3. Set environment variables:
   GITHUB_ENV=/tmp/gacils/jobs/<job-instance-id>/github_env
   GITHUB_PATH=/tmp/gacils/jobs/<job-instance-id>/github_path
   GITHUB_OUTPUT=/tmp/gacils/steps/<step-id>/github_output
4. Run step.
5. Parse GITHUB_OUTPUT and store step outputs.
6. Parse GITHUB_ENV and merge into job environment.
7. Truncate GITHUB_ENV to avoid reapplying values.
8. Parse GITHUB_PATH and prepend to job PATH.
9. Truncate GITHUB_PATH to avoid reapplying values.
```

Supported file formats:

```text
KEY=value
KEY<<DELIMITER
multiline value
DELIMITER
```

---

## 23. `GITHUB_STEP_SUMMARY`

Locked decision:

```text
GITHUB_STEP_SUMMARY is out of scope.
```

If detected, fail clearly.

Detection is best-effort static scanning of:

```text
step.run
step.env values
step.with values
job.env values
workflow.env values
```

If found:

```text
⚠️ job "test" step 2 references GITHUB_STEP_SUMMARY — Markdown step summaries are not supported locally
```

Exit code:

```text
3
```

---

## 24. Step Result Model

Each step has:

```text
outcome
conclusion
```

Definitions:

```text
outcome = actual result before continue-on-error
conclusion = final result used for job status and failure()
```

Logic:

```text
if exit code != 0:
  outcome = failure
else:
  outcome = success

conclusion = outcome

if continue-on-error == true and conclusion == failure:
  conclusion = success
```

---

## 25. `continue-on-error` Behavior

Example:

```yaml
steps:
  - id: fail
    run: exit 1
    continue-on-error: true

  - id: failure_step
    if: failure()
    run: echo "should not run"

  - id: success_step
    if: success()
    run: echo "should run"
```

Locked result:

```text
fail.outcome = failure
fail.conclusion = success
failure_step = skipped
success_step = success
job = success
```

Console output should show:

```text
✗ Step "fail" failed with exit code 1
⚠️ continue-on-error=true → step conclusion: success
```

---

## 26. Status Functions

Supported:

```text
success()
failure()
always()
cancelled()
```

Locked semantics:

```text
success()
  true if no previous step has conclusion failure or cancelled

failure()
  true if any previous step has conclusion failure

always()
  true regardless of previous step results

cancelled()
  true if the current run/job was cancelled
```

Important:

```text
failure() uses step conclusion, not raw outcome.
```

Therefore a failed step with `continue-on-error: true` does not make `failure()` true.

---

## 27. `needs.*` Context

Supported:

```text
needs.<job_id>.outputs.<name>
needs.<job_id>.result
```

Possible `result` values:

```text
success
failure
cancelled
skipped
```

Example:

```json
{
  "needs": {
    "build": {
      "outputs": {
        "version": "1.2.3"
      },
      "result": "success"
    }
  }
}
```

---

## 28. Job Outputs Flow

Given:

```yaml
jobs:
  build:
    outputs:
      version: ${{ steps.meta.outputs.version }}
    steps:
      - id: meta
        run: echo "version=1.2.3" >> "$GITHUB_OUTPUT"

  deploy:
    needs: build
    steps:
      - run: echo "${{ needs.build.outputs.version }}"
```

Flow:

```text
1. Run build job.
2. Parse step outputs from GITHUB_OUTPUT.
3. After build completes, resolve build.outputs.version.
4. Store job outputs in JobResult.
5. When deploy starts, inject needs.build.outputs.version.
```

Expected output:

```text
1.2.3
```

---

## 29. Dependency Output Edge Cases

### Needed job skipped

```text
needs.A.result = skipped
needs.A.outputs = {}
```

If referenced:

```text
${{ needs.A.outputs.version }}
```

Result:

```text
empty string
warning
```

Warning:

```text
⚠️ job "B" references needs.A.outputs.version, but job "A" was skipped; value is empty
```

### Needed job failed

```text
needs.A.result = failure
```

Outputs:

```text
Outputs set before failure are exposed.
Missing outputs resolve to empty string with warning.
```

Warning:

```text
⚠️ job "B" references needs.A.outputs.version, but output "version" was not set before job "A" failed
```

### Needed job cancelled

```text
needs.A.result = cancelled
needs.A.outputs = {}
```

Referenced outputs:

```text
empty string + warning
```

---

## 30. Matrix Job Output Policy

For a needed matrix job:

```yaml
jobs:
  test:
    strategy:
      matrix:
        python-version: ["3.11", "3.12", "3.13"]
```

### `needs.test.result`

Supported.

Rules:

```text
success   = all matrix legs success
failure   = any matrix leg failure
cancelled = cancelled and no failure
skipped   = all matrix legs skipped
```

### `needs.test.outputs.*`

Not supported when the needed matrix job has multiple legs.

Error:

```text
⚠️ job "report" references needs.test.outputs.coverage, but "test" is a matrix job with multiple legs.
   Output aggregation from matrix jobs is not supported locally.
```

Exit code:

```text
3
```

Exception:

```text
If matrix expansion produces only one leg, output access is allowed.
```

---

## 31. Expression Contexts

Supported contexts:

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

Examples:

```text
${{ matrix.python-version }}
${{ env.MY_VAR }}
${{ github.ref }}
${{ runner.os }}
${{ secrets.MY_SECRET }}
${{ steps.build.outputs.version }}
${{ needs.build.outputs.version }}
${{ needs.build.result }}
${{ job.status }}
```

---

## 32. Expression Support by Phase

| Feature | Phase 1 | Phase 2 | Phase 4 |
|---|---:|---:|---:|
| `matrix.*` | Yes | Yes | Yes |
| `env.*` | Yes | Yes | Yes |
| `github.*` | Yes | Yes | Yes |
| `runner.*` | Yes | Yes | Yes |
| `secrets.*` | Yes | Yes | Yes |
| `steps.*.outputs.*` | Yes | Yes | Yes |
| `needs.*.outputs.*` | Yes | Yes | Yes |
| `needs.*.result` | Yes | Yes | Yes |
| `job.status` | Yes | Yes | Yes |
| `success()` | Yes | Yes | Yes |
| `failure()` | Yes | Yes | Yes |
| `always()` | Yes | Yes | Yes |
| `cancelled()` | Yes | Yes | Yes |
| `contains()` | No | Yes | Yes |
| `startsWith()` | No | Yes | Yes |
| `endsWith()` | No | Yes | Yes |
| `format()` | No | Yes | Yes |
| `hashFiles()` | No | No | Yes |
| `fromJson()` | No | No | Yes |
| `toJson()` | No | No | Yes |
| `join()` | No | No | Yes |
| star expressions | No | No | Yes |

---

## 33. Common Function Semantics

### `contains(search, item)`

String behavior:

```text
Returns true if search contains item.
Case-insensitive.
```

Examples:

```text
contains('Hello World', 'world') -> true
contains('refs/heads/main', 'main') -> true
contains('refs/heads/dev', 'main') -> false
```

Array behavior:

```text
Returns true if item equals any array element.
```

Examples:

```text
contains(['a', 'b'], 'b') -> true
contains(['a', 'b'], 'z') -> false
```

Null behavior:

```text
contains(null, 'x') -> false
contains('abc', null) -> false
contains([], 'x') -> false
```

### `startsWith(search, item)`

```text
Returns true if search starts with item.
Case-insensitive.
```

Examples:

```text
startsWith('refs/heads/main', 'refs/heads/') -> true
startsWith('refs/tags/v1', 'refs/heads/') -> false
```

### `endsWith(search, item)`

```text
Returns true if search ends with item.
Case-insensitive.
```

Examples:

```text
endsWith('refs/heads/main', 'main') -> true
endsWith('refs/heads/main', 'dev') -> false
```

### `format(string, args...)`

Example:

```text
format('Hello {0}', 'world') -> Hello world
format('{0}/{1}', 'refs/heads', 'main') -> refs/heads/main
```

Escaping:

```text
{{ -> {
}} -> }
```

---

## 34. Unsupported Expression Functions

The following must fail clearly until implemented:

```text
hashFiles()
fromJson()
toJson()
join()
star expressions such as needs.*.result
object filtering such as github.event.labels.*.name
```

Example:

```yaml
- run: echo "${{ hashFiles('**/requirements.txt') }}"
```

Error:

```text
⚠️ expression function "hashFiles" is not supported locally yet
```

Example:

```yaml
if: contains(join(needs.*.result, ','), 'failure')
```

Error:

```text
⚠️ expression function "join" is not supported locally yet
```

or:

```text
⚠️ star expressions like "needs.*.result" are not supported locally yet
```

Exit code:

```text
3
```

---

## 35. CRLF Handling

Locked default:

```text
--crlf=convert
```

Modes:

| Mode | Behavior |
|---|---|
| `convert` | Convert CRLF to LF for likely text/script files and warn |
| `preserve` | Do not convert; closest to GitHub behavior |
| `error` | Fail if CRLF is detected in likely script files |

Warning example:

```text
⚠️ Converted CRLF → LF for scripts/test.sh
   GitHub Actions would fail here with /bin/bash^M: bad interpreter
```

Files to convert:

```text
Files with shebang #!
Files with executable bit in Git index
Files with extensions .sh, .bash, .zsh, .ksh
Text files detected as CRLF
```

Files not to convert:

```text
Binary files
.png, .jpg, .jpeg, .gif, .zip, .tar, .gz, .pdf
Files marked binary in .gitattributes
```

---

## 36. Workspace Copy Strategy

Do not use bind mounts for MVP.

Use tar stream copy:

```text
Host workspace -> tar stream -> Docker CopyToContainer -> /github/workspace
```

File selection:

```text
If inside Git repository:
  Use:
    git ls-files --cached --others --exclude-standard
Else:
  Copy all files except common generated directories.
```

Always exclude by default:

```text
.gacils/
node_modules/
__pycache__/
.venv/
dist/
build/
target/
```

Preserve POSIX executable bits when possible.

On Windows, use Git index executable bit when available.

---

## 37. Image Selection

Locked rule:

```text
If a job instance uses actions/setup-python with one resolved Python version:
  use python:<version>-slim
Else:
  use ubuntu image based on runs-on
```

Examples:

| Condition | Image |
|---|---|
| `ubuntu-latest`, no setup-python | `ubuntu:24.04` |
| `ubuntu-latest`, setup-python 3.11 | `python:3.11-slim` |
| `ubuntu-latest`, setup-python 3.12 | `python:3.12-slim` |
| `ubuntu-latest`, setup-python 3.13 | `python:3.13-slim` |
| `ubuntu-22.04`, no setup-python | `ubuntu:22.04` |
| `ubuntu-22.04`, setup-python | `python:<version>-slim` + warning |

Warning:

```text
⚠️ job "test" uses actions/setup-python.
   Local image is python:3.12-slim (Debian-based), not the full GitHub Ubuntu runner.
   Some preinstalled tools may differ.
```

Multiple Python versions in one job:

```text
Unsupported in MVP.
```

Error:

```text
⚠️ job "test" requests multiple Python versions in one job (3.11, 3.12).
   MVP supports one Python version per job instance.
   Use matrix strategy or run separate jobs.
```

---

## 38. `actions/checkout` Simulation

Supported:

```text
actions/checkout@v3
actions/checkout@v4
```

Local behavior:

```text
1. Workspace is already copied to /github/workspace.
2. If .git exists:
   git config --global --add safe.directory /github/workspace
3. If .git does not exist:
   optionally initialize a temporary Git repository:
     git init
     git add -A
     git commit -m "gacils local checkout"
4. Set GitHub environment variables:
   GITHUB_REPOSITORY
   GITHUB_SHA
   GITHUB_REF
   GITHUB_REF_NAME
```

Unsupported inputs:

```text
repository other than current repository
ssh-key
submodules: true
remote fetch requiring network
LFS
```

Example error:

```text
⚠️ actions/checkout input "submodules: true" requires Git remote access — not supported locally
```

---

## 39. `actions/setup-python` Simulation

Supported:

```text
actions/setup-python@v4
actions/setup-python@v5
```

Default mode:

```text
--python-mode=image
```

Behavior:

```text
1. Resolve python-version after matrix expansion.
2. If exactly one version:
   use python:<version>-slim as job image.
3. Set PATH so python/pip are available.
4. Set pythonLocation if appropriate.
```

Future toolcache mode:

```text
--python-mode=toolcache
```

Toolcache layout:

```text
/opt/hostedtoolcache/Python/<version>/x64/bin/python
```

---

## 40. Python Tool Cache

Host cache:

```text
~/.gacils/cache/python/<version>/
```

Metadata:

```text
~/.gacils/cache/python/<version>.json
```

Docker volume:

```text
gacils-python-cache
```

Mount path:

```text
/opt/hostedtoolcache
```

Command:

```sh
gacils setup python 3.12
```

Flow:

```text
1. Check Docker availability.
2. Create Docker volume if missing:
   docker volume create gacils-python-cache
3. Pull image:
   docker pull --platform linux/amd64 python:3.12-slim
4. Create temporary container:
   docker create --name gacils-python-extract-<uuid> python:3.12-slim
5. Copy /usr/local from container to host cache:
   docker cp gacils-python-extract-<uuid>:/usr/local/. ~/.gacils/cache/python/3.12/
6. Remove temporary container:
   docker rm gacils-python-extract-<uuid>
7. Populate named volume:
   Run temporary container with gacils-python-cache mounted at /opt/hostedtoolcache.
   Copy host cache into /opt/hostedtoolcache/Python/<version>/x64.
8. Write metadata:
   ~/.gacils/cache/python/3.12.json
```

Metadata example:

```json
{
  "version": "3.12",
  "image": "python:3.12-slim",
  "platform": "linux/amd64",
  "digest": "sha256:...",
  "created_at": "2026-07-30T12:00:00Z"
}
```

Offline error:

```text
⚠️ actions/setup-python requires Python 3.13, but it is not cached locally.
   Run once while online:
     gacils setup python 3.13
```

---

## 41. Secrets

Secrets are not read from host environment automatically.

Supported sources:

```sh
--secret-file .env
--secret KEY=value
```

Example:

```sh
gacils run --secret-file .env.secrets
```

Secret context:

```text
${{ secrets.KEY }}
```

Secrets must be masked in logs.

### `secrets.GITHUB_TOKEN`

Default:

```text
""
```

If referenced:

```text
⚠️ GitHub token is simulated as an empty string locally.
   GitHub API calls will fail.
```

User may override:

```sh
gacils run --secret GITHUB_TOKEN=token
```

Unsupported GitHub API actions still fail even if token is provided.

---

## 42. Timeout Handling

Support:

```yaml
jobs:
  test:
    timeout-minutes: 10
    steps:
      - run: sleep 9999
        timeout-minutes: 1
```

Default:

```text
360 minutes
```

Behavior:

```text
1. Create context with timeout.
2. If timeout expires:
   kill exec process
   mark step/job as timeout failure
   cleanup container
3. Exit code:
   5
```

Console example:

```text
✗ Step "Run tests" timed out after 1 minute
```

Flag:

```sh
--no-timeout
```

---

## 43. Logging

Default log directory:

```text
~/.gacils/logs/<run-id>/
```

Example:

```text
~/.gacils/logs/20260730-001/
├── gacils.log
├── plan.json
├── result.json
└── jobs/
    ├── lint/
    │   ├── job.log
    │   ├── step-01-checkout.log
    │   └── step-02-install.log
    └── test-python-3_12/
        ├── job.log
        ├── step-01-checkout.log
        ├── step-02-setup-python.log
        └── step-03-pytest.log
```

Flag:

```sh
--log-dir string
```

---

## 44. Parallel Execution

Flag:

```sh
--parallel N
```

Values:

| Value | Meaning |
|---:|---|
| `0` | Default. Run all independent jobs concurrently. |
| `1` | Sequential execution. |
| `N` | Maximum N concurrent jobs. |

Respect:

```yaml
strategy:
  max-parallel: 2
```

Effective parallelism:

```text
min(--parallel, strategy.max-parallel)
```

If `--parallel=0`:

```text
use strategy.max-parallel if present, otherwise all independent jobs
```

---

## 45. CLI Commands

```sh
gacils run
gacils list
gacils init
gacils doctor
gacils clean
gacils setup python <version>
gacils version
```

---

## 46. `gacils run` Flags

```text
-W, --workflow string
    Workflow file or directory.
    Default: .github/workflows

--job string
    Run only the specified job ID.

--dry-run
    Print execution plan without running.

--secret-file string
    Load secrets from a .env-style file.

--secret stringArray
    Provide a secret directly.

--platform string
    Docker platform.
    Default: linux/amd64

--parallel int
    Number of independent jobs to run concurrently.
    Default: 0 = all

--crlf string
    CRLF handling mode: convert | preserve | error
    Default: convert

--python-mode string
    Python mode: image | toolcache
    Default: image

--python-image-flavor string
    Python image flavor: slim | full
    Default: slim

--offline
    Do not pull images or download anything.

--log-dir string
    Log directory.
    Default: ~/.gacils/logs

--no-timeout
    Disable timeout handling.

--keep-containers
    Do not remove containers after execution.

--verbose
    Verbose output.

--no-color
    Disable colored output.

--strict
    Treat warnings as errors.
```

GitHub context override flags:

```text
--event string
    github.event_name
    Default: push

--event-payload string
    Path to JSON event payload.

--ref string
    github.ref

--sha string
    github.sha

--repository string
    github.repository

--actor string
    github.actor
    Default: gacils-local

--run-id string
    github.run_id
    Default: 1

--run-number string
    github.run_number
    Default: 1

--run-attempt string
    github.run_attempt
    Default: 1

--runner-name string
    runner.name
    Default: gacils-local
```

---

## 47. `gacils list`

```sh
gacils list
```

Example output:

```text
WORKFLOW FILE                  NAME        JOBS
.github/workflows/ci.yml       CI          lint, test, report
.github/workflows/deploy.yml   Deploy      deploy
```

With jobs:

```sh
gacils list -W .github/workflows/ci.yml --jobs
```

Example:

```text
JOB ID     NEEDS      MATRIX
lint       -          -
test       lint       python-version=3.11,3.12,3.13
report     test       -
```

---

## 48. `gacils init`

```sh
gacils init
```

Creates:

```text
.github/workflows/ci.yml
```

If file exists:

```text
✗ .github/workflows/ci.yml already exists.
  Use --force to overwrite.
```

Flags:

```text
--force
--python
--node
```

Default template:

```yaml
name: CI

on:
  push:
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-python@v5
        with:
          python-version: "3.12"

      - name: Show Python version
        run: python --version
```

---

## 49. `gacils doctor`

```sh
gacils doctor
```

Checks:

```text
Docker installed
Docker socket accessible
Docker API version negotiation
Platform supported
~/.gacils writable
Disk space sufficient
ubuntu:24.04 image present
Python images present
Docker volume gacils-python-cache present
Git available
```

Example output:

```text
Core
  ✅ Docker API available        OK
  ✅ Docker socket               npipe:////./pipe/docker_engine
  ✅ Docker platform             linux/amd64
  ✅ gacils home writable        C:\Users\you\.gacils
  ✅ Disk space                  58 GB available

Images
  ✅ ubuntu:24.04                present
  ✅ python:3.11-slim            present
  ✅ python:3.12-slim            present
  ⚠️ python:3.13-slim            missing
     Run:
       gacils setup python 3.13

Optional
  ✅ Git                         found
  ⚠️ Docker volume               gacils-python-cache not created
     Run:
       gacils setup python 3.12

Doctor finished with warnings.
```

Exit codes:

| Code | Meaning |
|---:|---|
| `0` | Ready |
| `1` | Warnings present |
| `2` | Usage or configuration error |
| `4` | Docker unavailable |

Flag:

```sh
--strict
```

---

## 50. `gacils clean`

```sh
gacils clean
```

Targets:

```text
logs
cache
containers
volumes
```

Flags:

```text
--logs
    Remove logs.

--cache
    Remove cache.

--containers
    Remove gacils containers.

--volumes
    Remove gacils Docker volumes.

--all
    Remove logs, cache, containers, and volumes.

--older-than duration
    Remove logs older than duration.

--force
    Do not ask for confirmation.

--include-config
    Also remove config file.
```

Examples:

```sh
gacils clean --logs
gacils clean --cache
gacils clean --all --force
gacils clean --logs --older-than 30d
```

Safety rule:

```text
--all must not remove ~/.gacils/config.yaml unless --include-config is used.
```

---

## 51. Exit Codes

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

## 52. Console Output

Example:

```text
▶ Workflow: CI
▶ File: .github/workflows/ci.yml
▶ Run ID: 1

▶ Job: lint
  ▶ Step: actions/checkout@v4
    ✓ actions/checkout@v4 (0.4s)
  ▶ Step: Install dependencies
    python -m pip install --upgrade pip
    pip install ruff
    ✓ Install dependencies (2.1s)
  ✓ Job lint succeeded (2.5s)

▶ Job: test (python-version=3.12)
  ✓ actions/checkout@v4 (0.3s)
  ✓ actions/setup-python@v5 (1.2s)
  ✗ Run tests (0.8s)
      AssertionError: expected 2, got 3
  ✗ Job test (python-version=3.12) failed (2.3s)

✗ Workflow CI failed
```

Color rules:

```text
green  = success
red    = failure
yellow = warning/skipped/unsupported
cyan   = step/job headers
```

Disable color when:

```text
--no-color
NO_COLOR=1
stdout is not a TTY
```

---

## 53. Cross-Platform Strategy

### Windows

```text
Handle C:\ paths.
Use npipe Docker socket.
Convert CRLF to LF by default with warning.
Enable ANSI color support.
Test on PowerShell, CMD, and Git Bash.
```

### macOS

```text
Support Apple Silicon and Intel.
Use Docker Desktop socket.
Default platform linux/amd64 for GitHub compatibility.
Allow linux/arm64 for speed.
```

### Linux

```text
Support Ubuntu and Debian.
Support Docker Engine.
Support rootless Docker if DOCKER_HOST is reachable.
Use unix socket by default.
```

---

## 54. Compatibility Policy

Locked policy:

```text
Target high compatibility for the supported feature subset.
Unsupported features must fail clearly.
Do not silently skip unsupported behavior.
```

Default mode:

```text
GitHub-compatible + Windows-friendly
```

CRLF default:

```text
convert + warning
```

Strict GitHub-like mode:

```sh
gacils run --crlf=preserve --strict
```

---

## 55. Build System

Go build:

```sh
go build -trimpath -ldflags="-s -w" -o dist/gacils ./cmd/gacils
```

Cross-build:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/gacils_linux_amd64 ./cmd/gacils
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o dist/gacils_linux_arm64 ./cmd/gacils
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o dist/gacils_darwin_amd64 ./cmd/gacils
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o dist/gacils_darwin_arm64 ./cmd/gacils
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/gacils_windows_amd64.exe ./cmd/gacils
CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -o dist/gacils_windows_arm64.exe ./cmd/gacils
```

GoReleaser:

```sh
goreleaser release --snapshot --clean
```

---

## 56. Installation

From source:

```sh
go install github.com/yourname/github-action-ci-local-simulator/cmd/gacils@latest
```

Verify:

```sh
gacils version
```

Homebrew:

```sh
brew install yourname/tap/gacils
```

Scoop:

```powershell
scoop bucket add yourname https://github.com/yourname/scoop-bucket
scoop install gacils
```

Linux tarball:

```sh
curl -fsSL https://github.com/yourname/github-action-ci-local-simulator/releases/latest/download/github-action-ci-local-simulator_linux_amd64.tar.gz \
  | tar -xz -C /tmp

sudo mv /tmp/gacils /usr/local/bin/gacils

gacils version
```

---

## 57. Phased Implementation Plan

### Phase 1 — Core Runner

```text
□ go mod init github.com/yourname/github-action-ci-local-simulator
□ Cobra CLI: run, version, doctor
□ Workflow loader
□ Basic validator
□ Docker client FromEnv + APIVersionNegotiation
□ Create ubuntu:24.04 container
□ Copy workspace via tar stream
□ Run steps with bash -e
□ Capture stdout/stderr
□ Capture exit code
□ Colored console output
□ Exit code 0/1
□ Basic github.* context
□ Basic runner.* context
□ Basic logs
□ gacils doctor minimum
```

### Phase 2 — Workflow Semantics

```text
□ env precedence
□ defaults.run.shell
□ defaults.run.working-directory
□ GITHUB_ENV job file
□ GITHUB_PATH job file
□ GITHUB_OUTPUT step file
□ job needs
□ DAG
□ topological sort
□ matrix expansion
□ include/exclude
□ if conditions
□ continue-on-error outcome/conclusion
□ timeout-minutes
□ logs directory
□ contains()
□ startsWith()
□ endsWith()
□ format()
```

### Phase 3 — Actions Simulation

```text
□ actions/checkout
□ actions/setup-python
□ Python image selection
□ gacils setup python
□ Python cache metadata
□ unsupported action errors
```

### Phase 4 — Advanced Features

```text
□ --parallel
□ strategy.max-parallel
□ fail-fast
□ secrets file
□ secrets masking
□ actions/cache
□ artifacts
□ service containers
□ hashFiles()
□ fromJson()
□ toJson()
□ join()
□ star expressions
```

### Phase 5 — Platform Polish

```text
□ Windows CRLF polish
□ Windows npipe polish
□ macOS Apple Silicon polish
□ Linux rootless Docker polish
□ log cleanup
□ gacils list
□ gacils init
□ installer packaging
□ documentation
```

---

## 58. Acceptance Tests

### Simple run

```yaml
name: simple-run

on: push

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo hello
      - run: pwd
      - run: ls -la
```

Expected:

```text
exit code 0
all steps success
```

---

### Failure

```yaml
name: failure

on: push

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: exit 1
```

Expected:

```text
exit code 1
step failure
job failure
```

---

### Env precedence

```yaml
name: env-precedence

on: push

env:
  A: workflow
  B: workflow

jobs:
  test:
    runs-on: ubuntu-latest
    env:
      B: job
      C: job
    steps:
      - name: Check env
        env:
          C: step
        run: |
          test "$A" = "workflow"
          test "$B" = "job"
          test "$C" = "step"
```

Expected:

```text
job success
```

---

### GitHub context

```yaml
name: github-context

on: push

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: |
          test "${{ github.actor }}" = "gacils-local"
          test "${{ github.workspace }}" = "/github/workspace"
          test "${{ github.run_id }}" = "1"
          test "${{ github.run_number }}" = "1"
          test "${GITHUB_ACTIONS}" = "true"
          test "${CI}" = "true"
```

Expected:

```text
job success
```

---

### Runner context

```yaml
name: runner-context

on: push

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: |
          test "${{ runner.os }}" = "Linux"
          test "${{ runner.temp }}" = "/tmp/gacils"
          test "${{ runner.tool_cache }}" = "/opt/hostedtoolcache"
          test "${RUNNER_OS}" = "Linux"
          test "${RUNNER_TOOL_CACHE}" = "/opt/hostedtoolcache"
```

Expected:

```text
job success
```

---

### continue-on-error

```yaml
name: continue-on-error

on: push

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - id: fail
        run: exit 1
        continue-on-error: true

      - id: should_not_run
        if: failure()
        run: echo "should not run"

      - id: should_run
        if: success()
        run: echo "should run"
```

Expected:

```text
fail.outcome = failure
fail.conclusion = success
should_not_run = skipped
should_run = success
job = success
```

---

### needs outputs

```yaml
name: needs-outputs

on: push

jobs:
  build:
    runs-on: ubuntu-latest
    outputs:
      version: ${{ steps.meta.outputs.version }}
    steps:
      - id: meta
        run: echo "version=1.2.3" >> "$GITHUB_OUTPUT"

  deploy:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - run: |
          test "${{ needs.build.outputs.version }}" = "1.2.3"
          test "${{ needs.build.result }}" = "success"
```

Expected:

```text
both jobs success
```

---

### contains

```yaml
name: contains-ref

on: push

jobs:
  deploy:
    runs-on: ubuntu-latest
    if: contains(github.ref, 'main')
    steps:
      - run: echo "deploy"
```

Command:

```sh
gacils run --ref refs/heads/main
```

Expected:

```text
deploy runs
```

Command:

```sh
gacils run --ref refs/heads/dev
```

Expected:

```text
deploy skipped
```

---

### Unsupported action

```yaml
name: unsupported-action

on: push

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/deploy-pages@v4
```

Expected:

```text
⚠️ workflow "unsupported-action" job "deploy" step 2 uses "actions/deploy-pages@v4": requires GitHub API — not supported locally
exit code 3
```

---

### Unsupported expression

```yaml
name: unsupported-expression

on: push

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "${{ hashFiles('**/requirements.txt') }}"
```

Expected:

```text
⚠️ expression function "hashFiles" is not supported locally yet
exit code 3
```

---

### GITHUB_STEP_SUMMARY

```yaml
name: step-summary

on: push

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "## Summary" >> $GITHUB_STEP_SUMMARY
```

Expected:

```text
⚠️ job "test" step 1 references GITHUB_STEP_SUMMARY — Markdown step summaries are not supported locally
exit code 3
```

---

## 59. Final Locked Decisions

```text
Repo name:
  github-action-ci-local-simulator

Binary name:
  gacils

Language:
  Go 1.26

Execution model:
  1 job instance = 1 container
  1 step = 1 docker exec

Environment files:
  GITHUB_ENV = per job
  GITHUB_PATH = per job
  GITHUB_OUTPUT = per step

Shell:
  bash -e {0}

CRLF:
  default convert + warning
  preserve and error modes available

setup-python:
  use python:<version>-slim when one Python version is requested

Python tool cache:
  host cache: ~/.gacils/cache/python/
  Docker volume: gacils-python-cache
  mount path: /opt/hostedtoolcache

runner.tool_cache:
  /opt/hostedtoolcache

continue-on-error:
  outcome may be failure
  conclusion becomes success
  failure() uses conclusion

env precedence:
  step > job > workflow > generated > container base

github context:
  deterministic defaults
  Git-derived values when available
  override flags supported

runner context:
  Linux
  X64 or ARM64
  gacils-local
  /tmp/gacils
  /opt/hostedtoolcache

needs outputs:
  collected after job completion
  skipped dependency outputs empty with warning
  matrix output aggregation unsupported

expression functions:
  Phase 1: status functions only
  Phase 2: contains, startsWith, endsWith, format
  Phase 4: hashFiles, fromJson, toJson, join, star expressions

unsupported features:
  fail clearly
  do not silently skip
  exit code 3
```

This document is the locked implementation specification for `github-action-ci-local-simulator`.