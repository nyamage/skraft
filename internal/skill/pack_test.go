package skill_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nyamage/skraft/internal/skill"
)

func TestPackSkill(t *testing.T) {
	skillDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: test\n---\n"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(skillDir, "scripts"), 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "scripts", "run.sh"), []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, ".DS_Store"), []byte(""), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	outDir := t.TempDir()
	zipPath := filepath.Join(outDir, "test-v1.0.0.zip")

	s := skill.Skill{Name: "test", DirName: "test", Dir: skillDir}
	if err := skill.Pack(s, zipPath); err != nil {
		t.Fatalf("Pack: %v", err)
	}

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

func TestPackSkill_ExcludedNames(t *testing.T) {
	excluded := []string{".git", "node_modules", ".gitignore", "dist"}

	for _, name := range excluded {
		t.Run(name, func(t *testing.T) {
			skillDir := t.TempDir()
			if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: test\n---\n"), 0644); err != nil {
				t.Fatalf("setup: %v", err)
			}

			// Create the excluded item (as a file for simple cases; .git as a dir)
			if name == ".git" || name == "node_modules" || name == "dist" {
				if err := os.MkdirAll(filepath.Join(skillDir, name), 0755); err != nil {
					t.Fatalf("setup: %v", err)
				}
				if err := os.WriteFile(filepath.Join(skillDir, name, "file.txt"), []byte("secret"), 0644); err != nil {
					t.Fatalf("setup: %v", err)
				}
			} else {
				if err := os.WriteFile(filepath.Join(skillDir, name), []byte("secret"), 0644); err != nil {
					t.Fatalf("setup: %v", err)
				}
			}

			zipPath := filepath.Join(t.TempDir(), "test.zip")
			s := skill.Skill{Name: "test", DirName: "test", Dir: skillDir}
			if err := skill.Pack(s, zipPath); err != nil {
				t.Fatalf("Pack: %v", err)
			}

			r, err := zip.OpenReader(zipPath)
			if err != nil {
				t.Fatalf("open zip: %v", err)
			}
			defer r.Close()

			for _, f := range r.File {
				if f.Name == name || len(f.Name) > len(name) && f.Name[:len(name)+1] == name+"/" {
					t.Errorf("zip should not contain %q (found %q)", name, f.Name)
				}
			}
		})
	}
}

func TestPackSkill_ExcludesTestsDir(t *testing.T) {
	dir := t.TempDir()
	s := skill.Skill{DirName: "my-skill", Dir: dir}
	testsDir := filepath.Join(dir, "tests")
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# skill"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testsDir, "test.yaml"), []byte("id: t"), 0644); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(dir, "out.zip")
	if err := skill.Pack(s, dest); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	zr, err := zip.OpenReader(dest)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "tests/") {
			t.Errorf("zip contains %q — tests/ should be excluded", f.Name)
		}
	}
}
