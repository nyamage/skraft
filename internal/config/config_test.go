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
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".claude", "skills")
	if cfg.SkillsDir != want {
		t.Errorf("SkillsDir = %q, want %q", cfg.SkillsDir, want)
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

func TestSkraftDirPath(t *testing.T) {
	dir := "/some/repo"
	want := "/some/repo/.skraft"
	if got := config.SkraftDirPath(dir); got != want {
		t.Errorf("SkraftDirPath = %q, want %q", got, want)
	}
}

func TestLedgerPath(t *testing.T) {
	dir := "/some/repo"
	want := "/some/repo/.skraft/ledger.db"
	if got := config.LedgerPath(dir); got != want {
		t.Errorf("LedgerPath = %q, want %q", got, want)
	}
}
