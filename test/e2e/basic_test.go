package e2e

import (
	"path/filepath"
	"testing"
)

func TestE2E_SingleJobWorkflow(t *testing.T) {
	skipIfNoDocker(t)

	workflows := map[string]string{
		"ci.yml": `
name: CI
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Step 1
        run: echo "First step"
      - name: Step 2
        run: echo "Second step"
      - name: Step 3
        run: echo "Third step"
`,
	}

	repoDir := createRepo(t, workflows)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(repoDir, ".github", "workflows", "ci.yml"))
	assertExitCode(t, err, 0)

	assertOutputContains(t, output, "Step 1")
	assertOutputContains(t, output, "Step 2")
	assertOutputContains(t, output, "Step 3")
	assertOutputContains(t, output, "First step")
	assertOutputContains(t, output, "Second step")
	assertOutputContains(t, output, "Third step")
}

func TestE2E_MultiStepWorkflow(t *testing.T) {
	skipIfNoDocker(t)

	workflows := map[string]string{
		"ci.yml": `
name: MultiStep CI
on: push
jobs:
  build_test:
    runs-on: ubuntu-latest
    steps:
      - name: Setup
        run: echo "SETUP_DONE" > /tmp/status.txt
      - name: Build
        run: |
          if [ -f /tmp/status.txt ]; then
            echo "BUILD_DONE" >> /tmp/status.txt
          fi
      - name: Test
        run: |
          cat /tmp/status.txt
`,
	}

	repoDir := createRepo(t, workflows)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(repoDir, ".github", "workflows", "ci.yml"))
	assertExitCode(t, err, 0)

	assertOutputContains(t, output, "SETUP_DONE")
	assertOutputContains(t, output, "BUILD_DONE")
}

func TestE2E_WorkflowWithEnv(t *testing.T) {
	skipIfNoDocker(t)

	workflows := map[string]string{
		"ci.yml": `
name: Env CI
on: push
env:
  GLOBAL_VAR: global_value
  OVERRIDE_VAR: original_global
jobs:
  test_env:
    runs-on: ubuntu-latest
    env:
      JOB_VAR: job_value
      OVERRIDE_VAR: job_override
    steps:
      - name: Check Env
        run: echo "GLOBAL=$GLOBAL_VAR JOB=$JOB_VAR OVERRIDE=$OVERRIDE_VAR"
`,
	}

	repoDir := createRepo(t, workflows)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(repoDir, ".github", "workflows", "ci.yml"))
	assertExitCode(t, err, 0)

	assertOutputContains(t, output, "GLOBAL=global_value")
	assertOutputContains(t, output, "JOB=job_value")
	assertOutputContains(t, output, "OVERRIDE=job_override")
}
