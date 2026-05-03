package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nyamage/skraft/internal/config"
	"github.com/nyamage/skraft/internal/ledger"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize skraft in the current git repository",
	Long: `Creates .skraft/ with config.toml and ledger.db.
Safe to run multiple times — does not overwrite existing config.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := findRepoRoot()
		if err != nil {
			return err
		}

		skraftDir := config.SkraftDirPath(repoRoot)
		if err := os.MkdirAll(skraftDir, 0755); err != nil {
			return fmt.Errorf("create .skraft/: %w", err)
		}

		// Write default config only if it doesn't exist
		cfgPath := filepath.Join(skraftDir, config.ConfigFile)
		if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
			if err := config.Save(repoRoot, config.DefaultConfig()); err != nil {
				return fmt.Errorf("write config: %w", err)
			}
			fmt.Println("created .skraft/config.toml")
		} else {
			fmt.Println(".skraft/config.toml already exists, skipping")
		}

		// Open (and migrate) the ledger
		l, err := ledger.Open(config.LedgerPath(repoRoot))
		if err != nil {
			return fmt.Errorf("initialize ledger: %w", err)
		}
		if err := l.Close(); err != nil {
			return fmt.Errorf("close ledger: %w", err)
		}
		fmt.Println("initialized .skraft/ledger.db")

		// Update .gitignore
		if err := ensureGitignore(repoRoot); err != nil {
			return fmt.Errorf("update .gitignore: %w", err)
		}

		fmt.Printf("\nskraft initialized in %s\n", repoRoot)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

// ensureGitignore appends skraft ledger entries to .gitignore if not present.
func ensureGitignore(repoRoot string) error {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	entries := []string{".skraft/ledger.db", ".skraft/ledger.db-shm", ".skraft/ledger.db-wal"}

	// Read existing content
	existing := map[string]bool{}
	if f, err := os.Open(gitignorePath); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			existing[strings.TrimSpace(scanner.Text())] = true
		}
		f.Close()
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("read .gitignore: %w", err)
		}
	}

	var toAdd []string
	for _, e := range entries {
		if !existing[e] {
			toAdd = append(toAdd, e)
		}
	}
	if len(toAdd) == 0 {
		return nil
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, "\n# skraft")
	for _, e := range toAdd {
		fmt.Fprintln(f, e)
	}
	fmt.Printf("updated .gitignore with %d entries\n", len(toAdd))
	return nil
}
