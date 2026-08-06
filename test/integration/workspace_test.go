package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceCopy_RepoRoot(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("test content"), 0644)
	wfContent := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: |
          ls -la /github/workspace/
          if [ ! -f /github/workspace/test.txt ]; then
            echo "ERROR: test.txt not found"
            exit 1
          fi
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("gacils failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "test.txt") {
		t.Errorf("workspace should contain test.txt, output: %s", output)
	}
}

func TestWorkspaceCopy_WithGitDir(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".git"), 0755)
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(tempDir, "git-repo-marker.txt"), []byte("git repo"), 0644)
	wfContent := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: cat /github/workspace/git-repo-marker.txt
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("gacils failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "git repo") {
		t.Errorf("expected output to contain 'git repo', output: %s", output)
	}
}

func TestWorkspaceCopy_NestedStructure(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".git"), 0755)
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows", "nested", "deep"), 0755)
	os.WriteFile(filepath.Join(tempDir, "root-file.txt"), []byte("root file"), 0644)
	wfContent := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: cat /github/workspace/root-file.txt
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "nested", "deep", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "nested", "deep", "ci.yml"))
	if err != nil {
		t.Fatalf("gacils failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "root file") {
		t.Errorf("expected output to contain 'root file', output: %s", output)
	}
}

func TestWorkspaceCopy_ExcludesGitDir(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".git", "objects"), 0755)
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)
	wfContent := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: |
          if [ -d /github/workspace/.git ]; then
            echo "ERROR: .git directory should not exist"
            exit 1
          fi
          echo "GIT_DIR_EXCLUDED_SUCCESS"
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("gacils failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "GIT_DIR_EXCLUDED_SUCCESS") {
		t.Errorf("expected GIT_DIR_EXCLUDED_SUCCESS, output: %s", output)
	}
}
