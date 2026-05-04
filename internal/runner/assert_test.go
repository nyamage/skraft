package runner

import (
	"testing"

	"github.com/nyamage/skraft/internal/testcase"
)

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

func TestEvaluate_AllPass(t *testing.T) {
	exp := testcase.Expectation{
		Triggered:         boolPtr(true),
		ToolsUsedIncludes: []string{"Skill"},
		ToolsUsedExcludes: []string{"Bash"},
		OutputContains:    "hello",
		OutputExcludes:    "error",
		OutputMatches:     "^hello",
		OutputLengthMin:   intPtr(3),
		OutputLengthMax:   intPtr(20),
		OutputLinesMin:    intPtr(1),
		OutputLinesMax:    intPtr(2),
	}
	obs := ObservedResult{
		Triggered: true,
		ToolsUsed: []string{"Skill"},
		Output:    "hello world",
	}
	failures := Evaluate("my-skill", exp, obs)
	if len(failures) != 0 {
		t.Errorf("expected no failures, got: %v", failures)
	}
}

func TestEvaluate_TriggeredMismatch(t *testing.T) {
	exp := testcase.Expectation{Triggered: boolPtr(true)}
	obs := ObservedResult{Triggered: false}
	failures := Evaluate("s", exp, obs)
	if len(failures) != 1 || failures[0].Field != "triggered" {
		t.Errorf("unexpected failures: %v", failures)
	}
}

func TestEvaluate_NegativeTriggered(t *testing.T) {
	exp := testcase.Expectation{Triggered: boolPtr(false)}
	obs := ObservedResult{Triggered: false}
	failures := Evaluate("s", exp, obs)
	if len(failures) != 0 {
		t.Errorf("expected no failures, got: %v", failures)
	}
}

func TestEvaluate_MissingIncludeTool(t *testing.T) {
	exp := testcase.Expectation{ToolsUsedIncludes: []string{"search_filings"}}
	obs := ObservedResult{ToolsUsed: []string{"Skill"}}
	failures := Evaluate("s", exp, obs)
	if len(failures) != 1 || failures[0].Field != "tools_used_includes" {
		t.Errorf("unexpected: %v", failures)
	}
}

func TestEvaluate_ForbiddenToolUsed(t *testing.T) {
	exp := testcase.Expectation{ToolsUsedExcludes: []string{"Bash"}}
	obs := ObservedResult{ToolsUsed: []string{"Skill", "Bash"}}
	failures := Evaluate("s", exp, obs)
	if len(failures) != 1 || failures[0].Field != "tools_used_excludes" {
		t.Errorf("unexpected: %v", failures)
	}
}

func TestEvaluate_OutputContains(t *testing.T) {
	exp := testcase.Expectation{OutputContains: "hello"}
	obs := ObservedResult{Output: "goodbye"}
	failures := Evaluate("s", exp, obs)
	if len(failures) != 1 {
		t.Errorf("expected 1 failure, got %v", failures)
	}
}

func TestEvaluate_OutputExcludes(t *testing.T) {
	exp := testcase.Expectation{OutputExcludes: "error"}
	obs := ObservedResult{Output: "an error occurred"}
	failures := Evaluate("s", exp, obs)
	if len(failures) != 1 {
		t.Errorf("expected 1 failure, got %v", failures)
	}
}

func TestEvaluate_OutputMatchesValid(t *testing.T) {
	exp := testcase.Expectation{OutputMatches: `^\d+$`}
	obs := ObservedResult{Output: "42"}
	if f := Evaluate("s", exp, obs); len(f) != 0 {
		t.Errorf("expected pass, got %v", f)
	}
	obs.Output = "abc"
	if f := Evaluate("s", exp, obs); len(f) != 1 {
		t.Errorf("expected 1 failure, got %v", f)
	}
}

func TestEvaluate_OutputMatchesInvalidRegex(t *testing.T) {
	exp := testcase.Expectation{OutputMatches: "[invalid"}
	obs := ObservedResult{Output: "x"}
	failures := Evaluate("s", exp, obs)
	if len(failures) != 1 || failures[0].Field != "output_matches" {
		t.Errorf("expected regex error failure, got %v", failures)
	}
}

func TestEvaluate_OutputLength(t *testing.T) {
	exp := testcase.Expectation{OutputLengthMin: intPtr(5), OutputLengthMax: intPtr(10)}
	obs := ObservedResult{Output: "hi"}
	f := Evaluate("s", exp, obs)
	if len(f) != 1 || f[0].Field != "output_length_min" {
		t.Errorf("expected length_min failure, got %v", f)
	}
	obs.Output = "hello world"
	f = Evaluate("s", exp, obs)
	if len(f) != 1 || f[0].Field != "output_length_max" {
		t.Errorf("expected length_max failure, got %v", f)
	}
}

func TestEvaluate_OutputLinesJapanese(t *testing.T) {
	exp := testcase.Expectation{OutputLengthMin: intPtr(3), OutputLengthMax: intPtr(3)}
	obs := ObservedResult{Output: "東京都"}
	f := Evaluate("s", exp, obs)
	if len(f) != 0 {
		t.Errorf("expected pass for 3 Japanese chars, got %v", f)
	}
}

func TestEvaluate_OutputLines(t *testing.T) {
	exp := testcase.Expectation{OutputLinesMin: intPtr(3), OutputLinesMax: intPtr(3)}
	obs := ObservedResult{Output: "line1\nline2\nline3"}
	if f := Evaluate("s", exp, obs); len(f) != 0 {
		t.Errorf("expected pass, got %v", f)
	}
	obs.Output = "line1\nline2"
	f := Evaluate("s", exp, obs)
	if len(f) != 1 || f[0].Field != "output_lines_min" {
		t.Errorf("expected lines_min failure, got %v", f)
	}
}

func TestEvaluate_EmptyOutput(t *testing.T) {
	exp := testcase.Expectation{OutputLinesMin: intPtr(1)}
	obs := ObservedResult{Output: ""}
	f := Evaluate("s", exp, obs)
	if len(f) != 1 || f[0].Field != "output_lines_min" {
		t.Errorf("expected lines_min failure for empty, got %v", f)
	}
}
