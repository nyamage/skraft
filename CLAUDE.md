# skraft

Claude Code skills, version-controlled.

## Overview

`skraft` is a Go CLI that manages Claude Agent Skills across three targets:

- **Git** — version truth via `git describe --tags --long`
- **Claude Code** — symlinks from `~/.claude/skills/<dirname>` to skill directories
- **Claude.ai** — zip generation for manual upload + SQLite ledger to track upload state

## Commands

```
skraft init            # Create .skraft/ with config.toml and ledger.db
skraft link [skill]    # Symlink skill(s) into Claude Code's skills directory
skraft unlink [skill]  # Remove symlink(s)
skraft status          # Show git version and per-skill link/upload state
skraft pack [skill]    # Generate dist/<skill>-<version>.zip
skraft mark-uploaded   # Record that a skill was uploaded to Claude.ai
skraft sync            # Check or fix drift (--check / --fix)
skraft config get/set  # Read or write config values
```

## Project Structure

```
main.go                          # Entry point → cmd.Execute()
cmd/                             # Cobra subcommands (thin; delegate to internal/)
internal/
  git/git.go                     # Version detection, repo root, HEAD SHA
  skill/skill.go                 # Skill discovery, SKILL.md frontmatter parsing, Pack()
  config/config.go               # .skraft/config.toml read/write
  ledger/ledger.go               # SQLite WAL ledger, upload_state CRUD
  ledger/migrations/             # Embedded SQL migrations (0001_initial.sql, ...)
```

## Key Design Decisions

- **SKILL.md is read-only**: skraft never writes frontmatter. Skills own their own metadata.
- **Version truth is git tags**: `git describe --tags --long` always produces `<tag>-<N>-g<sha>`. At an exact tag, the version is just the tag (e.g. `v1.0.0`).
- **CGo-free SQLite**: uses `modernc.org/sqlite` for easy cross-compilation.
- **Symlinks for Claude Code**: `~/.claude/skills/<dirname>` → skill directory. The `dirname` (not frontmatter `name`) is used for symlink naming.
- **Zips for Claude.ai**: manual upload workflow. `dist/` is gitignored.

## Development

```bash
go build ./...          # Build
go test ./...           # Run all tests
go build -o /tmp/skraft . && /tmp/skraft <command>  # Smoke test
```

Tests use temporary directories and `:memory:` SQLite — no external dependencies needed.

## Adding a Migration

1. Create `internal/ledger/migrations/NNNN_description.sql` (4-digit prefix, e.g. `0002_add_foo.sql`)
2. The migration runner parses the leading 4 digits as the version number and applies pending migrations on `ledger.Open()`
3. Each migration is applied atomically (SQL + `schema_version` update in one transaction)

## Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/spf13/cobra` | v1.10.2 | CLI framework |
| `gopkg.in/yaml.v3` | v3.0.1 | SKILL.md frontmatter parsing |
| `github.com/BurntSushi/toml` | v1.3.2 | config.toml read/write |
| `modernc.org/sqlite` | v1.29.9 | CGo-free SQLite driver |
