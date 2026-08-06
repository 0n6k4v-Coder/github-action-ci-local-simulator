package e2e

import (
	"path/filepath"
	"testing"
)

func TestE2E_LinearDependencies(t *testing.T) {
	skipIfNoDocker(t)

	workflows := map[string]string{
		"ci.yml": `
name: Linear CI
on: push
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Build step
        run: echo "BUILD_COMPLETE"
  test:
    runs-on: ubuntu-latest
    needs: build
    steps:
      - name: Test step
        run: echo "TEST_COMPLETE"
  deploy:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - name: Deploy step
        run: echo "DEPLOY_SUCCESSFUL"
`,
	}

	repoDir := createRepo(t, workflows)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(repoDir, ".github", "workflows", "ci.yml"))
	assertExitCode(t, err, 0)

	assertOutputContains(t, output, "BUILD_COMPLETE")
	assertOutputContains(t, output, "TEST_COMPLETE")
	assertOutputContains(t, output, "DEPLOY_SUCCESSFUL")
}

func TestE2E_DiamondDependencies(t *testing.T) {
	skipIfNoDocker(t)

	workflows := map[string]string{
		"ci.yml": `
name: Diamond CI
on: push
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo "BUILD_READY"

  lint:
    runs-on: ubuntu-latest
    needs: build
    steps:
      - run: echo "LINT_PASSED"

  test:
    runs-on: ubuntu-latest
    needs: build
    steps:
      - run: echo "TEST_PASSED"

  deploy:
    runs-on: ubuntu-latest
    needs: [lint, test]
    steps:
      - run: echo "DIAMOND_DEPLOY_PASSED"
`,
	}

	repoDir := createRepo(t, workflows)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(repoDir, ".github", "workflows", "ci.yml"))
	assertExitCode(t, err, 0)

	assertOutputContains(t, output, "BUILD_READY")
	assertOutputContains(t, output, "LINT_PASSED")
	assertOutputContains(t, output, "TEST_PASSED")
	assertOutputContains(t, output, "DIAMOND_DEPLOY_PASSED")
}

func TestE2E_FailurePropagation(t *testing.T) {
	skipIfNoDocker(t)

	workflows := map[string]string{
		"ci.yml": `
name: Failure CI
on: push
jobs:
  job1:
    runs-on: ubuntu-latest
    steps:
      - name: Fail step
        run: exit 1

  job2:
    runs-on: ubuntu-latest
    needs: job1
    steps:
      - name: Should skip
        run: echo "JOB2_SHOULD_NOT_RUN"
`,
	}

	repoDir := createRepo(t, workflows)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(repoDir, ".github", "workflows", "ci.yml"))
	assertExitCode(t, err, 1)

	assertOutputContains(t, output, "Job job2 skipped")
	assertOutputNotContains(t, output, "JOB2_SHOULD_NOT_RUN")
}
