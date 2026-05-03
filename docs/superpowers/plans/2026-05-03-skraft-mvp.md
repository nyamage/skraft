# skraft MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `skraft` — a Go CLI that manages Claude Agent Skills across Git, Claude Code (symlinks), and Claude.ai (zip + ledger).

**Architecture:** Single Go binary with Cobra subcommands. Core logic in `internal/` packages (`git`, `skill`, `config`, `ledger`). Commands in `cmd/` delegate to internal packages. SQLite WAL ledger at `.skraft/ledger.db` tracks Claude.ai upload state. Version truth is `git describe --tags --always`.

**Tech Stack:** Go 1.21+, `github.com/spf13/cobra` v1.8.1, `modernc.org/sqlite` v1.29.9, `gopkg.in/yaml.v3` v3.0.1, `github.com/BurntSushi/toml` v1.3.2

---

## File Map

| File | Responsibility |
|------|---------------|
| `main.go` | Entry point, delegates to `cmd.Execute()` |
| `cmd/root.go` | Root cobra command, global flags, `findRepoRoot()` helper |
| `cmd/init.go` | `skraft init` |
| `cmd/link.go` | `skraft link [skill]` |
| `cmd/unlink.go` | `skraft unlink [skill]` |
| `cmd/status.go` | `skraft status` |
| `cmd/pack.go` | `skraft pack [skill]` |
| `cmd/mark_uploaded.go` | `skraft mark-uploaded <skill> [--as version]` |
| `cmd/sync.go` | `skraft sync --check / --fix` |
| `cmd/config.go` | `skraft config get/set` |
| `internal/git/git.go` | `git describe`, repo root detection, HEAD info |
| `internal/git/git_test.go` | Tests using temp git repos |
| `internal/skill/skill.go` | Skill discovery, `SKILL.md` frontmatter parsing |
| `internal/skill/skill_test.go` | Tests with temp directories |
| `internal/config/config.go` | `.skraft/config.toml` read/write |
| `internal/config/config_test.go` | Tests with temp directories |
| `internal/ledger/ledger.go` | SQLite open, migrate, `upload_state` CRUD |
| `internal/ledger/ledger_test.go` | Tests with `:memory:` SQLite |
| `internal/ledger/migrations/0001_initial.sql` | Schema DDL |

---

## Task 1: Project Scaffold

**Files:**
- Create: `main.go`
- Create: `go.mod`
- Create: `cmd/root.go`

- [ ] **Step 1: Initialize Go module**

```bash
cd /path/to/skraft
go mod init github.com/nyamage/skraft
```

Expected: `go.mod` created with `module github.com/nyamage/skraft` and `go 1.21`

- [ ] **Step 2: Add dependencies**

```bash
go get github.com/spf13/cobra@v1.8.1
go get gopkg.in/yaml.v3@v3.0.1
go get github.com/BurntSushi/toml@v1.3.2
go get modernc.org/sqlite@v1.29.9
go mod tidy
```

- [ ] **Step 3: Create `main.go`**

```go
package main

import "github.com/nyamage/skraft/cmd"

func main() {
	cmd.Execute()
}
```

- [ ] **Step 4: Create `cmd/root.go`**

```go
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "skraft",
	Short: "Your Claude skills, version-controlled.",
	Long:  "skraft manages Claude Agent Skills across Git, Claude Code, and Claude.ai.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// findRepoRoot returns the git repository root from the current working directory.
func findRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not inside a git repository")
	}
	return strings.TrimSpace(string(out)), nil
}
```

- [ ] **Step 5: Build to verify it compiles**

```bash
go build ./...
```

Expected: no output (success)

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum main.go cmd/root.go
git commit -m "feat: scaffold project with cobra root command"
```

---

## Task 2: `internal/git` — Version and Repo Info

**Files:**
- Create: `internal/git/git.go`
- Create: `internal/git/git_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/git/git_test.go`:

```go
package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/nyamage/skraft/internal/git"
)

// makeTempRepo creates a temp git repo with one commit. Returns the repo dir.
func makeTempRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		c := exec.Command(args[0], args[1:]...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s", args, out)
		}
	}
	run("git", "init")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", ".")
	run("git", "commit", "-m", "initial")
	return dir
}

