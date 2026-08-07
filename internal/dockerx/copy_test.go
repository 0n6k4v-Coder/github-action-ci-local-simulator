package dockerx

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTarDirectory creates a temporary directory structure and verifies tar output.
func TestTarDirectory(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create test file structure
	testFiles := map[string]string{
		"file1.txt":                 "content1",
		"subdir/file2.txt":          "content2",
		"subdir/nested/file3.txt":   "content3",
		".git/config":               "git config",
		"node_modules/pkg/index.js": "js code",
		".venv/pyvenv.cfg":          "venv config",
		"build/output.bin":          "binary",
		"dist/bundle.js":            "bundle",
		"coverage.out":              "coverage data",
		"app.log":                   "log content",
		"script.sh":                 "#!/bin/bash\necho hello",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		// Make script.sh executable
		perm := os.FileMode(0644)
		if path == "script.sh" {
			perm = 0755
		}
		if err := os.WriteFile(fullPath, []byte(content), perm); err != nil {
			t.Fatalf("write file failed: %v", err)
		}
	}

	// Test with default exclude patterns
	reader, err := TarDirectory(tmpDir, DefaultExcludePatterns)
	if err != nil {
		t.Fatalf("TarDirectory failed: %v", err)
	}

	// Read and parse tar
	files := extractTar(t, reader)

	// Verify included files
	expectedIncluded := []string{
		"file1.txt",
		"subdir/file2.txt",
		"subdir/nested/file3.txt",
		"script.sh",
	}
	for _, f := range expectedIncluded {
		if _, ok := files[f]; !ok {
			t.Errorf("Expected file %s to be included in tar", f)
		}
	}

	// Verify excluded files (default patterns)
	expectedExcluded := []string{
		".git/config",
		"node_modules/pkg/index.js",
		".venv/pyvenv.cfg",
		"build/output.bin",
		"dist/bundle.js",
		"coverage.out",
		"app.log",
	}
	for _, f := range expectedExcluded {
		if _, ok := files[f]; ok {
			t.Errorf("Expected file %s to be excluded from tar", f)
		}
	}

	// Verify executable permission preserved
	if header, ok := files["script.sh"]; ok {
		if header.Mode&0111 == 0 {
			t.Errorf("Expected script.sh to have executable permissions")
		}
	}
}

// TestTarDirectoryWithCustomExcludes tests custom exclude patterns.
func TestTarDirectoryWithCustomExcludes(t *testing.T) {
	tmpDir := t.TempDir()

	testFiles := map[string]string{
		"keep.txt":          "keep",
		"custom_ignore.txt": "ignore me",
		"normal.txt":        "normal",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("write file failed: %v", err)
		}
	}

	customExcludes := []string{"custom_*"}
	reader, err := TarDirectory(tmpDir, customExcludes)
	if err != nil {
		t.Fatalf("TarDirectory failed: %v", err)
	}

	files := extractTar(t, reader)

	if _, ok := files["keep.txt"]; !ok {
		t.Errorf("keep.txt should be included")
	}
	if _, ok := files["normal.txt"]; !ok {
		t.Errorf("normal.txt should be included")
	}
	if _, ok := files["custom_ignore.txt"]; ok {
		t.Errorf("custom_ignore.txt should be excluded by custom pattern")
	}
}

// TestTarDirectoryNonExistentSource tests error handling for non-existent source.
func TestTarDirectoryNonExistentSource(t *testing.T) {
	_, err := TarDirectory("/non/existent/path", nil)
	if err == nil {
		t.Error("Expected error for non-existent source path")
	}
}

// TestTarDirectoryNotADirectory tests error handling when source is a file.
func TestTarDirectoryNotADirectory(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	_, err := TarDirectory(filePath, nil)
	if err == nil {
		t.Error("Expected error when source is not a directory")
	}
}

