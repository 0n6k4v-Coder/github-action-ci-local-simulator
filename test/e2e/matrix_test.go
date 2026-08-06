package e2e

import (
	"path/filepath"
	"testing"
)

func TestE2E_MatrixExpansion(t *testing.T) {
	skipIfNoDocker(t)

	workflows := map[string]string{
		"ci.yml": `
name: Matrix CI
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        os: ["ubuntu-latest", "ubuntu-22.04"]
        version: ["3.11", "3.12"]
    steps:
      - name: Print combination
        run: echo "MATRIX_COMBO os=${{ matrix.os }} version=${{ matrix.version }}"
`,
	}

	repoDir := createRepo(t, workflows)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacilsInDir(t, repoDir, "run", "-W", filepath.Join(repoDir, ".github", "workflows", "ci.yml"))
	assertExitCode(t, err, 0)

	assertOutputContains(t, output, "Job: test-os=ubuntu-latest,version=3.11")
	assertOutputContains(t, output, "Job: test-os=ubuntu-latest,version=3.12")
	assertOutputContains(t, output, "Job: test-os=ubuntu-22.04,version=3.11")
	assertOutputContains(t, output, "Job: test-os=ubuntu-22.04,version=3.12")
	assertOutputContains(t, output, "MATRIX_COMBO os=ubuntu-latest version=3.11")
	assertOutputContains(t, output, "MATRIX_COMBO os=ubuntu-latest version=3.12")
	assertOutputContains(t, output, "MATRIX_COMBO os=ubuntu-22.04 version=3.11")
	assertOutputContains(t, output, "MATRIX_COMBO os=ubuntu-22.04 version=3.12")
}

func TestE2E_MatrixWithInclude(t *testing.T) {
	skipIfNoDocker(t)

	workflows := map[string]string{
		"ci.yml": `
name: Matrix Include CI
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        os: ["ubuntu-latest"]
        version: ["3.11"]
        include:
          - os: "ubuntu-latest"
            version: "3.12"
            extra: "custom-include"
    steps:
      - name: Print combination
        run: echo "COMBO os=${{ matrix.os }} version=${{ matrix.version }} extra=${{ matrix.extra }}"
`,
	}

	repoDir := createRepo(t, workflows)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacilsInDir(t, repoDir, "run", "-W", filepath.Join(repoDir, ".github", "workflows", "ci.yml"))
	assertExitCode(t, err, 0)

	assertOutputContains(t, output, "Job: test-os=ubuntu-latest,version=3.11")
	assertOutputContains(t, output, "Job: test-extra=custom-include,os=ubuntu-latest,version=3.12")
	assertOutputContains(t, output, "COMBO os=ubuntu-latest version=3.11")
	assertOutputContains(t, output, "COMBO os=ubuntu-latest version=3.12 extra=custom-include")
}

func TestE2E_MatrixWithExclude(t *testing.T) {
	skipIfNoDocker(t)

	workflows := map[string]string{
		"ci.yml": `
name: Matrix Exclude CI
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        os: ["ubuntu-latest", "ubuntu-22.04"]
        version: ["3.11", "3.12"]
        exclude:
          - os: "ubuntu-22.04"
            version: "3.11"
    steps:
      - name: Print combination
        run: echo "ACTIVE_COMBO os=${{ matrix.os }} version=${{ matrix.version }}"
`,
	}

	repoDir := createRepo(t, workflows)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacilsInDir(t, repoDir, "run", "-W", filepath.Join(repoDir, ".github", "workflows", "ci.yml"))
	assertExitCode(t, err, 0)

	assertOutputContains(t, output, "Job: test-os=ubuntu-latest,version=3.11")
	assertOutputContains(t, output, "Job: test-os=ubuntu-latest,version=3.12")
	assertOutputContains(t, output, "Job: test-os=ubuntu-22.04,version=3.12")
	assertOutputNotContains(t, output, "test-os=ubuntu-22.04,version=3.11")
	assertOutputContains(t, output, "ACTIVE_COMBO os=ubuntu-latest version=3.11")
	assertOutputContains(t, output, "ACTIVE_COMBO os=ubuntu-latest version=3.12")
	assertOutputContains(t, output, "ACTIVE_COMBO os=ubuntu-22.04 version=3.12")
	assertOutputNotContains(t, output, "ACTIVE_COMBO os=ubuntu-22.04 version=3.11")
}
