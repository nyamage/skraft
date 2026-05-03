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
