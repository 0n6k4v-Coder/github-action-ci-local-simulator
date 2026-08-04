package actions

import (
	"context"
	"testing"
)

func TestParseActionRef(t *testing.T) {
	tests := []struct {
		input     string
		wantName  string
		wantOwner string
		wantRepo  string
		wantRef   string
		wantErr   bool
	}{
		{
			input:     "actions/checkout@v4",
			wantName:  "actions/checkout",
			wantOwner: "actions",
			wantRepo:  "checkout",
			wantRef:   "v4",
		},
		{
			input:     "actions/setup-python@v5",
			wantName:  "actions/setup-python",
			wantOwner: "actions",
			wantRepo:  "setup-python",
			wantRef:   "v5",
		},
		{
			input:     "actions/cache@v4",
			wantName:  "actions/cache",
			wantOwner: "actions",
			wantRepo:  "cache",
			wantRef:   "v4",
		},
		{
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ref, err := ParseActionRef(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseActionRef(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseActionRef(%q) unexpected error: %v", tt.input, err)
			}
			if ref.ActionName() != tt.wantName {
				t.Errorf("ActionName() = %q, want %q", ref.ActionName(), tt.wantName)
			}
			if ref.Owner != tt.wantOwner {
				t.Errorf("Owner = %q, want %q", ref.Owner, tt.wantOwner)
			}
			if ref.Repo != tt.wantRepo {
				t.Errorf("Repo = %q, want %q", ref.Repo, tt.wantRepo)
			}
			if ref.Ref != tt.wantRef {
				t.Errorf("Ref = %q, want %q", ref.Ref, tt.wantRef)
			}
		})
	}
}

func TestRegistrySupported(t *testing.T) {
	r := NewRegistry()

	checkoutRef, _ := ParseActionRef("actions/checkout@v4")
	if !r.IsSupported(checkoutRef) {
		t.Errorf("expected actions/checkout@v4 to be supported")
	}

	setupPythonRef, _ := ParseActionRef("actions/setup-python@v5")
	if !r.IsSupported(setupPythonRef) {
		t.Errorf("expected actions/setup-python@v5 to be supported")
	}

	cacheRef, _ := ParseActionRef("actions/cache@v4")
	if !r.IsSupported(cacheRef) {
		t.Errorf("expected actions/cache@v4 to be supported")
	}

	uploadRef, _ := ParseActionRef("actions/upload-artifact@v4")
	if !r.IsSupported(uploadRef) {
		t.Errorf("expected actions/upload-artifact@v4 to be supported")
	}

	downloadRef, _ := ParseActionRef("actions/download-artifact@v4")
	if !r.IsSupported(downloadRef) {
		t.Errorf("expected actions/download-artifact@v4 to be supported")
	}

	unknownRef, _ := ParseActionRef("actions/unknown@v1")
	if r.IsSupported(unknownRef) {
		t.Errorf("expected actions/unknown@v1 to be unsupported")
	}
}

func TestExecuteUnsupportedAction(t *testing.T) {
	r := NewRegistry()
	unknownRef, _ := ParseActionRef("actions/unknown@v1")
	ctx := context.Background()

	_, err := r.Execute(ctx, nil, "", "", unknownRef, nil)
	if err == nil {
		t.Fatalf("expected error for unsupported action, got nil")
	}

	uerr, ok := err.(*UnsupportedActionError)
	if !ok {
		t.Fatalf("expected *UnsupportedActionError, got %T: %v", err, err)
	}

	if uerr.Code() != 3 {
		t.Errorf("exit code = %d, want 3", uerr.Code())
	}

	if uerr.Error() != "unsupported action: actions/unknown" {
		t.Errorf("Error() = %q, want %q", uerr.Error(), "unsupported action: actions/unknown")
	}
}

func TestExecuteCheckoutSimulation(t *testing.T) {
	ctx := context.Background()
	res, err := ExecuteCheckout(ctx, nil, "", "/github/workspace", nil)
	if err != nil {
		t.Fatalf("ExecuteCheckout failed: %v", err)
	}
	if res == nil || res.Stdout == "" {
		t.Errorf("expected non-empty stdout result")
	}
}

func TestExecuteSetupPythonSimulation(t *testing.T) {
	ctx := context.Background()

	// Missing python-version
	_, err := ExecuteSetupPython(ctx, nil, "", "", nil)
	if err == nil {
		t.Fatalf("expected error for missing python-version, got nil")
	}
	if valErr, ok := err.(*ActionValidationError); !ok || valErr.Code() != 2 {
		t.Errorf("expected ActionValidationError with code 2, got: %v", err)
	}

	// With python-version
	with := map[string]any{"python-version": "3.12"}
	res, err := ExecuteSetupPython(ctx, nil, "", "", with)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Env["python-version"] != "3.12" {
		t.Errorf("env[python-version] = %q, want %q", res.Env["python-version"], "3.12")
	}
	if res.Env["python-location"] != "/usr/bin" {
		t.Errorf("env[python-location] = %q, want %q", res.Env["python-location"], "/usr/bin")
	}
}

func TestExecuteCache(t *testing.T) {
	ctx := context.Background()

	// Test missing key
	_, err := ExecuteCache(ctx, nil, "", "", map[string]any{"path": "/tmp/cache"})
	if err == nil {
		t.Fatalf("expected validation error for missing key")
	}

	// Test missing path
	_, err = ExecuteCache(ctx, nil, "", "", map[string]any{"key": "my-key"})
	if err == nil {
		t.Fatalf("expected validation error for missing path")
	}

	// Test cache execution (first run: cache miss)
	tempCacheDir := t.TempDir()
	ctx = WithHostDirs(ctx, tempCacheDir, t.TempDir())
	with := map[string]any{
		"key":  "my-key-1",
		"path": "/tmp/cache",
	}
	res, err := ExecuteCache(ctx, nil, "", "", with)
	if err != nil {
		t.Fatalf("ExecuteCache failed: %v", err)
	}
	if res.Env["cache-hit"] != "false" {
		t.Errorf("expected cache-hit=false on first run, got %s", res.Env["cache-hit"])
	}
}

func TestExecuteArtifacts(t *testing.T) {
	ctx := context.Background()

	// Upload validation
	_, err := ExecuteUploadArtifact(ctx, nil, "", "", map[string]any{"name": "my-art"})
	if err == nil {
		t.Fatalf("expected validation error for missing path in upload")
	}

	tempArtDir := t.TempDir()
	ctx = WithHostDirs(ctx, t.TempDir(), tempArtDir)
	ctx = WithJobID(ctx, "job1")

	// Upload artifact
	upRes, err := ExecuteUploadArtifact(ctx, nil, "", "/tmp", map[string]any{
		"name": "my-build",
		"path": "output.txt",
	})
	if err != nil {
		t.Fatalf("ExecuteUploadArtifact failed: %v", err)
	}
	if upRes == nil || upRes.Stdout == "" {
		t.Errorf("expected non-empty stdout")
	}

	// Download artifact
	downRes, err := ExecuteDownloadArtifact(ctx, nil, "", "/tmp", map[string]any{
		"name": "my-build",
	})
	if err != nil {
		t.Fatalf("ExecuteDownloadArtifact failed: %v", err)
	}
	if downRes == nil || downRes.Stdout == "" {
		t.Errorf("expected non-empty stdout")
	}
}

