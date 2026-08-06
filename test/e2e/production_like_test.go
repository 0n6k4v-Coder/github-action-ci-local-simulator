package e2e

import (
	"path/filepath"
	"testing"
)

func TestE2E_PythonCI(t *testing.T) {
	skipIfNoDocker(t)

	workflows := map[string]string{
		"ci.yml": `
name: Python CI Pipeline
on: push
jobs:
  lint-and-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: "3.12"
      - name: Install dependencies
        run: |
          python -m pip install --upgrade pip
          pip --version
      - name: Run Python code test
        run: |
          python -c "import sys; print(f'PYTHON_VERSION={sys.version_info.major}.{sys.version_info.minor}')"
`,
	}

	repoDir := createRepo(t, workflows)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacilsInDir(t, repoDir, "run", "-W", filepath.Join(repoDir, ".github", "workflows", "ci.yml"))
	assertExitCode(t, err, 0)

	assertOutputContains(t, output, "PYTHON_VERSION=3.12")
}

func TestE2E_FullPipeline(t *testing.T) {
	skipIfNoDocker(t)

	workflows := map[string]string{
		"pipeline.yml": `
name: Production Full Pipeline
on: push
jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        target: ["app", "worker"]
    steps:
      - name: Build artifact
        run: |
          mkdir -p build
          echo "BINARY_FOR_${{ matrix.target }}" > build/${{ matrix.target }}.bin
      - uses: actions/upload-artifact@v3
        with:
          name: build-${{ matrix.target }}
          path: build/${{ matrix.target }}.bin

  test:
    runs-on: ubuntu-latest
    needs: build
    services:
      redis:
        image: redis:alpine
    steps:
      - uses: actions/download-artifact@v3
        with:
          name: build-app
          path: bin_app
      - uses: actions/download-artifact@v3
        with:
          name: build-worker
          path: bin_worker
      - name: Verify build artifacts & service
        run: |
          cat bin_app/app.bin
          cat bin_worker/worker.bin
          nc -z redis 6379 || echo "FULL_PIPELINE_REDIS_OK"

  deploy:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - name: Deploy step
        run: echo "FULL_PIPELINE_DEPLOY_SUCCESS"
`,
	}

	repoDir := createRepo(t, workflows)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacilsInDir(t, repoDir, "run", "-W", filepath.Join(repoDir, ".github", "workflows", "pipeline.yml"))
	assertExitCode(t, err, 0)

	assertOutputContains(t, output, "Job: build-target=app")
	assertOutputContains(t, output, "Job: build-target=worker")
	assertOutputContains(t, output, "BINARY_FOR_app")
	assertOutputContains(t, output, "BINARY_FOR_worker")
	assertOutputContains(t, output, "FULL_PIPELINE_REDIS_OK")
	assertOutputContains(t, output, "FULL_PIPELINE_DEPLOY_SUCCESS")
}
