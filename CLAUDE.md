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

- `cmd/jot/main.go` — Entry point, invokes `commands.Execute()`
- `internal/commands/` — Cobra CLI commands (one file per command, register via `init()`)
- `internal/entry/` — Core Entry type, parsing, validation, slug generation
- `internal/store/` — File-based storage, CRUD, search, filtering, tag summaries
- `internal/config/` — Configuration loading, journal root discovery (`FindRoot()`)
- `internal/fuzzy/` — Levenshtein distance fuzzy matching (used by `--fuzzy` flag)

### Command Groups

Commands are organized into 6 Cobra groups defined in `root.go`:

| Group | ID | Commands |
|-------|----|----------|
| Create | `create` | add, new |
| View and Query | `query` | list, show, search, tags |
| Modify | `modify` | edit, status |
| Lifecycle | `lifecycle` | stale, archive, purge, rm |
| Data | `data` | export, import |
| Admin | `admin` | init, config, which, lint, claude |

Every command must have `GroupID` set. `TestCommandGroups` validates this.

### Key Design Patterns

**Journal Resolution**: `config.FindRoot()` walks up from cwd looking for `.jot/` directories. Falls back to `~/.jot` (global journal). In practice, the global journal is used exclusively with tags for project scoping.

**Entry Storage**: Markdown files with YAML frontmatter at `entries/YYYY/MM/YYYYMMDD-slug.md`. Sidecar metadata in `.meta.yaml` files. Asset directories at `YYYYMMDD-slug_assets/`.

**Output Flags**: Global boolean flags `--json`, `--table`, `--markdown`/`--md` (mutually exclusive). `--fuzzy` enables Levenshtein matching. Helper `getOutputFormat()` returns the format string. Default is table.

**Filter System**: `store.Filter` struct with AND-combined fields (Type, Tag, Status, Priority, Since, Until, Limit, Fuzzy). `ParseDuration()` handles relative strings like "7d", "2w", "1m".

**Slug Resolution**: `resolve.go` provides partial slug matching with interactive selection for ambiguous matches. Used by show, status, edit, rm, lint.

### Data Model

Entry fields: `title`, `type` (idea/task/note/plan/log), `tags`, `status` (open/in_progress/done/blocked/archived), `priority` (low/medium/high/critical), `due`, `created`, `modified`. Task-specific: `blocked_by`, `depends_on`. Unknown frontmatter preserved in `Extensions` map.

### Command Implementation Pattern

Commands use shared helpers from `root.go`: `getStore()`, `getConfig()`, `getOutputFormat()`, `shouldPrettyPrint()`, `getFuzzy()`. Each command file registers via `rootCmd.AddCommand()` in its `init()`.

### Claude Code Plugin

The `jot claude` subcommand is deprecated — it redirects users to the standalone plugin at `github.com/queelius/claude-anvil` (the `jot/` directory). The plugin provides: skill (CLI reference), commands (/jot, /jot triage), agent (journal-analyst), and a SessionStart hook for ambient project context.
