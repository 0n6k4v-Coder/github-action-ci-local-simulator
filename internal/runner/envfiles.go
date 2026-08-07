package runner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GitHubEnvFile handles the GITHUB_ENV file for a job.
type GitHubEnvFile struct {
	path string
}

// NewGitHubEnvFile creates a new GitHubEnvFile for a job.
func NewGitHubEnvFile(jobInstanceID string) *GitHubEnvFile {
	path := filepath.Join("/tmp/gacils/jobs", jobInstanceID, "github_env")
	return &GitHubEnvFile{path: path}
}

// Path returns the path to the GITHUB_ENV file.
func (f *GitHubEnvFile) Path() string {
	return f.path
}

// EnsureDir creates the directory for the file.
func (f *GitHubEnvFile) EnsureDir() error {
	return os.MkdirAll(filepath.Dir(f.path), 0755)
}

// Write writes a key-value pair to the GITHUB_ENV file.
// Supports both single-line and multiline formats.
func (f *GitHubEnvFile) Write(key, value string) error {
	if err := f.EnsureDir(); err != nil {
		return err
	}
	file, err := os.OpenFile(f.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Check if value contains newlines (multiline)
	if strings.Contains(value, "\n") {
		// Use heredoc format with a delimiter
		delimiter := "EOF_" + key
		_, err = fmt.Fprintf(file, "%s<<%s\n%s\n%s\n", key, delimiter, value, delimiter)
	} else {
		_, err = fmt.Fprintf(file, "%s=%s\n", key, value)
	}
	return err
}

// ReadAndParse reads the GITHUB_ENV file, parses it, and returns the parsed variables.
// It also truncates the file to avoid reapplying values.
// Supports both single-line (KEY=value) and multiline (KEY<<DELIMITER ... DELIMITER) formats.
func (f *GitHubEnvFile) ReadAndParse() (map[string]string, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	vars := make(map[string]string)
	content := string(data)
	lines := strings.Split(content, "\n")
	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			i++
			continue
		}

		// Check for multiline format (KEY<<DELIMITER)
		if strings.Contains(line, "<<") && !strings.HasPrefix(line, "<<") {
			// This is a heredoc start: KEY<<DELIMITER
			parts := strings.SplitN(line, "<<", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				delimiter := strings.TrimSpace(parts[1])
				i++
				var valueLines []string
				for i < len(lines) {
					if strings.TrimSpace(lines[i]) == delimiter {
						break
					}
					valueLines = append(valueLines, lines[i])
					i++
				}
				vars[key] = strings.Join(valueLines, "\n")
			}
		} else if strings.Contains(line, "=") {
			// Single-line format: KEY=value
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				vars[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
		i++
	}

	// Truncate the file
	if err := os.WriteFile(f.path, []byte(""), 0644); err != nil {
		return nil, err
	}

	return vars, nil
}

// GitHubPathFile handles the GITHUB_PATH file for a job.
type GitHubPathFile struct {
	path string
}

// NewGitHubPathFile creates a new GitHubPathFile for a job.
func NewGitHubPathFile(jobInstanceID string) *GitHubPathFile {
	path := filepath.Join("/tmp/gacils/jobs", jobInstanceID, "github_path")
	return &GitHubPathFile{path: path}
}

// Path returns the path to the GITHUB_PATH file.
func (f *GitHubPathFile) Path() string {
	return f.path
}

// EnsureDir creates the directory for the file.
func (f *GitHubPathFile) EnsureDir() error {
	return os.MkdirAll(filepath.Dir(f.path), 0755)
}

// Write writes a path to the GITHUB_PATH file.
func (f *GitHubPathFile) Write(path string) error {
	if err := f.EnsureDir(); err != nil {
		return err
	}
	file, err := os.OpenFile(f.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = fmt.Fprintf(file, "%s\n", path)
	return err
}

// ReadAndParse reads the GITHUB_PATH file, parses it, and returns the paths.
// It also truncates the file to avoid reapplying values.
func (f *GitHubPathFile) ReadAndParse() ([]string, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var paths []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			paths = append(paths, line)
		}
	}

	// Truncate the file
	if err := os.WriteFile(f.path, []byte(""), 0644); err != nil {
		return nil, err
	}

	return paths, scanner.Err()
}

// GitHubOutputFile handles the GITHUB_OUTPUT file for a step.
type GitHubOutputFile struct {
	path string
}

// NewGitHubOutputFile creates a new GitHubOutputFile for a step.
func NewGitHubOutputFile(jobInstanceID, stepID string) *GitHubOutputFile {
	path := filepath.Join("/tmp/gacils/steps", stepID, "github_output")
	return &GitHubOutputFile{path: path}
}

// Path returns the path to the GITHUB_OUTPUT file.
func (f *GitHubOutputFile) Path() string {
	return f.path
}

// EnsureDir creates the directory for the file.
func (f *GitHubOutputFile) EnsureDir() error {
	return os.MkdirAll(filepath.Dir(f.path), 0755)
}

// Write writes a key-value pair to the GITHUB_OUTPUT file.
// Supports both single-line and multiline formats.
func (f *GitHubOutputFile) Write(key, value string) error {
	if err := f.EnsureDir(); err != nil {
		return err
	}
	file, err := os.OpenFile(f.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Check if value contains newlines (multiline)
	if strings.Contains(value, "\n") {
		// Use heredoc format with a delimiter
		delimiter := "EOF_" + key
		_, err = fmt.Fprintf(file, "%s<<%s\n%s\n%s\n", key, delimiter, value, delimiter)
	} else {
		_, err = fmt.Fprintf(file, "%s=%s\n", key, value)
	}
	return err
}

// ReadAndParse reads the GITHUB_OUTPUT file, parses it, and returns the outputs.
func (f *GitHubOutputFile) ReadAndParse() (map[string]string, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	outputs := make(map[string]string)
	content := string(data)

	// First, handle multiline format (KEY<<DELIMITER ... DELIMITER)
	// We'll use a simple state machine
	lines := strings.Split(content, "\n")
	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			i++
			continue
		}

		// Check for multiline format
		if strings.Contains(line, "<<") && !strings.Contains(line, "=") {
			// This is a heredoc start: KEY<<DELIMITER
			parts := strings.SplitN(line, "<<", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				delimiter := strings.TrimSpace(parts[1])
				i++
				var valueLines []string
				for i < len(lines) {
					if strings.TrimSpace(lines[i]) == delimiter {
						break
					}
					valueLines = append(valueLines, lines[i])
					i++
				}
				outputs[key] = strings.Join(valueLines, "\n")
			}
		} else if strings.Contains(line, "=") {
			// Single-line format: KEY=value
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				outputs[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
		i++
	}

	return outputs, nil
}
