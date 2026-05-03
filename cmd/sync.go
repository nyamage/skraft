package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nyamage/skraft/internal/config"
	"github.com/nyamage/skraft/internal/git"
	"github.com/nyamage/skraft/internal/ledger"
	"github.com/nyamage/skraft/internal/skill"
	"github.com/spf13/cobra"
)

var (
	syncCheck bool
	syncFix   bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Check or fix drift between Git, Claude Code, and Claude.ai",
	Long: `--check reports skills that are out of sync.
--fix automatically re-links Claude Code skills and prints instructions for Claude.ai.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !syncCheck && !syncFix {
			return fmt.Errorf("specify --check or --fix")
		}

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

		type drift struct {
			skill       skill.Skill
			claudeCode  bool // needs re-link
			claudeAI    bool // needs re-upload
			uploadedVer string
		}
		var drifts []drift

		fmt.Println("Claude Code:")
		for _, s := range skills {
			linked := skill.IsLinked(cfg.SkillsDir, s)
			if linked {
				fmt.Printf("  %-20s ✓ linked\n", s.Name)
			} else {
				fmt.Printf("  %-20s ✗ NOT linked\n", s.Name)
			}
			d := drift{skill: s, claudeCode: !linked}

			state, err := l.GetUploadState(s.Name, "claudeai")
			if err != nil {
				return err
			}
			if state == nil {
				d.claudeAI = true
				d.uploadedVer = ""
			} else if latestTag != "" && state.Version != latestTag {
				d.claudeAI = true
				d.uploadedVer = state.Version
			}
			drifts = append(drifts, d)
		}

		fmt.Println("\nClaude.ai:")
		for _, d := range drifts {
			if d.uploadedVer == "" && d.claudeAI {
				fmt.Printf("  %-20s ✗ never uploaded\n", d.skill.Name)
			} else if d.claudeAI {
				fmt.Printf("  %-20s ✗ outdated (uploaded: %s, current: %s)\n", d.skill.Name, d.uploadedVer, latestTag)
			} else {
				state, _ := l.GetUploadState(d.skill.Name, "claudeai")
				v := ""
				if state != nil {
					v = state.Version
				}
				fmt.Printf("  %-20s ✓ %s\n", d.skill.Name, v)
			}
		}

		if !syncFix {
			return nil
		}

		// Fix: re-link Claude Code
		var needsUpload []drift
		relinked := 0
		fmt.Println("\nFixing Claude Code...")
		if err := os.MkdirAll(cfg.SkillsDir, 0755); err != nil {
			return err
		}
		for _, d := range drifts {
			if d.claudeCode {
				linkPath := skill.LinkPath(cfg.SkillsDir, d.skill)
				if info, err := os.Lstat(linkPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
					os.Remove(linkPath)
				}
				if err := os.Symlink(d.skill.Dir, linkPath); err != nil {
					return fmt.Errorf("symlink %s: %w", d.skill.Name, err)
				}
				fmt.Printf("  %-20s re-linked\n", d.skill.Name)
				relinked++
			}
			if d.claudeAI {
				needsUpload = append(needsUpload, d)
			}
		}
		if relinked == 0 {
			fmt.Println("  (nothing to fix)")
		}

		if len(needsUpload) > 0 {
			fmt.Println("\nClaude.ai: manual action required.")
			for _, d := range needsUpload {
				zipName := fmt.Sprintf("%s-%s.zip", d.skill.DirName, version)
				zipPath := filepath.Join(repoRoot, "dist", zipName)
				fmt.Printf("  %s:\n", d.skill.Name)
				fmt.Printf("    1. Run: skraft pack %s\n", d.skill.DirName)
				fmt.Printf("    2. Upload %s to claude.ai skill settings\n", zipPath)
				fmt.Printf("    3. Run: skraft mark-uploaded %s\n", d.skill.DirName)
			}
		}
		return nil
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncCheck, "check", false, "report drift without making changes")
	syncCmd.Flags().BoolVar(&syncFix, "fix", false, "fix Claude Code drift and print Claude.ai instructions")
	rootCmd.AddCommand(syncCmd)
}
