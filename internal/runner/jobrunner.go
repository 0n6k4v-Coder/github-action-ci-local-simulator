package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/dockerx"
	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
	"github.com/docker/docker/client"
)

// JobRunner handles execution of jobs within Docker containers.
type JobRunner struct {
	cli *client.Client
}

// NewJobRunner creates a new job runner.
func NewJobRunner(cli *client.Client) *JobRunner {
	return &JobRunner{
		cli: cli,
	}
}

// RunJob executes a job in a Docker container.
func (jr *JobRunner) RunJob(ctx context.Context, job workflow.Job, jobID string, workflowEnv map[string]string, workflowDefaults *workflow.Defaults, wf *workflow.Workflow, workspacePath string) (*JobResult, error) {
	// Generate job instance ID
	jobInstanceID := fmt.Sprintf("%s-%s", jobID, generateShortID())
	job.SetInstanceID(jobInstanceID)

	// Resolve image from runs-on
	runsOn := getRunsOn(job)
	imageName, err := dockerx.ResolveImage(runsOn)
	if err != nil {
		return &JobResult{JobID: jobID, ExitCode: 1, Error: err}, nil
	}

	// Ensure image exists
	if err := dockerx.EnsureImage(ctx, jr.cli, imageName); err != nil {
		return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("ensure image: %w", err)}, nil
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

	// Ensure /tmp/gacils exists on host for bind mount
	if err := os.MkdirAll("/tmp/gacils", 0755); err != nil {
		return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("create /tmp/gacils: %w", err)}, nil
	}

	// Create container with GitHub and runner context
	workingDir := "/github/workspace"
	containerID, err := dockerx.CreateContainer(ctx, jr.cli, imageName, workingDir, githubEnvFile.Path(), githubPathFile.Path(), githubContext, runnerContext)
	if err != nil {
		return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("create container: %w", err)}, nil
	}

	// Start container
	if err := dockerx.StartContainer(ctx, jr.cli, containerID); err != nil {
		_ = dockerx.RemoveContainer(ctx, jr.cli, containerID)
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

	// Create step runner
	stepRunner := NewStepRunner(jr.cli, containerID, workingDir)

	// Prepare base job environment (workflow > job > github > container)
	jobEnv := make(map[string]string)
	// Workflow env
	for k, v := range workflowEnv {
		jobEnv[k] = fmt.Sprintf("%v", v)
	}
	// Job env (overrides workflow)
	for k, v := range job.Env {
		jobEnv[k] = fmt.Sprintf("%v", v)
	}
	
	// Ensure PATH is initialized from container
	if jobEnv["PATH"] == "" {
		jobEnv["PATH"] = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	}

	// Create step outputs store
	stepOutputs := NewStepOutputs()

	// Create expression context
	exprContext := NewExpressionContext(jobEnv, githubContext, runnerContext, stepOutputs)

	// Run steps sequentially
	var stepResults []*StepResult
	exitCode := 0
	var firstError error

	for i, step := range job.Steps {
		// Set current step ID for expression context
		exprContext.SetCurrentStepID(step.ID)

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
			exitCode = 1
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

		// Run step
		result, err := stepRunner.RunStep(ctx, step, stepEnv, shell, stepWorkingDir, githubOutputFile, exprContext)
		if err != nil {
			// Check if continue-on-error
			if step.ContinueOnError {
				stepResults = append(stepResults, &StepResult{
					ExitCode: 1,
					Stdout:   "",
					Stderr:   err.Error(),
				})
				continue
			}
			firstError = err
			exitCode = 1
			break
		}

		fmt.Printf("    Step %d stdout:\n%s", i+1, result.Stdout)
		if result.Stderr != "" {
			fmt.Printf("    Step %d stderr:\n%s", i+1, result.Stderr)
		}

		// Check if file exists in container
		checkCmd := []string{"ls", "-la", githubEnvFile.Path()}
		_, _ = dockerx.ExecCommand(ctx, jr.cli, containerID, "/", checkCmd, nil)

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
		
		// Update expression context
		exprContext = NewExpressionContext(jobEnv, githubContext, runnerContext, stepOutputs)

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
		
		// Parse the content (supports both single-line and multiline)
		outputs := make(map[string]string)
		content := catOutputResult.Stdout
		outputLines := strings.Split(content, "\n")
		i := 0
		for i < len(outputLines) {
			line := strings.TrimSpace(outputLines[i])
			if line == "" {
				i++
				continue
			}

			// Check for multiline format
			if strings.Contains(line, "<<") && !strings.HasPrefix(line, "<<") {
				parts := strings.SplitN(line, "<<", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					delimiter := strings.TrimSpace(parts[1])
					i++
					var valueLines []string
					for i < len(outputLines) {
						if strings.TrimSpace(outputLines[i]) == delimiter {
							break
						}
						valueLines = append(valueLines, outputLines[i])
						i++
					}
					outputs[key] = strings.Join(valueLines, "\n")
				}
			} else if strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					outputs[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
				}
			}
			i++
		}
		
		if len(outputs) > 0 {
			stepOutputs.SetOutputs(step.ID, outputs)
		}
		
		// Clear the file in container
		clearOutputCmd := []string{"truncate", "-s", "0", githubOutputFile.Path()}
		if _, err := dockerx.ExecCommand(ctx, jr.cli, containerID, "/", clearOutputCmd, nil); err != nil {
			firstError = fmt.Errorf("clear GITHUB_OUTPUT: %w", err)
			exitCode = 1
			break
		}

		stepResults = append(stepResults, result)
		if result.ExitCode != 0 {
			if step.ContinueOnError {
				continue
			}
			exitCode = result.ExitCode
			firstError = fmt.Errorf("step %d failed with exit code %d", i+1, result.ExitCode)
			break
		}
	}

	// Cleanup container
	if err := dockerx.RemoveContainer(ctx, jr.cli, containerID); err != nil {
		// Log but don't fail
		fmt.Printf("Warning: failed to remove container: %v\n", err)
	}

	return &JobResult{
		JobID:    jobID,
		Steps:    stepResults,
		ExitCode: exitCode,
		Error:    firstError,
	}, nil
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
		"GITHUB_WORKFLOW":      "workflow",
		"GITHUB_JOB":           jobID,
		"GITHUB_RUN_ID":        jobInstanceID,
		"GITHUB_RUN_NUMBER":    "1",
		"GITHUB_RUN_ATTEMPT":   "1",
		"GITHUB_REPOSITORY":    "local/repo",
		"GITHUB_REF":           "refs/heads/main",
		"GITHUB_REF_NAME":      "main",
		"GITHUB_REF_TYPE":      "branch",
		"GITHUB_SHA":           "0000000000000000000000000000000000000000",
		"GITHUB_ACTOR":         "gacils",
		"GITHUB_WORKSPACE":     "/github/workspace",
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