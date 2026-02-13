package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Embedded skill content - self-contained in the binary
// Must have YAML frontmatter with name and description for Claude Code to recognize it
const skillContent = `---
name: jot
description: Use jot to manage ideas, tasks, plans, and notes. Use when the user wants to capture thoughts, track tasks, search their journal, or work with their personal knowledge base. NOTE - jot is the user's PERSISTENT journal stored on disk, separate from Claude Code's built-in TodoWrite tool which is for session-only task tracking.
---

# jot — Personal Idea Toolkit

jot is a CLI tool for managing ideas, tasks, plans, and notes in plaintext markdown format.

**IMPORTANT**: jot is the user's PERSISTENT personal journal/todo system stored on disk at ~/.jot/.
This is DIFFERENT from Claude Code's built-in TodoWrite tool:
- **jot**: User's permanent journal. Persists across sessions. Use when user says "add to my todos", "jot this down", "remember this", "add to my journal".
- **TodoWrite**: Claude's session-only task tracking. Disappears after session. Use for tracking YOUR progress on multi-step coding tasks.

## Quick Reference

` + "```bash" + `
# Create entries
jot add "Quick thought"                    # One-liner capture
jot add "Fix bug" --type=task --priority=high
jot new                                    # Open editor for new entry
jot new --type=idea --tags=api

# List and filter
jot list                                   # All entries
jot list --type=task --status=open         # Open tasks
jot list --tag=api --since=7d              # Recent API-tagged entries
jot list --json                            # JSON output
jot list --table                           # Table output

# View and edit
jot show <slug>                            # View entry (rendered markdown)
jot show <slug> --json                     # Structured output
jot show <slug> --raw                      # Raw markdown
jot edit <slug>                            # Open in editor

# Search
jot search "query"                         # Full-text search
jot search "query" --context=3             # With context lines
jot tags                                   # List all tags

# Tasks
jot list --type=task --status=open         # List open tasks
jot list --type=task --priority=high       # High priority tasks
jot list --type=task --due=today           # Due today or overdue
jot status <slug> done                     # Mark complete
jot status <slug> in_progress              # Change status

# Lifecycle
jot stale                                  # Find entries not touched in 90 days
jot stale --days 30 --type=idea            # Stale ideas (30+ days)
jot archive --stale --confirm              # Archive stale entries
jot archive --status done --confirm        # Archive all done entries
jot purge --all --force                    # Permanently delete archived entries

# Maintenance
jot lint                                   # Validate all entries
jot export > backup.json                   # Export to JSON
jot import backup.json                     # Import from JSON
jot config                                 # View config
jot which                                  # Show active journal (local vs global)
` + "```" + `

## Entry Location

Entries are stored at: ` + "`entries/YYYY/MM/YYYYMMDD-slug.md`" + `

Example: ` + "`entries/2024/01/20240102-api-redesign.md`" + `

## Entry Format

` + "```markdown" + `
---
title: Entry Title
type: idea | task | note | plan | log
tags: [tag1, tag2]
status: open | in_progress | done | blocked | archived
priority: low | medium | high | critical
due: 2024-01-15
created: 2024-01-02T10:30:00Z
modified: 2024-01-02T10:30:00Z
---

Markdown content here.
` + "```" + `

## Global vs Local Journals

jot automatically resolves which journal to use:
1. Walks up from cwd looking for ` + "`.jot/`" + ` directory
2. If found → local journal
3. If not found → falls back to ` + "`~/.jot/`" + ` (global journal)

Use ` + "`jot which`" + ` to see which journal is active.

## Common Patterns

### Bulk operations
Read files directly from ` + "`entries/`" + ` directory. Modify with ` + "`jot edit`" + ` or write directly.

### Creating entries programmatically
` + "```bash" + `
jot add "Title here" --type=task --tags=tag1,tag2
` + "```" + `

Or write directly to ` + "`entries/YYYY/MM/YYYYMMDD-slug.md`" + ` with proper frontmatter.

## Filter Options

| Flag | Description |
|------|-------------|
| ` + "`--type`" + ` | Filter by type (idea, task, note, plan, log) |
| ` + "`--tag`" + ` | Filter by tag |
| ` + "`--status`" + ` | Filter by status (open, in_progress, done, blocked, archived) |
| ` + "`--priority`" + ` | Filter by priority (low, medium, high, critical) |
| ` + "`--since`" + ` | Created since (7d, 2w, 2024-01-01) |
| ` + "`--until`" + ` | Created until (date) |
| ` + "`--limit`" + ` | Limit results |

## Output Formats

- ` + "`--json`" + ` — JSON (one object per entry)
- ` + "`--table`" + ` — ASCII table (default)
- ` + "`--markdown`" + ` or ` + "`--md`" + ` — Markdown list

## Notes for LLM Usage

- All commands support ` + "`--json`" + ` for structured output
- Entry slugs are stable identifiers: ` + "`YYYYMMDD-title-slug`" + `
- Use ` + "`jot which --json`" + ` to determine active journal
- When modifying entries directly, run ` + "`jot lint`" + ` to validate
`

var claudeCmd = &cobra.Command{
	Use:     "claude",
	Short:   "Claude Code integration",
	GroupID: "admin",
	Long:  `Commands for integrating jot with Claude Code.`,
}

var claudeInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install jot skill for Claude Code",
	Long: `Install the jot skill file for Claude Code.

By default, installs globally to ~/.claude/skills/jot/SKILL.md
Use --local to install to the current directory's .claude/skills/jot/

Examples:
  jot claude install           # Install globally
  jot claude install --local   # Install to current project`,
	RunE: runClaudeInstall,
}

var claudeShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print the jot skill content",
	Long: `Print the jot skill content to stdout.

Useful for inspection or appending to CLAUDE.md:
  jot claude show >> ~/.claude/CLAUDE.md`,
	RunE: runClaudeShow,
}

var claudeInstallLocal bool

func init() {
	claudeInstallCmd.Flags().BoolVar(&claudeInstallLocal, "local", false, "install to current project instead of globally")

	claudeCmd.AddCommand(claudeInstallCmd)
	claudeCmd.AddCommand(claudeShowCmd)
	rootCmd.AddCommand(claudeCmd)
}

func runClaudeInstall(cmd *cobra.Command, args []string) error {
	var targetDir string

	if claudeInstallLocal {
		// Install to current directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
		targetDir = filepath.Join(cwd, ".claude", "skills", "jot")
	} else {
		// Install globally
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting home directory: %w", err)
		}
		targetDir = filepath.Join(home, ".claude", "skills", "jot")
	}

	// Create directory (skills must be in their own subdirectory)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Write skill file (must be named SKILL.md)
	skillPath := filepath.Join(targetDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0644); err != nil {
		return fmt.Errorf("writing skill file: %w", err)
	}

	fmt.Printf("Installed: %s\n", skillPath)
	return nil
}

func runClaudeShow(cmd *cobra.Command, args []string) error {
	fmt.Print(skillContent)
	return nil
}
