package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/dockerx"
	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
	"github.com/docker/docker/client"
)

// JobRunner handles execution of jobs within Docker containers.
type JobRunner struct {
	cli      *client.Client
	offline  bool
	platform string
	crlf     string // "convert", "preserve", or "error"
}

// NewJobRunner creates a new job runner.
func NewJobRunner(cli *client.Client, offline bool, platform string, crlf string) *JobRunner {
	return &JobRunner{
		cli:      cli,
		offline:  offline,
		platform: platform,
		crlf:     crlf,
	}
}

// RunJob executes a job in a Docker container.
func (jr *JobRunner) RunJob(ctx context.Context, job workflow.Job, jobID string, workflowEnv map[string]string, workflowDefaults *workflow.Defaults, wf *workflow.Workflow, workspacePath string, needsCtx map[string]JobNeedsData) (*JobResult, error) {
	// Generate job instance ID
	jobInstanceID := fmt.Sprintf("%s-%s", jobID, generateShortID())
	job.SetInstanceID(jobInstanceID)

	// Evaluate job-level if condition
	if job.If != "" {
		// Create a temporary expression context for job-level if
		githubContext := buildGitHubContext(jobID, jobInstanceID)
		runnerContext := buildRunnerContext()
		exprContext := NewExpressionContext(workflowEnv, githubContext, runnerContext, NewStepOutputs())
		exprContext.SetNeedsContext(needsCtx)
		if job.GetMatrixContext() != nil {
			exprContext.SetMatrix(job.GetMatrixContext())
		}

		shouldRun, err := evaluateIfConditionStatic(job.If, exprContext)
		if err != nil {
			return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("evaluate job if condition: %w", err)}, nil
		}
		if !shouldRun {
			return &JobResult{JobID: jobID, ExitCode: 0, Status: StatusSkipped}, nil
		}
	}

	// Resolve image from runs-on
	runsOn := getRunsOn(job)
	imageName, err := dockerx.ResolveImage(runsOn)
	if err != nil {
		return &JobResult{JobID: jobID, ExitCode: 1, Error: err}, nil
	}

	// Check if job has setup-python
	if setupStep := findSetupPythonStep(job); setupStep != nil {
		pythonVersion := extractPythonVersion(setupStep)
		pythonImage := fmt.Sprintf("python:%s-slim", pythonVersion)

		// Try to pull/use python image
		if err := dockerx.EnsureImage(ctx, jr.cli, pythonImage, jr.offline, jr.platform); err == nil {
			imageName = pythonImage
			fmt.Printf("  ℹ️ Using %s for actions/setup-python\n", pythonImage)
		} else {
			fmt.Printf("  ⚠️ Could not use %s, will auto-install Python\n", pythonImage)
		}
	}

	// Ensure image exists
	if err := dockerx.EnsureImage(ctx, jr.cli, imageName, jr.offline, jr.platform); err != nil {
		return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("ensure image %s: %w\n  Hint: Check image name and run 'docker login' if private", imageName, err)}, nil
	}

	// Create GITHUB_ENV and GITHUB_PATH files
	githubEnvFile := NewGitHubEnvFile(jobInstanceID)
	githubPathFile := NewGitHubPathFile(jobInstanceID)

	// Set paths on job
	job.SetGitHubEnvPath(githubEnvFile.Path())
	job.SetGitHubPathPath(githubPathFile.Path())

	// Prepare GitHub context
	githubContext := buildGitHubContext(jobID, jobInstanceID)
	runnerContext := buildRunnerContext()

	// Setup tracking variables and defer teardown for containers and network
	var networkID string
	var serviceContainerIDs []string
	var containerID string

	defer func() {
		if containerID != "" {
			_ = dockerx.RemoveContainer(ctx, jr.cli, containerID)
		}
		for _, svcID := range serviceContainerIDs {
			_ = dockerx.RemoveContainer(ctx, jr.cli, svcID)
		}
		if networkID != "" {
			_ = dockerx.RemoveNetwork(ctx, jr.cli, networkID)
		}
	}()

	// Ensure /tmp/gacils exists on host for bind mount
	if err := os.MkdirAll("/tmp/gacils", 0755); err != nil {
		return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("create /tmp/gacils: %w", err)}, nil
	}

	// Prepare base job environment (workflow > job > github > container)
	jobEnv := make(map[string]string)
	for k, v := range workflowEnv {
		jobEnv[k] = fmt.Sprintf("%v", v)
	}
	for k, v := range job.Env {
		jobEnv[k] = fmt.Sprintf("%v", v)
	}

	// Setup service containers and custom network if job has services
	var networkName string
	if len(job.Services) > 0 {
		networkName = fmt.Sprintf("gacils-net-%s", jobInstanceID)
		netID, err := dockerx.CreateNetwork(ctx, jr.cli, networkName)
		if err != nil {
			return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("create network: %w", err)}, nil
		}
		networkID = netID

		var svcNames []string
		for sName := range job.Services {
			svcNames = append(svcNames, sName)
		}
		sort.Strings(svcNames)

		for _, sName := range svcNames {
			svc := job.Services[sName]

			if err := dockerx.EnsureImage(ctx, jr.cli, svc.Image, jr.offline, jr.platform); err != nil {
				return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("ensure service image %s: %w\n  Hint: Check image name and run 'docker login' if private", svc.Image, err)}, nil
			}

			svcConfig := dockerx.ServiceConfig{
				Name:    sName,
				Image:   svc.Image,
				Env:     svc.Env,
				Ports:   svc.Ports,
				Options: svc.Options,
			}

			svcID, primaryPort, err := dockerx.CreateServiceContainer(ctx, jr.cli, networkName, sName, svcConfig)
			if err != nil {
				return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("create service %s: %w", sName, err)}, nil
			}
			serviceContainerIDs = append(serviceContainerIDs, svcID)

			if err := dockerx.StartContainer(ctx, jr.cli, svcID); err != nil {
				return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("start service %s: %w", sName, err)}, nil
			}

			if err := dockerx.WaitForServiceReady(ctx, jr.cli, svcID, primaryPort, 30*time.Second); err != nil {
				return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("wait for service %s: %w\n  Hint: Check Docker daemon is running and network connectivity", sName, err)}, nil
			}

			svcUpper := strings.ReplaceAll(strings.ToUpper(sName), "-", "_")
			jobEnv[svcUpper+"_HOST"] = sName
			if primaryPort != "" {
				jobEnv[svcUpper+"_PORT"] = primaryPort
			}
		}
	}

	// Create container with GitHub and runner context
	workingDir := "/github/workspace"
	cID, err := dockerx.CreateContainer(ctx, jr.cli, imageName, workingDir, githubEnvFile.Path(), githubPathFile.Path(), githubContext, runnerContext)
	if err != nil {
		return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("create container: %w", err)}, nil
	}
	containerID = cID

	// Connect main container to custom network if network was created
	if networkID != "" {
		if err := dockerx.ConnectNetwork(ctx, jr.cli, networkID, containerID); err != nil {
			return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("connect main container to network: %w", err)}, nil
		}
	}

	// Start container
	if err := dockerx.StartContainer(ctx, jr.cli, containerID); err != nil {
		return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("start container: %w", err)}, nil
	}

	// Copy workspace into container
	if err := dockerx.CopyWorkspace(ctx, jr.cli, dockerx.CopyConfig{
		SourcePath:  workspacePath,
		TargetPath:  "/github/workspace",
		ContainerID: containerID,
	}); err != nil {
		_ = dockerx.RemoveContainer(ctx, jr.cli, containerID)
		return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("copy workspace: %w", err)}, nil
	}

	// Create directory structure for GITHUB_ENV, GITHUB_PATH, GITHUB_OUTPUT in the container
	// The volume is mounted at /tmp/gacils, so we need to create the job-specific directories
	setupCmd := []string{"mkdir", "-p", filepath.Join("/tmp/gacils/jobs", jobInstanceID), filepath.Join("/tmp/gacils/steps")}
	if _, err := dockerx.ExecCommand(ctx, jr.cli, containerID, "/", setupCmd, nil); err != nil {
		_ = dockerx.RemoveContainer(ctx, jr.cli, containerID)
		return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("create directories in container: %w", err)}, nil
	}

	// Create step-specific output directories
	for _, step := range job.Steps {
		if step.ID != "" {
			stepDir := filepath.Join("/tmp/gacils/steps", step.ID)
			mkdirCmd := []string{"mkdir", "-p", stepDir}
			if _, err := dockerx.ExecCommand(ctx, jr.cli, containerID, "/", mkdirCmd, nil); err != nil {
				_ = dockerx.RemoveContainer(ctx, jr.cli, containerID)
				return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("create step output dir: %w", err)}, nil
			}
		}
	}

	// Pre-create working directories from workflow/job defaults
	if workflowDefaults != nil && workflowDefaults.Run != nil && workflowDefaults.Run.WorkingDirectory != "" {
		wfDir := workflowDefaults.Run.WorkingDirectory
		if !filepath.IsAbs(wfDir) {
			wfDir = filepath.Join("/github/workspace", wfDir)
		}
		mkdirCmd := []string{"mkdir", "-p", wfDir}
		if _, err := dockerx.ExecCommand(ctx, jr.cli, containerID, "/", mkdirCmd, nil); err != nil {
			_ = dockerx.RemoveContainer(ctx, jr.cli, containerID)
			return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("create workflow working dir: %w", err)}, nil
		}
	}

	// Create job working directory
	jobWorkingDir := workingDir
	if job.Defaults != nil && job.Defaults.Run != nil && job.Defaults.Run.WorkingDirectory != "" {
		jobDir := job.Defaults.Run.WorkingDirectory
		if !filepath.IsAbs(jobDir) {
			jobDir = filepath.Join("/github/workspace", jobDir)
		}
		mkdirCmd := []string{"mkdir", "-p", jobDir}
		if _, err := dockerx.ExecCommand(ctx, jr.cli, containerID, "/", mkdirCmd, nil); err != nil {
			_ = dockerx.RemoveContainer(ctx, jr.cli, containerID)
			return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("create job working dir: %w", err)}, nil
		}
		jobWorkingDir = jobDir
	}

	// Create step working directories
	for _, step := range job.Steps {
		if step.WorkingDirectory != "" {
			stepDir := step.WorkingDirectory
			if !filepath.IsAbs(stepDir) {
				stepDir = filepath.Join(jobWorkingDir, stepDir)
			}
			mkdirCmd := []string{"mkdir", "-p", stepDir}
			if _, err := dockerx.ExecCommand(ctx, jr.cli, containerID, "/", mkdirCmd, nil); err != nil {
				_ = dockerx.RemoveContainer(ctx, jr.cli, containerID)
				return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("create step working dir: %w", err)}, nil
			}
		}
	}

	// Auto-install Python if python/pip commands are detected on ubuntu-based images without setup-python
	if jobHasPythonCommands(job) {
		if err := ensurePythonInstalled(ctx, jr.cli, containerID, jobHasSetupPython(job), imageName); err != nil {
			fmt.Printf("  ⚠️ Could not auto-install python: %v\n", err)
			fmt.Printf("  ⚠️ Workflow uses Python commands but no actions/setup-python found.\n     Consider adding setup-python for better local compatibility.\n")
		}
	}

	// Auto-install Docker CLI if docker commands are detected
	if jobHasDockerCommands(job) {
		if err := ensureDockerCLIAvailable(ctx, jr.cli, containerID, imageName); err != nil {
			fmt.Printf("  ⚠️ Could not auto-install docker CLI: %v\n", err)
		}
	}

	// Create step runner
	stepRunner := NewStepRunner(jr.cli, containerID, workingDir, jr.crlf)

	// Ensure PATH is initialized from container
	if jobEnv["PATH"] == "" {
		jobEnv["PATH"] = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	}

	// Create step outputs store
	stepOutputs := NewStepOutputs()

	// Create expression context with matrix support
	exprContext := NewExpressionContext(jobEnv, githubContext, runnerContext, stepOutputs)
	exprContext.SetNeedsContext(needsCtx)
	if job.GetMatrixContext() != nil {
		exprContext.SetMatrix(job.GetMatrixContext())
	}

	// Parse job timeout
	var jobTimeout time.Duration
	if job.TimeoutMinutes > 0 {
		jobTimeout, _ = ParseTimeoutMinutes(job.TimeoutMinutes)
	}

	// Run steps sequentially
	var stepResults []*StepResult
	exitCode := 0
	var firstError error

	for i, step := range job.Steps {
		// Set current step ID for expression context
		exprContext.SetCurrentStepID(step.ID)
		exprContext.SetStepResults(stepResults)

		// Resolve shell with precedence: step.shell > job.defaults.run.shell > workflow.defaults.run.shell > bash
		shell := resolveShell(step, job.Defaults, workflowDefaults)

		// Resolve working directory with precedence: step.working-directory > job.defaults.run.working-directory > workflow.defaults.run.working-directory > /github/workspace
		// Note: For step.working-directory, if relative, it's relative to the parent working directory
		// First, determine the parent working directory (job or workflow default)
		parentWorkingDir := workingDir
		if job.Defaults != nil && job.Defaults.Run != nil && job.Defaults.Run.WorkingDirectory != "" {
			parentWorkingDir = resolveWorkingDirectory(workflow.Step{WorkingDirectory: job.Defaults.Run.WorkingDirectory}, nil, workflowDefaults, workingDir)
		} else if workflowDefaults != nil && workflowDefaults.Run != nil && workflowDefaults.Run.WorkingDirectory != "" {
			parentWorkingDir = resolveWorkingDirectory(workflow.Step{WorkingDirectory: workflowDefaults.Run.WorkingDirectory}, nil, workflowDefaults, workingDir)
		}

		// Now resolve the step's working directory relative to parent
		stepWorkingDir := resolveWorkingDirectory(step, nil, nil, parentWorkingDir)

		// Create step-specific GITHUB_OUTPUT file
		githubOutputFile := NewGitHubOutputFile(jobInstanceID, step.ID)

		// Build step environment with env precedence: step.env > job.env > workflow.env > github > container
		stepEnv := make(map[string]string)
		for k, v := range jobEnv {
			stepEnv[k] = v
		}
		for k, v := range step.Env {
			stepEnv[k] = fmt.Sprintf("%v", v)
		}

		// Interpolate expressions in step.env
		interpolatedStepEnv, err := exprContext.InterpolateMap(stepEnv)
		if err != nil {
			firstError = fmt.Errorf("interpolate step env: %w", err)
			if ecErr, ok := err.(interface{ Code() int }); ok {
				exitCode = ecErr.Code()
			} else {
				exitCode = 1
			}
			break
		}
		stepEnv = interpolatedStepEnv

		// Add GITHUB_OUTPUT to step environment
		stepEnv["GITHUB_OUTPUT"] = githubOutputFile.Path()

		// Add GITHUB_ENV and GITHUB_PATH to step environment (from container env)
		if githubEnvFile != nil {
			stepEnv["GITHUB_ENV"] = githubEnvFile.Path()
		}
		if githubPathFile != nil {
			stepEnv["GITHUB_PATH"] = githubPathFile.Path()
		}

		// Collect secrets to mask in stdout/stderr
		var wfSecrets map[string]any
		if wf != nil {
			wfSecrets = wf.Secrets
		}
		stepSecrets := CollectSecrets(wfSecrets, workflowEnv, jobEnv, stepEnv)
		stepRunner.SetSecrets(stepSecrets)

		// Run step with timeout
		result, err := stepRunner.RunStep(ctx, step, stepEnv, shell, stepWorkingDir, githubOutputFile, exprContext, jobTimeout, 0)
		if err != nil {
			firstError = err
			var uerr *UnsupportedError
			if errors.As(err, &uerr) {
				exitCode = uerr.Code()
			} else if ecErr, ok := err.(interface{ Code() int }); ok {
				exitCode = ecErr.Code()
			} else {
				exitCode = 1
			}
			break
		}

		// Update step results for status function evaluation
		stepResults = append(stepResults, result)
		exprContext.SetStepResults(stepResults)

		fmt.Printf("    Step %d stdout:\n%s", i+1, result.Stdout)
		if result.Stderr != "" {
			fmt.Printf("    Step %d stderr:\n%s", i+1, result.Stderr)
		}

		// Check for timeout
		if result.ExitCode == 5 {
			exitCode = 5
			firstError = fmt.Errorf("step %d timed out", i+1)
			break
		}

		// Parse GITHUB_ENV after step - read from container since we're using a volume
		catCmd := []string{"cat", githubEnvFile.Path()}
		catResult, err := dockerx.ExecCommand(ctx, jr.cli, containerID, "/", catCmd, nil)
		if err != nil {
			firstError = fmt.Errorf("read GITHUB_ENV from container: %w", err)
			exitCode = 1
			break
		}

		// Parse the content
		envVars := make(map[string]string)
		lines := strings.Split(catResult.Stdout, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				envVars[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}

		// Merge into job environment
		for k, v := range envVars {
			jobEnv[k] = v
		}

		// Clear the file in container
		clearCmd := []string{"truncate", "-s", "0", githubEnvFile.Path()}
		if _, err := dockerx.ExecCommand(ctx, jr.cli, containerID, "/", clearCmd, nil); err != nil {
			firstError = fmt.Errorf("clear GITHUB_ENV: %w", err)
			exitCode = 1
			break
		}

		// Update expression context with new env
		exprContext = NewExpressionContext(jobEnv, githubContext, runnerContext, stepOutputs)
		if job.GetMatrixContext() != nil {
			exprContext.SetMatrix(job.GetMatrixContext())
		}
		exprContext.SetStepResults(stepResults)

		// Parse GITHUB_PATH after step - read from container since we're using a volume
		catPathCmd := []string{"cat", githubPathFile.Path()}
		catPathResult, err := dockerx.ExecCommand(ctx, jr.cli, containerID, "/", catPathCmd, nil)
		if err != nil {
			firstError = fmt.Errorf("read GITHUB_PATH from container: %w", err)
			exitCode = 1
			break
		}

		// Parse the content
		var paths []string
		pathLines := strings.Split(catPathResult.Stdout, "\n")
		for _, line := range pathLines {
			line = strings.TrimSpace(line)
			if line != "" {
				paths = append(paths, line)
			}
		}

		if len(paths) > 0 {
			// Prepend to PATH
			existingPath := jobEnv["PATH"]
			newPath := strings.Join(paths, ":") + ":" + existingPath
			jobEnv["PATH"] = newPath
			exprContext = NewExpressionContext(jobEnv, githubContext, runnerContext, stepOutputs)
			if job.GetMatrixContext() != nil {
				exprContext.SetMatrix(job.GetMatrixContext())
			}
			exprContext.SetStepResults(stepResults)
		}

		// Clear the file in container
		clearPathCmd := []string{"truncate", "-s", "0", githubPathFile.Path()}
		if _, err := dockerx.ExecCommand(ctx, jr.cli, containerID, "/", clearPathCmd, nil); err != nil {
			firstError = fmt.Errorf("clear GITHUB_PATH: %w", err)
			exitCode = 1
			break
		}

		// Parse GITHUB_OUTPUT after step - read from container since we're using a volume
		catOutputCmd := []string{"cat", githubOutputFile.Path()}
		catOutputResult, err := dockerx.ExecCommand(ctx, jr.cli, containerID, "/", catOutputCmd, nil)
		if err != nil {
			firstError = fmt.Errorf("read GITHUB_OUTPUT from container: %w", err)
			exitCode = 1
			break
		}

		// Parse the content
		outputVars := make(map[string]string)
		outputLines := strings.Split(catOutputResult.Stdout, "\n")
		for _, line := range outputLines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				outputVars[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}

		// Store step outputs
		stepOutputs.SetOutputs(step.ID, outputVars)

		// Clear the file in container
		clearOutputCmd := []string{"truncate", "-s", "0", githubOutputFile.Path()}
		if _, err := dockerx.ExecCommand(ctx, jr.cli, containerID, "/", clearOutputCmd, nil); err != nil {
			firstError = fmt.Errorf("clear GITHUB_OUTPUT: %w", err)
			exitCode = 1
			break
		}

		if result.ExitCode != 0 {
			if step.ContinueOnError {
				fmt.Printf("    Step %d failed with exit code %d (continue-on-error)\n", i+1, result.ExitCode)
			} else {
				exitCode = result.ExitCode
				firstError = fmt.Errorf("step %d failed with exit code %d", i+1, result.ExitCode)
				break
			}
		}
	}

	// Stop the container gracefully (not wait) — it runs `tail -f /dev/null`
	// as a keepalive, so it will never exit on its own. Stop after steps complete.
	if err := dockerx.StopContainer(ctx, jr.cli, containerID, 3); err != nil {
		// If stop fails (e.g., already stopped), force remove
		_ = dockerx.RemoveContainer(ctx, jr.cli, containerID)
		if firstError == nil {
			firstError = fmt.Errorf("stop container: %w", err)
			exitCode = 1
		}
	}

	// Note: We do NOT inspect the container exit code here because the container's
	// entrypoint is `tail -f /dev/null` which is killed by the stop call above.
	// The exit code from that kill (e.g., 137) is meaningless — the job's
	// actual success/failure is determined by the step results already collected.

	status := StatusSuccess
	if exitCode != 0 {
		status = StatusFailure
	}

	return &JobResult{
		JobID:    jobID,
		ExitCode: exitCode,
		Status:   status,
		Error:    firstError,
		Steps:    stepResults,
	}, nil
}

