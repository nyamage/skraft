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
			// Remove stale symlink if present
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
