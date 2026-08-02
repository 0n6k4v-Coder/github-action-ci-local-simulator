// Package fsx provides filesystem utilities for the workflow runner.
// It currently contains placeholder implementations for CRLF conversion
// and git ls-files workspace listing which will be implemented in future phases.
//
// TODO(phase-5): Implement workspace file listing using git ls-files.
// Fallback to filtered filesystem walk when not inside a Git repository.
package fsx