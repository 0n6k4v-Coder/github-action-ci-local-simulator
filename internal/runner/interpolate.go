package runner

import (
	"fmt"
	"regexp"
	"strconv"
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
	re := regexp.MustCompile(`\$\{\{\s*(.*?)\s*\}\}`)
	result := input

	for {
		loc := re.FindStringSubmatchIndex(result)
		if loc == nil {
			break
		}

		fullMatch := result[loc[0]:loc[1]]
		expr := strings.TrimSpace(result[loc[2]:loc[3]])

		value, err := ec.evaluateExpression(expr)
		if err != nil {
			return "", fmt.Errorf("interpolate %q: %w", fullMatch, err)
		}

		result = result[:loc[0]] + value + result[loc[1]:]
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

func isTrue(val string) bool {
	v := strings.ToLower(strings.TrimSpace(val))
	return v != "false" && v != "" && v != "0"
}

func isEnclosedInParens(expr string) bool {
	expr = strings.TrimSpace(expr)
	if len(expr) < 2 || !strings.HasPrefix(expr, "(") || !strings.HasSuffix(expr, ")") {
		return false
	}

	var inSingleQuote, inDoubleQuote, escaped bool
	parenDepth := 0
	n := len(expr)

	for i := 0; i < n; i++ {
		ch := expr[i]

		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' && (inSingleQuote || inDoubleQuote) {
			escaped = true
			continue
		}

		if ch == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			continue
		}

		if ch == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			continue
		}

		if inSingleQuote || inDoubleQuote {
			continue
		}

		if ch == '(' {
			parenDepth++
		} else if ch == ')' {
			parenDepth--
			if parenDepth == 0 && i < n-1 {
				return false
			}
		}
	}

	return parenDepth == 0
}

type opMatch struct {
	op    string
	index int
}

func findTopLevelOps(expr string) ([]opMatch, error) {
	var (
		inSingleQuote bool
		inDoubleQuote bool
		escaped       bool
		parenDepth    int
		matches       []opMatch
	)

	n := len(expr)
	for i := 0; i < n; i++ {
		ch := expr[i]

		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' && (inSingleQuote || inDoubleQuote) {
			escaped = true
			continue
		}

		if ch == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			continue
		}

		if ch == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			continue
		}

		if inSingleQuote || inDoubleQuote {
			continue
		}

		if ch == '(' {
			parenDepth++
			continue
		}

		if ch == ')' {
			if parenDepth > 0 {
				parenDepth--
			}
			continue
		}

		if parenDepth == 0 {
			// Check 2-character operators
			if i+1 < n {
				two := expr[i : i+2]
				switch two {
				case "&&", "||", "==", "!=", "<=", ">=":
					matches = append(matches, opMatch{op: two, index: i})
					i++ // skip next char
					continue
				}
			}

			// Check 1-character operators
			switch ch {
			case '<', '>':
				matches = append(matches, opMatch{op: string(ch), index: i})
			case '!':
				if i+1 >= n || expr[i+1] != '=' {
					matches = append(matches, opMatch{op: "!", index: i})
				}
			}
		}
	}

	if inSingleQuote || inDoubleQuote {
		return nil, fmt.Errorf("unterminated string literal in expression: %s", expr)
	}

	return matches, nil
}

