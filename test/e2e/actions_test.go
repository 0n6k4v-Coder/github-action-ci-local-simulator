package e2e

import (
	"path/filepath"
	"testing"
)

func TestE2E_CheckoutAndSetupPython(t *testing.T) {
	skipIfNoDocker(t)

	workflows := map[string]string{
		"ci.yml": `
name: Python Ecosystem CI
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: "3.12"
      - run: python --version
      - run: pip --version
`,
	}

	repoDir := createRepo(t, workflows)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacilsInDir(t, repoDir, "run", "-W", filepath.Join(repoDir, ".github", "workflows", "ci.yml"))
	assertExitCode(t, err, 0)

	assertOutputContains(t, output, "Python 3")
	assertOutputContains(t, output, "pip")
}

func TestE2E_CacheAndArtifacts(t *testing.T) {
	skipIfNoDocker(t)

	workflows := map[string]string{
		"ci1.yml": `
name: Cache & Artifacts Save
on: push
jobs:
  build_cache:
    runs-on: ubuntu-latest
    steps:
      - run: |
          mkdir -p my-cache dist
          echo "cached-data-content" > my-cache/data.txt
          echo "artifact-file-content" > dist/app.bin
      - uses: actions/cache@v3
        with:
          path: my-cache
          key: e2e-cache-key-1
      - uses: actions/upload-artifact@v3
        with:
          name: build-artifact
          path: dist/app.bin
`,
		"ci2.yml": `
name: Cache & Artifacts Restore
on: push
jobs:
  restore_test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/cache@v3
        with:
          path: my-cache
          key: e2e-cache-key-1
      - uses: actions/download-artifact@v3
        with:
          name: build-artifact
          path: downloaded
      - run: cat my-cache/data.txt
      - run: cat downloaded/app.bin
`,
	}

	repoDir := createRepo(t, workflows)

	buildGacils(t)
	defer cleanupGacils(t)

	// Step 1: Save cache and upload artifact
	output1, err1 := runGacilsInDir(t, repoDir, "run", "-W", filepath.Join(repoDir, ".github", "workflows", "ci1.yml"))
	assertExitCode(t, err1, 0)
	assertOutputContains(t, output1, "actions/cache simulation: Cache not found for key: e2e-cache-key-1")

	// Step 2: Restore cache and download artifact in second run
	output2, err2 := runGacilsInDir(t, repoDir, "run", "-W", filepath.Join(repoDir, ".github", "workflows", "ci2.yml"))
	assertExitCode(t, err2, 0)
	assertOutputContains(t, output2, "cached-data-content")
	assertOutputContains(t, output2, "artifact-file-content")
}

func TestE2E_UploadDownloadCrossJob(t *testing.T) {
	skipIfNoDocker(t)

	workflows := map[string]string{
		"ci.yml": `
name: Cross Job Artifact CI
on: push
jobs:
  producer:
    runs-on: ubuntu-latest
    steps:
      - run: |
          mkdir -p out
          echo "CROSS_JOB_PAYLOAD" > out/data.json
      - uses: actions/upload-artifact@v3
        with:
          name: payload
          path: out/data.json

  consumer:
    runs-on: ubuntu-latest
    needs: producer
    steps:
      - uses: actions/download-artifact@v3
        with:
          name: payload
          path: received
      - run: cat received/data.json
`,
	}

	repoDir := createRepo(t, workflows)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacilsInDir(t, repoDir, "run", "-W", filepath.Join(repoDir, ".github", "workflows", "ci.yml"))
	assertExitCode(t, err, 0)

	assertOutputContains(t, output, "CROSS_JOB_PAYLOAD")
}

func TestE2E_UnsupportedAction(t *testing.T) {
	skipIfNoDocker(t)

	workflows := map[string]string{
		"ci.yml": `
name: Unsupported Action CI
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: non-existent/unsupported-action@v1
`,
	}

	repoDir := createRepo(t, workflows)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacilsInDir(t, repoDir, "run", "-W", filepath.Join(repoDir, ".github", "workflows", "ci.yml"))
	assertExitCode(t, err, 3)

	assertOutputContains(t, output, "unsupported action")
}
