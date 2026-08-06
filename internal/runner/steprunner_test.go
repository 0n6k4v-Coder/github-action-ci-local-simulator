package runner

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/dockerx"
	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
	"github.com/docker/docker/client"
)

// Helper to mock execCommand for testing
func mockExecCommand(fn func(ctx context.Context, cli *client.Client, containerID, workingDir string, cmd []string, env map[string]string) (*dockerx.ExecResult, error)) func() {
	orig := execCommand
	execCommand = fn
	return func() { execCommand = orig }
}

// Shell detection tests
func TestDetectShell_Bash(t *testing.T) {
	sr := &StepRunner{}
	cmd := sr.buildCommand("echo hi", "bash")
	if len(cmd) < 4 || cmd[0] != "bash" {
		t.Errorf("expected bash command, got %v", cmd)
	}
}

func TestDetectShell_Sh(t *testing.T) {
	sr := &StepRunner{}
	cmd := sr.buildCommand("echo hi", "sh")
	if len(cmd) < 4 || cmd[0] != "sh" {
		t.Errorf("expected sh command, got %v", cmd)
	}
}

func TestDetectShell_Pwsh(t *testing.T) {
	sr := &StepRunner{}
	cmd := sr.buildCommand("echo hi", "pwsh")
	if len(cmd) < 4 || cmd[0] != "bash" {
		t.Errorf("expected bash fallback for pwsh, got %v", cmd)
	}
}

func TestDetectShell_Cmd(t *testing.T) {
	sr := &StepRunner{}
	cmd := sr.buildCommand("echo hi", "cmd")
	if len(cmd) < 4 || cmd[0] != "bash" {
		t.Errorf("expected bash fallback for cmd, got %v", cmd)
	}
}

func TestDetectShell_Default(t *testing.T) {
	sr := &StepRunner{}
	cmd := sr.buildCommand("echo hi", "")
	if len(cmd) < 4 || cmd[0] != "bash" {
		t.Errorf("expected default bash command, got %v", cmd)
	}
}

