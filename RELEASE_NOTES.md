# 🎉 gacils v1.0.0 — Production-Ready Local GitHub Actions Simulator

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
