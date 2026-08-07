# gacils - GitHub Actions CI Local Simulator

[![Go Report Card](https://goreportcard.com/badge/github.com/0n6k4v-Coder/github-action-ci-local-simulator)](https://goreportcard.com/report/github.com/0n6k4v-Coder/github-action-ci-local-simulator)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/0n6k4v-Coder/github-action-ci-local-simulator)](https://golang.org)

> Run your GitHub Actions workflows locally with Docker — including matrix jobs, parallel execution, service containers, cache, artifacts, and automatic secrets masking.

## ✨ Why gacils?

| Problem | `gacils` Solution |
| --- | --- |
| Waiting for GitHub runners | Local Docker execution |
| Paying for CI minutes | 100% free |
| Slow iteration | Instant feedback |
| Matrix testing pain | Full matrix expansion |
| Secrets in logs | Automatic masking |
| No local cache | Host-mapped cache |

## 🚀 Features

### Core Execution
- **Docker Isolation**: Runs job steps inside isolated Docker containers using offical or custom images.
- **Workspace Context**: Mounts your local project repository directly into container environments (`/github/workspace`).
- **Multi-Step Execution**: Executes sequential `run` steps, `uses` actions, and inline shell commands.

### Workflow Logic
- **Matrix Expansion**: Full support for `strategy.matrix`, including `include` and `exclude` combinations.
- **Job DAG & Dependencies**: Inter-job dependency resolution (`needs`) running independent jobs concurrently in parallel goroutines.
- **Conditional Execution**: Evaluates `if` expressions at job and step levels (e.g. `success()`, `failure()`, `always()`).
- **Timeouts & Controls**: Per-job step timeout enforcement and failure handling (`continue-on-error`).

### Expression Engine
- **Context Access**: Expression parsing for `github`, `matrix`, `env`, `vars`, `secrets`, `steps`, `needs`, and `job`.
- **Built-in Functions**: Supports `contains()`, `startsWith()`, `endsWith()`, `format()`, `join()`, `toJSON()`, `fromJSON()`, `hashFiles()`, etc.
- **Operators & Precedence**: Complete logical, equality, and relational operators respecting standard evaluation order.

### Actions Ecosystem
- **Local Action Runner**: Native emulation for popular actions like `actions/checkout`, `actions/setup-python`, `actions/cache`, and `actions/upload-artifact`/`download-artifact`.
- **Composite Actions**: Supports executing local or remote composite actions.

### Infrastructure
- **Service Containers**: Runs background service containers (PostgreSQL, Redis, MySQL, etc.) linked over custom Docker bridge networks.
- **Health Checking**: Automatic container health inspection (`test`, `interval`, `timeout`, `retries`) before step execution.
- **Lifecycle & Cleanup**: Reliable cleanup of created networks and containers upon workflow completion or interruption.

### Security & UX
- **Automatic Secrets Masking**: Length-descending secret replacement engine preventing partial leakage in terminal logs.
- **Rich Error Hints**: Actionable troubleshooting suggestions for YAML syntax, missing context keys, or Docker errors.
- **CI Exit Codes**: Accurate process exit codes reflecting overall workflow execution results.

## 📦 Installation

### From Source
```bash
go install github.com/0n6k4v-Coder/github-action-ci-local-simulator/cmd/gacils@latest
```

### From Binary Release
Download the pre-compiled binary for your operating system and architecture from the [Releases](https://github.com/0n6k4v-Coder/github-action-ci-local-simulator/releases) page and place it in your `PATH`.

### Prerequisites
- [Docker Engine](https://docs.docker.com/get-docker/) running locally.
- [Go 1.22+](https://golang.org/dl/) (only required if building from source).

## 🐧 WSL2 + Docker Desktop Support

gacils v1.3.1+ fully supports WSL2 with Docker Desktop integration.

### Installation on WSL2

```bash
# Install Go (if not already installed)
sudo apt-get update
sudo apt-get install -y golang-go

# Install gacils
go install github.com/0n6k4v-Coder/github-action-ci-local-simulator/cmd/gacils@latest

# Verify installation
gacils --version
```

### Running on WSL2

gacils automatically detects the correct Docker socket path on WSL2.
No manual configuration required:

```bash
cd /path/to/your/repo
gacils run -W .github/workflows/ci.yml
```

### How It Works

gacils uses Docker SDK's `FromEnv` pattern which:
- Reads `DOCKER_HOST` environment variable if set
- Falls back to Docker context inspection
- Automatically detects WSL2 socket paths:
  - `~/.docker/desktop/docker.sock`
  - `/mnt/wsl/docker-desktop/docker.sock`
- Also works with Linux native Docker (`/var/run/docker.sock`)

### Troubleshooting

If you encounter Docker connection issues on WSL2:

1. Verify Docker Desktop is running and WSL2 integration is enabled
2. Check Docker context:
   ```bash
   docker context ls
   ```
3. Verify socket exists:
   ```bash
   ls -la ~/.docker/desktop/docker.sock
   ```
4. If needed, manually set DOCKER_HOST:
   ```bash
   export DOCKER_HOST="unix://$(docker context inspect --format '{{.Endpoints.docker.Host}}' | sed 's|unix://||')"
   ```

### Platform Support

| Platform | Status | Notes |
|----------|--------|-------|
| Linux (native Docker) | ✅ Full support | Default socket: `/var/run/docker.sock` |
| WSL2 + Docker Desktop | ✅ Full support | Auto-detects WSL2 socket paths |
| macOS + Docker Desktop | ⚠️ Untested | Should work with `/var/run/docker.sock` |
| Windows (native) | ❌ Not supported | Requires WSL2 |

## 🎯 Quick Start

1. **Navigate to your repository**:
   ```bash
   cd /path/to/your/repo
   ```

2. **Verify your workflow file**:
   Ensure you have a workflow defined at `.github/workflows/ci.yml`.

3. **Run `gacils`**:
   ```bash
   gacils run
   ```

## 📖 Usage

```text
gacils [command] [flags]

Commands:
  run         Run GitHub Actions workflows locally

Flags:
  -w, --workflow string   Path to target workflow file (default ".github/workflows")
  -j, --job string        Run specific job by ID
  -s, --secret string     Provide secret in KEY=VAL format (can be specified multiple times)
      --env string        Provide environment variable in KEY=VAL format
      --dry-run           Validate and print execution plan without running containers
  -h, --help              Help for gacils
```

## CLI Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--workflow` | `-W` | Path to workflow file or directory | `.github/workflows/` |
| `--job` | `-j` | Run only the specified job | (all jobs) |
| `--dry-run` | | Print execution plan without running | false |
| `--offline` | | Skip image pulls, use local images only | false |
| `--parallel` | `-p` | Max concurrent jobs (0 = unlimited) | 0 |
| `--platform` | | Docker platform for image pulls | (host platform) |
| `--crlf` | | Line ending handling: convert, preserve, error | convert |

### Examples

Run default workflows:
```bash
gacils run
```

Run a specific job with secrets:
```bash
gacils run -j test -s DB_PASSWORD=supersecret -s API_KEY=xyz123
```

Dry-run validation:
```bash
gacils run --dry-run
```

## 🧹 Cleanup

gacils uses Docker to run workflows. Over time, Docker resources (containers, images, volumes) can accumulate. Use the `clean` command to manage these resources.

### Quick Reference

```bash
gacils clean --dry-run          # See what would be removed (safe)
gacils clean                    # Light cleanup (with confirmation)
gacils clean --all              # Full cleanup (volumes + build cache)
gacils clean --images           # Also remove unused images (24h filter)
gacils clean --prune-images     # Remove ALL images (aggressive)
gacils clean --force            # Skip confirmation prompt
```

### Usage Examples

**Preview what would be removed (recommended first step):**
```bash
gacils clean --dry-run
```
Output:
```
=== Dry Run Mode ===
Showing what would be removed (no action taken)

Found: 5 containers, 12 images, 3 volumes

[DRY RUN] Would remove:
  - 5 containers

Run without --dry-run to actually remove these resources.
```

**Light cleanup (safe default):**
```bash
gacils clean
```
- Removes stopped containers
- Removes dangling images only
- Asks for confirmation
- Does NOT remove volumes or build cache

**Full cleanup (between development phases):**
```bash
gacils clean --all
```
- Removes ALL containers (running + stopped)
- Removes unused images (24h safety filter)
- Removes unused volumes
- Cleans build cache
- Asks for confirmation

**Aggressive cleanup (truly clean slate):**
```bash
gacils clean --all --prune-images
```
- Everything from `--all`
- Removes ALL unused images (no filter)
- Will need to re-pull images on next run

**Skip confirmation (for automation):**
```bash
gacils clean --force
```
- Skips the confirmation prompt
- Use with caution in automation

### Safety Features

- **Default is safe:** Only removes stopped containers and dangling images
- **Confirmation prompt:** Asks before removing resources (unless `--force`)
- **Dry-run mode:** Preview what would be removed before actually removing
- **24h safety filter:** `--images` keeps images from the last 24h

### When to Use Each Mode

| Mode | Use Case | What's Removed |
|------|----------|----------------|
| `--dry-run` | Preview before cleanup | Nothing (just shows) |
| (default) | Regular maintenance | Stopped containers + dangling images |
| `--all` | Between development phases | All containers + images + volumes + cache |
| `--prune-images` | Truly clean slate | ALL unused images (no filter) |
| `--force` | Automation/scripting | Same as default, but no confirmation |

## 🔬 Example Workflows

### 1. Matrix Build with Parallel Jobs

```yaml
name: CI Matrix Build
on: [push]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: ['1.21', '1.22']
        os: [ubuntu-latest]
    steps:
      - uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: ${{ matrix.go-version }}
      - name: Run Tests
        run: go test -v ./...
```

### 2. Service Containers with Secrets Masking

```yaml
name: Integration Tests
on: [push]

jobs:
  integration:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: ${{ secrets.DB_PASS }}
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v4
      - name: Run Database Migration
        env:
          DB_PASS: ${{ secrets.DB_PASS }}
        run: |
          echo "Connecting to DB with secret key: $DB_PASS"
          go run ./cmd/migrate
```

## ⚠️ Limitations

- **JavaScript Actions Execution**: Only NodeJS runtime actions are emulated or executed inside containers; complex JS actions relying on GitHub API tokens may require local overrides.
- **GitHub Hosted Runner Architecture**: macOS and Windows native host execution are not supported; jobs run inside Linux Docker containers.
- **GitHub API Context**: Contexts requiring remote GitHub API calls (e.g. `github.event.pull_request.labels`) are simulated using mock or local git metadata.

## 🗺️ Roadmap

- [ ] Complete GitHub Event Payload Generator
- [ ] Interactive UI / Terminal Dashboard (`tui`)
- [ ] Support for reusable workflows (`jobs.<job_id>.uses`)
- [ ] Pre-built runner image caching engine

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

Distributed under the MIT License. See [`LICENSE`](file:///home/kawee/Code/project/github-action-ci-local-simulator/LICENSE) for more information.

## 🙏 Acknowledgments

- [Docker Engine API](https://docs.docker.com/engine/api/) for container management.
- [nektos/act](https://github.com/nektos/act) for pioneering local GitHub Actions simulation.
- [gopkg.in/yaml.v3](https://github.com/go-yaml/yaml) for YAML parsing capabilities.