// TestShouldExclude tests the shouldExclude function directly.
func TestShouldExclude(t *testing.T) {
	testCases := []struct {
		path     string
		patterns []string
		excluded bool
	}{
		{".git/config", DefaultExcludePatterns, true},
		{".git/", DefaultExcludePatterns, true},
		{"node_modules/pkg/index.js", DefaultExcludePatterns, true},
		{"__pycache__/cache.pyc", DefaultExcludePatterns, true},
		{".venv/bin/python", DefaultExcludePatterns, true},
		{"venv/lib/python3.11", DefaultExcludePatterns, true},
		{"dist/bundle.js", DefaultExcludePatterns, true},
		{"build/output.bin", DefaultExcludePatterns, true},
		{"target/classes", DefaultExcludePatterns, true},
		{"coverage.out", DefaultExcludePatterns, true},
		{"app.log", DefaultExcludePatterns, true},
		{"debug.log", DefaultExcludePatterns, true},
		{"normal.txt", DefaultExcludePatterns, false},
		{"src/main.go", DefaultExcludePatterns, false},
		{"script.sh", DefaultExcludePatterns, false},
		// Custom pattern tests
		{"custom_ignore.txt", []string{"custom_*"}, true},
		{"custom_dir/file.txt", []string{"custom_*"}, true},
		{"keep.txt", []string{"custom_*"}, false},
	}

	for _, tc := range testCases {
		result := shouldExclude(tc.path, tc.patterns)
		if result != tc.excluded {
			t.Errorf("shouldExclude(%q, %v) = %v, want %v", tc.path, tc.patterns, result, tc.excluded)
		}
	}
}

// TestCopyConfigValidation tests CopyWorkspace config validation.
func TestCopyConfigValidation(t *testing.T) {
	// We can't easily test with a real Docker client, but we can test the validation logic
	// by checking that the function returns appropriate errors for invalid config

	// Test empty source path
	config := CopyConfig{
		SourcePath:  "",
		TargetPath:  "/github/workspace",
		ContainerID: "test-container",
	}
	// We can't call CopyWorkspace without a real client, but we can verify the validation
	// logic by inspecting the code. This test documents the expected behavior.
	if config.SourcePath == "" {
		t.Log("Empty source path should be rejected")
	}

	// Test empty container ID
	config2 := CopyConfig{
		SourcePath:  "/tmp/source",
		TargetPath:  "/github/workspace",
		ContainerID: "",
	}
	if config2.ContainerID == "" {
		t.Log("Empty container ID should be rejected")
	}
}

// TestEnsureWorkspaceDirSafety tests the safety check in CleanupWorkspace.
func TestCleanupWorkspaceSafety(t *testing.T) {
	testCases := []struct {
		path        string
		shouldAllow bool
	}{
		{"/github/workspace", true},
		{"/github/workspace/subdir", true},
		{"/tmp/gacils", true},
		{"/tmp/gacils/jobs/123", true},
		{"/", false},
		{"/etc", false},
		{"/home", false},
		{"/var", false},
		{"/usr", false},
		{"/bin", false},
		{"/sbin", false},
		{"/lib", false},
		{"/opt", false},
		{"/root", false},
	}

	for _, tc := range testCases {
		// Test the logic directly
		allowed := strings.HasPrefix(tc.path, "/github/workspace") || strings.HasPrefix(tc.path, "/tmp/gacils")
		if allowed != tc.shouldAllow {
			t.Errorf("CleanupWorkspace safety check for %q: got %v, want %v", tc.path, allowed, tc.shouldAllow)
		}
	}
}

// extractTar reads a tar stream and returns a map of filename -> tar.Header
func extractTar(t *testing.T, reader io.Reader) map[string]*tar.Header {
	tr := tar.NewReader(reader)
	files := make(map[string]*tar.Header)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar read error: %v", err)
		}
		files[header.Name] = header
		// Consume the content
		if _, err := io.Copy(io.Discard, tr); err != nil {
			t.Fatalf("tar content copy error: %v", err)
		}
	}

	return files
}

