package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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
