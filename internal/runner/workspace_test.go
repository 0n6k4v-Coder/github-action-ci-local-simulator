package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRepoRoot_WithGitDir(t *testing.T) {
	// Create temp repo structure
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")
	os.MkdirAll(gitDir, 0755)

	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	os.MkdirAll(workflowsDir, 0755)

	workflowPath := filepath.Join(workflowsDir, "ci.yml")
	os.WriteFile(workflowPath, []byte("name: test"), 0644)

	// Also create some repo files
	os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module test"), 0644)

	root, err := findRepoRoot(workflowPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if root != tempDir {
		t.Errorf("expected repo root %s, got %s", tempDir, root)
	}
}

func TestFindRepoRoot_Fallback(t *testing.T) {
	// Workflow in directory with no .git/ or .github/
	tempDir := t.TempDir()
	workflowPath := filepath.Join(tempDir, "workflow.yml")
	os.WriteFile(workflowPath, []byte("name: test"), 0644)

	root, err := findRepoRoot(workflowPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Fallback: workflow's directory
	expected := filepath.Dir(workflowPath)
	if root != expected {
		t.Errorf("expected fallback %s, got %s", expected, root)
	}
}

func TestFindRepoRoot_WithGithubDir(t *testing.T) {
	tempDir := t.TempDir()
	githubDir := filepath.Join(tempDir, ".github")
	os.MkdirAll(githubDir, 0755)

	workflowsDir := filepath.Join(githubDir, "workflows")
	os.MkdirAll(workflowsDir, 0755)

	workflowPath := filepath.Join(workflowsDir, "ci.yml")
	os.WriteFile(workflowPath, []byte("name: test"), 0644)

	root, err := findRepoRoot(workflowPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if root != tempDir {
		t.Errorf("expected %s, got %s", tempDir, root)
	}
}

func TestFindRepoRoot_NestedWorkflows(t *testing.T) {
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")
	os.MkdirAll(gitDir, 0755)

	nestedDir := filepath.Join(tempDir, ".github", "workflows", "ci", "deploy")
	os.MkdirAll(nestedDir, 0755)

	workflowPath := filepath.Join(nestedDir, "production.yml")
	os.WriteFile(workflowPath, []byte("name: test"), 0644)

	root, err := findRepoRoot(workflowPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if root != tempDir {
		t.Errorf("expected %s, got %s", tempDir, root)
	}
}

func TestFindRepoRoot_AbsolutePath(t *testing.T) {
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")
	os.MkdirAll(gitDir, 0755)

	absWorkflowPath := filepath.Join(tempDir, "ci.yml")
	os.WriteFile(absWorkflowPath, []byte("name: test"), 0644)

	root, err := FindRepoRoot(absWorkflowPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != tempDir {
		t.Errorf("expected %s, got %s", tempDir, root)
	}
}

func TestFindRepoRoot_RelativePath(t *testing.T) {
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")
	os.MkdirAll(gitDir, 0755)

	workflowPath := filepath.Join(tempDir, "ci.yml")
	os.WriteFile(workflowPath, []byte("name: test"), 0644)

	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	_ = os.Chdir(tempDir)

	root, err := FindRepoRoot("ci.yml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	evalRoot, _ := filepath.EvalSymlinks(root)
	evalTemp, _ := filepath.EvalSymlinks(tempDir)
	if evalRoot != evalTemp {
		t.Errorf("expected %s, got %s", evalTemp, evalRoot)
	}
}

func TestFindRepoRoot_Symlink(t *testing.T) {
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")
	os.MkdirAll(gitDir, 0755)

	realWorkflow := filepath.Join(tempDir, "ci.yml")
	os.WriteFile(realWorkflow, []byte("name: test"), 0644)

	symlinkPath := filepath.Join(tempDir, "link_ci.yml")
	if err := os.Symlink(realWorkflow, symlinkPath); err != nil {
		t.Skip("symlinks not supported on this environment")
	}

	root, err := FindRepoRoot(symlinkPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	evalRoot, _ := filepath.EvalSymlinks(root)
	evalTempDir, _ := filepath.EvalSymlinks(tempDir)
	if evalRoot != evalTempDir {
		t.Errorf("expected %s, got %s", evalTempDir, evalRoot)
	}
}

func TestFindRepoRoot_PermissionDenied(t *testing.T) {
	tempDir := t.TempDir()
	restrictedDir := filepath.Join(tempDir, "restricted")
	os.MkdirAll(restrictedDir, 0000)
	defer os.Chmod(restrictedDir, 0755)

	// FindRepoRoot should still handle permission errors gracefully without crashing
	_, err := FindRepoRoot(filepath.Join(restrictedDir, "ci.yml"))
	// Error or fallback expected, shouldn't panic
	if err != nil {
		t.Logf("got expected error or result: %v", err)
	}
}

func TestFindRepoRoot_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	workflowPath := filepath.Join(tempDir, "ci.yml")
	os.WriteFile(workflowPath, []byte("name: test"), 0644)

	root, err := FindRepoRoot(workflowPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != tempDir {
		t.Errorf("expected fallback to %s, got %s", tempDir, root)
	}
}
