package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEvents_SkillTriggered(t *testing.T) {
	lines := []string{
		`{"hook_event_name":"PreToolUse","tool_name":"Skill","tool_input":{"skill":"weather-haiku"}}`,
		`{"hook_event_name":"Stop","last_assistant_message":"晴れです"}`,
	}
	path := writeEvents(t, lines)
	obs, err := parseEvents(path, "weather-haiku", "weather-haiku")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !obs.Triggered {
		t.Error("expected Triggered=true")
	}
	if obs.Output != "晴れです" {
		t.Errorf("Output = %q", obs.Output)
	}
	if !hasTool(obs.ToolsUsed, "Skill") {
		t.Errorf("expected Skill in ToolsUsed, got %v", obs.ToolsUsed)
	}
}

func TestParseEvents_SkillNotTriggered(t *testing.T) {
	lines := []string{
		`{"hook_event_name":"PreToolUse","tool_name":"TodoWrite","tool_input":{}}`,
		`{"hook_event_name":"Stop","last_assistant_message":"2"}`,
	}
	path := writeEvents(t, lines)
	obs, err := parseEvents(path, "weather-haiku", "weather-haiku")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Triggered {
		t.Error("expected Triggered=false")
	}
	if obs.Output != "2" {
		t.Errorf("Output = %q", obs.Output)
	}
}

func TestParseEvents_MatchesFrontmatterName(t *testing.T) {
	lines := []string{
		`{"hook_event_name":"PreToolUse","tool_name":"Skill","tool_input":{"skill":"Weather Haiku"}}`,
		`{"hook_event_name":"Stop","last_assistant_message":"ok"}`,
	}
	path := writeEvents(t, lines)
	obs, err := parseEvents(path, "weather-haiku", "Weather Haiku")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !obs.Triggered {
		t.Error("expected Triggered=true when matching frontmatter name")
	}
}

func TestParseEvents_NoFile(t *testing.T) {
	obs, err := parseEvents(filepath.Join(t.TempDir(), "none.jsonl"), "s", "s")
	if err != nil {
		t.Fatalf("unexpected error for missing file: %v", err)
	}
	if obs.Triggered || len(obs.ToolsUsed) > 0 || obs.Output != "" {
		t.Errorf("expected zero result, got %+v", obs)
	}
}

func TestParseEvents_MalformedLineSkipped(t *testing.T) {
	lines := []string{
		`not-json`,
		`{"hook_event_name":"Stop","last_assistant_message":"ok"}`,
	}
	path := writeEvents(t, lines)
	obs, err := parseEvents(path, "s", "s")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Output != "ok" {
		t.Errorf("Output = %q, want ok", obs.Output)
	}
}

func writeEvents(t *testing.T, lines []string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "events.jsonl")
	var content string
	for _, l := range lines {
		content += l + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
