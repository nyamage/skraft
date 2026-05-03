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
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Printf("skipped %s (not linked)\n", s.Name)
					continue
				}
				return fmt.Errorf("stat %s: %w", linkPath, err)
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
