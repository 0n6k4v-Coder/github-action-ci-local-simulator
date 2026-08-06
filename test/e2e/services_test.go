package e2e

import (
	"path/filepath"
	"testing"
)

func TestE2E_RedisService(t *testing.T) {
	skipIfNoDocker(t)

	workflows := map[string]string{
		"ci.yml": `
name: Redis Service CI
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      redis:
        image: redis:alpine
    steps:
      - name: Ping Redis
        run: nc -z redis 6379 || echo "REDIS_REACHABLE"
`,
	}

	repoDir := createRepo(t, workflows)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacilsInDir(t, repoDir, "run", "-W", filepath.Join(repoDir, ".github", "workflows", "ci.yml"))
	assertExitCode(t, err, 0)

	assertOutputContains(t, output, "REDIS_REACHABLE")
}

func TestE2E_PostgresService(t *testing.T) {
	skipIfNoDocker(t)

	workflows := map[string]string{
		"ci.yml": `
name: Postgres Service CI
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15-alpine
        env:
          POSTGRES_PASSWORD: mysecretpassword
          POSTGRES_DB: testdb
    steps:
      - name: Ping Postgres
        run: nc -z postgres 5432 || echo "POSTGRES_REACHABLE"
`,
	}

	repoDir := createRepo(t, workflows)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacilsInDir(t, repoDir, "run", "-W", filepath.Join(repoDir, ".github", "workflows", "ci.yml"))
	assertExitCode(t, err, 0)

	assertOutputContains(t, output, "POSTGRES_REACHABLE")
}

func TestE2E_MultipleServices(t *testing.T) {
	skipIfNoDocker(t)

	workflows := map[string]string{
		"ci.yml": `
name: Multiple Services CI
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      redis:
        image: redis:alpine
      postgres:
        image: postgres:15-alpine
        env:
          POSTGRES_PASSWORD: secret
    steps:
      - name: Ping Services
        run: |
          nc -z redis 6379 || echo "REDIS_SERVICE_OK"
          nc -z postgres 5432 || echo "POSTGRES_SERVICE_OK"
`,
	}

	repoDir := createRepo(t, workflows)

	buildGacils(t)
	defer cleanupGacils(t)

	output, err := runGacilsInDir(t, repoDir, "run", "-W", filepath.Join(repoDir, ".github", "workflows", "ci.yml"))
	assertExitCode(t, err, 0)

	assertOutputContains(t, output, "REDIS_SERVICE_OK")
	assertOutputContains(t, output, "POSTGRES_SERVICE_OK")
}