func TestRepoRoot(t *testing.T) {
	dir := makeTempRepo(t)
	sub := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	got, err := git.RepoRoot(sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != dir {
		t.Errorf("RepoRoot(%q) = %q, want %q", sub, got, dir)
	}
}

func TestLatestTag_NoTags(t *testing.T) {
	dir := makeTempRepo(t)
	tag, err := git.LatestTag(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag != "" {
		t.Errorf("expected empty tag, got %q", tag)
	}
}

func TestLatestTag_WithTag(t *testing.T) {
	dir := makeTempRepo(t)
	run := func(args ...string) {
		c := exec.Command(args[0], args[1:]...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s", args, out)
		}
	}
	run("git", "tag", "v1.2.3")
	tag, err := git.LatestTag(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag != "v1.2.3" {
		t.Errorf("LatestTag = %q, want v1.2.3", tag)
	}
}

func TestVersion_NoTags(t *testing.T) {
	dir := makeTempRepo(t)
	v, err := git.Version(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should start with "untagged-"
	if len(v) < 9 || v[:9] != "untagged-" {
		t.Errorf("Version with no tags = %q, want prefix 'untagged-'", v)
	}
}

func TestVersion_AtTag(t *testing.T) {
	dir := makeTempRepo(t)
	run := func(args ...string) {
		c := exec.Command(args[0], args[1:]...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s", args, out)
		}
	}
	run("git", "tag", "v2.0.0")
	v, err := git.Version(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "v2.0.0" {
		t.Errorf("Version at tag = %q, want v2.0.0", v)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/git/... 2>&1 | head -5
```

Expected: compile error — package `github.com/nyamage/skraft/internal/git` does not exist

- [ ] **Step 3: Implement `internal/git/git.go`**

```go
package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// RepoRoot returns the git repository root detected from dir.
func RepoRoot(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// LatestTag returns the most recent semver tag reachable from HEAD,
// or empty string if no tags exist.
func LatestTag(repoRoot string) (string, error) {
	out, err := exec.Command("git", "-C", repoRoot, "describe", "--tags", "--abbrev=0").Output()
	if err != nil {
		// No tags — not an error for callers
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
}

// ShortSHA returns the short SHA of HEAD prefixed with "untagged-".
func ShortSHA(repoRoot string) (string, error) {
	out, err := exec.Command("git", "-C", repoRoot, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --short HEAD: %w", err)
	}
	return "untagged-" + strings.TrimSpace(string(out)), nil
}

// Version returns the version string for the repository.
// At a tag: "v1.2.0". Ahead of a tag: "v1.2.0-3-gabcdef".
// No tags: "untagged-abcdef".
func Version(repoRoot string) (string, error) {
	out, err := exec.Command("git", "-C", repoRoot, "describe", "--tags", "--always").Output()
	if err != nil {
		// --always falls back to SHA; if that also fails, use ShortSHA
		return ShortSHA(repoRoot)
	}
	v := strings.TrimSpace(string(out))
	// If the result looks like a bare SHA (no v prefix, no hyphen-number), it has no tags
	if !strings.HasPrefix(v, "v") && !strings.Contains(v, "-") {
		return "untagged-" + v, nil
	}
	return v, nil
}

// HeadSHA returns the full SHA of HEAD.
func HeadSHA(repoRoot string) (string, error) {
	out, err := exec.Command("git", "-C", repoRoot, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/git/... -v
```

Expected: all 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/git/
git commit -m "feat: add internal/git package for version detection"
```

---

## Task 3: `internal/skill` — Discovery and Frontmatter Parsing

**Files:**
- Create: `internal/skill/skill.go`
- Create: `internal/skill/skill_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/skill/skill_test.go`:

```go
package skill_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nyamage/skraft/internal/skill"
)

func TestParseFrontmatter_Basic(t *testing.T) {
	dir := t.TempDir()
	content := "---\nname: my-skill\ndescription: Does stuff\nlicense: MIT\n---\n\n# Body\n"
	path := filepath.Join(dir, "SKILL.md")
	os.WriteFile(path, []byte(content), 0644)

	fm, err := skill.ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.Name != "my-skill" {
		t.Errorf("Name = %q, want my-skill", fm.Name)
	}
	if fm.Description != "Does stuff" {
		t.Errorf("Description = %q, want 'Does stuff'", fm.Description)
	}
	if fm.License != "MIT" {
		t.Errorf("License = %q, want MIT", fm.License)
	}
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	os.WriteFile(path, []byte("# Just a body\n"), 0644)

	fm, err := skill.ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.Name != "" {
		t.Errorf("expected empty name, got %q", fm.Name)
	}
}

func TestDiscover_FindsSkillDirs(t *testing.T) {
	root := t.TempDir()

	// Create skill-a with SKILL.md
	os.MkdirAll(filepath.Join(root, "skill-a"), 0755)
	os.WriteFile(filepath.Join(root, "skill-a", "SKILL.md"),
		[]byte("---\nname: skill-a\ndescription: Alpha\n---\n"), 0644)

	// Create skill-b with SKILL.md
	os.MkdirAll(filepath.Join(root, "skill-b"), 0755)
	os.WriteFile(filepath.Join(root, "skill-b", "SKILL.md"),
		[]byte("---\nname: skill-b\ndescription: Beta\n---\n"), 0644)

	// Create a dir without SKILL.md — should be ignored
	os.MkdirAll(filepath.Join(root, "not-a-skill"), 0755)

	// Create a hidden dir — should be ignored
	os.MkdirAll(filepath.Join(root, ".skraft"), 0755)

	skills, err := skill.Discover(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d: %v", len(skills), skills)
	}
}

func TestDiscover_FallbackDirName(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "my-skill"), 0755)
	// SKILL.md without a name field
	os.WriteFile(filepath.Join(root, "my-skill", "SKILL.md"),
		[]byte("---\ndescription: no name\n---\n"), 0644)

	skills, err := skill.Discover(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Name != "my-skill" {
		t.Errorf("Name = %q, want my-skill (dir fallback)", skills[0].Name)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/skill/... 2>&1 | head -5
```

Expected: compile error — package does not exist

- [ ] **Step 3: Implement `internal/skill/skill.go`**

```go
package skill

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Frontmatter holds the YAML frontmatter fields from SKILL.md.
// skraft reads these but never writes them (ADR 0011).
type Frontmatter struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	License      string   `yaml:"license"`
	AllowedTools []string `yaml:"allowed-tools"`
}

// Skill represents a single skill found in the repository.
type Skill struct {
	Name        string // from frontmatter.Name or directory basename
	Description string
	DirName     string // directory basename (used for symlink naming)
	Dir         string // absolute path to skill directory
	SkillMDPath string // absolute path to SKILL.md
}

// ParseFrontmatter extracts YAML frontmatter from a SKILL.md file.
// Returns zero-value Frontmatter if no frontmatter block is present.
func ParseFrontmatter(path string) (Frontmatter, error) {
	f, err := os.Open(path)
	if err != nil {
		return Frontmatter{}, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var yamlLines []string
	inBlock := false

	for scanner.Scan() {
		line := scanner.Text()
		if !inBlock {
			if line == "---" {
				inBlock = true
			}
			continue
		}
		if line == "---" {
			break
		}
		yamlLines = append(yamlLines, line)
	}
	if err := scanner.Err(); err != nil {
		return Frontmatter{}, err
	}
	if len(yamlLines) == 0 {
		return Frontmatter{}, nil
	}

	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), &fm); err != nil {
		return Frontmatter{}, fmt.Errorf("invalid frontmatter in %s: %w", path, err)
	}
	return fm, nil
}

// Discover finds all skills in repoRoot by scanning for directories that
// contain a SKILL.md file. Hidden directories (prefixed with ".") are skipped.
func Discover(repoRoot string) ([]Skill, error) {
	entries, err := os.ReadDir(repoRoot)
	if err != nil {
		return nil, err
	}

	var skills []Skill
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		skillMDPath := filepath.Join(repoRoot, e.Name(), "SKILL.md")
		if _, err := os.Stat(skillMDPath); os.IsNotExist(err) {
			continue
		}

		fm, err := ParseFrontmatter(skillMDPath)
		if err != nil {
			// Malformed frontmatter: skip with warning, don't abort discovery
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", e.Name(), err)
			continue
		}

		name := fm.Name
		if name == "" {
			name = e.Name()
		}
		skills = append(skills, Skill{
			Name:        name,
			Description: fm.Description,
			DirName:     e.Name(),
			Dir:         filepath.Join(repoRoot, e.Name()),
			SkillMDPath: skillMDPath,
		})
	}
	return skills, nil
}

// Find returns the skill with the given name or directory basename.
// Returns nil if not found.
func Find(skills []Skill, nameOrDir string) *Skill {
	for i := range skills {
		if skills[i].Name == nameOrDir || skills[i].DirName == nameOrDir {
			return &skills[i]
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/skill/... -v
```

Expected: all 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/skill/
git commit -m "feat: add internal/skill package for discovery and frontmatter parsing"
```

---

## Task 4: `internal/config` — Config File

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/config/config_test.go`:

```go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nyamage/skraft/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	dir := t.TempDir()
	// No .skraft/config.toml — should get defaults
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SkillsDir == "" {
		t.Error("SkillsDir should have a default value")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".skraft"), 0755)

	cfg := config.DefaultConfig()
	cfg.SkillsDir = "/custom/skills"

	if err := config.Save(dir, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.SkillsDir != "/custom/skills" {
		t.Errorf("SkillsDir = %q, want /custom/skills", loaded.SkillsDir)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/config/... 2>&1 | head -5
```

Expected: compile error

- [ ] **Step 3: Implement `internal/config/config.go`**

```go
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	SkraftDir  = ".skraft"
	ConfigFile = "config.toml"
	LedgerFile = "ledger.db"
)

// Config holds skraft's persistent settings.
type Config struct {
	SkillsDir string `toml:"skills_dir"` // path to Claude Code skills directory
}

// DefaultConfig returns config with sensible defaults.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		SkillsDir: filepath.Join(home, ".claude", "skills"),
	}
}

// Load reads the config from repoRoot/.skraft/config.toml.
// Returns defaults if the file does not exist.
func Load(repoRoot string) (Config, error) {
	cfg := DefaultConfig()
	path := filepath.Join(repoRoot, SkraftDir, ConfigFile)
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	return cfg, nil
}

// Save writes cfg to repoRoot/.skraft/config.toml.
// The .skraft directory must already exist.
func Save(repoRoot string, cfg Config) error {
	path := filepath.Join(repoRoot, SkraftDir, ConfigFile)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

// SkraftDirPath returns the absolute path to the .skraft directory.
func SkraftDirPath(repoRoot string) string {
	return filepath.Join(repoRoot, SkraftDir)
}

// LedgerPath returns the absolute path to the SQLite ledger.
func LedgerPath(repoRoot string) string {
	return filepath.Join(repoRoot, SkraftDir, LedgerFile)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/config/... -v
```

Expected: both tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add internal/config package for .skraft/config.toml management"
```

---

## Task 5: `internal/ledger` — SQLite Ledger

**Files:**
- Create: `internal/ledger/migrations/0001_initial.sql`
- Create: `internal/ledger/ledger.go`
- Create: `internal/ledger/ledger_test.go`

- [ ] **Step 1: Create migration SQL**

Create `internal/ledger/migrations/0001_initial.sql`:

```sql
CREATE TABLE IF NOT EXISTS metadata (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

INSERT OR IGNORE INTO metadata (key, value) VALUES ('schema_version', '0');

CREATE TABLE IF NOT EXISTS upload_state (
    skill_name   TEXT NOT NULL,
    target       TEXT NOT NULL,  -- 'claudeai' | 'claude_code'
    version      TEXT NOT NULL,  -- git tag or short SHA
    content_hash TEXT NOT NULL,  -- SHA256 of zip (recorded for future use)
    uploaded_at  TEXT NOT NULL,  -- ISO 8601
    PRIMARY KEY (skill_name, target)
);

CREATE TABLE IF NOT EXISTS events (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    otel_event_id TEXT UNIQUE,
    timestamp     TEXT NOT NULL,
    skill_name    TEXT NOT NULL,
    event_type    TEXT NOT NULL,
    payload       TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_events_skill     ON events(skill_name);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
CREATE INDEX IF NOT EXISTS idx_events_type      ON events(event_type);
```

- [ ] **Step 2: Write failing tests**

Create `internal/ledger/ledger_test.go`:

```go
package ledger_test

import (
	"testing"
	"time"

	"github.com/nyamage/skraft/internal/ledger"
)

func openMemory(t *testing.T) *ledger.Ledger {
	t.Helper()
	l, err := ledger.Open(":memory:")
	if err != nil {
		t.Fatalf("ledger.Open: %v", err)
	}
	t.Cleanup(func() { l.Close() })
	return l
}

func TestOpen_CreatesSchema(t *testing.T) {
	l := openMemory(t)
	// If schema creation failed, Open would have returned an error.
	_ = l
}

func TestSetAndGetUploadState(t *testing.T) {
	l := openMemory(t)

	state := ledger.UploadState{
		SkillName:   "skill-a",
		Target:      "claudeai",
		Version:     "v1.2.0",
		ContentHash: "abc123",
		UploadedAt:  time.Now().UTC().Truncate(time.Second),
	}
	if err := l.SetUploadState(state); err != nil {
		t.Fatalf("SetUploadState: %v", err)
	}

	got, err := l.GetUploadState("skill-a", "claudeai")
	if err != nil {
		t.Fatalf("GetUploadState: %v", err)
	}
	if got == nil {
		t.Fatal("expected state, got nil")
	}
	if got.Version != "v1.2.0" {
		t.Errorf("Version = %q, want v1.2.0", got.Version)
	}
	if got.ContentHash != "abc123" {
		t.Errorf("ContentHash = %q, want abc123", got.ContentHash)
	}
}

func TestGetUploadState_NotFound(t *testing.T) {
	l := openMemory(t)
	got, err := l.GetUploadState("nonexistent", "claudeai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestSetUploadState_Upsert(t *testing.T) {
	l := openMemory(t)

	first := ledger.UploadState{SkillName: "skill-a", Target: "claudeai", Version: "v1.0.0", ContentHash: "hash1", UploadedAt: time.Now().UTC()}
	second := ledger.UploadState{SkillName: "skill-a", Target: "claudeai", Version: "v1.1.0", ContentHash: "hash2", UploadedAt: time.Now().UTC()}

	if err := l.SetUploadState(first); err != nil {
		t.Fatal(err)
	}
	if err := l.SetUploadState(second); err != nil {
		t.Fatal(err)
	}

	got, _ := l.GetUploadState("skill-a", "claudeai")
	if got.Version != "v1.1.0" {
		t.Errorf("after upsert: Version = %q, want v1.1.0", got.Version)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/ledger/... 2>&1 | head -5
```

Expected: compile error

- [ ] **Step 4: Implement `internal/ledger/ledger.go`**

```go
package ledger

import (
	"database/sql"
	"embed"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Ledger wraps the SQLite database.
type Ledger struct {
	db *sql.DB
}

// UploadState records the last upload to a target environment.
type UploadState struct {
	SkillName   string
	Target      string // "claudeai" | "claude_code"
	Version     string
	ContentHash string
	UploadedAt  time.Time
}

// Open opens (or creates) the SQLite ledger at path and runs pending migrations.
// Use ":memory:" for tests.
func Open(path string) (*Ledger, error) {
	dsn := path
	if path != ":memory:" {
		dsn = "file:" + path
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open ledger %s: %w", path, err)
	}
	// Enable WAL for concurrent access safety (no-op for :memory:)
	if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	l := &Ledger{db: db}
	if err := l.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return l, nil
}

// Close closes the database connection.
func (l *Ledger) Close() error {
	return l.db.Close()
}

// GetUploadState returns the last recorded upload state for a skill+target pair.
// Returns nil (not an error) if no record exists.
func (l *Ledger) GetUploadState(skillName, target string) (*UploadState, error) {
	row := l.db.QueryRow(`
		SELECT skill_name, target, version, content_hash, uploaded_at
		FROM upload_state
		WHERE skill_name = ? AND target = ?`, skillName, target)

	var s UploadState
	var uploadedAt string
	err := row.Scan(&s.SkillName, &s.Target, &s.Version, &s.ContentHash, &uploadedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.UploadedAt, err = time.Parse(time.RFC3339, uploadedAt)
	if err != nil {
		return nil, fmt.Errorf("parse uploaded_at: %w", err)
	}
	return &s, nil
}

// SetUploadState inserts or replaces the upload state for a skill+target pair.
func (l *Ledger) SetUploadState(state UploadState) error {
	_, err := l.db.Exec(`
		INSERT OR REPLACE INTO upload_state (skill_name, target, version, content_hash, uploaded_at)
		VALUES (?, ?, ?, ?, ?)`,
		state.SkillName,
		state.Target,
		state.Version,
		state.ContentHash,
		state.UploadedAt.UTC().Format(time.RFC3339),
	)
	return err
}

// migrate runs any SQL migration files not yet applied.
func (l *Ledger) migrate() error {
	// Ensure metadata table exists (bootstrap for schema_version)
	if _, err := l.db.Exec(`CREATE TABLE IF NOT EXISTS metadata (key TEXT PRIMARY KEY, value TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("create metadata table: %w", err)
	}
	if _, err := l.db.Exec(`INSERT OR IGNORE INTO metadata (key, value) VALUES ('schema_version', '0')`); err != nil {
		return fmt.Errorf("init schema_version: %w", err)
	}

	var currentVersion int
	if err := l.db.QueryRow(`SELECT CAST(value AS INTEGER) FROM metadata WHERE key = 'schema_version'`).Scan(&currentVersion); err != nil {
		return fmt.Errorf("read schema_version: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	for i, entry := range entries {
		migrationNum := i + 1
		if migrationNum <= currentVersion {
			continue
		}
		sql, err := migrationsFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		if _, err := l.db.Exec(string(sql)); err != nil {
			return fmt.Errorf("run migration %s: %w", entry.Name(), err)
		}
		if _, err := l.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'schema_version'`, migrationNum); err != nil {
			return fmt.Errorf("update schema_version: %w", err)
		}
	}
	return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/ledger/... -v
```

Expected: all 4 tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/ledger/
git commit -m "feat: add internal/ledger package with SQLite WAL and migrations"
```

---

## Task 6: `skraft init`

**Files:**
- Create: `cmd/init.go`

- [ ] **Step 1: Write integration test**

Add to `internal/config/config_test.go`:

```go
func TestSkraftDirPath(t *testing.T) {
	dir := "/some/repo"
	want := "/some/repo/.skraft"
	if got := config.SkraftDirPath(dir); got != want {
		t.Errorf("SkraftDirPath = %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run test to verify it passes**

```bash
go test ./internal/config/... -run TestSkraftDirPath -v
```

Expected: PASS

- [ ] **Step 3: Implement `cmd/init.go`**

```go
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nyamage/skraft/internal/config"
	"github.com/nyamage/skraft/internal/ledger"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize skraft in the current git repository",
	Long: `Creates .skraft/ with config.toml and ledger.db.
Safe to run multiple times — does not overwrite existing config.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := findRepoRoot()
		if err != nil {
			return err
		}

		skraftDir := config.SkraftDirPath(repoRoot)
		if err := os.MkdirAll(skraftDir, 0755); err != nil {
			return fmt.Errorf("create .skraft/: %w", err)
		}

		// Write default config only if it doesn't exist
		cfgPath := filepath.Join(skraftDir, config.ConfigFile)
		if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
			if err := config.Save(repoRoot, config.DefaultConfig()); err != nil {
				return fmt.Errorf("write config: %w", err)
			}
			fmt.Println("created .skraft/config.toml")
		} else {
			fmt.Println(".skraft/config.toml already exists, skipping")
		}

		// Open (and migrate) the ledger
		l, err := ledger.Open(config.LedgerPath(repoRoot))
		if err != nil {
			return fmt.Errorf("initialize ledger: %w", err)
		}
		l.Close()
		fmt.Println("initialized .skraft/ledger.db")

		// Update .gitignore
		if err := ensureGitignore(repoRoot); err != nil {
			return fmt.Errorf("update .gitignore: %w", err)
		}

		fmt.Printf("\nskraft initialized in %s\n", repoRoot)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

// ensureGitignore appends skraft ledger entries to .gitignore if not present.
func ensureGitignore(repoRoot string) error {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	entries := []string{".skraft/ledger.db", ".skraft/ledger.db-shm", ".skraft/ledger.db-wal"}

	// Read existing content
	existing := map[string]bool{}
	if f, err := os.Open(gitignorePath); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			existing[strings.TrimSpace(scanner.Text())] = true
		}
		f.Close()
	}

	var toAdd []string
	for _, e := range entries {
		if !existing[e] {
			toAdd = append(toAdd, e)
		}
	}
	if len(toAdd) == 0 {
		return nil
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, "\n# skraft")
	for _, e := range toAdd {
		fmt.Fprintln(f, e)
	}
	fmt.Printf("updated .gitignore with %d entries\n", len(toAdd))
	return nil
}
```

- [ ] **Step 4: Build and smoke-test**

```bash
go build -o /tmp/skraft .
cd /tmp && mkdir test-repo && cd test-repo && git init && git commit --allow-empty -m "init"
/tmp/skraft init
```

Expected output:
```
created .skraft/config.toml
initialized .skraft/ledger.db
updated .gitignore with 3 entries

skraft initialized in /tmp/test-repo
```

- [ ] **Step 5: Verify idempotency**

```bash
/tmp/skraft init
```

Expected: `.skraft/config.toml already exists, skipping` (no errors)

- [ ] **Step 6: Commit**

```bash
cd /path/to/skraft
git add cmd/init.go
git commit -m "feat: add skraft init command"
```

---

## Task 7: `skraft link` and `skraft unlink`

**Files:**
- Create: `cmd/link.go`
- Create: `cmd/unlink.go`

- [ ] **Step 1: Write tests for link logic**

Add `internal/skill/link_test.go`:

```go
package skill_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nyamage/skraft/internal/skill"
)

func TestLinkPath(t *testing.T) {
	s := skill.Skill{DirName: "my-skill"}
	got := skill.LinkPath("/target/skills", s)
	want := "/target/skills/my-skill"
	if got != want {
		t.Errorf("LinkPath = %q, want %q", got, want)
	}
}

func TestIsLinked(t *testing.T) {
	skillDir := t.TempDir()
	skillsDir := t.TempDir()
	s := skill.Skill{DirName: "test-skill", Dir: skillDir}

	// Before linking
	if skill.IsLinked(skillsDir, s) {
		t.Error("expected not linked before symlink creation")
	}

	// Create symlink
	os.Symlink(skillDir, filepath.Join(skillsDir, "test-skill"))

	if !skill.IsLinked(skillsDir, s) {
		t.Error("expected linked after symlink creation")
	}
}
```

- [ ] **Step 2: Add link helpers to `internal/skill/skill.go`**

Append to `internal/skill/skill.go`:

```go
// LinkPath returns the path where a skill's symlink should be placed.
func LinkPath(skillsDir string, s Skill) string {
	return filepath.Join(skillsDir, s.DirName)
}

// IsLinked reports whether the skill is symlinked in skillsDir.
func IsLinked(skillsDir string, s Skill) bool {
	info, err := os.Lstat(LinkPath(skillsDir, s))
	return err == nil && info.Mode()&os.ModeSymlink != 0
}
```

- [ ] **Step 3: Run tests to verify they pass**

```bash
go test ./internal/skill/... -v
```

Expected: all tests PASS (including new link tests)

- [ ] **Step 4: Implement `cmd/link.go`**

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/nyamage/skraft/internal/config"
	"github.com/nyamage/skraft/internal/skill"
	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link [skill]",
	Short: "Symlink skills into Claude Code's skills directory",
	Long: `Creates symlinks from ~/.claude/skills/<skill> to each skill directory.
With no argument, links all discovered skills.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := findRepoRoot()
		if err != nil {
			return err
		}
		cfg, err := config.Load(repoRoot)
		if err != nil {
			return err
		}
		skills, err := skill.Discover(repoRoot)
		if err != nil {
			return err
		}

		// Filter to specified skill if provided
		if len(args) == 1 {
			s := skill.Find(skills, args[0])
			if s == nil {
				return fmt.Errorf("skill %q not found", args[0])
			}
			skills = []skill.Skill{*s}
		}

		if err := os.MkdirAll(cfg.SkillsDir, 0755); err != nil {
			return fmt.Errorf("create skills dir: %w", err)
		}

		for _, s := range skills {
			linkPath := skill.LinkPath(cfg.SkillsDir, s)
			// Remove stale symlink
			if info, err := os.Lstat(linkPath); err == nil {
				if info.Mode()&os.ModeSymlink == 0 {
					return fmt.Errorf("%s exists and is not a symlink; remove it manually", linkPath)
				}
				os.Remove(linkPath)
			}
			if err := os.Symlink(s.Dir, linkPath); err != nil {
				return fmt.Errorf("symlink %s: %w", s.Name, err)
			}
			fmt.Printf("linked  %s → %s\n", s.Name, linkPath)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(linkCmd)
}
```

- [ ] **Step 5: Implement `cmd/unlink.go`**

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/nyamage/skraft/internal/config"
	"github.com/nyamage/skraft/internal/skill"
	"github.com/spf13/cobra"
)

var unlinkCmd = &cobra.Command{
	Use:   "unlink [skill]",
	Short: "Remove skill symlinks from Claude Code's skills directory",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := findRepoRoot()
		if err != nil {
			return err
		}
		cfg, err := config.Load(repoRoot)
		if err != nil {
			return err
		}
		skills, err := skill.Discover(repoRoot)
		if err != nil {
			return err
		}

		if len(args) == 1 {
			s := skill.Find(skills, args[0])
			if s == nil {
				return fmt.Errorf("skill %q not found", args[0])
			}
			skills = []skill.Skill{*s}
		}

		for _, s := range skills {
			linkPath := skill.LinkPath(cfg.SkillsDir, s)
			info, err := os.Lstat(linkPath)
			if os.IsNotExist(err) {
				fmt.Printf("skipped %s (not linked)\n", s.Name)
				continue
			}
			if info.Mode()&os.ModeSymlink == 0 {
				fmt.Printf("skipped %s (%s is not a symlink)\n", s.Name, linkPath)
				continue
			}
			if err := os.Remove(linkPath); err != nil {
				return fmt.Errorf("remove %s: %w", linkPath, err)
			}
			fmt.Printf("unlinked %s\n", s.Name)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(unlinkCmd)
}
```

- [ ] **Step 6: Build and verify**

```bash
go build ./...
```

Expected: no errors

- [ ] **Step 7: Commit**

```bash
git add cmd/link.go cmd/unlink.go internal/skill/skill.go internal/skill/link_test.go
git commit -m "feat: add skraft link and skraft unlink commands"
```

---

## Task 8: `skraft status`

**Files:**
- Create: `cmd/status.go`

- [ ] **Step 1: Implement `cmd/status.go`**

```go
package cmd

import (
	"fmt"
	"strings"

	"github.com/nyamage/skraft/internal/config"
	"github.com/nyamage/skraft/internal/git"
	"github.com/nyamage/skraft/internal/ledger"
	"github.com/nyamage/skraft/internal/skill"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync status of all skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := findRepoRoot()
		if err != nil {
			return err
		}

		version, err := git.Version(repoRoot)
		if err != nil {
			return err
		}
		latestTag, err := git.LatestTag(repoRoot)
		if err != nil {
			return err
		}
		headSHA, err := git.HeadSHA(repoRoot)
		if err != nil {
			return err
		}

		cfg, err := config.Load(repoRoot)
		if err != nil {
			return err
		}
		skills, err := skill.Discover(repoRoot)
		if err != nil {
			return err
		}

		l, err := ledger.Open(config.LedgerPath(repoRoot))
		if err != nil {
			return fmt.Errorf("open ledger (run 'skraft init' first): %w", err)
		}
		defer l.Close()

		// Header
		fmt.Printf("Repository: %s\n", repoRoot)
		if latestTag != "" {
			fmt.Printf("Latest tag: %s\n", latestTag)
		} else {
			fmt.Printf("Latest tag: (none)\n")
		}
		fmt.Printf("HEAD:       %s\n", headSHA[:7])
		fmt.Printf("Version:    %s\n\n", version)

		if len(skills) == 0 {
			fmt.Println("No skills found (add directories with SKILL.md)")
			return nil
		}

		fmt.Println("Skills:")
		for _, s := range skills {
			var parts []string

			// Claude Code status
			if skill.IsLinked(cfg.SkillsDir, s) {
				parts = append(parts, "Claude Code: linked")
			} else {
				parts = append(parts, "Claude Code: NOT linked ⚠")
			}

			// Claude.ai status
			state, err := l.GetUploadState(s.Name, "claudeai")
			if err != nil {
				return err
			}
			if state == nil {
				parts = append(parts, "Claude.ai: never uploaded ⚠")
			} else if latestTag != "" && state.Version != latestTag {
				parts = append(parts, fmt.Sprintf("Claude.ai: %s ⚠ (current: %s)", state.Version, latestTag))
			} else {
				parts = append(parts, fmt.Sprintf("Claude.ai: %s ✓", state.Version))
			}

			fmt.Printf("  %-20s %s\n", s.Name, strings.Join(parts, " | "))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
```

- [ ] **Step 2: Build and verify**

```bash
go build ./...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add cmd/status.go
git commit -m "feat: add skraft status command"
```

---

## Task 9: `skraft pack`

**Files:**
- Create: `cmd/pack.go`

- [ ] **Step 1: Write pack helper test**

Add `internal/skill/pack_test.go`:

```go
package skill_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/nyamage/skraft/internal/skill"
)

func TestPackSkill(t *testing.T) {
	// Create a fake skill dir
	skillDir := t.TempDir()
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: test\n---\n"), 0644)
	os.MkdirAll(filepath.Join(skillDir, "scripts"), 0755)
	os.WriteFile(filepath.Join(skillDir, "scripts", "run.sh"), []byte("#!/bin/sh\n"), 0755)
	// These should be excluded
	os.WriteFile(filepath.Join(skillDir, ".DS_Store"), []byte(""), 0644)

	outDir := t.TempDir()
	zipPath := filepath.Join(outDir, "test-v1.0.0.zip")

	s := skill.Skill{Name: "test", DirName: "test", Dir: skillDir}
	if err := skill.Pack(s, zipPath); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	// Verify zip contents
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer r.Close()

	names := map[string]bool{}
	for _, f := range r.File {
		names[f.Name] = true
	}
	if !names["SKILL.md"] {
		t.Error("zip missing SKILL.md")
	}
	if !names["scripts/run.sh"] {
		t.Error("zip missing scripts/run.sh")
	}
	if names[".DS_Store"] {
		t.Error("zip should not contain .DS_Store")
	}
}
```

- [ ] **Step 2: Add `Pack` function to `internal/skill/skill.go`**

Append to `internal/skill/skill.go`:

```go
// excludedNames are file/directory names excluded from pack zips.
var excludedNames = map[string]bool{
	".git":         true,
	".DS_Store":    true,
	"node_modules": true,
	".gitignore":   true,
	"dist":         true,
}

// Pack creates a zip of the skill directory at destPath.
// Excludes .git, .DS_Store, node_modules, and similar non-distributable files.
func Pack(s Skill, destPath string) error {
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	return filepath.WalkDir(s.Dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if excludedNames[d.Name()] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(s.Dir, path)
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// Use forward slashes in zip entries
		zipName := filepath.ToSlash(rel)
		fw, err := w.Create(zipName)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = fw.Write(data)
		return err
	})
}
```

Also add the necessary import `"io/fs"` and `"archive/zip"` to `internal/skill/skill.go`:

Update the imports block in `internal/skill/skill.go`:
```go
import (
	"archive/zip"
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)
```

- [ ] **Step 3: Run tests to verify they pass**

```bash
go test ./internal/skill/... -v
```

Expected: all tests PASS

- [ ] **Step 4: Implement `cmd/pack.go`**

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nyamage/skraft/internal/config"
	"github.com/nyamage/skraft/internal/git"
	"github.com/nyamage/skraft/internal/skill"
	"github.com/spf13/cobra"
)

var packCmd = &cobra.Command{
	Use:   "pack [skill]",
	Short: "Generate Claude.ai upload zip(s) in dist/",
	Long: `Creates dist/<skill>-<version>.zip for each skill.
With no argument, packs all discovered skills.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := findRepoRoot()
		if err != nil {
			return err
		}
		version, err := git.Version(repoRoot)
		if err != nil {
			return err
		}
		skills, err := skill.Discover(repoRoot)
		if err != nil {
			return err
		}

		if len(args) == 1 {
			s := skill.Find(skills, args[0])
			if s == nil {
				return fmt.Errorf("skill %q not found", args[0])
			}
			skills = []skill.Skill{*s}
		}

		distDir := filepath.Join(repoRoot, "dist")
		if err := os.MkdirAll(distDir, 0755); err != nil {
			return fmt.Errorf("create dist/: %w", err)
		}

		for _, s := range skills {
			zipName := fmt.Sprintf("%s-%s.zip", s.DirName, version)
			zipPath := filepath.Join(distDir, zipName)
			if err := skill.Pack(s, zipPath); err != nil {
				return fmt.Errorf("pack %s: %w", s.Name, err)
			}
			fmt.Printf("packed  %s → dist/%s\n", s.Name, zipName)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(packCmd)
}
```

- [ ] **Step 5: Build and verify**

```bash
go build ./...
```

Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add cmd/pack.go internal/skill/skill.go internal/skill/pack_test.go
git commit -m "feat: add skraft pack command with zip generation"
```

---

## Task 10: `skraft mark-uploaded`

**Files:**
- Create: `cmd/mark_uploaded.go`

- [ ] **Step 1: Implement `cmd/mark_uploaded.go`**

```go
package cmd

import (
	"fmt"
	"time"

	"github.com/nyamage/skraft/internal/config"
	"github.com/nyamage/skraft/internal/git"
	"github.com/nyamage/skraft/internal/ledger"
	"github.com/nyamage/skraft/internal/skill"
	"github.com/spf13/cobra"
)

var markUploadedAs string

var markUploadedCmd = &cobra.Command{
	Use:   "mark-uploaded <skill>",
	Short: "Record that a skill has been uploaded to Claude.ai",
	Long: `Records the current git version as uploaded for the given skill.
Use --as to override the version (for emergency use only).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := findRepoRoot()
		if err != nil {
			return err
		}

		skills, err := skill.Discover(repoRoot)
		if err != nil {
			return err
		}
		s := skill.Find(skills, args[0])
		if s == nil {
			return fmt.Errorf("skill %q not found", args[0])
		}

		version := markUploadedAs
		if version == "" {
			version, err = git.Version(repoRoot)
			if err != nil {
				return err
			}
		}

		l, err := ledger.Open(config.LedgerPath(repoRoot))
		if err != nil {
			return fmt.Errorf("open ledger (run 'skraft init' first): %w", err)
		}
		defer l.Close()

		state := ledger.UploadState{
			SkillName:   s.Name,
			Target:      "claudeai",
			Version:     version,
			ContentHash: "", // populated in future when pack hash tracking is added
			UploadedAt:  time.Now().UTC(),
		}
		if err := l.SetUploadState(state); err != nil {
			return fmt.Errorf("record upload state: %w", err)
		}

		fmt.Printf("recorded: %s uploaded to Claude.ai at %s\n", s.Name, version)
		return nil
	},
}

func init() {
	markUploadedCmd.Flags().StringVar(&markUploadedAs, "as", "", "override version string (default: current git describe)")
	rootCmd.AddCommand(markUploadedCmd)
}
```

- [ ] **Step 2: Build and verify**

```bash
go build ./...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add cmd/mark_uploaded.go
git commit -m "feat: add skraft mark-uploaded command"
```

---

## Task 11: `skraft sync`

**Files:**
- Create: `cmd/sync.go`

- [ ] **Step 1: Implement `cmd/sync.go`**

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nyamage/skraft/internal/config"
	"github.com/nyamage/skraft/internal/git"
	"github.com/nyamage/skraft/internal/ledger"
	"github.com/nyamage/skraft/internal/skill"
	"github.com/spf13/cobra"
)

var (
	syncCheck bool
	syncFix   bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Check or fix drift between Git, Claude Code, and Claude.ai",
	Long: `--check reports skills that are out of sync.
--fix automatically re-links Claude Code skills and prints instructions for Claude.ai.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !syncCheck && !syncFix {
			return fmt.Errorf("specify --check or --fix")
		}

		repoRoot, err := findRepoRoot()
		if err != nil {
			return err
		}
		version, err := git.Version(repoRoot)
		if err != nil {
			return err
		}
		latestTag, err := git.LatestTag(repoRoot)
		if err != nil {
			return err
		}
		cfg, err := config.Load(repoRoot)
		if err != nil {
			return err
		}
		skills, err := skill.Discover(repoRoot)
		if err != nil {
			return err
		}
		l, err := ledger.Open(config.LedgerPath(repoRoot))
		if err != nil {
			return fmt.Errorf("open ledger (run 'skraft init' first): %w", err)
		}
		defer l.Close()

		type drift struct {
			skill       skill.Skill
			claudeCode  bool // needs re-link
			claudeAI    bool // needs re-upload
			uploadedVer string
		}
		var drifts []drift

		fmt.Println("Claude Code:")
		for _, s := range skills {
			linked := skill.IsLinked(cfg.SkillsDir, s)
			if linked {
				fmt.Printf("  %-20s ✓ linked\n", s.Name)
			} else {
				fmt.Printf("  %-20s ✗ NOT linked\n", s.Name)
			}
			d := drift{skill: s, claudeCode: !linked}

			state, err := l.GetUploadState(s.Name, "claudeai")
			if err != nil {
				return err
			}
			if state == nil {
				d.claudeAI = true
				d.uploadedVer = ""
			} else if latestTag != "" && state.Version != latestTag {
				d.claudeAI = true
				d.uploadedVer = state.Version
			}
			drifts = append(drifts, d)
		}

		fmt.Println("\nClaude.ai:")
		for _, d := range drifts {
			if d.uploadedVer == "" && d.claudeAI {
				fmt.Printf("  %-20s ✗ never uploaded\n", d.skill.Name)
			} else if d.claudeAI {
				fmt.Printf("  %-20s ✗ outdated (uploaded: %s, current: %s)\n", d.skill.Name, d.uploadedVer, latestTag)
			} else {
				state, _ := l.GetUploadState(d.skill.Name, "claudeai")
				v := ""
				if state != nil {
					v = state.Version
				}
				fmt.Printf("  %-20s ✓ %s\n", d.skill.Name, v)
			}
		}

		if !syncFix {
			return nil
		}

		// Fix: re-link Claude Code
		var needsUpload []drift
		relinked := 0
		fmt.Println("\nFixing Claude Code...")
		if err := os.MkdirAll(cfg.SkillsDir, 0755); err != nil {
			return err
		}
		for _, d := range drifts {
			if d.claudeCode {
				linkPath := skill.LinkPath(cfg.SkillsDir, d.skill)
				if info, err := os.Lstat(linkPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
					os.Remove(linkPath)
				}
				if err := os.Symlink(d.skill.Dir, linkPath); err != nil {
					return fmt.Errorf("symlink %s: %w", d.skill.Name, err)
				}
				fmt.Printf("  %-20s re-linked\n", d.skill.Name)
				relinked++
			}
			if d.claudeAI {
				needsUpload = append(needsUpload, d)
			}
		}
		if relinked == 0 {
			fmt.Println("  (nothing to fix)")
		}

		if len(needsUpload) > 0 {
			fmt.Println("\nClaude.ai: manual action required.")
			for _, d := range needsUpload {
				zipName := fmt.Sprintf("%s-%s.zip", d.skill.DirName, version)
				zipPath := filepath.Join(repoRoot, "dist", zipName)
				fmt.Printf("  %s:\n", d.skill.Name)
				fmt.Printf("    1. Run: skraft pack %s\n", d.skill.DirName)
				fmt.Printf("    2. Upload %s to claude.ai skill settings\n", zipPath)
				fmt.Printf("    3. Run: skraft mark-uploaded %s\n", d.skill.DirName)
			}
		}
		return nil
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncCheck, "check", false, "report drift without making changes")
	syncCmd.Flags().BoolVar(&syncFix, "fix", false, "fix Claude Code drift and print Claude.ai instructions")
	rootCmd.AddCommand(syncCmd)
}
```

- [ ] **Step 2: Build and verify**

```bash
go build ./...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add cmd/sync.go
git commit -m "feat: add skraft sync --check/--fix command"
```

---

## Task 12: `skraft config`

**Files:**
- Create: `cmd/config.go`

- [ ] **Step 1: Implement `cmd/config.go`**

```go
package cmd

import (
	"fmt"

	skraftconfig "github.com/nyamage/skraft/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Read or write skraft configuration",
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Print a config value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := findRepoRoot()
		if err != nil {
			return err
		}
		cfg, err := skraftconfig.Load(repoRoot)
		if err != nil {
			return err
		}
		switch args[0] {
		case "skills_dir":
			fmt.Println(cfg.SkillsDir)
		default:
			return fmt.Errorf("unknown key %q (known keys: skills_dir)", args[0])
		}
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := findRepoRoot()
		if err != nil {
			return err
		}
		cfg, err := skraftconfig.Load(repoRoot)
		if err != nil {
			return err
		}
		switch args[0] {
		case "skills_dir":
			cfg.SkillsDir = args[1]
		default:
			return fmt.Errorf("unknown key %q (known keys: skills_dir)", args[0])
		}
		if err := skraftconfig.Save(repoRoot, cfg); err != nil {
			return err
		}
		fmt.Printf("set %s = %s\n", args[0], args[1])
		return nil
	},
}

func init() {
	configCmd.AddCommand(configGetCmd, configSetCmd)
	rootCmd.AddCommand(configCmd)
}
```

- [ ] **Step 2: Build and verify**

```bash
go build ./...
```

Expected: no errors

- [ ] **Step 3: Run all tests**

```bash
go test ./...
```

Expected: all tests PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/config.go
git commit -m "feat: add skraft config get/set command"
```

---

## Task 13: End-to-End Smoke Test

**Files:** none created

- [ ] **Step 1: Build final binary**

```bash
go build -o /tmp/skraft .
```

- [ ] **Step 2: Create a test skill repo and run full workflow**

```bash
# Setup
mkdir -p /tmp/e2e-test
cd /tmp/e2e-test
git init
git config user.email "test@test.com"
git config user.name "Test"

# Create two skills
mkdir -p skill-hello skill-world
cat > skill-hello/SKILL.md <<'EOF'
---
name: hello
description: Says hello
---
A greeting skill.
EOF
cat > skill-world/SKILL.md <<'EOF'
---
name: world
description: World skill
---
World content.
EOF

git add .
git commit -m "add skills"
git tag v0.1.0
```

- [ ] **Step 3: Run skraft workflow**

```bash
/tmp/skraft init
```
Expected: `.skraft/` created, `.gitignore` updated

```bash
/tmp/skraft status
```
Expected: shows `v0.1.0`, both skills listed as NOT linked, never uploaded

```bash
/tmp/skraft link
```
Expected: `linked hello` and `linked world`

```bash
/tmp/skraft status
```
Expected: both skills show "Claude Code: linked"

```bash
/tmp/skraft pack
```
Expected: `dist/skill-hello-v0.1.0.zip` and `dist/skill-world-v0.1.0.zip` created

```bash
/tmp/skraft mark-uploaded hello
/tmp/skraft mark-uploaded world
```
Expected: `recorded: hello uploaded to Claude.ai at v0.1.0`

```bash
/tmp/skraft status
```
Expected: both skills show `Claude.ai: v0.1.0 ✓`

```bash
/tmp/skraft sync --check
```
Expected: all ✓, no drift

```bash
/tmp/skraft unlink hello
/tmp/skraft sync --check
```
Expected: hello shows ✗ NOT linked

```bash
/tmp/skraft sync --fix
```
Expected: hello re-linked, no Claude.ai drift

```bash
/tmp/skraft config get skills_dir
/tmp/skraft config set skills_dir /tmp/custom-skills
/tmp/skraft config get skills_dir
```
Expected: shows current dir, then `/tmp/custom-skills`

- [ ] **Step 4: Final test run**

```bash
cd /path/to/skraft
go test ./... -v
```

Expected: all tests PASS

- [ ] **Step 5: Final commit**

```bash
git add .
git commit -m "chore: end-to-end verified, MVP complete"
```
