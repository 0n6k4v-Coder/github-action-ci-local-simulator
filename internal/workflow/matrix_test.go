package workflow

import (
	"testing"
)

func TestComputeCartesianProduct(t *testing.T) {
	dimensions := map[string][]any{
		"os":      {"ubuntu", "debian"},
		"version": {"1", "2"},
	}

	result := computeCartesianProduct(dimensions)

	if len(result) != 4 {
		t.Errorf("expected 4 combinations, got %d", len(result))
	}

	expected := []map[string]any{
		{"os": "ubuntu", "version": "1"},
		{"os": "ubuntu", "version": "2"},
		{"os": "debian", "version": "1"},
		{"os": "debian", "version": "2"},
	}

	for i, exp := range expected {
		if result[i]["os"] != exp["os"] || result[i]["version"] != exp["version"] {
			t.Errorf("combination %d: expected %v, got %v", i, exp, result[i])
		}
	}
}

func TestComputeCartesianProductEmpty(t *testing.T) {
	dimensions := map[string][]any{}
	result := computeCartesianProduct(dimensions)

	if len(result) != 1 {
		t.Errorf("expected 1 empty combination, got %d", len(result))
	}
	if len(result[0]) != 0 {
		t.Errorf("expected empty map, got %v", result[0])
	}
}

func TestApplyExcludesFullMatch(t *testing.T) {
	combinations := []map[string]any{
		{"os": "ubuntu", "version": "1"},
		{"os": "ubuntu", "version": "2"},
		{"os": "debian", "version": "1"},
		{"os": "debian", "version": "2"},
	}

	excludes := []map[string]any{
		{"os": "debian", "version": "1"},
	}

	result := applyExcludes(combinations, excludes)

	if len(result) != 3 {
		t.Errorf("expected 3 combinations after exclude, got %d", len(result))
	}

	// Check debian/1 is removed
	for _, combo := range result {
		if combo["os"] == "debian" && combo["version"] == "1" {
			t.Error("expected debian/1 to be excluded")
		}
	}
}

func TestApplyExcludesPartialMatch(t *testing.T) {
	combinations := []map[string]any{
		{"os": "ubuntu", "version": "1"},
		{"os": "ubuntu", "version": "2"},
		{"os": "debian", "version": "1"},
		{"os": "debian", "version": "2"},
	}

	excludes := []map[string]any{
		{"os": "debian"},
	}

	result := applyExcludes(combinations, excludes)

	if len(result) != 2 {
		t.Errorf("expected 2 combinations after exclude, got %d", len(result))
	}

	// Check all debian are removed
	for _, combo := range result {
		if combo["os"] == "debian" {
			t.Error("expected all debian to be excluded")
		}
	}
}

func TestApplyIncludesNewKey(t *testing.T) {
	combinations := []map[string]any{
		{"os": "ubuntu", "version": "1"},
		{"os": "ubuntu", "version": "2"},
		{"os": "debian", "version": "1"},
		{"os": "debian", "version": "2"},
	}

	includes := []map[string]any{
		{"color": "red"},
	}

	baseKeys := []string{"os", "version"}
	result := applyIncludes(combinations, includes, baseKeys)

	if len(result) != 4 {
		t.Errorf("expected 4 combinations, got %d", len(result))
	}

	for _, combo := range result {
		if combo["color"] != "red" {
			t.Errorf("expected color=red in all combinations, got %v", combo)
		}
	}
}

func TestApplyIncludesPartialMatch(t *testing.T) {
	combinations := []map[string]any{
		{"os": "ubuntu", "version": "1"},
		{"os": "ubuntu", "version": "2"},
		{"os": "debian", "version": "1"},
		{"os": "debian", "version": "2"},
	}

	includes := []map[string]any{
		{"os": "ubuntu", "extra": "yes"},
	}

	baseKeys := []string{"os", "version"}
	result := applyIncludes(combinations, includes, baseKeys)

	if len(result) != 4 {
		t.Errorf("expected 4 combinations, got %d", len(result))
	}

	// ubuntu should have extra=yes
	for _, combo := range result {
		if combo["os"] == "ubuntu" {
			if combo["extra"] != "yes" {
				t.Errorf("expected ubuntu to have extra=yes, got %v", combo)
			}
		} else if combo["os"] == "debian" {
			if _, ok := combo["extra"]; ok {
				t.Errorf("expected debian to not have extra, got %v", combo)
			}
		}
	}
}

