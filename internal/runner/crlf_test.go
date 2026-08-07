package runner

import (
	"strings"
	"testing"
)

func TestNormalizeLineEndings_CRLFtoLF(t *testing.T) {
	input := "line1\r\nline2\r\nline3\r\n"
	expected := "line1\nline2\nline3\n"

	result, err := NormalizeLineEndings(input, "convert")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected LF line endings, got %q", result)
	}
}

func TestNormalizeLineEndings_Preserve(t *testing.T) {
	input := "line1\r\nline2\r\nline3\r\n"

	result, err := NormalizeLineEndings(input, "preserve")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != input {
		t.Errorf("expected preserved line endings, got %q", result)
	}
}

func TestNormalizeLineEndings_Error(t *testing.T) {
	input := "line1\r\nline2\r\nline3\r\n"

	_, err := NormalizeLineEndings(input, "error")
	if err == nil {
		t.Error("expected error for CRLF in error mode")
	}

	if !strings.Contains(err.Error(), "CRLF") {
		t.Errorf("error should mention CRLF, got: %v", err)
	}
}

func TestNormalizeLineEndings_NoCRLF(t *testing.T) {
	input := "line1\nline2\nline3\n"

	// All modes should work with LF-only input
	for _, mode := range []string{"convert", "preserve", "error"} {
		result, err := NormalizeLineEndings(input, mode)
		if err != nil {
			t.Errorf("mode %s should not error on LF input: %v", mode, err)
		}
		if result != input {
			t.Errorf("mode %s should preserve LF input", mode)
		}
	}
}

func TestNormalizeLineEndings_InvalidMode(t *testing.T) {
	input := "line1\nline2\n"

	_, err := NormalizeLineEndings(input, "invalid")
	if err == nil {
		t.Error("expected error for invalid mode")
	}
}