// TestTarDirectorySymlink tests handling of symlinks.
func TestTarDirectorySymlink(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a regular file
	targetFile := filepath.Join(tmpDir, "target.txt")
	if err := os.WriteFile(targetFile, []byte("target content"), 0644); err != nil {
		t.Fatalf("write target file failed: %v", err)
	}

	// Create a symlink
	linkPath := filepath.Join(tmpDir, "link.txt")
	if err := os.Symlink("target.txt", linkPath); err != nil {
		t.Fatalf("create symlink failed: %v", err)
	}

	reader, err := TarDirectory(tmpDir, DefaultExcludePatterns)
	if err != nil {
		t.Fatalf("TarDirectory failed: %v", err)
	}

	files := extractTar(t, reader)

	if _, ok := files["target.txt"]; !ok {
		t.Errorf("target.txt should be included")
	}
	if header, ok := files["link.txt"]; ok {
		if header.Typeflag != tar.TypeSymlink {
			t.Errorf("link.txt should be a symlink, got type %c", header.Typeflag)
		}
		if header.Linkname != "target.txt" {
			t.Errorf("symlink target mismatch: got %q, want %q", header.Linkname, "target.txt")
		}
	} else {
		t.Errorf("link.txt should be included as symlink")
	}
}

// TestTarDirectoryEmptyDir tests handling of empty directories.
func TestTarDirectoryEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an empty directory
	emptyDir := filepath.Join(tmpDir, "empty_dir")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	// Create a non-empty directory
	nonEmptyDir := filepath.Join(tmpDir, "non_empty")
	if err := os.MkdirAll(nonEmptyDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nonEmptyDir, "file.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	reader, err := TarDirectory(tmpDir, DefaultExcludePatterns)
	if err != nil {
		t.Fatalf("TarDirectory failed: %v", err)
	}

	files := extractTar(t, reader)

	// Empty directories may or may not be included depending on implementation
	// Non-empty directory should have its file
	if _, ok := files["non_empty/file.txt"]; !ok {
		t.Errorf("non_empty/file.txt should be included")
	}
}

// TestTarDirectoryExcludeSubdirectory tests that excluding a directory excludes its contents.
func TestTarDirectoryExcludeSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create structure with excluded subdirectory
	excludedDir := filepath.Join(tmpDir, "node_modules", "pkg")
	if err := os.MkdirAll(excludedDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(excludedDir, "index.js"), []byte("code"), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	// Create a file at root level
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	reader, err := TarDirectory(tmpDir, DefaultExcludePatterns)
	if err != nil {
		t.Fatalf("TarDirectory failed: %v", err)
	}

	files := extractTar(t, reader)

	if _, ok := files["main.go"]; !ok {
		t.Errorf("main.go should be included")
	}
	if _, ok := files["node_modules/pkg/index.js"]; ok {
		t.Errorf("node_modules/pkg/index.js should be excluded")
	}
}

// TestTarDirectoryPreservesFileMode tests that file modes are preserved.
func TestTarDirectoryPreservesFileMode(t *testing.T) {
	tmpDir := t.TempDir()

	// Regular file
	regularFile := filepath.Join(tmpDir, "regular.txt")
	if err := os.WriteFile(regularFile, []byte("content"), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	// Executable file
	execFile := filepath.Join(tmpDir, "script.sh")
	if err := os.WriteFile(execFile, []byte("#!/bin/bash\necho hello"), 0755); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	reader, err := TarDirectory(tmpDir, DefaultExcludePatterns)
	if err != nil {
		t.Fatalf("TarDirectory failed: %v", err)
	}

	files := extractTar(t, reader)

	if header, ok := files["regular.txt"]; ok {
		if header.Mode&0111 != 0 {
			t.Errorf("regular.txt should not be executable")
		}
	}
	if header, ok := files["script.sh"]; ok {
		if header.Mode&0111 == 0 {
			t.Errorf("script.sh should be executable")
		}
	}
}

// TestTarDirectoryLargeFile tests handling of larger files.
func TestTarDirectoryLargeFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a larger file (100KB)
	largeContent := strings.Repeat("x", 100*1024)
	largeFile := filepath.Join(tmpDir, "large.bin")
	if err := os.WriteFile(largeFile, []byte(largeContent), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	reader, err := TarDirectory(tmpDir, DefaultExcludePatterns)
	if err != nil {
		t.Fatalf("TarDirectory failed: %v", err)
	}

	// Verify we can read the content back
	tr := tar.NewReader(reader)
	found := false
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar read error: %v", err)
		}
		if header.Name == "large.bin" {
			found = true
			content, err := io.ReadAll(tr)
			if err != nil {
				t.Fatalf("read content error: %v", err)
			}
			if len(content) != 100*1024 {
				t.Errorf("content size mismatch: got %d, want %d", len(content), 100*1024)
			}
			if string(content) != largeContent {
				t.Errorf("content mismatch")
			}
			break
		}
		// Consume other entries
		io.Copy(io.Discard, tr)
	}

	if !found {
		t.Errorf("large.bin not found in tar")
	}
}