func TestApplyIncludesNoMatchCreatesNew(t *testing.T) {
	combinations := []map[string]any{
		{"os": "ubuntu", "version": "1"},
		{"os": "ubuntu", "version": "2"},
		{"os": "debian", "version": "1"},
		{"os": "debian", "version": "2"},
	}

	includes := []map[string]any{
		{"os": "windows", "version": "special"},
	}

	baseKeys := []string{"os", "version"}
	result := applyIncludes(combinations, includes, baseKeys)

	if len(result) != 5 {
		t.Errorf("expected 5 combinations, got %d", len(result))
	}

	// Check new combination exists
	found := false
	for _, combo := range result {
		if combo["os"] == "windows" && combo["version"] == "special" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected new windows/special combination to be created")
	}
}

func TestApplyIncludesOverwriteBehavior(t *testing.T) {
	combinations := []map[string]any{
		{"fruit": "apple", "animal": "cat"},
		{"fruit": "apple", "animal": "dog"},
		{"fruit": "pear", "animal": "cat"},
		{"fruit": "pear", "animal": "dog"},
	}

	includes := []map[string]any{
		{"color": "green"},
		{"color": "pink", "animal": "cat"},
		{"fruit": "apple", "shape": "circle"},
	}

	baseKeys := []string{"fruit", "animal"}
	result := applyIncludes(combinations, includes, baseKeys)

	if len(result) != 4 {
		t.Errorf("expected 4 combinations, got %d", len(result))
	}

	// Check expected results
	expected := []map[string]any{
		{"fruit": "apple", "animal": "cat", "color": "pink", "shape": "circle"},
		{"fruit": "apple", "animal": "dog", "color": "green", "shape": "circle"},
		{"fruit": "pear", "animal": "cat", "color": "pink"},
		{"fruit": "pear", "animal": "dog", "color": "green"},
	}

	for i, exp := range expected {
		for k, v := range exp {
			if result[i][k] != v {
				t.Errorf("combination %s: key %s expected %v, got %v", comboToString(result[i]), k, v, result[i][k])
			}
		}
	}
}

func TestExcludeBeforeInclude(t *testing.T) {
	combinations := []map[string]any{
		{"os": "ubuntu"},
		{"os": "debian"},
	}

	excludes := []map[string]any{
		{"os": "debian"},
	}

	includes := []map[string]any{
		{"os": "debian", "version": "special"},
	}

	baseKeys := []string{"os"}

	// Apply exclude first
	result := applyExcludes(combinations, excludes)
	// Then apply include
	result = applyIncludes(result, includes, baseKeys)

	if len(result) != 2 {
		t.Errorf("expected 2 combinations, got %d", len(result))
	}

	// Should have ubuntu and debian/special
	foundUbuntu := false
	foundDebianSpecial := false
	for _, combo := range result {
		if combo["os"] == "ubuntu" && len(combo) == 1 {
			foundUbuntu = true
		}
		if combo["os"] == "debian" && combo["version"] == "special" {
			foundDebianSpecial = true
		}
	}

	if !foundUbuntu {
		t.Error("expected ubuntu combination")
	}
	if !foundDebianSpecial {
		t.Error("expected debian/special combination")
	}
}

func TestDeduplicateCombinations(t *testing.T) {
	combinations := []map[string]any{
		{"os": "ubuntu"},
		{"os": "ubuntu"},
		{"os": "ubuntu"},
	}

	result := deduplicateCombinations(combinations)

	if len(result) != 1 {
		t.Errorf("expected 1 combination after deduplication, got %d", len(result))
	}
	if result[0]["os"] != "ubuntu" {
		t.Errorf("expected ubuntu, got %v", result[0])
	}
}

