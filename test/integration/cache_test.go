package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCache_SaveAndRestore(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)

	wf1Content := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: |
          mkdir -p my-cache
          echo "cached data" > my-cache/data.txt
      - uses: actions/cache@v3
        with:
          path: my-cache
          key: cache-key-1
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci1.yml"), []byte(wf1Content), 0644)

	wf2Content := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/cache@v3
        with:
          path: my-cache
          key: cache-key-1
      - run: |
          if [ -f my-cache/data.txt ]; then
            echo "CACHE_RESTORED_SUCCESS"
          fi
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci2.yml"), []byte(wf2Content), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	_ = copyFile("gacils", filepath.Join(tempDir, "gacils"))

	// First run: create directory and execute actions/cache
	output1, err := runGacilsInDir(t, tempDir, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci1.yml"))
	if err != nil {
		t.Fatalf("first run failed: %v\nOutput: %s", err, output1)
	}

	// Second run: restore cache
	output2, err := runGacilsInDir(t, tempDir, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci2.yml"))
	if err != nil {
		t.Fatalf("second run failed: %v\nOutput: %s", err, output2)
	}
	if !strings.Contains(output2, "Cache restored from key") && !strings.Contains(output2, "cache-hit=true") {
		t.Errorf("expected second run to restore cache, output: %s", output2)
	}
}

func TestCache_CacheHit(t *testing.T) {
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
          mkdir -p my-cache-hit
          touch my-cache-hit/item.txt
      - id: cache-step
        uses: actions/cache@v3
        with:
          path: my-cache-hit
          key: hit-key-1
      - run: |
          echo "cache-hit=${{ steps.cache-step.outputs.cache-hit }}"
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	_ = copyFile("gacils", filepath.Join(tempDir, "gacils"))

	// First run
	_, _ = runGacilsInDir(t, tempDir, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))

	// Second run
	output2, err := runGacilsInDir(t, tempDir, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("second run failed: %v\nOutput: %s", err, output2)
	}
	if !strings.Contains(output2, "Cache restored from key") && !strings.Contains(output2, "cache-hit=true") {
		t.Errorf("expected cache hit on second run, output: %s", output2)
	}
}

func TestCache_DifferentKeys(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)

	wf1Content := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/cache@v3
        with:
          path: my-diff-cache
          key: key-alpha
      - run: |
          mkdir -p my-diff-cache
          echo "alpha" > my-diff-cache/alpha.txt
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci1.yml"), []byte(wf1Content), 0644)

	wf2Content := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/cache@v3
        with:
          path: my-diff-cache
          key: key-beta
      - run: |
          if [ -f my-diff-cache/alpha.txt ]; then
            echo "ERROR: SHOULD_NOT_RESTORE_DIFF_KEY"
            exit 1
          fi
          echo "DIFFERENT_KEYS_ISOLATED"
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci2.yml"), []byte(wf2Content), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	_ = copyFile("gacils", filepath.Join(tempDir, "gacils"))

	// Run 1 with key-alpha
	_, _ = runGacilsInDir(t, tempDir, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci1.yml"))

	// Run 2 with key-beta
	output2, err := runGacilsInDir(t, tempDir, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci2.yml"))
	if err != nil {
		t.Fatalf("run with key-beta failed: %v\nOutput: %s", err, output2)
	}
	if !strings.Contains(output2, "DIFFERENT_KEYS_ISOLATED") {
		t.Errorf("expected DIFFERENT_KEYS_ISOLATED, output: %s", output2)
	}
}
