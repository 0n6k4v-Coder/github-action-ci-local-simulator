package runner

import (
	"fmt"
	"regexp"
	"strings"
)

// ExpressionContext holds the context for expression interpolation.
type ExpressionContext struct {
	env           map[string]string
	github        map[string]string
	runner        map[string]string
	stepOutputs   *StepOutputs
	currentStepID string
}

// NewExpressionContext creates a new ExpressionContext.
func NewExpressionContext(
	env map[string]string,
	github map[string]string,
	runner map[string]string,
	stepOutputs *StepOutputs,
) *ExpressionContext {
	return &ExpressionContext{
		env:         env,
		github:      github,
		runner:      runner,
		stepOutputs: stepOutputs,
	}
}

// SetCurrentStepID sets the current step ID for steps context.
func (ec *ExpressionContext) SetCurrentStepID(stepID string) {
	ec.currentStepID = stepID
}

// Interpolate interpolates expressions in a string.
// Supports: ${{ env.KEY }}, ${{ github.KEY }}, ${{ runner.KEY }}, ${{ steps.STEP_ID.outputs.KEY }}
func (ec *ExpressionContext) Interpolate(input string) (string, error) {
	if input == "" {
		return input, nil
	}

	// Regex to match ${{ ... }}
	re := regexp.MustCompile(`\$\{\{\s*([^}]+)\s*\}\}`)
	result := input

	matches := re.FindAllStringSubmatch(input, -1)
	for _, match := range matches {
		fullMatch := match[0]
		expr := strings.TrimSpace(match[1])

		value, err := ec.evaluateExpression(expr)
		if err != nil {
			return "", fmt.Errorf("interpolate %q: %w", fullMatch, err)
		}

		result = strings.Replace(result, fullMatch, value, 1)
	}

	return result, nil
}

// InterpolateMap interpolates expressions in all values of a map.
func (ec *ExpressionContext) InterpolateMap(input map[string]string) (map[string]string, error) {
	result := make(map[string]string, len(input))
	for k, v := range input {
		interpolated, err := ec.Interpolate(v)
		if err != nil {
			return nil, fmt.Errorf("interpolate %s: %w", k, err)
		}
		result[k] = interpolated
	}
	return result, nil
}

// evaluateExpression evaluates a single expression.
func (ec *ExpressionContext) evaluateExpression(expr string) (string, error) {
	// Check for unsupported expressions first
	if strings.Contains(expr, "(") || strings.Contains(expr, ")") ||
		strings.Contains(expr, "&&") || strings.Contains(expr, "||") ||
		strings.Contains(expr, "==") || strings.Contains(expr, "!=") ||
		strings.Contains(expr, "<") || strings.Contains(expr, ">") {
		return "", fmt.Errorf("unsupported expression: %s (functions, operators, and comparisons not yet supported)", expr)
	}

	// Split by dot notation
	parts := strings.Split(expr, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid expression format: %s", expr)
	}

	context := parts[0]
	switch context {
	case "env":
		if len(parts) != 2 {
			return "", fmt.Errorf("env context requires exactly one key: %s", expr)
		}
		key := parts[1]
		if value, ok := ec.env[key]; ok {
			return value, nil
		}
		return "", nil // Return empty string for missing env vars (GitHub behavior)

	case "github":
		if len(parts) != 2 {
			return "", fmt.Errorf("github context requires exactly one key: %s", expr)
		}
		key := parts[1]
		if value, ok := ec.github[key]; ok {
			return value, nil
		}
		return "", nil

	case "runner":
		if len(parts) != 2 {
			return "", fmt.Errorf("runner context requires exactly one key: %s", expr)
		}
		key := parts[1]
		if value, ok := ec.runner[key]; ok {
			return value, nil
		}
		return "", nil

	case "steps":
		if len(parts) != 4 || parts[2] != "outputs" {
			return "", fmt.Errorf("steps context requires step ID and outputs key: %s", expr)
		}
		stepID := parts[1]
		key := parts[3]
		if value, ok := ec.stepOutputs.GetOutput(stepID, key); ok {
			return value, nil
		}
		return "", nil

	default:
		return "", fmt.Errorf("unsupported context: %s", context)
	}
}