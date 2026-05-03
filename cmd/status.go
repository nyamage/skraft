package cmd

import (
	"fmt"
	"strings"

	"github.com/nyamage/skraft/internal/config"
	"github.com/nyamage/skraft/internal/git"
	"github.com/nyamage/skraft/internal/ledger"
	"github.com/nyamage/skraft/internal/skill"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync status of all skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := findRepoRoot()
		if err != nil {
			return err
		}

		version, err := git.Version(repoRoot)
		if err != nil {
			return err
		}
		latestTag, err := git.LatestTag(repoRoot)
		if err != nil {
			return err
		}
		headSHA, err := git.HeadSHA(repoRoot)
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

		l, err := ledger.Open(config.LedgerPath(repoRoot))
		if err != nil {
			return fmt.Errorf("open ledger (run 'skraft init' first): %w", err)
		}
		defer l.Close()

		// Header
		fmt.Printf("Repository: %s\n", repoRoot)
		if latestTag != "" {
			fmt.Printf("Latest tag: %s\n", latestTag)
		} else {
			fmt.Printf("Latest tag: (none)\n")
		}
		fmt.Printf("HEAD:       %s\n", headSHA[:7])
		fmt.Printf("Version:    %s\n\n", version)

		if len(skills) == 0 {
			fmt.Println("No skills found (add directories with SKILL.md)")
			return nil
		}

		fmt.Println("Skills:")
		for _, s := range skills {
			var parts []string

			// Claude Code status
			if skill.IsLinked(cfg.SkillsDir, s) {
				parts = append(parts, "Claude Code: linked")
			} else {
				parts = append(parts, "Claude Code: NOT linked ⚠")
			}

			// Claude.ai status
			state, err := l.GetUploadState(s.Name, "claudeai")
			if err != nil {
				return err
			}
			if state == nil {
				parts = append(parts, "Claude.ai: never uploaded ⚠")
			} else if latestTag != "" && state.Version != latestTag {
				parts = append(parts, fmt.Sprintf("Claude.ai: %s ⚠ (current: %s)", state.Version, latestTag))
			} else {
				parts = append(parts, fmt.Sprintf("Claude.ai: %s ✓", state.Version))
			}

			fmt.Printf("  %-20s %s\n", s.Name, strings.Join(parts, " | "))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