// evaluateIfConditionStatic evaluates an if condition without running a step.
func evaluateIfConditionStatic(condition string, exprContext *ExpressionContext) (bool, error) {
	interpolated, err := exprContext.Interpolate(condition)
	if err != nil {
		return false, err
	}

	// Also handle bare expressions (without ${{ }})
	trimmed := strings.TrimSpace(interpolated)
	if !strings.HasPrefix(trimmed, "${{") {
		interpolated = "${{" + trimmed + "}}"
	}

	result, err := exprContext.Interpolate(interpolated)
	if err != nil {
		return false, err
	}

	switch strings.ToLower(strings.TrimSpace(result)) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	default:
		return false, fmt.Errorf("if condition evaluated to non-boolean: %s", result)
	}
}

// getRunsOn extracts the runs-on value from a job.
func getRunsOn(job workflow.Job) string {
	// runs-on can be string or []string - handle both
	switch v := job.RunsOn.(type) {
	case string:
		return v
	case []interface{}:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				return s
			}
		}
	case []string:
		if len(v) > 0 {
			return v[0]
		}
	}
	return "ubuntu-latest" // default
}

// generateShortID generates a short unique ID.
func generateShortID() string {
	return fmt.Sprintf("%d", os.Getpid())
}

// buildGitHubContext builds the GitHub context environment variables.
func buildGitHubContext(jobID, jobInstanceID string) map[string]string {
	return map[string]string{
		"GITHUB_WORKFLOW":    "workflow",
		"GITHUB_JOB":         jobID,
		"GITHUB_RUN_ID":      jobInstanceID,
		"GITHUB_RUN_NUMBER":  "1",
		"GITHUB_RUN_ATTEMPT": "1",
		"GITHUB_REPOSITORY":  "local/repo",
		"GITHUB_REF":         "refs/heads/main",
		"GITHUB_REF_NAME":    "main",
		"GITHUB_REF_TYPE":    "branch",
		"GITHUB_SHA":         "0000000000000000000000000000000000000000",
		"GITHUB_ACTOR":       "gacils",
		"GITHUB_WORKSPACE":   "/github/workspace",
	}
}

