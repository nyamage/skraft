package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nyamage/skraft/internal/config"
	"github.com/spf13/cobra"
)

var adoptFrom string

var adoptCmd = &cobra.Command{
	Use:   "adopt <name>",
	Short: "Adopt an existing skill directory into the skraft repo",
	Long: `Moves a skill from cfg.SkillsDir/<name> (or --from <path>) into the git
repo root, then creates a symlink at the original location. This is the
inverse of the manual copy-delete-link workflow.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := findRepoRoot()
		if err != nil {
			return err
		}
		cfg, err := config.Load(repoRoot)
		if err != nil {
			return err
		}

		srcDir := adoptFrom
		if srcDir == "" {
			srcDir = filepath.Join(cfg.SkillsDir, args[0])
		}
		srcDir, err = filepath.Abs(srcDir)
		if err != nil {
			return fmt.Errorf("resolve source path: %w", err)
		}

		srcInfo, err := os.Lstat(srcDir)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("source %s does not exist", srcDir)
			}
			return fmt.Errorf("stat source: %w", err)
		}
		if srcInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is already a symlink — already managed by skraft?", srcDir)
		}
		if !srcInfo.IsDir() {
			return fmt.Errorf("%s is not a directory", srcDir)
		}

		if _, err := os.Stat(filepath.Join(srcDir, "SKILL.md")); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("%s has no SKILL.md — is this a skill directory?", srcDir)
			}
			return fmt.Errorf("stat SKILL.md: %w", err)
		}

		dirName := filepath.Base(srcDir)
		destDir := filepath.Join(repoRoot, dirName)

		if _, err := os.Lstat(destDir); err == nil {
			return fmt.Errorf("%s already exists in the repo", destDir)
		}

		if err := os.Rename(srcDir, destDir); err != nil {
			return fmt.Errorf("move %s → %s: %w", srcDir, destDir, err)
		}

		if err := os.MkdirAll(cfg.SkillsDir, 0755); err != nil {
			return fmt.Errorf("create skills dir: %w", err)
		}
		if err := os.Symlink(destDir, srcDir); err != nil {
			_ = os.Rename(destDir, srcDir) // best-effort rollback
			return fmt.Errorf("create symlink: %w", err)
		}

		fmt.Printf("adopted  %s\n", dirName)
		fmt.Printf("  moved:   %s\n", destDir)
		fmt.Printf("  linked:  %s → %s\n", srcDir, destDir)
		return nil
	},
}

func init() {
	adoptCmd.Flags().StringVar(&adoptFrom, "from", "", "source path (default: cfg.SkillsDir/<name>)")
	rootCmd.AddCommand(adoptCmd)
}
