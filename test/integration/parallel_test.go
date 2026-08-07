package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParallel_IndependentJobs(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)

	wfContent := `name: test
on: push
jobs:
  job1:
    runs-on: ubuntu-latest
    steps:
      - run: sleep 3
  job2:
    runs-on: ubuntu-latest
    steps:
      - run: sleep 3
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	start := time.Now()
	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("gacils failed: %v\nOutput: %s", err, output)
	}

	// Two 3s jobs running in parallel should complete well before 10s
	// (3s sleep + ~5s Docker container overhead, running concurrently)
	if duration >= 10000*time.Millisecond {
		t.Errorf("expected parallel execution (<10s), total duration was: %v", duration)
	}
}

func TestParallel_DependentJobsWait(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)

	wfContent := `name: test
on: push
jobs:
  job1:
    runs-on: ubuntu-latest
    steps:
      - run: |
          mkdir -p /tmp/gacils/shared
          echo "JOB1_DONE" > /tmp/gacils/shared/job1.txt

  job2:
    runs-on: ubuntu-latest
    needs: [job1]
    steps:
      - run: |
          if [ ! -f /tmp/gacils/shared/job1.txt ]; then
            echo "ERROR: job1 output not ready"
            exit 1
          fi
          echo "JOB2_DEPENDENCY_PASSED"
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("gacils failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "JOB2_DEPENDENCY_PASSED") {
		t.Errorf("expected JOB2_DEPENDENCY_PASSED, got: %s", output)
	}
}

func TestParallel_MatrixBarrier(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)

	wfContent := `name: test
on: push
jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        version: ["1", "2"]
    steps:
      - run: echo "MATRIX_INSTANCE_${{ matrix.version }}_DONE"

  downstream:
    runs-on: ubuntu-latest
    needs: build
    if: always()
    steps:
      - run: echo "ALL_MATRIX_INSTANCES_PASSED"
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("gacils failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "ALL_MATRIX_INSTANCES_PASSED") {
		t.Errorf("expected ALL_MATRIX_INSTANCES_PASSED output, got: %s", output)
	}
}
