# jot

A plaintext idea toolkit for the LLM era.

Most note-taking tools put intelligence in the app — smart editors, AI assistants baked into the UI, proprietary sync. jot does the opposite: **keep the data dumb, let the LLM be smart.**

Entries are markdown files with YAML frontmatter. That's it. No database, no lock-in, no magic. Claude Code (or any LLM) can read, write, search, and reason over your journal using the CLI — because the data format is trivially machine-readable, and the CLI produces structured JSON.

```bash
jot add "Consider GraphQL for API v2"
jot add "Fix auth bug" --type=task --priority=high --due=2024-01-15
jot list --type=task --status=open --json
```

## Installation

```bash
go install github.com/queelius/jot/cmd/jot@latest
```

Or build from source:

```bash
git clone https://github.com/queelius/jot.git
cd jot && go build -o jot ./cmd/jot
```

## Quick Start

```bash
# Initialize a journal
jot init ~/ideas

# Quick capture
jot add "Consider GraphQL for API v2"
jot add "Fix auth bug" --type=task --priority=high

# Create a detailed entry in your editor
jot new --type=idea --tags=api,architecture

# List and filter
jot list                              # Everything
jot list --type=task --status=open    # Open tasks
jot list --tags=api --since=7d        # Recent API-tagged entries

# View, edit, search
jot show api-redesign                 # Partial slug match
jot edit api-redesign
jot search "authentication"

# Task lifecycle
jot status fix-auth-bug in_progress
jot status fix-auth-bug done

# Housekeeping
jot stale                             # Entries untouched in 90 days
jot archive --status done --confirm   # Archive completed work
jot export > backup.json              # Backup
```

## Entry Format

Entries are markdown files with YAML frontmatter, stored in a date-based hierarchy:

```
~/.jot/entries/2024/01/20240102-api-redesign-proposal.md
```

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

Types: `idea`, `task`, `note`, `plan`, `log`. Statuses: `open`, `in_progress`, `done`, `blocked`, `archived`. Priorities: `low`, `medium`, `high`, `critical`.

## Commands

```
Create:
  add         Quick capture a new entry
  new         Create a new entry in your editor

View and Query:
  list        List and filter entries (alias: ls)
  show        Display an entry
  search      Search entry content
  tags        List all tags (or entries with a tag)

Modify:
  edit        Edit an entry in your editor
  status      Change entry status

Lifecycle:
  stale       Find entries that haven't been touched recently
  archive     Bulk archive entries
  purge       Permanently delete archived entries
  rm          Remove an entry

Data:
  export      Export entries
  import      Import entries

Admin:
  init        Initialize a new jot journal
  config      View or modify configuration
  which       Show which journal is active
  lint        Validate entries
  claude      Claude Code integration
```

## Global vs Local Journals

jot resolves which journal to use by walking up from the current directory:

1. If `.jot/` exists in the cwd or any parent, that's the **local journal**
2. Otherwise, falls back to `~/.jot/` — the **global journal**

```bash
jot which                 # Show which journal is active

cd ~/projects/myapp
jot init                  # Create local journal for this project
jot add "Project idea"    # Goes to ./entries/...

cd /tmp
jot add "Random thought"  # Goes to ~/.jot/entries/...
```

This means project-scoped and global journals coexist naturally.

## Claude Code Integration

jot ships with an embedded Claude Code skill. Install it and Claude learns to use jot as your persistent journal:

```bash
jot claude install          # Install globally
jot claude install --local  # Install to current project
jot claude show             # View skill content
```

Once installed, Claude Code can:

- **Capture on your behalf**: "jot down that I need to revisit the caching layer"
- **Query your journal**: "what are my open tasks related to the API?"
- **Manage lifecycle**: "archive everything that's done, show me what's stale"
- **Reason over entries**: relate ideas, find duplicates, decompose plans

The skill file lives at `~/.claude/skills/jot/SKILL.md`.

## Design Philosophy

jot is deliberately simple. It provides CRUD, filtering, and structured output. That's the contract. Intelligence — semantic search, relationship discovery, summarization, gap analysis — comes from the LLM layer, not the tool.

This means:
- **No lock-in**: entries are files, readable by anything
- **No sync**: use git, Syncthing, rsync, whatever you already use
- **No AI baked in**: the LLM talks to jot through the CLI, not through an SDK
- **Composable**: `jot list --json | jq '.[] | select(.priority == "high")'`

## License

MIT