func TestExpandMatrixBasic(t *testing.T) {
	job := Job{
		RunsOn: []string{"ubuntu-latest"},
		Steps:  []Step{{Run: "echo hello"}},
		Strategy: &Strategy{
			Matrix: map[string]any{
				"version": []any{"a", "b", "c"},
			},
		},
	}

	expanded, err := ExpandMatrix("test", job)
	if err != nil {
		t.Fatalf("ExpandMatrix failed: %v", err)
	}

	if len(expanded) != 3 {
		t.Errorf("expected 3 expanded jobs, got %d", len(expanded))
	}

	for i, expJob := range expanded {
		matrix := expJob.GetMatrixContext()
		if matrix == nil {
			t.Errorf("job %d: missing matrix context", i)
			continue
		}
		expectedVersion := string(rune('a' + i))
		if matrix["version"] != expectedVersion {
			t.Errorf("job %d: expected version=%s, got %v", i, expectedVersion, matrix["version"])
		}
	}
}

func TestExpandMatrixExclude(t *testing.T) {
	job := Job{
		RunsOn: []string{"ubuntu-latest"},
		Steps:  []Step{{Run: "echo hello"}},
		Strategy: &Strategy{
			Matrix: map[string]any{
				"os":      []any{"ubuntu", "debian"},
				"version": []any{"1", "2"},
				"exclude": []any{map[string]any{"os": "debian", "version": "1"}},
			},
		},
	}

	expanded, err := ExpandMatrix("test", job)
	if err != nil {
		t.Fatalf("ExpandMatrix failed: %v", err)
	}

	if len(expanded) != 3 {
		t.Errorf("expected 3 expanded jobs after exclude, got %d", len(expanded))
	}

	// Check debian/1 is excluded
	for _, expJob := range expanded {
		matrix := expJob.GetMatrixContext()
		if matrix["os"] == "debian" && matrix["version"] == "1" {
			t.Error("expected debian/1 to be excluded")
		}
	}
}

func TestExpandMatrixIncludeNewKey(t *testing.T) {
	job := Job{
		RunsOn: []string{"ubuntu-latest"},
		Steps:  []Step{{Run: "echo hello"}},
		Strategy: &Strategy{
			Matrix: map[string]any{
				"os":      []any{"ubuntu", "debian"},
				"version": []any{"1", "2"},
				"include": []any{map[string]any{"color": "red"}},
			},
		},
	}

	expanded, err := ExpandMatrix("test", job)
	if err != nil {
		t.Fatalf("ExpandMatrix failed: %v", err)
	}

	if len(expanded) != 4 {
		t.Errorf("expected 4 expanded jobs, got %d", len(expanded))
	}

	for _, expJob := range expanded {
		matrix := expJob.GetMatrixContext()
		if matrix["color"] != "red" {
			t.Errorf("expected color=red in all combinations, got %v", matrix)
		}
	}
}

func TestExpandMatrixIncludePartialMatch(t *testing.T) {
	job := Job{
		RunsOn: []string{"ubuntu-latest"},
		Steps:  []Step{{Run: "echo hello"}},
		Strategy: &Strategy{
			Matrix: map[string]any{
				"os":      []any{"ubuntu", "debian"},
				"version": []any{"1", "2"},
				"include": []any{map[string]any{"os": "ubuntu", "extra": "yes"}},
			},
		},
	}

	expanded, err := ExpandMatrix("test", job)
	if err != nil {
		t.Fatalf("ExpandMatrix failed: %v", err)
	}

	if len(expanded) != 4 {
		t.Errorf("expected 4 expanded jobs, got %d", len(expanded))
	}

	for _, expJob := range expanded {
		matrix := expJob.GetMatrixContext()
		if matrix["os"] == "ubuntu" {
			if matrix["extra"] != "yes" {
				t.Errorf("expected ubuntu to have extra=yes, got %v", matrix)
			}
		} else if matrix["os"] == "debian" {
			if _, ok := matrix["extra"]; ok {
				t.Errorf("expected debian to not have extra, got %v", matrix)
			}
		}
	}
}