// evaluateExpression evaluates a single expression.
func (ec *ExpressionContext) evaluateExpression(expr string) (string, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", nil
	}

	for isEnclosedInParens(expr) {
		expr = strings.TrimSpace(expr[1 : len(expr)-1])
	}
	if expr == "" {
		return "", nil
	}

	matches, err := findTopLevelOps(expr)
	if err != nil {
		return "", err
	}

	// Level 1: Logical OR (||)
	for _, m := range matches {
		if m.op == "||" {
			left := strings.TrimSpace(expr[:m.index])
			right := strings.TrimSpace(expr[m.index+2:])

			leftVal, err := ec.evaluateExpression(left)
			if err != nil {
				return "", err
			}
			if isTrue(leftVal) {
				return "true", nil
			}

			rightVal, err := ec.evaluateExpression(right)
			if err != nil {
				return "", err
			}
			if isTrue(rightVal) {
				return "true", nil
			}
			return "false", nil
		}
	}

	// Level 2: Logical AND (&&)
	for _, m := range matches {
		if m.op == "&&" {
			left := strings.TrimSpace(expr[:m.index])
			right := strings.TrimSpace(expr[m.index+2:])

			leftVal, err := ec.evaluateExpression(left)
			if err != nil {
				return "", err
			}
			if !isTrue(leftVal) {
				return "false", nil
			}

			rightVal, err := ec.evaluateExpression(right)
			if err != nil {
				return "", err
			}
			if !isTrue(rightVal) {
				return "false", nil
			}
			return "true", nil
		}
	}

	// Level 3: Comparison operators (==, !=, <, >, <=, >=)
	var compMatches []opMatch
	for _, m := range matches {
		switch m.op {
		case "==", "!=", "<", ">", "<=", ">=":
			compMatches = append(compMatches, m)
		}
	}

	if len(compMatches) > 0 {
		if len(compMatches) > 1 {
			return "", NewUnsupportedError("nested boolean logic and multiple operators are not supported")
		}
		m := compMatches[0]
		switch m.op {
		case "<", ">", "<=", ">=":
			return "", NewUnsupportedError(fmt.Sprintf("unsupported expression operator: %s", m.op))
		case "==", "!=":
			left := strings.TrimSpace(expr[:m.index])
			right := strings.TrimSpace(expr[m.index+len(m.op):])

			leftVal, err := ec.evaluateExpression(left)
			if err != nil {
				return "", err
			}
			rightVal, err := ec.evaluateExpression(right)
			if err != nil {
				return "", err
			}

			if m.op == "==" {
				if leftVal == rightVal {
					return "true", nil
				}
				return "false", nil
			} else {
				if leftVal != rightVal {
					return "true", nil
				}
				return "false", nil
			}
		}
	}

	// Level 4: Logical NOT (!)
	for _, m := range matches {
		if m.op == "!" {
			if m.index == 0 {
				target := strings.TrimSpace(expr[1:])
				val, err := ec.evaluateExpression(target)
				if err != nil {
					return "", err
				}
				if isTrue(val) {
					return "false", nil
				}
				return "true", nil
			} else {
				return "", NewUnsupportedError("unsupported expression operator: !")
			}
		}
	}

	// Level 5: Primary expression / operand
	return ec.evaluateOperand(expr)
}

// evaluateOperand evaluates a single operand token (string literal, number, bool, function call, or context path).
func (ec *ExpressionContext) evaluateOperand(arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "", nil
	}

	// Single quoted string literal
	if strings.HasPrefix(arg, "'") && strings.HasSuffix(arg, "'") && len(arg) >= 2 {
		val := arg[1 : len(arg)-1]
		val = strings.ReplaceAll(val, "''", "'")
		return val, nil
	}

	// Double quoted string literal (convenience)
	if strings.HasPrefix(arg, "\"") && strings.HasSuffix(arg, "\"") && len(arg) >= 2 {
		return arg[1 : len(arg)-1], nil
	}

	// Boolean literal
	if strings.EqualFold(arg, "true") || strings.EqualFold(arg, "false") {
		return strings.ToLower(arg), nil
	}

	// Number literal
	if _, err := strconv.ParseFloat(arg, 64); err == nil {
		return arg, nil
	}

	// Function call (e.g. success(), contains(...), etc.)
	if strings.Contains(arg, "(") {
		return ec.evaluateFunction(arg)
	}

	// Otherwise, evaluate as context path
	return ec.evaluateContextPath(arg)
}


