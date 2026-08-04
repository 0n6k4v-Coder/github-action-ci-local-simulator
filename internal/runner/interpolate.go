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
	matrix        map[string]any
	stepResults   []*StepResult
	jobStatus     JobStatus
	cancelled     bool
	needs         map[string]JobNeedsData
}

// JobStatus represents the current job execution status.
type JobStatus struct {
	hasFailure   bool
	hasCancelled bool
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
		jobStatus:   JobStatus{},
	}
}

// SetNeedsContext sets the needs context for expression evaluation.
func (ec *ExpressionContext) SetNeedsContext(needs map[string]JobNeedsData) {
	ec.needs = needs
}

// SetMatrix sets the matrix context for expression evaluation.
func (ec *ExpressionContext) SetMatrix(matrix map[string]any) {
	ec.matrix = matrix
}

// SetStepResults sets the step results for status function evaluation.
func (ec *ExpressionContext) SetStepResults(results []*StepResult) {
	ec.stepResults = results
	ec.updateJobStatus()
}

// SetCancelled sets the cancelled state.
func (ec *ExpressionContext) SetCancelled(cancelled bool) {
	ec.cancelled = cancelled
}

func (ec *ExpressionContext) updateJobStatus() {
	ec.jobStatus.hasFailure = false
	ec.jobStatus.hasCancelled = false
	for _, result := range ec.stepResults {
		if result != nil {
			if result.Status == StatusFailure || result.Status == StatusCancelled {
				ec.jobStatus.hasFailure = true
			}
			if result.Status == StatusCancelled {
				ec.jobStatus.hasCancelled = true
			}
		}
	}
}

// SetCurrentStepID sets the current step ID for steps context.
func (ec *ExpressionContext) SetCurrentStepID(stepID string) {
	ec.currentStepID = stepID
}

// Interpolate interpolates expressions in a string.
// Supports: ${{ env.KEY }}, ${{ github.KEY }}, ${{ runner.KEY }}, ${{ steps.STEP_ID.outputs.KEY }}, ${{ matrix.KEY }}, ${{ needs.JOB_ID.result }}, ${{ needs.JOB_ID.outputs.KEY }}
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
	// Check for unsupported expressions first (operators, comparisons)
	if strings.Contains(expr, "&&") || strings.Contains(expr, "||") ||
		strings.Contains(expr, "==") || strings.Contains(expr, "!=") ||
		strings.Contains(expr, "<") || strings.Contains(expr, ">") {
		return "", fmt.Errorf("unsupported expression: %s (functions, operators, and comparisons not yet supported)", expr)
	}

	// Handle status functions
	if strings.Contains(expr, "(") || strings.Contains(expr, ")") {
		return ec.evaluateFunction(expr)
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

	case "matrix":
		if len(parts) != 2 {
			return "", fmt.Errorf("matrix context requires exactly one key: %s", expr)
		}
		key := parts[1]
		if ec.matrix != nil {
			if value, ok := ec.matrix[key]; ok {
				return fmt.Sprintf("%v", value), nil
			}
		}
		return "", nil

	case "needs":
		if len(parts) < 3 {
			return "", fmt.Errorf("invalid needs expression: %s", expr)
		}
		jobID := parts[1]
		data, ok := ec.needs[jobID]

		if parts[2] == "result" {
			if len(parts) != 3 {
				return "", fmt.Errorf("invalid needs.result expression: %s", expr)
			}
			if ok {
				return data.Result, nil
			}
			return "skipped", nil
		}

		if parts[2] == "outputs" {
			if len(parts) != 4 {
				return "", fmt.Errorf("invalid needs.outputs expression: %s", expr)
			}
			key := parts[3]
			if ok && data.IsMatrix {
				return "", NewUnsupportedError(fmt.Sprintf("needs.outputs from matrix job '%s' is not supported yet", jobID))
			}
			if ok && data.Outputs != nil {
				if val, exists := data.Outputs[key]; exists {
					return val, nil
				}
			}
			return "", nil
		}

		return "", fmt.Errorf("unsupported needs field: %s", parts[2])

	default:
		return "", fmt.Errorf("unsupported context: %s", context)
	}
}

// evaluateFunction evaluates status functions like success(), failure(), always(), cancelled().
func (ec *ExpressionContext) evaluateFunction(expr string) (string, error) {
	trimmed := strings.TrimSpace(expr)

	// Check for supported functions
	switch {
	case strings.HasPrefix(trimmed, "success()"):
		return fmt.Sprintf("%v", ec.evalSuccess()), nil
	case strings.HasPrefix(trimmed, "failure()"):
		return fmt.Sprintf("%v", ec.evalFailure()), nil
	case strings.HasPrefix(trimmed, "always()"):
		return "true", nil
	case strings.HasPrefix(trimmed, "cancelled()"):
		return fmt.Sprintf("%v", ec.evalCancelled()), nil
	default:
		return "", fmt.Errorf("unsupported expression: %s (functions, operators, and comparisons not yet supported)", expr)
	}
}

func (ec *ExpressionContext) evalSuccess() bool {
	if len(ec.needs) > 0 {
		for _, data := range ec.needs {
			if data.Result != "success" {
				return false
			}
		}
		return true
	}
	return !ec.jobStatus.hasFailure && !ec.jobStatus.hasCancelled
}

func (ec *ExpressionContext) evalFailure() bool {
	if len(ec.needs) > 0 {
		for _, data := range ec.needs {
			if data.Result == "failure" {
				return true
			}
		}
		return false
	}
	return ec.jobStatus.hasFailure
}

func (ec *ExpressionContext) evalCancelled() bool {
	return ec.cancelled
}