// buildRunnerContext builds the runner context environment variables.
func buildRunnerContext() map[string]string {
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "X64"
	} else if arch == "arm64" {
		arch = "ARM64"
	}
	return map[string]string{
		"RUNNER_OS":         "Linux",
		"RUNNER_ARCH":       arch,
		"RUNNER_NAME":       "gacils-local",
		"RUNNER_TEMP":       "/tmp/gacils",
		"RUNNER_TOOL_CACHE": "/opt/hostedtoolcache",
	}
}

// resolveShell resolves the shell with precedence.
func resolveShell(step workflow.Step, jobDefaults, workflowDefaults *workflow.Defaults) string {
	if step.Shell != "" {
		return step.Shell
	}
	if jobDefaults != nil && jobDefaults.Run != nil && jobDefaults.Run.Shell != "" {
		return jobDefaults.Run.Shell
	}
	if workflowDefaults != nil && workflowDefaults.Run != nil && workflowDefaults.Run.Shell != "" {
		return workflowDefaults.Run.Shell
	}
	return "bash"
}

// resolveWorkingDirectory resolves the working directory with precedence.
func resolveWorkingDirectory(step workflow.Step, jobDefaults, workflowDefaults *workflow.Defaults, defaultWorkingDir string) string {
	var dir string
	if step.WorkingDirectory != "" {
		dir = step.WorkingDirectory
	} else if jobDefaults != nil && jobDefaults.Run != nil && jobDefaults.Run.WorkingDirectory != "" {
		dir = jobDefaults.Run.WorkingDirectory
	} else if workflowDefaults != nil && workflowDefaults.Run != nil && workflowDefaults.Run.WorkingDirectory != "" {
		dir = workflowDefaults.Run.WorkingDirectory
	} else {
		return defaultWorkingDir
	}

	// If relative, resolve against defaultWorkingDir
	if !filepath.IsAbs(dir) {
		return filepath.Join(defaultWorkingDir, dir)
	}
	return dir
}

// findSetupPythonStep finds the actions/setup-python step in a job
func findSetupPythonStep(job workflow.Job) *workflow.Step {
	for i := range job.Steps {
		step := &job.Steps[i]
		if step.Uses != "" && strings.Contains(step.Uses, "actions/setup-python") {
			return step
		}
	}
	return nil
}

// extractPythonVersion gets python-version from setup-python step
func extractPythonVersion(step *workflow.Step) string {
	if step == nil || step.With == nil {
		return "3.12" // default
	}
	if version, ok := step.With["python-version"]; ok {
		v := fmt.Sprintf("%v", version)
		if v != "" && !strings.Contains(v, "${{") {
			return v
		}
	}
	return "3.12" // default
}
