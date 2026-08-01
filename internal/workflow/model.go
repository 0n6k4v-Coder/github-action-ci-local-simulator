package workflow

// Workflow represents a GitHub Actions workflow file.
type Workflow struct {
	Name     string         `yaml:"name"`
	On       interface{}    `yaml:"on"`
	Env      map[string]any `yaml:"env"`
	Jobs     map[string]Job `yaml:"jobs"`
	Defaults *Defaults      `yaml:"defaults"`
}

// Job represents a job in a workflow.
type Job struct {
	Name            string            `yaml:"name"`
	RunsOn          interface{}       `yaml:"runs-on"`
	Needs           interface{}       `yaml:"needs"`
	If              string            `yaml:"if"`
	Env             map[string]any    `yaml:"env"`
	Steps           []Step            `yaml:"steps"`
	Strategy        *Strategy         `yaml:"strategy"`
	Outputs         map[string]string `yaml:"outputs"`
	TimeoutMinutes  int               `yaml:"timeout-minutes"`
	Defaults        *Defaults         `yaml:"defaults"`
	// Runtime fields
	instanceID      string
	githubEnvPath   string
	githubPathPath  string
}

// Strategy represents the strategy configuration for a job.
type Strategy struct {
	Matrix        map[string]any `yaml:"matrix"`
	FailFast      bool           `yaml:"fail-fast"`
	MaxParallel   int            `yaml:"max-parallel"`
}

// Defaults represents default settings for a job.
type Defaults struct {
	Run *RunDefaults `yaml:"run"`
}

// RunDefaults represents default run settings.
type RunDefaults struct {
	Shell            string `yaml:"shell"`
	WorkingDirectory string `yaml:"working-directory"`
}

// Step represents a step in a job.
type Step struct {
	ID                string         `yaml:"id"`
	Name              string         `yaml:"name"`
	If                string         `yaml:"if"`
	Run               string         `yaml:"run"`
	Uses              string         `yaml:"uses"`
	With              map[string]any `yaml:"with"`
	Env               map[string]any `yaml:"env"`
	Shell             string         `yaml:"shell"`
	WorkingDirectory  string         `yaml:"working-directory"`
	ContinueOnError   bool           `yaml:"continue-on-error"`
	TimeoutMinutes    int            `yaml:"timeout-minutes"`
}