func (ec *ExpressionContext) evaluateContextPath(expr string) (string, error) {
	expr = strings.TrimSpace(expr)

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
		// Try case-insensitive key lookup for github context variables like github.ref / GITHUB_REF
		upperKey := strings.ToUpper(key)
		if value, ok := ec.github[upperKey]; ok {
			return value, nil
		}
		if value, ok := ec.github["GITHUB_"+upperKey]; ok {
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
		upperKey := strings.ToUpper(key)
		if value, ok := ec.runner[upperKey]; ok {
			return value, nil
		}
		if value, ok := ec.runner["RUNNER_"+upperKey]; ok {
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

// parseFunctionCall parses a function call like funcName(arg1, arg2, ...) into funcName and args.
func parseFunctionCall(expr string) (string, []string, error) {
	trimmed := strings.TrimSpace(expr)
	idx := strings.Index(trimmed, "(")
	if idx == -1 || !strings.HasSuffix(trimmed, ")") {
		return "", nil, fmt.Errorf("invalid function call syntax: %s", expr)
	}

	funcName := strings.TrimSpace(trimmed[:idx])
	argsStr := trimmed[idx+1 : len(trimmed)-1]

	args, err := parseFunctionArgs(argsStr)
	if err != nil {
		return "", nil, err
	}

	return funcName, args, nil
}

// parseFunctionArgs parses comma-separated function arguments respecting quotes and nested parentheses.
func parseFunctionArgs(argsStr string) ([]string, error) {
	trimmed := strings.TrimSpace(argsStr)
	if trimmed == "" {
		return []string{}, nil
	}

	var args []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false
	parenDepth := 0

	for i := 0; i < len(trimmed); i++ {
		ch := trimmed[i]

		if escaped {
			current.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' && (inSingleQuote || inDoubleQuote) {
			escaped = true
			current.WriteByte(ch)
			continue
		}

		if ch == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			current.WriteByte(ch)
			continue
		}

		if ch == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			current.WriteByte(ch)
			continue
		}

		if !inSingleQuote && !inDoubleQuote {
			if ch == '(' {
				parenDepth++
			} else if ch == ')' {
				parenDepth--
			} else if ch == ',' && parenDepth == 0 {
				args = append(args, strings.TrimSpace(current.String()))
				current.Reset()
				continue
			}
		}

		current.WriteByte(ch)
	}

	if inSingleQuote || inDoubleQuote {
		return nil, fmt.Errorf("unterminated string in function arguments")
	}

	args = append(args, strings.TrimSpace(current.String()))
	return args, nil
}

// evaluateArgument evaluates a single argument token (literal string, number, bool, or context path).
func (ec *ExpressionContext) evaluateArgument(arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "", nil
	}

	// Check if arg is a nested function call
	if strings.Contains(arg, "(") && strings.Contains(arg, ")") {
		return "", NewUnsupportedError("nested expression functions are not supported")
	}

	// Single quoted string literal
	if strings.HasPrefix(arg, "'") && strings.HasSuffix(arg, "'") && len(arg) >= 2 {
		val := arg[1 : len(arg)-1]
		val = strings.ReplaceAll(val, "''", "'")
		return val, nil
	}

	// Double quoted string literal (convenience)
	if strings.HasPrefix(arg, "\"") && strings.HasSuffix(arg, "\"") && len(arg) >= 2 {
		return arg[1 : len(arg)-1], nil
	}

	// Boolean literal
	if strings.EqualFold(arg, "true") || strings.EqualFold(arg, "false") {
		return strings.ToLower(arg), nil
	}

	// Number literal
	if _, err := strconv.ParseFloat(arg, 64); err == nil {
		return arg, nil
	}

	// Otherwise, evaluate as context path
	return ec.evaluateContextPath(arg)
}

