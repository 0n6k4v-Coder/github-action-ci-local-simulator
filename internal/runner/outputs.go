package runner

// StepOutputs stores outputs from steps for expression interpolation.
type StepOutputs struct {
	outputs map[string]map[string]string // stepID -> key -> value
}

// NewStepOutputs creates a new StepOutputs store.
func NewStepOutputs() *StepOutputs {
	return &StepOutputs{
		outputs: make(map[string]map[string]string),
	}
}

// SetOutputs sets the outputs for a step.
func (so *StepOutputs) SetOutputs(stepID string, outputs map[string]string) {
	so.outputs[stepID] = outputs
}

// GetOutputs returns the outputs for a step.
func (so *StepOutputs) GetOutputs(stepID string) map[string]string {
	return so.outputs[stepID]
}

// GetOutput returns a specific output value for a step.
func (so *StepOutputs) GetOutput(stepID, key string) (string, bool) {
	if stepOutputs, ok := so.outputs[stepID]; ok {
		value, ok := stepOutputs[key]
		return value, ok
	}
	return "", false
}
