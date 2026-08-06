package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecrets_MaskedInOutput(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)

	wfContent := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    env:
      MY_PASSWORD: super-secret-value-123
    steps:
      - run: echo "The secret password is $MY_PASSWORD"
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("gacils failed: %v\nOutput: %s", err, output)
	}

	if strings.Contains(output, "super-secret-value-123") {
		t.Errorf("secret leaked in output! output: %s", output)
	}
	if !strings.Contains(output, "***") {
		t.Errorf("expected masked secret *** in output, got: %s", output)
	}
}

func TestSecrets_MultipleSecrets(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)

	wfContent := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    env:
      DB_SECRET: db-secret-999
      API_KEY: api-token-888
    steps:
      - run: echo "DB=$DB_SECRET API=$API_KEY"
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("gacils failed: %v\nOutput: %s", err, output)
	}

	if strings.Contains(output, "db-secret-999") || strings.Contains(output, "api-token-888") {
		t.Errorf("secret leaked in output! output: %s", output)
	}
}

func TestSecrets_LongestFirst(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)

	wfContent := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    env:
      SHORT_SECRET: token
      LONG_SECRET: token-extended-secret
    steps:
      - run: echo "Value=$LONG_SECRET"
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("gacils failed: %v\nOutput: %s", err, output)
	}

	if strings.Contains(output, "token-extended-secret") {
		t.Errorf("long secret leaked in output! output: %s", output)
	}
}