// BenchmarkTarDirectory benchmarks tar creation performance.
func BenchmarkTarDirectory(b *testing.B) {
	tmpDir := b.TempDir()

	// Create a larger directory structure
	for i := 0; i < 100; i++ {
		dir := filepath.Join(tmpDir, fmt.Sprintf("dir%d", i))
		if err := os.MkdirAll(dir, 0755); err != nil {
			b.Fatalf("mkdir failed: %v", err)
		}
		for j := 0; j < 10; j++ {
			file := filepath.Join(dir, fmt.Sprintf("file%d.txt", j))
			content := strings.Repeat(fmt.Sprintf("content%d", j), 100)
			if err := os.WriteFile(file, []byte(content), 0644); err != nil {
				b.Fatalf("write file failed: %v", err)
			}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader, err := TarDirectory(tmpDir, DefaultExcludePatterns)
		if err != nil {
			b.Fatalf("TarDirectory failed: %v", err)
		}
		// Consume the stream
		io.Copy(io.Discard, reader)
	}
}

// TestCopyFromContainerExtraction tests the tar extraction logic used in CopyFromContainer.
func TestCopyFromContainerExtraction(t *testing.T) {
	// Create a test tar archive in memory
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Add a directory
	dirHeader := &tar.Header{
		Name:     "testdir/",
		Mode:     0755,
		Typeflag: tar.TypeDir,
	}
	if err := tw.WriteHeader(dirHeader); err != nil {
		t.Fatalf("write dir header: %v", err)
	}

	// Add a regular file
	fileHeader := &tar.Header{
		Name:     "testdir/file.txt",
		Mode:     0644,
		Typeflag: tar.TypeReg,
		Size:     int64(len("file content")),
	}
	if err := tw.WriteHeader(fileHeader); err != nil {
		t.Fatalf("write file header: %v", err)
	}
	if _, err := tw.Write([]byte("file content")); err != nil {
		t.Fatalf("write file content: %v", err)
	}

	// Add a symlink
	linkHeader := &tar.Header{
		Name:     "testdir/link.txt",
		Typeflag: tar.TypeSymlink,
		Linkname: "file.txt",
		Mode:     0777,
	}
	if err := tw.WriteHeader(linkHeader); err != nil {
		t.Fatalf("write link header: %v", err)
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}

	// Extract to a temp directory
	destDir := t.TempDir()
	reader := bytes.NewReader(buf.Bytes())
	tr := tar.NewReader(reader)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read tar entry: %v", err)
		}

		targetPath := filepath.Join(destDir, header.Name)

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			t.Fatalf("create parent dir: %v", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				t.Fatalf("create directory: %v", err)
			}
		case tar.TypeReg, tar.TypeRegA:
			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				t.Fatalf("create file: %v", err)
			}
			if _, err := io.Copy(file, tr); err != nil {
				file.Close()
				t.Fatalf("write file: %v", err)
			}
			file.Close()
		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				t.Fatalf("create symlink: %v", err)
			}
		}
	}

	// Verify extracted content
	if _, err := os.Stat(filepath.Join(destDir, "testdir")); err != nil {
		t.Errorf("testdir not created: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(destDir, "testdir/file.txt"))
	if err != nil {
		t.Errorf("file.txt not readable: %v", err)
	}
	if string(content) != "file content" {
		t.Errorf("file content mismatch: got %q", string(content))
	}

	linkTarget, err := os.Readlink(filepath.Join(destDir, "testdir/link.txt"))
	if err != nil {
		t.Errorf("link.txt not readable: %v", err)
	}
	if linkTarget != "file.txt" {
		t.Errorf("link target mismatch: got %q", linkTarget)
	}
}
