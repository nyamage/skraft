package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nyamage/skraft/internal/config"
	"github.com/nyamage/skraft/internal/skill"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Scaffold a new skill in the repo and link it immediately",
	Long: `Creates <repo-root>/<name>/SKILL.md with minimal frontmatter and
immediately creates a symlink at cfg.SkillsDir/<name> so the skill is
available in Claude Code right away.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		repoRoot, err := findRepoRoot()
		if err != nil {
			return err
		}
		cfg, err := config.Load(repoRoot)
		if err != nil {
			return err
		}

		skillDir := filepath.Join(repoRoot, name)
		skillMDPath := filepath.Join(skillDir, "SKILL.md")

		if _, err := os.Lstat(skillDir); err == nil {
			return fmt.Errorf("directory %s already exists", skillDir)
		}

		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return fmt.Errorf("create skill dir: %w", err)
		}

		stub := fmt.Sprintf("---\nname: %s\ndescription: \"\"\n---\n\n# %s\n\nDescribe what this skill does.\n", name, name)
		if err := os.WriteFile(skillMDPath, []byte(stub), 0644); err != nil {
			return fmt.Errorf("write SKILL.md: %w", err)
		}

		if err := os.MkdirAll(cfg.SkillsDir, 0755); err != nil {
			return fmt.Errorf("create skills dir: %w", err)
		}
		s := skill.Skill{Name: name, DirName: name, Dir: skillDir, SkillMDPath: skillMDPath}
		linkPath := skill.LinkPath(cfg.SkillsDir, s)
		if info, err := os.Lstat(linkPath); err == nil {
			if info.Mode()&os.ModeSymlink == 0 {
				return fmt.Errorf("%s exists and is not a symlink; remove it manually", linkPath)
			}
			if err := os.Remove(linkPath); err != nil {
				return fmt.Errorf("remove stale symlink %s: %w", linkPath, err)
			}
		}
		if err := os.Symlink(skillDir, linkPath); err != nil {
			return fmt.Errorf("symlink %s: %w", name, err)
		}

		fmt.Printf("created  %s\n", skillDir)
		fmt.Printf("linked   %s → %s\n", name, linkPath)
		fmt.Printf("\nEdit %s to complete setup.\n", skillMDPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
}
