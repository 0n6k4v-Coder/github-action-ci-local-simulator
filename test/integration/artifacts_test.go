package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArtifact_UploadAndDownload(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)

	wfContent := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: |
          mkdir -p dist
          echo "artifact build content" > dist/app.txt
      - uses: actions/upload-artifact@v3
        with:
          name: build-artifact
          path: dist/app.txt
      - run: rm -rf dist
      - uses: actions/download-artifact@v3
        with:
          name: build-artifact
          path: downloaded
      - run: cat downloaded/app.txt
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("gacils failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "artifact build content") {
		t.Errorf("expected downloaded artifact content, output: %s", output)
	}
}

func TestArtifact_CrossJob(t *testing.T) {
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
          mkdir -p build
          echo "cross job data" > build/file.txt
      - uses: actions/upload-artifact@v3
        with:
          name: cross-job-art
          path: build/file.txt

  job2:
    runs-on: ubuntu-latest
    needs: job1
    steps:
      - uses: actions/download-artifact@v3
        with:
          name: cross-job-art
          path: out
      - run: cat out/file.txt
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("gacils failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "cross job data") {
		t.Errorf("expected cross job artifact data in job2, output: %s", output)
	}
}

func TestArtifact_MissingArtifact(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)

	wfContent := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/download-artifact@v3
        with:
          name: non-existent-artifact
          path: out
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	if err == nil {
		t.Fatalf("expected error when downloading non-existent artifact, got success output: %s", output)
	}

	if !strings.Contains(output, "not found") && !strings.Contains(output, "error") && !strings.Contains(output, "failed") {
		t.Errorf("expected error message about missing artifact, output: %s", output)
	}
}
