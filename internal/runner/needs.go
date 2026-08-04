package runner

import (
	"fmt"
	"sort"
	"strings"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
)

// JobNeedsData contains the aggregated result and resolved outputs of a needed job.
type JobNeedsData struct {
	Result   string
	Outputs  map[string]string
	IsMatrix bool
}

// ValidateNeeds validates the job needs graph in a workflow.
func ValidateNeeds(wf *workflow.Workflow) error {
	if wf == nil || len(wf.Jobs) == 0 {
		return nil
	}

	// 1. Check all referenced needed jobs exist
	// Collect job IDs sorted for deterministic output
	jobIDs := make([]string, 0, len(wf.Jobs))
	for id := range wf.Jobs {
		jobIDs = append(jobIDs, id)
	}
	sort.Strings(jobIDs)

	for _, jobID := range jobIDs {
		job := wf.Jobs[jobID]
		for _, needID := range getJobNeedsList(job) {
			if _, exists := wf.Jobs[needID]; !exists {
				return workflow.NewValidationErrorWithCode(
					jobID,
					fmt.Sprintf("job '%s' depends on unknown job '%s'", jobID, needID),
					2,
				)
			}
		}
	}

	// 2. Cycle detection using DFS
	if cycle := findDependencyCycle(wf); cycle != "" {
		return workflow.NewValidationErrorWithCode(
			"",
			fmt.Sprintf("dependency cycle detected: %s", cycle),
			2,
		)
	}

	return nil
}

// TopologicalSort returns job IDs of the workflow sorted in topological execution order.
func TopologicalSort(wf *workflow.Workflow) ([]string, error) {
	if err := ValidateNeeds(wf); err != nil {
		return nil, err
	}

	// Calculate in-degree (number of dependencies each job has)
	inDegree := make(map[string]int)
	dependents := make(map[string][]string)

	jobIDs := make([]string, 0, len(wf.Jobs))
	for id := range wf.Jobs {
		jobIDs = append(jobIDs, id)
		inDegree[id] = 0
	}
	sort.Strings(jobIDs)

	for id, job := range wf.Jobs {
		needs := getJobNeedsList(job)
		inDegree[id] = len(needs)
		for _, needID := range needs {
			dependents[needID] = append(dependents[needID], id)
		}
	}

	// Find all jobs with in-degree 0
	var queue []string
	for _, id := range jobIDs {
		if inDegree[id] == 0 {
			queue = append(queue, id)
		}
	}

	var result []string
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		result = append(result, curr)

		deps := dependents[curr]
		sort.Strings(deps)

		for _, dep := range deps {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
				sort.Strings(queue)
			}
		}
	}

	if len(result) != len(wf.Jobs) {
		return nil, workflow.NewValidationErrorWithCode("", "dependency cycle detected", 2)
	}

	return result, nil
}

// AggregateJobResults aggregates job instance results per original job ID.
func AggregateJobResults(instanceResults []*JobResult) Status {
	if len(instanceResults) == 0 {
		return StatusSkipped
	}

	hasFailure := false
	allSkipped := true

	for _, res := range instanceResults {
		if res == nil {
			continue
		}
		if res.Status == StatusSkipped {
			continue
		}
		allSkipped = false
		if res.ExitCode != 0 || res.Status == StatusFailure {
			hasFailure = true
		}
	}

	if hasFailure {
		return StatusFailure
	}
	if allSkipped {
		return StatusSkipped
	}
	return StatusSuccess
}

// BuildNeedsContext builds the needs context map for expression evaluation.
func BuildNeedsContext(
	job workflow.Job,
	jobResults map[string]Status,
	jobOutputs map[string]map[string]string,
	matrixJobs map[string]bool,
) map[string]JobNeedsData {
	needsCtx := make(map[string]JobNeedsData)
	neededJobIDs := getJobNeedsList(job)
	for _, needID := range neededJobIDs {
		status, exists := jobResults[needID]
		if !exists {
			status = StatusSkipped
		}
		outputs := jobOutputs[needID]
		if outputs == nil {
			outputs = make(map[string]string)
		}
		needsCtx[needID] = JobNeedsData{
			Result:   string(status),
			Outputs:  outputs,
			IsMatrix: matrixJobs[needID],
		}
	}
	return needsCtx
}

// getJobNeedsList returns the needs list for a job.
func getJobNeedsList(job workflow.Job) []string {
	if job.Needs == nil {
		return nil
	}
	switch v := job.Needs.(type) {
	case []string:
		return v
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	case []interface{}:
		var result []string
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

// findDependencyCycle detects cycles in job dependencies using DFS.
func findDependencyCycle(wf *workflow.Workflow) string {
	jobIDs := make([]string, 0, len(wf.Jobs))
	for id := range wf.Jobs {
		jobIDs = append(jobIDs, id)
	}
	sort.Strings(jobIDs)

	visited := make(map[string]bool)
	onPath := make(map[string]bool)
	var path []string
	var cycleStr string

	var dfs func(u string) bool
	dfs = func(u string) bool {
		visited[u] = true
		onPath[u] = true
		path = append(path, u)

		deps := getJobNeedsList(wf.Jobs[u])
		sort.Strings(deps)

		for _, v := range deps {
			if onPath[v] {
				cycleStart := -1
				for i, node := range path {
					if node == v {
						cycleStart = i
						break
					}
				}
				if cycleStart != -1 {
					cycleNodes := append([]string{}, path[cycleStart:]...)
					cycleNodes = append(cycleNodes, v)
					cycleStr = strings.Join(cycleNodes, " -> ")
					return true
				}
			}
			if !visited[v] {
				if dfs(v) {
					return true
				}
			}
		}

		path = path[:len(path)-1]
		onPath[u] = false
		return false
	}

	for _, id := range jobIDs {
		if !visited[id] {
			if dfs(id) {
				return cycleStr
			}
		}
	}

	return ""
}