// evaluateFunction evaluates expression functions.
func (ec *ExpressionContext) evaluateFunction(expr string) (string, error) {
	trimmed := strings.TrimSpace(expr)

	funcName, rawArgs, err := parseFunctionCall(trimmed)
	if err != nil {
		return "", err
	}

	switch funcName {
	case "success":
		if len(rawArgs) != 0 {
			return "", fmt.Errorf("success() takes no arguments")
		}
		return fmt.Sprintf("%v", ec.evalSuccess()), nil
	case "failure":
		if len(rawArgs) != 0 {
			return "", fmt.Errorf("failure() takes no arguments")
		}
		return fmt.Sprintf("%v", ec.evalFailure()), nil
	case "always":
		if len(rawArgs) != 0 {
			return "", fmt.Errorf("always() takes no arguments")
		}
		return "true", nil
	case "cancelled":
		if len(rawArgs) != 0 {
			return "", fmt.Errorf("cancelled() takes no arguments")
		}
		return fmt.Sprintf("%v", ec.evalCancelled()), nil

	case "contains":
		if len(rawArgs) != 2 {
			return "", fmt.Errorf("contains() requires exactly 2 arguments")
		}
		arg1, err := ec.evaluateArgument(rawArgs[0])
		if err != nil {
			return "", err
		}
		arg2, err := ec.evaluateArgument(rawArgs[1])
		if err != nil {
			return "", err
		}
		if arg1 == "" || arg2 == "" {
			return "false", nil
		}
		res := strings.Contains(strings.ToLower(arg1), strings.ToLower(arg2))
		return fmt.Sprintf("%v", res), nil

	case "startsWith":
		if len(rawArgs) != 2 {
			return "", fmt.Errorf("startsWith() requires exactly 2 arguments")
		}
		arg1, err := ec.evaluateArgument(rawArgs[0])
		if err != nil {
			return "", err
		}
		arg2, err := ec.evaluateArgument(rawArgs[1])
		if err != nil {
			return "", err
		}
		if arg1 == "" || arg2 == "" {
			return "false", nil
		}
		res := strings.HasPrefix(strings.ToLower(arg1), strings.ToLower(arg2))
		return fmt.Sprintf("%v", res), nil

	case "endsWith":
		if len(rawArgs) != 2 {
			return "", fmt.Errorf("endsWith() requires exactly 2 arguments")
		}
		arg1, err := ec.evaluateArgument(rawArgs[0])
		if err != nil {
			return "", err
		}
		arg2, err := ec.evaluateArgument(rawArgs[1])
		if err != nil {
			return "", err
		}
		if arg1 == "" || arg2 == "" {
			return "false", nil
		}
		res := strings.HasSuffix(strings.ToLower(arg1), strings.ToLower(arg2))
		return fmt.Sprintf("%v", res), nil

	case "format":
		if len(rawArgs) < 1 {
			return "", fmt.Errorf("format() requires at least 1 argument")
		}
		formatStr, err := ec.evaluateArgument(rawArgs[0])
		if err != nil {
			return "", err
		}
		evalArgs := make([]string, len(rawArgs)-1)
		for i := 1; i < len(rawArgs); i++ {
			val, err := ec.evaluateArgument(rawArgs[i])
			if err != nil {
				return "", err
			}
			evalArgs[i-1] = val
		}
		res := formatStr
		for i, val := range evalArgs {
			placeholder := fmt.Sprintf("{%d}", i)
			res = strings.ReplaceAll(res, placeholder, val)
		}
		// If there are unreplaced placeholders like {X}, replace with empty string
		rePlaceholder := regexp.MustCompile(`\{\d+\}`)
		res = rePlaceholder.ReplaceAllString(res, "")
		return res, nil

	case "hashFiles", "fromJson", "toJson", "join":
		return "", NewUnsupportedError(fmt.Sprintf("unsupported expression function: %s", funcName))

	default:
		return "", NewUnsupportedError(fmt.Sprintf("unsupported expression function: %s", funcName))
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