# jot

A CLI-first, plaintext-native toolkit for capturing and organizing ideas, plans, tasks, and notes.

**Designed for the LLM era**: simple primitives, predictable structure, machine-readable output. Intelligence lives in the LLM layer (Claude Code).

## Installation

```bash
go install github.com/queelius/jot/cmd/jot@latest
```

Or build from source:

```bash
git clone https://github.com/queelius/jot.git
cd jot
go build -o jot ./cmd/jot
```

## Quick Start

```bash
# Initialize a journal
jot init ~/ideas

# Quick capture
jot add "Consider GraphQL for API v2"

# Create a task
jot add "Fix auth bug" --type=task --priority=high --due=2024-01-15

# Create detailed entry with editor
jot new --type=idea --tags=api,architecture

# List entries
jot list                           # All entries
jot list --type=task --status=open # Open tasks
jot list --format=table            # Table view

# View entry
jot show 20240102-api-redesign

# Task workflow
jot list --type=task --status=open # List open tasks
jot status 20240102-fix-auth-bug done
jot status 20240102-idea in_progress

# Search
jot search "authentication"

# Export/backup
jot export > backup.json
```

## Philosophy

jot is deliberately "dumb"—it provides:
- CRUD operations with predictable behavior
- Structured JSON output for parsing
- Consistent file layout for direct file access
- Schema validation for data integrity

Intelligence comes from LLM orchestration (Claude Code):
- Semantic understanding and relationship discovery
- Decomposition of complex entries
- Synthesis and summarization
- Gap analysis and deduplication

## Data Format

Entries are markdown files with YAML frontmatter:

```markdown
---
title: API Redesign Proposal
type: idea
tags: [api, architecture]
status: open
priority: high
created: 2024-01-02T10:30:00Z
modified: 2024-01-02T10:30:00Z
---

We should consider GraphQL for the v2 API...
```

Files are stored in a date-based hierarchy:
```
entries/2024/01/20240102-api-redesign-proposal.md
```

## Commands

| Command | Description |
|---------|-------------|
| `jot init` | Initialize a new journal |
| `jot add` | Quick capture entry |
| `jot new` | Create entry in editor |
| `jot list` | List entries with filters |
| `jot show` | Display entry |
| `jot edit` | Edit entry in editor |
| `jot rm` | Remove entry |
| `jot search` | Full-text search |
| `jot tags` | List tags |
| `jot status` | Change entry status |
| `jot stale` | Find stale entries |
| `jot archive` | Bulk archive entries |
| `jot purge` | Delete archived entries |
| `jot lint` | Validate entries |
| `jot export` | Export to JSON |
| `jot import` | Import from JSON |
| `jot config` | View/set configuration |

## Global vs Local Journals

jot automatically resolves which journal to use:

1. **Local journal**: If `.jot/` exists in the current directory (or any parent), uses that
2. **Global journal**: Otherwise, falls back to `~/.jot/` (auto-created on first use)

```bash
jot which              # Shows which journal is active

cd ~/projects/myapp
jot init               # Create local journal for this project
jot add "Project idea" # Goes to ./jot

cd /tmp
jot add "Random thought"  # Goes to ~/.jot (global)
```

## Claude Code Integration

jot is designed to work seamlessly with Claude Code. The `.claude/skills/jot.md` file teaches Claude how to use jot effectively.

### Installing the Skill

jot embeds its Claude Code skill and can install it directly:

```bash
# Install globally (recommended)
jot claude install

# Install to current project only
jot claude install --local

# View skill content
jot claude show
```

The skill is installed to `~/.claude/skills/jot/SKILL.md` (global) or `./.claude/skills/jot/SKILL.md` (local).

### What Claude Code Can Do

With jot's structured output, Claude Code can:
- **Create & manage entries**: `jot add "idea"`, `jot new --type=task`
- **Search & filter**: `jot search "query"`, `jot list --type=task --status=open`
- **Task workflow**: `jot list --type=task`, `jot status <slug> done`
- **Semantic operations**: Relate entries, find duplicates, decompose complex ideas
- **Bulk operations**: Read JSON output, modify entries programmatically

### Example Workflow

```
You: "What are my open tasks related to the API?"

Claude Code:
1. jot list --type=task --status=open --json
2. jot search "API" --format=json
3. Correlates and summarizes results
```

## License

MIT
