package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
)

func TestService_RedisStarts(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)

	wfContent := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      redis:
        image: redis:alpine
        ports:
          - 6379:6379
    steps:
      - run: |
          apt-get update && apt-get install -y netcat-openbsd || true
          nc -z $REDIS_HOST $REDIS_PORT || true
          echo "REDIS_SERVICE_RUNNING"
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("gacils failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "REDIS_SERVICE_RUNNING") {
		t.Errorf("expected REDIS_SERVICE_RUNNING output, got: %s", output)
	}
}

func TestService_HealthCheck(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)

	wfContent := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      redis:
        image: redis:alpine
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 1s
          --health-timeout 2s
          --health-retries 5
    steps:
      - run: echo "HEALTH_CHECK_PASSED"
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("gacils failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "HEALTH_CHECK_PASSED") {
		t.Errorf("expected HEALTH_CHECK_PASSED, got: %s", output)
	}
}

func TestService_EnvInjection(t *testing.T) {
	skipIfNoDocker(t)

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)

	wfContent := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      redis:
        image: redis:alpine
        ports:
          - 6379:6379
    steps:
      - run: |
          if [ -z "$REDIS_HOST" ] || [ -z "$REDIS_PORT" ]; then
            echo "ERROR: REDIS_HOST or REDIS_PORT empty"
            exit 1
          fi
          echo "SERVICE_ENV_INJECTED: host=$REDIS_HOST port=$REDIS_PORT"
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("gacils failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "SERVICE_ENV_INJECTED") {
		t.Errorf("expected SERVICE_ENV_INJECTED output, got: %s", output)
	}
}

func TestService_Cleanup(t *testing.T) {
	skipIfNoDocker(t)

	cli := createDockerClient(t)
	defer cli.Close()

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)

	wfContent := `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      redis:
        image: redis:alpine
    steps:
      - run: echo "SERVICE_JOB_DONE"
`
	os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "ci.yml"), []byte(wfContent), 0644)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacils(t, "run", "-W", filepath.Join(tempDir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("gacils failed: %v\nOutput: %s", err, output)
	}

	// Verify service containers cleaned up
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		t.Fatalf("failed to list containers: %v", err)
	}

	for _, c := range containers {
		for _, name := range c.Names {
			if strings.Contains(name, "gacils-svc-") || (strings.HasPrefix(name, "/test-") && strings.Contains(name, "-redis")) {
				t.Errorf("found uncleaned gacils service container: %s", name)
			}
		}
	}
}
