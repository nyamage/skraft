package testcase_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nyamage/skraft/internal/testcase"
)

func writeYAML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestLoad_Minimal(t *testing.T) {
	path := writeYAML(t, "id: trigger-basic\nquery: \"東京の天気\"\n")
	tc, err := testcase.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tc.ID != "trigger-basic" {
		t.Errorf("ID = %q, want %q", tc.ID, "trigger-basic")
	}
	if tc.Query != "東京の天気" {
		t.Errorf("Query = %q", tc.Query)
	}
}

func TestLoad_AllExpectFields(t *testing.T) {
	content := `
id: full
query: "test"
expect:
  triggered: true
  tools_used_includes: [Skill, search_filings]
  tools_used_excludes: [Bash]
  output_contains: "hello"
  output_excludes: "error"
  output_matches: "^hello"
  output_length_min: 5
  output_length_max: 100
  output_lines_min: 1
  output_lines_max: 3
`
	path := writeYAML(t, content)
	tc, err := testcase.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tc.Expect.Triggered == nil || !*tc.Expect.Triggered {
		t.Errorf("Triggered should be true")
	}
	if len(tc.Expect.ToolsUsedIncludes) != 2 {
		t.Errorf("ToolsUsedIncludes = %v, want 2 items", tc.Expect.ToolsUsedIncludes)
	}
	if tc.Expect.OutputContains != "hello" {
		t.Errorf("OutputContains = %q", tc.Expect.OutputContains)
	}
	if tc.Expect.OutputLengthMin == nil || *tc.Expect.OutputLengthMin != 5 {
		t.Errorf("OutputLengthMin wrong")
	}
}

func TestLoad_TriggeredFalse(t *testing.T) {
	path := writeYAML(t, "id: neg\nquery: \"1+1\"\nexpect:\n  triggered: false\n")
	tc, err := testcase.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tc.Expect.Triggered == nil || *tc.Expect.Triggered {
		t.Errorf("Triggered should be false")
	}
}

func TestLoad_UnknownFieldError(t *testing.T) {
	path := writeYAML(t, "id: x\nquery: \"q\"\nrubric: bad_field\n")
	_, err := testcase.Load(path)
	if err == nil {
		t.Error("expected error for unknown field 'rubric', got nil")
	}
}

func TestLoad_MissingID(t *testing.T) {
	path := writeYAML(t, "query: \"q\"\n")
	_, err := testcase.Load(path)
	if err == nil {
		t.Error("expected error for missing id")
	}
}

func TestLoad_MissingQuery(t *testing.T) {
	path := writeYAML(t, "id: x\n")
	_, err := testcase.Load(path)
	if err == nil {
		t.Error("expected error for missing query")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := testcase.Load(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err == nil {
		t.Error("expected error for missing file")
	}
}
