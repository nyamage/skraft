package cmd

import (
	"fmt"

	skraftconfig "github.com/nyamage/skraft/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Read or write skraft configuration",
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Print a config value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := findRepoRoot()
		if err != nil {
			return err
		}
		cfg, err := skraftconfig.Load(repoRoot)
		if err != nil {
			return err
		}
		switch args[0] {
		case "skills_dir":
			fmt.Println(cfg.SkillsDir)
		default:
			return fmt.Errorf("unknown key %q (known keys: skills_dir)", args[0])
		}
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := findRepoRoot()
		if err != nil {
			return err
		}
		cfg, err := skraftconfig.Load(repoRoot)
		if err != nil {
			return err
		}
		switch args[0] {
		case "skills_dir":
			cfg.SkillsDir = args[1]
		default:
			return fmt.Errorf("unknown key %q (known keys: skills_dir)", args[0])
		}
		if err := skraftconfig.Save(repoRoot, cfg); err != nil {
			return err
		}
		fmt.Printf("set %s = %s\n", args[0], args[1])
		return nil
	},
}

func init() {
	configCmd.AddCommand(configGetCmd, configSetCmd)
	rootCmd.AddCommand(configCmd)
}
