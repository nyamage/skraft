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

func TestSkraftDirPath(t *testing.T) {
	dir := "/some/repo"
	want := "/some/repo/.skraft"
	if got := config.SkraftDirPath(dir); got != want {
		t.Errorf("SkraftDirPath = %q, want %q", got, want)
	}
}
