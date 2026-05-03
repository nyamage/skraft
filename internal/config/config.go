package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	SkraftDir  = ".skraft"
	ConfigFile = "config.toml"
	LedgerFile = "ledger.db"
)

// Config holds skraft's persistent settings.
type Config struct {
	SkillsDir string `toml:"skills_dir"` // path to Claude Code skills directory
}

// DefaultConfig returns config with sensible defaults.
// If the user's home directory cannot be determined, SkillsDir is set to
// ".claude/skills" relative to the process working directory.
func DefaultConfig() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return Config{
		SkillsDir: filepath.Join(home, ".claude", "skills"),
	}
}

// Load reads the config from repoRoot/.skraft/config.toml.
// Returns defaults if the file does not exist.
func Load(repoRoot string) (Config, error) {
	cfg := DefaultConfig()
	path := filepath.Join(repoRoot, SkraftDir, ConfigFile)
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	return cfg, nil
}

// Save writes cfg to repoRoot/.skraft/config.toml.
// The .skraft directory must already exist.
func Save(repoRoot string, cfg Config) error {
	path := filepath.Join(repoRoot, SkraftDir, ConfigFile)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

// SkraftDirPath returns the absolute path to the .skraft directory.
func SkraftDirPath(repoRoot string) string {
	return filepath.Join(repoRoot, SkraftDir)
}

// LedgerPath returns the absolute path to the SQLite ledger.
func LedgerPath(repoRoot string) string {
	return filepath.Join(repoRoot, SkraftDir, LedgerFile)
}
