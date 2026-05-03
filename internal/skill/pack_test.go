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
