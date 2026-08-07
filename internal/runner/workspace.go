package runner

import (
	"os"
	"path/filepath"
)

// FindRepoRoot finds the git repository root by walking up from startPath.
// Returns the directory containing .git/ or .github/.
// Falls back to the workflow file's directory if neither is found.
func FindRepoRoot(startPath string) (string, error) {
	return findRepoRoot(startPath)
}

// findRepoRoot finds the git repository root by walking up from startPath.
// Returns the directory containing .git/ or .github/.
// Falls back to the workflow file's directory if neither is found.
func findRepoRoot(startPath string) (string, error) {
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(absPath)

	// Walk up until we find .git/
	for {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Fallback: Walk up until we find .github/
	dir = filepath.Dir(absPath)
	for {
		githubDir := filepath.Join(dir, ".github")
		if info, err := os.Stat(githubDir); err == nil && info.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Final fallback: use workflow file's directory
	return filepath.Dir(absPath), nil
}
