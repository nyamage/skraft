package cmd

import (
	"fmt"
	"time"

	"github.com/nyamage/skraft/internal/config"
	"github.com/nyamage/skraft/internal/git"
	"github.com/nyamage/skraft/internal/ledger"
	"github.com/nyamage/skraft/internal/skill"
	"github.com/spf13/cobra"
)

var markUploadedAs string

var markUploadedCmd = &cobra.Command{
	Use:   "mark-uploaded <skill>",
	Short: "Record that a skill has been uploaded to Claude.ai",
	Long: `Records the current git version as uploaded for the given skill.
Use --as to override the version (for emergency use only).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := findRepoRoot()
		if err != nil {
			return err
		}

		skills, err := skill.Discover(repoRoot)
		if err != nil {
			return err
		}
		s := skill.Find(skills, args[0])
		if s == nil {
			return fmt.Errorf("skill %q not found", args[0])
		}

		version := markUploadedAs
		if version == "" {
			version, err = git.Version(repoRoot)
			if err != nil {
				return err
			}
		}

		l, err := ledger.Open(config.LedgerPath(repoRoot))
		if err != nil {
			return fmt.Errorf("open ledger (run 'skraft init' first): %w", err)
		}
		defer l.Close()

		state := ledger.UploadState{
			SkillName:   s.Name,
			Target:      "claudeai",
			Version:     version,
			ContentHash: "", // populated in future when pack hash tracking is added
			UploadedAt:  time.Now().UTC(),
		}
		if err := l.SetUploadState(state); err != nil {
			return fmt.Errorf("record upload state: %w", err)
		}

		fmt.Printf("recorded: %s uploaded to Claude.ai at %s\n", s.Name, version)
		return nil
	},
}

func init() {
	markUploadedCmd.Flags().StringVar(&markUploadedAs, "as", "", "override version string (default: current git describe)")
	rootCmd.AddCommand(markUploadedCmd)
}
