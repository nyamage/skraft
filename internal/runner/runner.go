package runner

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/nyamage/skraft/internal/skill"
	"github.com/nyamage/skraft/internal/testcase"
)

// Result holds the outcome of running a single test case.
type Result struct {
	TestCase   testcase.TestCase
	Pass       bool
	Failures   []Failure
	Duration   time.Duration
	Skipped    bool
	SkipReason string
	Observed   ObservedResult
}

// Run executes tc against skill s using claudeBin and returns the result.
// The test runs in an isolated temp HOME to avoid polluting the user's environment.
func Run(s skill.Skill, tc testcase.TestCase, claudeBin string) (Result, error) {
	if s.Context != "" && s.Context != "inline" {
		return Result{
			TestCase:   tc,
			Skipped:    true,
			SkipReason: fmt.Sprintf("'context: %s' is not supported in this version of skraft", s.Context),
		}, nil
	}

	tmpDir, err := os.MkdirTemp("", "skraft-test-*")
	if err != nil {
		return Result{}, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	eventsPath := filepath.Join(tmpDir, "events.jsonl")

	if err := setupEnv(tmpDir, s, eventsPath); err != nil {
		return Result{}, fmt.Errorf("setup test env: %w", err)
	}

	start := time.Now()
	if err := runClaude(claudeBin, tc.Query, tmpDir); err != nil {
		return Result{}, fmt.Errorf("claude --print: %w", err)
	}
	duration := time.Since(start)

	obs, err := parseEvents(eventsPath, s.DirName, s.Name)
	if err != nil {
		return Result{}, fmt.Errorf("parse events: %w", err)
	}

	failures := Evaluate(s.DirName, tc.Expect, obs)

	return Result{
		TestCase: tc,
		Pass:     len(failures) == 0,
		Failures: failures,
		Duration: duration,
		Observed: obs,
	}, nil
}

// hookConfig matches the Claude Code settings.json hooks schema.
type hookConfig struct {
	Hooks map[string][]hookEntryConfig `json:"hooks"`
}

type hookEntryConfig struct {
	Hooks []hookDefConfig `json:"hooks"`
}

type hookDefConfig struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// setupEnv creates the isolated .claude directory with injected hooks and
// a copy of the skill in .claude/skills/<dirName>/.
func setupEnv(tmpDir string, s skill.Skill, eventsPath string) error {
	claudeDir := filepath.Join(tmpDir, ".claude")
	skillsDir := filepath.Join(claudeDir, "skills", s.DirName)
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return err
	}

	if err := copyDir(s.Dir, skillsDir); err != nil {
		return fmt.Errorf("copy skill %s: %w", s.DirName, err)
	}

	// Append all PreToolUse and Stop events to eventsPath via hook.
	hookCmd := fmt.Sprintf("cat >> %s", eventsPath)
	cfg := hookConfig{
		Hooks: map[string][]hookEntryConfig{
			"PreToolUse": {{Hooks: []hookDefConfig{{Type: "command", Command: hookCmd}}}},
			"Stop":       {{Hooks: []hookDefConfig{{Type: "command", Command: hookCmd}}}},
		},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644)
}

// runClaude runs claude --print <query> with HOME=homeDir (isolated environment).
func runClaude(claudeBin, query, homeDir string) error {
	cmd := exec.Command(claudeBin, "--print", query)
	// Override HOME so Claude Code uses our injected settings.json.
	env := make([]string, 0, len(os.Environ()))
	for _, e := range os.Environ() {
		if len(e) >= 5 && e[:5] == "HOME=" {
			continue
		}
		env = append(env, e)
	}
	cmd.Env = append(env, "HOME="+homeDir)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

// copyDir recursively copies src directory into dst.
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
