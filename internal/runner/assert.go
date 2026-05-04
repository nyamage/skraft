package runner

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/nyamage/skraft/internal/testcase"
)

// ObservedResult holds what was actually observed during a test run.
type ObservedResult struct {
	Triggered bool
	ToolsUsed []string
	Output    string
}

// Failure describes a single assertion that did not pass.
type Failure struct {
	Field   string
	Message string
}

// Evaluate checks obs against exp and returns all assertion failures.
// All assertions are AND-combined: every failing check produces a Failure.
func Evaluate(skillName string, exp testcase.Expectation, obs ObservedResult) []Failure {
	var failures []Failure

	if exp.Triggered != nil && obs.Triggered != *exp.Triggered {
		failures = append(failures, Failure{
			Field:   "triggered",
			Message: fmt.Sprintf("expected %v, got %v", *exp.Triggered, obs.Triggered),
		})
	}

	for _, tool := range exp.ToolsUsedIncludes {
		if !hasTool(obs.ToolsUsed, tool) {
			failures = append(failures, Failure{
				Field:   "tools_used_includes",
				Message: fmt.Sprintf("missing tool %q (observed: %v)", tool, obs.ToolsUsed),
			})
		}
	}

	for _, tool := range exp.ToolsUsedExcludes {
		if hasTool(obs.ToolsUsed, tool) {
			failures = append(failures, Failure{
				Field:   "tools_used_excludes",
				Message: fmt.Sprintf("unexpected tool %q was used", tool),
			})
		}
	}

	if exp.OutputContains != "" && !strings.Contains(obs.Output, exp.OutputContains) {
		failures = append(failures, Failure{
			Field:   "output_contains",
			Message: fmt.Sprintf("output does not contain %q", exp.OutputContains),
		})
	}

	if exp.OutputExcludes != "" && strings.Contains(obs.Output, exp.OutputExcludes) {
		failures = append(failures, Failure{
			Field:   "output_excludes",
			Message: fmt.Sprintf("output contains forbidden string %q", exp.OutputExcludes),
		})
	}

	if exp.OutputMatches != "" {
		re, err := regexp.Compile(exp.OutputMatches)
		if err != nil {
			failures = append(failures, Failure{
				Field:   "output_matches",
				Message: fmt.Sprintf("invalid regex %q: %v", exp.OutputMatches, err),
			})
		} else if !re.MatchString(obs.Output) {
			failures = append(failures, Failure{
				Field:   "output_matches",
				Message: fmt.Sprintf("output does not match /%s/", exp.OutputMatches),
			})
		}
	}

	runeCount := utf8.RuneCountInString(obs.Output)
	if exp.OutputLengthMin != nil && runeCount < *exp.OutputLengthMin {
		failures = append(failures, Failure{
			Field:   "output_length_min",
			Message: fmt.Sprintf("output length %d < min %d", runeCount, *exp.OutputLengthMin),
		})
	}
	if exp.OutputLengthMax != nil && runeCount > *exp.OutputLengthMax {
		failures = append(failures, Failure{
			Field:   "output_length_max",
			Message: fmt.Sprintf("output length %d > max %d", runeCount, *exp.OutputLengthMax),
		})
	}

	lineCount := countLines(obs.Output)
	if exp.OutputLinesMin != nil && lineCount < *exp.OutputLinesMin {
		failures = append(failures, Failure{
			Field:   "output_lines_min",
			Message: fmt.Sprintf("line count %d < min %d", lineCount, *exp.OutputLinesMin),
		})
	}
	if exp.OutputLinesMax != nil && lineCount > *exp.OutputLinesMax {
		failures = append(failures, Failure{
			Field:   "output_lines_max",
			Message: fmt.Sprintf("line count %d > max %d", lineCount, *exp.OutputLinesMax),
		})
	}

	return failures
}

func hasTool(tools []string, name string) bool {
	for _, t := range tools {
		if t == name {
			return true
		}
	}
	return false
}

// countLines returns the number of lines in s.
// Empty string = 0, "a" = 1, "a\nb" = 2, "a\n" = 1.
func countLines(s string) int {
	if s == "" {
		return 0
	}
	s = strings.TrimRight(s, "\n")
	return len(strings.Split(s, "\n"))
}
