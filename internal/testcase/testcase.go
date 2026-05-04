package testcase

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// TestCase is a single skraft test case loaded from a YAML file.
type TestCase struct {
	ID     string      `yaml:"id"`
	Query  string      `yaml:"query"`
	Expect Expectation `yaml:"expect"`
}

// Expectation holds the assertions for a test case.
// All fields are optional; omitted fields are not checked.
// Unknown fields cause a parse error (future-proofing).
type Expectation struct {
	// Layer A: skill trigger
	Triggered *bool `yaml:"triggered"`

	// Layer B: tool usage
	ToolsUsedIncludes []string `yaml:"tools_used_includes"`
	ToolsUsedExcludes []string `yaml:"tools_used_excludes"`

	// Layer E1: output format (string comparisons)
	OutputContains string `yaml:"output_contains"`
	OutputExcludes string `yaml:"output_excludes"`
	OutputMatches  string `yaml:"output_matches"` // Go regexp syntax

	// Layer E1: output size
	OutputLengthMin *int `yaml:"output_length_min"` // rune count
	OutputLengthMax *int `yaml:"output_length_max"`
	OutputLinesMin  *int `yaml:"output_lines_min"`
	OutputLinesMax  *int `yaml:"output_lines_max"`
}

// Load reads and validates a test case from a YAML file.
// Unknown fields in the YAML are rejected to prevent silent misuse.
func Load(path string) (TestCase, error) {
	f, err := os.Open(path)
	if err != nil {
		return TestCase{}, err
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)

	var tc TestCase
	if err := dec.Decode(&tc); err != nil {
		return TestCase{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if tc.ID == "" {
		return TestCase{}, fmt.Errorf("%s: missing required field 'id'", path)
	}
	if tc.Query == "" {
		return TestCase{}, fmt.Errorf("%s: missing required field 'query'", path)
	}
	return tc, nil
}