func TestExpandMatrixIncludeNoMatch(t *testing.T) {
	job := Job{
		RunsOn: []string{"ubuntu-latest"},
		Steps:  []Step{{Run: "echo hello"}},
		Strategy: &Strategy{
			Matrix: map[string]any{
				"os":      []any{"ubuntu", "debian"},
				"version": []any{"1", "2"},
				"include": []any{map[string]any{"os": "windows", "version": "special"}},
			},
		},
	}

	expanded, err := ExpandMatrix("test", job)
	if err != nil {
		t.Fatalf("ExpandMatrix failed: %v", err)
	}

	if len(expanded) != 5 {
		t.Errorf("expected 5 expanded jobs, got %d", len(expanded))
	}

	found := false
	for _, expJob := range expanded {
		matrix := expJob.GetMatrixContext()
		if matrix["os"] == "windows" && matrix["version"] == "special" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected windows/special combination")
	}
}

func TestExpandMatrixZeroJobsEmptyArray(t *testing.T) {
	job := Job{
		RunsOn: []string{"ubuntu-latest"},
		Steps:  []Step{{Run: "echo hello"}},
		Strategy: &Strategy{
			Matrix: map[string]any{
				"version": []any{},
			},
		},
	}

	_, err := ExpandMatrix("test", job)
	if err == nil {
		t.Error("expected error for empty matrix array")
	}
}

func TestExpandMatrixZeroJobsExcludeAll(t *testing.T) {
	job := Job{
		RunsOn: []string{"ubuntu-latest"},
		Steps:  []Step{{Run: "echo hello"}},
		Strategy: &Strategy{
			Matrix: map[string]any{
				"os":      []any{"ubuntu"},
				"exclude": []any{map[string]any{"os": "ubuntu"}},
			},
		},
	}

	_, err := ExpandMatrix("test", job)
	if err == nil {
		t.Error("expected error for exclude all")
	}
}

func TestExpandMatrixSequentialIncludes(t *testing.T) {
	job := Job{
		RunsOn: []string{"ubuntu-latest"},
		Steps:  []Step{{Run: "echo hello"}},
		Strategy: &Strategy{
			Matrix: map[string]any{
				"fruit":  []any{"apple", "pear"},
				"animal": []any{"cat", "dog"},
				"include": []any{
					map[string]any{"color": "green"},
					map[string]any{"color": "pink", "animal": "cat"},
					map[string]any{"fruit": "apple", "shape": "circle"},
				},
			},
		},
	}

	expanded, err := ExpandMatrix("test", job)
	if err != nil {
		t.Fatalf("ExpandMatrix failed: %v", err)
	}

	if len(expanded) != 4 {
		t.Errorf("expected 4 expanded jobs, got %d", len(expanded))
	}

	// Verify all combinations have expected values
	for _, expJob := range expanded {
		matrix := expJob.GetMatrixContext()
		switch {
		case matrix["fruit"] == "apple" && matrix["animal"] == "cat":
			if matrix["color"] != "pink" || matrix["shape"] != "circle" {
				t.Errorf("apple/cat: expected color=pink shape=circle, got %v", matrix)
			}
		case matrix["fruit"] == "apple" && matrix["animal"] == "dog":
			if matrix["color"] != "green" || matrix["shape"] != "circle" {
				t.Errorf("apple/dog: expected color=green shape=circle, got %v", matrix)
			}
		case matrix["fruit"] == "pear" && matrix["animal"] == "cat":
			if matrix["color"] != "pink" {
				t.Errorf("pear/cat: expected color=pink, got %v", matrix)
			}
		case matrix["fruit"] == "pear" && matrix["animal"] == "dog":
			if matrix["color"] != "green" {
				t.Errorf("pear/dog: expected color=green, got %v", matrix)
			}
		}
	}
}

func TestExpandMatrixDuplicateInclude(t *testing.T) {
	job := Job{
		RunsOn: []string{"ubuntu-latest"},
		Steps:  []Step{{Run: "echo hello"}},
		Strategy: &Strategy{
			Matrix: map[string]any{
				"os": []any{"ubuntu"},
				"include": []any{
					map[string]any{"os": "ubuntu"},
					map[string]any{"os": "ubuntu"},
				},
			},
		},
	}

	expanded, err := ExpandMatrix("test", job)
	if err != nil {
		t.Fatalf("ExpandMatrix failed: %v", err)
	}

	if len(expanded) != 1 {
		t.Errorf("expected 1 expanded job after deduplication, got %d", len(expanded))
	}
}