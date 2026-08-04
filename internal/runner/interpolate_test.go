package runner

import (
	"errors"
	"strings"
	"testing"
)

func TestInterpolateExpressionFunctions(t *testing.T) {
	githubCtx := map[string]string{
		"ref":   "refs/heads/main",
		"actor": "octocat",
	}
	matrixCtx := map[string]any{
		"version": "1.20",
	}
	stepOutputs := NewStepOutputs()
	stepOutputs.SetOutputs("sample", map[string]string{
		"ref":  "refs/heads/main",
		"name": "world",
	})

	ec := NewExpressionContext(nil, githubCtx, nil, stepOutputs)
	ec.SetMatrix(matrixCtx)

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "contains matching substring",
			input:    "${{ contains('Hello World', 'world') }}",
			expected: "true",
		},
		{
			name:     "contains non-matching substring",
			input:    "${{ contains('Hello World', 'xyz') }}",
			expected: "false",
		},
		{
			name:     "contains case insensitivity",
			input:    "${{ contains('HELLO WORLD', 'hello') }}",
			expected: "true",
		},
		{
			name:     "contains with context ref",
			input:    "${{ contains(github.ref, 'refs/heads/') }}",
			expected: "true",
		},
		{
			name:     "startsWith true case",
			input:    "${{ startsWith('refs/heads/main', 'refs/heads/') }}",
			expected: "true",
		},
		{
			name:     "startsWith false case",
			input:    "${{ startsWith('refs/pull/1', 'refs/heads/') }}",
			expected: "false",
		},
		{
			name:     "startsWith case insensitivity",
			input:    "${{ startsWith('REFS/HEADS/MAIN', 'refs/heads/') }}",
			expected: "true",
		},
		{
			name:     "endsWith true case",
			input:    "${{ endsWith('refs/heads/main', 'main') }}",
			expected: "true",
		},
		{
			name:     "endsWith false case",
			input:    "${{ endsWith('refs/heads/develop', 'main') }}",
			expected: "false",
		},
		{
			name:     "endsWith case insensitivity",
			input:    "${{ endsWith('refs/heads/MAIN', 'main') }}",
			expected: "true",
		},
		{
			name:     "format with one argument",
			input:    "${{ format('Hello {0}', steps.sample.outputs.name) }}",
			expected: "Hello world",
		},
		{
			name:     "format with two arguments",
			input:    "${{ format('{0}/{1}', github.actor, matrix.version) }}",
			expected: "octocat/1.20",
		},
		{
			name:     "format with missing argument",
			input:    "${{ format('Hello {0} {1}', github.actor) }}",
			expected: "Hello octocat ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ec.Interpolate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Interpolate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.expected {
				t.Errorf("Interpolate() got = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestInterpolateUnsupportedFunctions(t *testing.T) {
	ec := NewExpressionContext(nil, nil, nil, NewStepOutputs())

	unsupportedInputs := []string{
		"${{ hashFiles('**/go.mod') }}",
		"${{ fromJson('{}') }}",
		"${{ toJson(github) }}",
		"${{ join(github, ',') }}",
	}

	for _, input := range unsupportedInputs {
		t.Run(input, func(t *testing.T) {
			_, err := ec.Interpolate(input)
			if err == nil {
				t.Fatalf("expected error for unsupported input %s, got nil", input)
			}
			var uErr *UnsupportedError
			if !errors.As(err, &uErr) {
				t.Fatalf("expected UnsupportedError type, got %T: %v", err, err)
			}
			if uErr.Code() != 3 {
				t.Errorf("expected exit code 3, got %d", uErr.Code())
			}
		})
	}
}

func TestInterpolateExpressionComparisons(t *testing.T) {
	githubCtx := map[string]string{
		"ref": "refs/heads/main",
	}
	stepOutputs := NewStepOutputs()
	stepOutputs.SetOutputs("sample", map[string]string{
		"ref":  "refs/heads/main",
		"name": "world",
	})

	ec := NewExpressionContext(nil, githubCtx, nil, stepOutputs)

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "equality with matching string literal",
			input:    "${{ 'refs/heads/main' == 'refs/heads/main' }}",
			expected: "true",
		},
		{
			name:     "equality with non-matching string literal",
			input:    "${{ 'refs/heads/main' == 'refs/heads/develop' }}",
			expected: "false",
		},
		{
			name:     "inequality with matching string literal",
			input:    "${{ 'refs/heads/main' != 'refs/heads/main' }}",
			expected: "false",
		},
		{
			name:     "inequality with non-matching string literal",
			input:    "${{ 'refs/heads/main' != 'refs/heads/develop' }}",
			expected: "true",
		},
		{
			name:     "equality between context path and literal",
			input:    "${{ github.ref == 'refs/heads/main' }}",
			expected: "true",
		},
		{
			name:     "inequality between context path and literal",
			input:    "${{ github.ref != 'refs/heads/develop' }}",
			expected: "true",
		},
		{
			name:     "equality with boolean literal",
			input:    "${{ true == true }}",
			expected: "true",
		},
		{
			name:     "equality with number literal and string literal",
			input:    "${{ '1' == 1 }}",
			expected: "true",
		},
		{
			name:     "function result compared to boolean literal",
			input:    "${{ contains(github.ref, 'refs/') == true }}",
			expected: "true",
		},
		{
			name:     "status function result compared to boolean literal",
			input:    "${{ success() == true }}",
			expected: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ec.Interpolate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Interpolate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.expected {
				t.Errorf("Interpolate() got = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestInterpolateLogicalOperators(t *testing.T) {
	ec := NewExpressionContext(nil, nil, nil, NewStepOutputs())

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "&& with both true",
			input:    "${{ 'a' == 'a' && 'b' == 'b' }}",
			expected: "true",
		},
		{
			name:     "&& with left false (short-circuit)",
			input:    "${{ false && hashFiles('**/go.mod') }}",
			expected: "false",
		},
		{
			name:     "&& with right false",
			input:    "${{ true && false }}",
			expected: "false",
		},
		{
			name:     "|| with both false",
			input:    "${{ 'a' == 'b' || 'c' == 'd' }}",
			expected: "false",
		},
		{
			name:     "|| with left true (short-circuit)",
			input:    "${{ true || hashFiles('**/go.mod') }}",
			expected: "true",
		},
		{
			name:     "|| with right true",
			input:    "${{ false || true }}",
			expected: "true",
		},
		{
			name:     "! with true",
			input:    "${{ !success() }}",
			expected: "false",
		},
		{
			name:     "! with false",
			input:    "${{ !failure() }}",
			expected: "true",
		},
		{
			name:     "Precedence A || B && C",
			input:    "${{ false || true && false }}",
			expected: "false",
		},
		{
			name:     "Precedence !A && B",
			input:    "${{ !false && true }}",
			expected: "true",
		},
		{
			name:     "Combined with context and functions",
			input:    "${{ github.ref == 'refs/heads/main' && success() }}",
			expected: "true",
		},
	}

	githubCtx := map[string]string{
		"ref": "refs/heads/main",
	}
	ecWithGithub := NewExpressionContext(nil, githubCtx, nil, NewStepOutputs())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctxToUse := ec
			if strings.Contains(tt.input, "github.ref") {
				ctxToUse = ecWithGithub
			}
			got, err := ctxToUse.Interpolate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Interpolate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.expected {
				t.Errorf("Interpolate() got = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestInterpolateUnsupportedOperators(t *testing.T) {
	ec := NewExpressionContext(nil, nil, nil, NewStepOutputs())

	unsupportedInputs := []string{
		"${{ github.ref < 'refs/heads/main' }}",
		"${{ 1 <= 2 }}",
		"${{ 1 >= 2 }}",
		"${{ 1 > 2 }}",
		"${{ github.ref < 'refs/heads/main' && success() }}",
	}

	for _, input := range unsupportedInputs {
		t.Run(input, func(t *testing.T) {
			_, err := ec.Interpolate(input)
			if err == nil {
				t.Fatalf("expected error for unsupported operator input %s, got nil", input)
			}
			var uErr *UnsupportedError
			if !errors.As(err, &uErr) {
				t.Fatalf("expected UnsupportedError type, got %T: %v", err, err)
			}
			if uErr.Code() != 3 {
				t.Errorf("expected exit code 3, got %d", uErr.Code())
			}
		})
	}
}


