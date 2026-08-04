package runner

import (
	"testing"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
)

func TestJobRunner_ServiceDefinitionParsing(t *testing.T) {
	job := workflow.Job{
		RunsOn: "ubuntu-latest",
		Services: map[string]workflow.Service{
			"redis": {
				Image: "redis:alpine",
				Ports: []any{"6379:6379"},
			},
			"postgres": {
				Image:   "postgres:14",
				Env:     map[string]any{"POSTGRES_PASSWORD": "secret"},
				Ports:   []any{"5432:5432"},
				Options: `--health-cmd "pg_isready" --health-interval 10s --health-timeout 5s --health-retries 5`,
			},
		},
		Steps: []workflow.Step{
			{
				Name: "Verify env vars",
				Run:  "echo $REDIS_HOST $POSTGRES_HOST",
			},
		},
	}

	if len(job.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(job.Services))
	}

	redis, ok := job.Services["redis"]
	if !ok {
		t.Fatal("expected redis service")
	}
	if redis.Image != "redis:alpine" {
		t.Errorf("expected redis:alpine, got %s", redis.Image)
	}

	postgres, ok := job.Services["postgres"]
	if !ok {
		t.Fatal("expected postgres service")
	}
	if postgres.Options == "" {
		t.Error("expected non-empty options for postgres")
	}
}
