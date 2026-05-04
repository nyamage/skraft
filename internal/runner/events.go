package runner

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

// hookEvent is the JSON structure Claude Code sends to hooks via stdin.
type hookEvent struct {
	HookEventName        string          `json:"hook_event_name"`
	ToolName             string          `json:"tool_name"`
	ToolInput            json.RawMessage `json:"tool_input"`
	LastAssistantMessage string          `json:"last_assistant_message"`
}

type skillToolInput struct {
	Skill string `json:"skill"`
}

// parseEvents reads events.jsonl and extracts observed behavior for a skill.
// skillDirName and skillName are both checked against tool_input.skill
// (Claude Code may use either the directory name or the frontmatter name).
// Returns zero ObservedResult (not error) if the file does not exist.
func parseEvents(path, skillDirName, skillName string) (ObservedResult, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return ObservedResult{}, nil
	}
	if err != nil {
		return ObservedResult{}, err
	}
	defer f.Close()

	var result ObservedResult
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev hookEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue // skip malformed lines
		}
		switch ev.HookEventName {
		case "PreToolUse":
			result.ToolsUsed = append(result.ToolsUsed, ev.ToolName)
			if ev.ToolName == "Skill" {
				var input skillToolInput
				if json.Unmarshal(ev.ToolInput, &input) == nil {
					if input.Skill == skillDirName || input.Skill == skillName {
						result.Triggered = true
					}
				}
			}
		case "Stop":
			result.Output = ev.LastAssistantMessage
		}
	}
	return result, scanner.Err()
}