// Step execution tests
func TestRunStep_SimpleCommand(t *testing.T) {
	cleanup := mockExecCommand(func(ctx context.Context, cli *client.Client, containerID, workingDir string, cmd []string, env map[string]string) (*dockerx.ExecResult, error) {
		return &dockerx.ExecResult{ExitCode: 0, Stdout: "hello\n"}, nil
	})
	defer cleanup()

	sr := NewStepRunner(nil, "fake-container", "/workspace")
	exprCtx := NewExpressionContext(nil, nil, nil, NewStepOutputs())
	step := workflow.Step{Run: "echo hello"}

	res, err := sr.RunStep(context.Background(), step, map[string]string{}, "bash", "", nil, exprCtx, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusSuccess || strings.TrimSpace(res.Stdout) != "hello" {
		t.Errorf("expected StatusSuccess and 'hello', got status %s stdout %q", res.Status, res.Stdout)
	}
}

func TestRunStep_WithEnv(t *testing.T) {
	var capturedEnv map[string]string
	cleanup := mockExecCommand(func(ctx context.Context, cli *client.Client, containerID, workingDir string, cmd []string, env map[string]string) (*dockerx.ExecResult, error) {
		capturedEnv = env
		return &dockerx.ExecResult{ExitCode: 0, Stdout: "bar\n"}, nil
	})
	defer cleanup()

	sr := NewStepRunner(nil, "fake-container", "/workspace")
	exprCtx := NewExpressionContext(nil, nil, nil, NewStepOutputs())
	step := workflow.Step{Run: "echo $FOO"}
	env := map[string]string{"FOO": "bar"}

	_, err := sr.RunStep(context.Background(), step, env, "bash", "", nil, exprCtx, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedEnv["FOO"] != "bar" {
		t.Errorf("expected FOO=bar in env, got %v", capturedEnv)
	}
}

func TestRunStep_WithWorkingDirectory(t *testing.T) {
	var capturedDir string
	cleanup := mockExecCommand(func(ctx context.Context, cli *client.Client, containerID, workingDir string, cmd []string, env map[string]string) (*dockerx.ExecResult, error) {
		capturedDir = workingDir
		return &dockerx.ExecResult{ExitCode: 0}, nil
	})
	defer cleanup()

	sr := NewStepRunner(nil, "fake-container", "/workspace")
	exprCtx := NewExpressionContext(nil, nil, nil, NewStepOutputs())
	step := workflow.Step{Run: "pwd"}

	_, err := sr.RunStep(context.Background(), step, map[string]string{}, "bash", "/custom/dir", nil, exprCtx, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedDir != "/custom/dir" {
		t.Errorf("expected workingDir /custom/dir, got %s", capturedDir)
	}
}

func TestRunStep_ContinueOnError(t *testing.T) {
	cleanup := mockExecCommand(func(ctx context.Context, cli *client.Client, containerID, workingDir string, cmd []string, env map[string]string) (*dockerx.ExecResult, error) {
		return &dockerx.ExecResult{ExitCode: 1, Stderr: "failed"}, nil
	})
	defer cleanup()

	sr := NewStepRunner(nil, "fake-container", "/workspace")
	exprCtx := NewExpressionContext(nil, nil, nil, NewStepOutputs())
	step := workflow.Step{Run: "exit 1", ContinueOnError: true}

	res, err := sr.RunStep(context.Background(), step, map[string]string{}, "bash", "", nil, exprCtx, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Outcome != StatusFailure {
		t.Errorf("expected Outcome Failure, got %s", res.Outcome)
	}
	if res.Conclusion != StatusSuccess || res.Status != StatusSuccess {
		t.Errorf("expected Conclusion Success due to continue-on-error, got conclusion %s status %s", res.Conclusion, res.Status)
	}
}

func TestRunStep_Timeout(t *testing.T) {
	cleanup := mockExecCommand(func(ctx context.Context, cli *client.Client, containerID, workingDir string, cmd []string, env map[string]string) (*dockerx.ExecResult, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
			return &dockerx.ExecResult{ExitCode: 0}, nil
		}
	})
	defer cleanup()

	sr := NewStepRunner(nil, "fake-container", "/workspace")
	exprCtx := NewExpressionContext(nil, nil, nil, NewStepOutputs())
	step := workflow.Step{Run: "sleep 10"}

	// Use a tiny jobTimeout (1ms) to test timeout behavior quickly
	res, err := sr.RunStep(context.Background(), step, map[string]string{}, "bash", "", nil, exprCtx, 1*time.Millisecond, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusFailure {
		t.Errorf("expected StatusFailure on timeout, got %s", res.Status)
	}
	if !strings.Contains(res.Stderr, "step timed out") {
		t.Errorf("expected 'step timed out' in stderr, got %q", res.Stderr)
	}
}

// If conditions tests
func TestRunStep_IfCondition_True(t *testing.T) {
	cleanup := mockExecCommand(func(ctx context.Context, cli *client.Client, containerID, workingDir string, cmd []string, env map[string]string) (*dockerx.ExecResult, error) {
		return &dockerx.ExecResult{ExitCode: 0}, nil
	})
	defer cleanup()

	sr := NewStepRunner(nil, "fake-container", "/workspace")
	exprCtx := NewExpressionContext(nil, nil, nil, NewStepOutputs())
	step := workflow.Step{
		If:  "true",
		Run: "echo run",
	}

	res, err := sr.RunStep(context.Background(), step, map[string]string{}, "bash", "", nil, exprCtx, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusSuccess {
		t.Errorf("expected StatusSuccess, got %s", res.Status)
	}
}

func TestRunStep_IfCondition_False(t *testing.T) {
	sr := NewStepRunner(nil, "fake-container", "/workspace")
	exprCtx := NewExpressionContext(nil, nil, nil, NewStepOutputs())
	step := workflow.Step{
		If:  "false",
		Run: "echo skip",
	}

	res, err := sr.RunStep(context.Background(), step, map[string]string{}, "bash", "", nil, exprCtx, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusSkipped {
		t.Errorf("expected StatusSkipped, got %s", res.Status)
	}
}

func TestRunStep_IfCondition_Empty(t *testing.T) {
	cleanup := mockExecCommand(func(ctx context.Context, cli *client.Client, containerID, workingDir string, cmd []string, env map[string]string) (*dockerx.ExecResult, error) {
		return &dockerx.ExecResult{ExitCode: 0}, nil
	})
	defer cleanup()

	sr := NewStepRunner(nil, "fake-container", "/workspace")
	exprCtx := NewExpressionContext(nil, nil, nil, NewStepOutputs())
	step := workflow.Step{
		If:  "",
		Run: "echo run",
	}

	res, err := sr.RunStep(context.Background(), step, map[string]string{}, "bash", "", nil, exprCtx, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusSuccess {
		t.Errorf("expected StatusSuccess, got %s", res.Status)
	}
}

// Interpolation tests
func TestRunStep_WithInterpolation(t *testing.T) {
	var capturedCmd []string
	cleanup := mockExecCommand(func(ctx context.Context, cli *client.Client, containerID, workingDir string, cmd []string, env map[string]string) (*dockerx.ExecResult, error) {
		capturedCmd = cmd
		return &dockerx.ExecResult{ExitCode: 0}, nil
	})
	defer cleanup()

	sr := NewStepRunner(nil, "fake-container", "/workspace")
	env := map[string]string{"GREETING": "hello"}
	exprCtx := NewExpressionContext(env, nil, nil, NewStepOutputs())
	step := workflow.Step{
		Run: "echo ${{ env.GREETING }}",
	}

	_, err := sr.RunStep(context.Background(), step, map[string]string{}, "bash", "", nil, exprCtx, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(capturedCmd) < 4 || capturedCmd[3] != "echo hello" {
		t.Errorf("expected interpolated command 'echo hello', got %v", capturedCmd)
	}
}

func TestRunStep_InterpolationError(t *testing.T) {
	cleanup := mockExecCommand(func(ctx context.Context, cli *client.Client, containerID, workingDir string, cmd []string, env map[string]string) (*dockerx.ExecResult, error) {
		return &dockerx.ExecResult{ExitCode: 0}, nil
	})
	defer cleanup()

	sr := NewStepRunner(nil, "fake-container", "/workspace")
	exprCtx := NewExpressionContext(nil, nil, nil, NewStepOutputs())
	step := workflow.Step{
		Run: "echo ${{ env.INVALID..NAME }}",
	}

	_, err := sr.RunStep(context.Background(), step, map[string]string{}, "bash", "", nil, exprCtx, 0, 0)
	if err == nil || !strings.Contains(err.Error(), "interpolate step run") {
		t.Fatalf("expected interpolation error, got %v", err)
	}
}

// Error handling tests
func TestRunStep_CommandNotFound(t *testing.T) {
	cleanup := mockExecCommand(func(ctx context.Context, cli *client.Client, containerID, workingDir string, cmd []string, env map[string]string) (*dockerx.ExecResult, error) {
		return &dockerx.ExecResult{ExitCode: 127, Stderr: "command not found"}, nil
	})
	defer cleanup()

	sr := NewStepRunner(nil, "fake-container", "/workspace")
	exprCtx := NewExpressionContext(nil, nil, nil, NewStepOutputs())
	step := workflow.Step{
		Run: "nonexistent_command_12345",
	}

	res, err := sr.RunStep(context.Background(), step, map[string]string{}, "bash", "", nil, exprCtx, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 127 || res.Status != StatusFailure {
		t.Errorf("expected ExitCode 127 and StatusFailure, got exitcode %d status %s", res.ExitCode, res.Status)
	}
}

func TestRunStep_ExitCode(t *testing.T) {
	cleanup := mockExecCommand(func(ctx context.Context, cli *client.Client, containerID, workingDir string, cmd []string, env map[string]string) (*dockerx.ExecResult, error) {
		return &dockerx.ExecResult{ExitCode: 42, Stderr: "error exit"}, nil
	})
	defer cleanup()

	sr := NewStepRunner(nil, "fake-container", "/workspace")
	exprCtx := NewExpressionContext(nil, nil, nil, NewStepOutputs())
	step := workflow.Step{
		Run: "exit 42",
	}

	res, err := sr.RunStep(context.Background(), step, map[string]string{}, "bash", "", nil, exprCtx, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 42 || res.Status != StatusFailure {
		t.Errorf("expected ExitCode 42 and StatusFailure, got exitcode %d status %s", res.ExitCode, res.Status)
	}
}

// Additional test cases for step runner secrets and unsupported actions
func TestRunStep_UnsupportedAction(t *testing.T) {
	sr := NewStepRunner(nil, "fake-container", "/workspace")
	exprCtx := NewExpressionContext(nil, nil, nil, NewStepOutputs())
	step := workflow.Step{
		Uses: "actions/unknown-action@v1",
	}

	_, err := sr.RunStep(context.Background(), step, map[string]string{}, "bash", "", nil, exprCtx, 0, 0)
	if err == nil {
		t.Fatal("expected error for unsupported action")
	}
	var target *UnsupportedError
	if !errors.As(err, &target) {
		t.Fatalf("expected UnsupportedError, got %T: %v", err, err)
	}
}

func TestStepRunner_SetSecrets(t *testing.T) {
	sr := NewStepRunner(nil, "fake-container", "/workspace")
	sr.SetSecrets([]string{"MY_SECRET"})
	if len(sr.secrets) != 1 || sr.secrets[0] != "MY_SECRET" {
		t.Errorf("SetSecrets failed, got %v", sr.secrets)
	}
}

func TestRunStep_NoRunNoUses(t *testing.T) {
	sr := NewStepRunner(nil, "fake-container", "/workspace")
	exprCtx := NewExpressionContext(nil, nil, nil, NewStepOutputs())
	step := workflow.Step{}

	_, err := sr.RunStep(context.Background(), step, map[string]string{}, "bash", "", nil, exprCtx, 0, 0)
	if err == nil || !strings.Contains(err.Error(), "neither run nor uses") {
		t.Fatalf("expected error about neither run nor uses, got %v", err)
	}
}

func TestRunStep_SupportedAction_SetupPython(t *testing.T) {
	sr := NewStepRunner(nil, "fake-container", "/workspace")
	exprCtx := NewExpressionContext(nil, nil, nil, NewStepOutputs())
	jobEnv := make(map[string]string)
	step := workflow.Step{
		Uses: "actions/setup-python@v5",
		With: map[string]any{"python-version": "3.12"},
	}

	res, err := sr.RunStep(context.Background(), step, jobEnv, "bash", "", nil, exprCtx, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusSuccess {
		t.Errorf("expected StatusSuccess, got %s", res.Status)
	}
	if jobEnv["python-version"] != "3.12" {
		t.Errorf("expected python-version=3.12 in jobEnv, got %v", jobEnv)
	}
}

func TestRunStep_IfCondition_NonBoolean(t *testing.T) {
	sr := NewStepRunner(nil, "fake-container", "/workspace")
	exprCtx := NewExpressionContext(nil, nil, nil, NewStepOutputs())
	step := workflow.Step{
		If:  "12345",
		Run: "echo test",
	}

	_, err := sr.RunStep(context.Background(), step, map[string]string{}, "bash", "", nil, exprCtx, 0, 0)
	if err == nil || !strings.Contains(err.Error(), "non-boolean") {
		t.Fatalf("expected non-boolean error, got %v", err)
	}
}

