# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
# Build
go build -o jot ./cmd/jot

# Install
go install ./cmd/jot

# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run a single test
go test -run TestParse ./internal/entry

# Run tests for a specific package
go test ./internal/entry
```

## Architecture

jot is a CLI-first plaintext journal tool designed for LLM orchestration. The intelligence lives in the LLM layer; jot provides simple CRUD operations with JSON output.

### Package Structure

- `cmd/jot/main.go` - Entry point, invokes `commands.Execute()`
- `internal/commands/` - Cobra CLI commands (one file per command)
- `internal/entry/` - Core Entry type, parsing, validation, slug generation
- `internal/store/` - File-based storage, CRUD operations, search, filtering
- `internal/config/` - Configuration loading, journal root discovery

### Key Design Patterns

**Journal Resolution**: `config.FindRoot()` walks up from cwd looking for `.jot/` directories. Falls back to `~/.jot` (global journal) if no local journal found.

**Entry Storage**: Entries are markdown files with YAML frontmatter stored at `entries/YYYY/MM/YYYYMMDD-slug.md`. Sidecar metadata goes in `.meta.yaml` files.

**Slug Format**: `YYYYMMDD-title-slug` (e.g., `20240102-api-redesign`). Generated from timestamp + slugified title.

**Output**: All commands default to JSON output for machine consumption. Use `--format=table` or `--format=markdown` for human-readable output, `--pretty` for formatted JSON.

### Data Model

Entry core fields: `title`, `type` (idea/task/note/plan/log), `tags`, `status` (open/in_progress/done/blocked/archived), `priority` (low/medium/high/critical), `due`, `created`, `modified`. Task-specific: `blocked_by`, `depends_on`. Extensions preserved in `Extensions` map.

### Command Implementation Pattern

Commands use shared helpers from `root.go`: `getStore()`, `getConfig()`, `getOutputFormat()`, `shouldPrettyPrint()`. Commands register themselves in their `init()` functions via `rootCmd.AddCommand()`.
