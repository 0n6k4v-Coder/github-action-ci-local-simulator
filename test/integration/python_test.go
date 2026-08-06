package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupPython_ImageSwitch(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)
	wfContent := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-python@v5
        with:
          python-version: "3.12"
      - run: python --version
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("gacils failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "3.12") {
		t.Errorf("expected output to contain python version 3.12, got: %s", output)
	}
}

func TestSetupPython_FallbackInstall(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)
	wfContent := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-python@v5
        with:
          python-version: "3.12"
      - run: python3 --version
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("gacils failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Python 3") {
		t.Errorf("expected output to contain Python 3, got: %s", output)
	}
}

func TestSetupPython_PipWorks(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)
	wfContent := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-python@v5
        with:
          python-version: "3.12"
      - run: pip --version
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("gacils failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "pip") {
		t.Errorf("expected pip output, got: %s", output)
	}
}

func TestAutoInstall_PythonCommands(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)
	wfContent := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: python3 -c "print('AUTO_PYTHON_SUCCESS')"
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("gacils failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "AUTO_PYTHON_SUCCESS") {
		t.Errorf("expected AUTO_PYTHON_SUCCESS, got: %s", output)
	}
}
