package skill

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

// Frontmatter holds the parsed YAML front matter from a SKILL.md file.
type Frontmatter struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	License      string   `yaml:"license"`
	AllowedTools []string `yaml:"allowed-tools"`
}

// Skill represents a discovered skill with metadata.
type Skill struct {
	Name        string // from frontmatter.Name or directory basename
	Description string
	DirName     string // directory basename (used for symlink naming)
	Dir         string // absolute path to skill directory
	SkillMDPath string // absolute path to SKILL.md
}

// ParseFrontmatter reads a SKILL.md file and extracts YAML between the first
// pair of "---" delimiters. Returns zero-value Frontmatter (not an error) if
// no frontmatter block is found.
func ParseFrontmatter(path string) (Frontmatter, error) {
	f, err := os.Open(path)
	if err != nil {
		return Frontmatter{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	// Expect the very first line to be "---"
	if !scanner.Scan() {
		return Frontmatter{}, nil
	}
	firstLine := scanner.Text()
	if firstLine != "---" {
		return Frontmatter{}, nil
	}

	// Collect lines until the closing "---"
	var yamlLines []string
	closed := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			closed = true
			break
		}
		yamlLines = append(yamlLines, line)
	}
	if err := scanner.Err(); err != nil {
		return Frontmatter{}, fmt.Errorf("read %s: %w", path, err)
	}
	if !closed || len(yamlLines) == 0 {
		return Frontmatter{}, nil
	}

	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), &fm); err != nil {
		return Frontmatter{}, fmt.Errorf("parse frontmatter in %s: %w", path, err)
	}
	return fm, nil
}

// Discover scans repoRoot for subdirectories that contain a SKILL.md file.
// Hidden directories (starting with ".") and directories without SKILL.md are
// skipped. Each valid skill directory is parsed and returned as a Skill.
func Discover(repoRoot string) ([]Skill, error) {
	entries, err := os.ReadDir(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", repoRoot, err)
	}

	var skills []Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		skillMDPath := filepath.Join(repoRoot, name, "SKILL.md")
		if _, err := os.Stat(skillMDPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat %s: %w", skillMDPath, err)
		}

		fm, err := ParseFrontmatter(skillMDPath)
		if err != nil {
			return nil, fmt.Errorf("parse frontmatter for %s: %w", name, err)
		}

		skillName := fm.Name
		if skillName == "" {
			skillName = name
		}

		skills = append(skills, Skill{
			Name:        skillName,
			Description: fm.Description,
			DirName:     name,
			Dir:         filepath.Join(repoRoot, name),
			SkillMDPath: skillMDPath,
		})
	}
	return skills, nil
}

// Find returns the first skill whose Name or DirName matches nameOrDir, or nil
// if no match is found.
func Find(skills []Skill, nameOrDir string) *Skill {
	for i := range skills {
		if skills[i].Name == nameOrDir || skills[i].DirName == nameOrDir {
			return &skills[i]
		}
	}
	return nil
}

// LinkPath returns the path where a symlink for the skill would be created
// inside skillsDir.
func LinkPath(skillsDir string, s Skill) string {
	return filepath.Join(skillsDir, s.DirName)
}

// IsLinked reports whether a symlink for the skill already exists in skillsDir.
func IsLinked(skillsDir string, s Skill) bool {
	info, err := os.Lstat(LinkPath(skillsDir, s))
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

// excludedNames are file/directory names excluded from pack zips.
var excludedNames = map[string]bool{
	".git":         true,
	".DS_Store":    true,
	"node_modules": true,
	".gitignore":   true,
	"dist":         true,
}

// Pack creates a zip archive of the skill directory at destPath.
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

		// Use forward slashes in zip entries (cross-platform)
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
