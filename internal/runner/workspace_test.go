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
