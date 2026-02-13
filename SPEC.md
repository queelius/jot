# jot — A Plaintext Idea Toolkit for the LLM Era

## Executive Summary

**jot** is a CLI-first, plaintext-native toolkit for capturing and organizing ideas, plans, tasks, and notes. Unlike traditional personal knowledge management (PKM) tools that bolt AI features onto existing architectures, jot is designed from the ground up for the LLM era—where an intelligent agent like Claude Code can orchestrate complex operations over a simple, transparent data layer.

### The Core Insight

The advent of conversational AI fundamentally changes what a notes/ideas tool needs to be:

- **Old paradigm**: Tool must be smart (complex querying, auto-linking, visualization)
- **New paradigm**: Tool must be *transparent*—simple primitives, predictable structure, machine-readable output. Intelligence lives in the LLM layer.

jot embraces this by being deliberately "dumb": it provides robust CRUD operations, structured JSON output, and clean plaintext storage. Claude Code (or any LLM) provides the intelligence: relating entries, decomposing complex ideas, identifying duplicates, synthesizing new insights, and filling conceptual gaps.

---

## Motivation: Why Build This?

### Prior Art Limitations

| Tool | Limitation in LLM Era |
|------|----------------------|
| **Obsidian** | GUI-heavy, proprietary plugin system, vault format optimized for humans not machines |
| **Notion** | Proprietary format, API-heavy, not plaintext, vendor lock-in |
| **org-mode** | Requires Emacs, syntax is human-oriented, complex for LLMs to modify safely |
| **Taskwarrior** | Task-only focus, no freeform ideas, custom binary format |
| **Plain markdown files** | No structure, no metadata, no search, no CLI tooling |

### The jot Difference

1. **LLM-native design**: Every decision optimizes for LLM interoperability—JSON output, predictable file layout, clear schemas, explicit primitives
2. **CLI-first simplicity**: No GUI to load, no Electron overhead, composable Unix-style commands
3. **Plaintext purity**: Human-readable markdown, grep-able, future-proof, git-friendly
4. **Git as history**: Version control via git, not reinvented. Diffs, branches, remote backup for free
5. **Separation of concerns**: jot manages data, LLMs provide intelligence

---

## Design Principles

1. **KISS**: Every feature must justify its complexity. When in doubt, leave it out.
2. **Plaintext first**: A human with `cat` and `grep` can use the data without jot installed.
3. **JSON for machines**: Structured output for programmatic access. `--pretty` for humans.
4. **Git-native**: Assume git is available. Use it for history, sync, and backup.
5. **LLM-friendly**: Predictable structure, explicit IDs, no magic. An LLM should be able to use jot by reading `jot --help`.
6. **No lock-in**: Data format is documented. Migration scripts should be trivial to write.

---

## Data Model

### Entry

The fundamental unit is an **entry**: a markdown file with optional YAML frontmatter.

```markdown
---
title: API Redesign for v2
type: idea
tags: [api, architecture, v2]
status: open
priority: high
due: 2024-02-15
created: 2024-01-02T10:30:00Z
modified: 2024-01-02T14:22:00Z
---

We should consider a GraphQL-first approach for the v2 API.

## Key considerations

- Current REST API has N+1 query problems
- Mobile clients would benefit from selective field fetching
- Need to evaluate Hasura vs. custom implementation

## Open questions

- [ ] Benchmark current REST performance
- [ ] Prototype GraphQL layer
- [ ] Evaluate authentication story
```

### Core Frontmatter Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | Yes (auto-generated if omitted) | Human-readable title |
| `type` | string | No | Entry type: `idea`, `task`, `note`, `plan`, `log` (extensible) |
| `tags` | string[] | No | Categorization tags |
| `status` | string | No | Workflow status: `open`, `in_progress`, `done`, `blocked`, `archived` |
| `priority` | string | No | Priority: `low`, `medium`, `high`, `critical` |
| `due` | date | No | Due date (ISO 8601: YYYY-MM-DD) |
| `created` | datetime | Auto | Creation timestamp (ISO 8601) |
| `modified` | datetime | Auto | Last modification timestamp |

**Extension fields**: Any additional YAML fields are preserved. jot validates core fields but passes through unknown fields. This allows user-defined or LLM-generated metadata.

### Entry Identity

Entries are identified by their **slug**: `YYYYMMDD-title-slug`

- Generated from timestamp + title (slugified)
- Example: `20240102-api-redesign-for-v2`
- Stable for referencing, human-readable, greppable
- Claude Code can suggest titles; user or auto-generation creates slugs

### Sidecar Metadata

LLM-generated metadata lives in sidecar files to keep entries clean:

```
entries/2024/01/20240102-api-redesign.md           # Human content
entries/2024/01/20240102-api-redesign.meta.yaml    # Machine metadata
```

Sidecar files may contain:
- Embeddings (for semantic search)
- Computed relationships to other entries
- Auto-generated summaries
- Classification results
- Any LLM-generated annotations

```yaml
# 20240102-api-redesign.meta.yaml
embedding: [0.123, -0.456, ...]  # Vector embedding
summary: "Proposal to migrate v2 API to GraphQL for better mobile performance"
relations:
  - target: 20231215-mobile-app-perf
    type: addresses
    confidence: 0.92
  - target: 20240101-graphql-evaluation
    type: related
    confidence: 0.78
generated_at: 2024-01-02T15:00:00Z
generator: claude-3-opus
```

### Attachments

Entries may have associated assets stored in a sibling directory:

```
entries/2024/01/20240102-api-redesign.md
entries/2024/01/20240102-api-redesign.meta.yaml
entries/2024/01/20240102-api-redesign/
  ├── diagram.png
  ├── benchmark-results.csv
  └── prototype.py
```

Markdown content references assets with relative paths:
```markdown
![Architecture diagram](./20240102-api-redesign/diagram.png)
```

---

## Directory Structure

```
my-journal/
├── .jot/
│   └── config.yaml          # Journal configuration
├── entries/
│   ├── 2024/
│   │   ├── 01/
│   │   │   ├── 20240102-api-redesign.md
│   │   │   ├── 20240102-api-redesign.meta.yaml
│   │   │   ├── 20240102-api-redesign/
│   │   │   │   └── diagram.png
│   │   │   ├── 20240102-quick-thought.md
│   │   │   └── 20240103-meeting-notes.md
│   │   └── 02/
│   │       └── ...
│   └── 2023/
│       └── ...
├── .gitignore
└── README.md                 # Optional: journal description
```

### Configuration

`.jot/config.yaml`:

```yaml
# Journal metadata
name: "Personal Ideas"
description: "Technical ideas, plans, and notes"

# Default values for new entries
defaults:
  type: note
  tags: []

# Editor preference (falls back to $EDITOR)
editor: nvim

# Output preferences
output:
  format: json          # json | markdown | table
  pretty: false         # Pretty-print JSON when true
  color: auto           # auto | always | never

# Date format for display
date_format: "2006-01-02"  # Go time format
```

---

## CLI Commands

### Initialization

```bash
jot init [directory]
```

Creates a new jot journal in the specified directory (default: current directory).
- Creates `.jot/config.yaml` with defaults
- Creates `entries/` directory
- Optionally initializes git repository

### Entry Creation

```bash
# Quick capture (one-liner)
jot add "Quick thought about API caching"
jot add "Fix the login bug" --type=task --priority=high --due=2024-01-15

# Full editor experience
jot new
jot new --type=idea --tags=api,architecture
jot new --title="API Redesign Proposal"
```

**Behavior**:
- `jot add`: Creates entry from argument, opens editor only if `--edit` flag
- `jot new`: Opens `$EDITOR` with template, saves on close
- Auto-generates slug from title or first line
- Sets `created` timestamp

### Entry Listing

```bash
# List all entries
jot list

# Filter by attributes
jot list --type=task
jot list --tag=api
jot list --status=open
jot list --priority=high
jot list --since=7d              # Last 7 days
jot list --until=2024-01-01      # Before date
jot list --type=task --status=open --tag=urgent   # Combine filters

# Output control
jot list --format=json           # JSON array
jot list --format=table          # ASCII table
jot list --format=markdown       # Markdown list
jot list --pretty                # Pretty-print JSON
jot list --limit=10              # Limit results
jot list --sort=created          # Sort by field (created, modified, title, priority)
jot list --reverse               # Reverse sort order
```

**Default output** (JSON, one object per line for streaming):
```json
{"slug":"20240102-api-redesign","title":"API Redesign for v2","type":"idea","status":"open","created":"2024-01-02T10:30:00Z"}
{"slug":"20240102-quick-thought","title":"Quick thought about caching","type":"note","created":"2024-01-02T11:00:00Z"}
```

### Entry Display

```bash
jot show <slug>
jot show 20240102-api-redesign

# Output options
jot show <slug> --raw            # Raw file contents (no rendering)
jot show <slug> --json           # Structured JSON with content + frontmatter
jot show <slug> --meta           # Include sidecar metadata if present
```

**Default**: Renders markdown in terminal using glamour or similar.

### Entry Editing

```bash
jot edit <slug>
jot edit 20240102-api-redesign
```

Opens entry in `$EDITOR`. Updates `modified` timestamp on save.

### Entry Deletion

```bash
jot rm <slug>
jot rm 20240102-api-redesign

# With confirmation bypass
jot rm <slug> --force
```

Deletes entry file, sidecar, and asset directory if present.

### Content Search

```bash
jot search <query>
jot search "GraphQL implementation"
jot search "authentication" --type=task
jot search "api" --tag=v2

# Output matches with context
jot search "query" --context=3    # 3 lines before/after match
```

Full-text search across entry content and titles.

### Tag Management

```bash
# List all tags with counts
jot tags

# List entries with specific tag
jot tags api                      # Same as: jot list --tag=api
```

### Task Workflow

```bash
# List open tasks
jot list --type=task --status=open
jot list --type=task --status=blocked
jot list --type=task --priority=high
jot list --type=task --due=today  # Due today or overdue
jot list --type=task --due=week   # Due within 7 days

# Mark task complete
jot status <slug> done
jot status 20240102-fix-login-bug done

# Change status
jot status <slug> <status>
jot status 20240102-api-redesign in_progress
jot status 20240102-api-redesign blocked
```

### Validation

```bash
# Lint all entries
jot lint

# Lint specific entry
jot lint <slug>

# Output
jot lint --format=json           # Machine-readable errors
```

Validates:
- YAML frontmatter syntax
- Core field types (date formats, enum values)
- File structure integrity
- Orphaned sidecar/asset files

### Import/Export

```bash
# Export all entries
jot export > backup.json
jot export --format=markdown > backup.md

# Export with filters
jot export --type=task --status=open > open-tasks.json

# Import entries
jot import backup.json
jot import --format=markdown notes.md
```

### Configuration

```bash
# Show current config
jot config

# Get specific value
jot config defaults.type

# Set value
jot config set editor "code --wait"
jot config set defaults.type idea
```

---

## Task Workflow Semantics

Tasks are entries with `type: task` and task-specific fields.

### Status Lifecycle

```
open → in_progress → done
  ↓         ↓
blocked ←───┘
  ↓
archived
```

### Task-Specific Fields

| Field | Description |
|-------|-------------|
| `status` | `open`, `in_progress`, `done`, `blocked`, `archived` |
| `priority` | `low`, `medium`, `high`, `critical` |
| `due` | Due date (YYYY-MM-DD) |
| `blocked_by` | Slug of blocking entry |
| `depends_on` | List of prerequisite slugs |

### Example Task

```markdown
---
title: Implement GraphQL prototype
type: task
status: in_progress
priority: high
due: 2024-01-20
depends_on:
  - 20240102-api-redesign
tags: [api, v2, prototype]
---

Build a minimal GraphQL endpoint to validate the architecture proposal.

## Acceptance criteria

- [ ] Basic query support
- [ ] Authentication working
- [ ] Performance benchmarks vs REST
```

---

## Search & Discovery

### List Filtering (Simple Flags)

```bash
jot list --type=task --status=open --tag=api --since=30d
```

Filters are AND-combined. For complex queries, pipe to `jq`:

```bash
jot list --format=json | jq 'select(.priority == "high" or .due <= "2024-01-15")'
```

### Full-Text Search

```bash
jot search "authentication flow"
```

Searches:
- Entry titles
- Entry body content
- Does NOT search frontmatter values (use `--list` filters for that)

### Graph Traversal (via sidecar relations)

If sidecar metadata contains relations, Claude Code can traverse:

```bash
# Get relations for an entry
jot show 20240102-api-redesign --meta --json | jq '.meta.relations'
```

---

## LLM Integration Philosophy

### jot's Role: Dumb Data Layer

jot provides:
- ✅ CRUD operations with predictable behavior
- ✅ Structured JSON output for parsing
- ✅ Consistent file layout for direct file access
- ✅ Schema validation for data integrity
- ✅ Search primitives (text, filters)

jot does NOT provide:
- ❌ Semantic search (embeddings)
- ❌ Auto-linking or relation inference
- ❌ Summarization or synthesis
- ❌ Duplicate detection
- ❌ Decomposition of complex entries

### Claude Code's Role: Intelligent Orchestrator

Claude Code (or similar LLMs) provides:
- Semantic understanding of entry content
- Relationship discovery across entries
- Decomposition: breaking large entries into atomic pieces
- Synthesis: combining related entries into coherent summaries
- Gap analysis: identifying missing pieces in a knowledge area
- Deduplication: finding semantically equivalent entries
- Smart querying: "show me all entries related to API performance"

### Example Workflows

**User**: "What are all my open ideas related to the API?"

**Claude Code**:
1. `jot list --type=idea --status=open --format=json`
2. `jot search "API" --format=json`
3. Semantically filters and ranks results
4. Presents synthesized summary

**User**: "This entry is too big, break it down"

**Claude Code**:
1. `jot show <slug> --json` to read content
2. Analyzes and identifies atomic sub-ideas
3. `jot add "..." --type=idea` for each sub-idea
4. Updates original to reference children
5. `jot edit <original>` to mark as decomposed

**User**: "Find duplicate entries"

**Claude Code**:
1. `jot list --format=json` to get all entries
2. `jot show <slug> --json` for each (or reads files directly)
3. Computes semantic similarity
4. Presents candidates for merging
5. User decides; Claude executes merges

---

## Claude Code Skill Definition

Below is the skill file to be placed at `.claude/skills/jot.md` or included in `CLAUDE.md`:

````markdown
# jot — Personal Idea Toolkit

jot is a CLI tool for managing ideas, tasks, plans, and notes in plaintext markdown format.

## Quick Reference

```bash
# Create entries
jot add "Quick thought"                    # One-liner capture
jot add "Fix bug" --type=task --priority=high
jot new                                    # Open editor for new entry
jot new --type=idea --tags=api

# List and filter
jot list                                   # All entries
jot list --type=task --status=open        # Open tasks
jot list --tag=api --since=7d             # Recent API-tagged entries
jot list --format=json                    # JSON output

# View and edit
jot show <slug>                           # View entry (rendered markdown)
jot show <slug> --json                    # Structured output
jot edit <slug>                           # Open in editor

# Search
jot search "query"                        # Full-text search
jot tags                                  # List all tags

# Tasks
jot list --type=task --status=open        # List open tasks
jot status <slug> done                    # Mark complete
jot status <slug> in_progress             # Change status

# Maintenance
jot lint                                  # Validate all entries
jot export > backup.json                  # Export
```

## Entry Location

Entries are stored at: `entries/YYYY/MM/YYYYMMDD-slug.md`

Sidecar metadata (if present): `entries/YYYY/MM/YYYYMMDD-slug.meta.yaml`

## Entry Format

```markdown
---
title: Entry Title
type: idea | task | note | plan | log
tags: [tag1, tag2]
status: open | in_progress | done | blocked | archived
priority: low | medium | high | critical
due: 2024-01-15
---

Markdown content here.
```

## Common Patterns

### Finding related entries
```bash
jot list --format=json | jq -r '.slug' | while read slug; do
  jot show "$slug" --json
done
```

### Bulk operations
Read files directly from `entries/` directory. Modify with `jot edit` or write directly.

### Creating entries programmatically
```bash
jot add "Title here" --type=task --tags=tag1,tag2
```

Or write directly to `entries/YYYY/MM/YYYYMMDD-slug.md` with proper frontmatter.

## Notes for LLM Usage

- All commands support `--format=json` for structured output
- Entry slugs are stable identifiers: `YYYYMMDD-title-slug`
- Sidecar `.meta.yaml` files can store computed metadata (embeddings, relations)
- Git tracks history—no need to preserve old versions manually
- When modifying entries directly, run `jot lint` to validate
````

---

## Implementation Notes

### Language Recommendation

**Go** is recommended for implementation:
- Single static binary (no runtime dependencies)
- Excellent CLI libraries (cobra, viper, glamour)
- Fast startup time
- Cross-platform compilation
- Good YAML/JSON handling

Alternatives considered:
- **Rust**: Faster, but steeper learning curve, slower compilation
- **Python**: Faster prototyping, but requires runtime, slower startup

### Key Dependencies (Go)

- `github.com/spf13/cobra` — CLI framework
- `github.com/spf13/viper` — Configuration
- `github.com/charmbracelet/glamour` — Markdown rendering
- `gopkg.in/yaml.v3` — YAML parsing
- `github.com/gosimple/slug` — Slug generation
- `github.com/blevesearch/bleve` — Full-text search (optional, for indexed search at scale)

### Performance Considerations

For thousands of entries:
- File listing is fast (filesystem is the index)
- Full-text search should use an index (bleve or SQLite FTS)
- Consider caching frontmatter in `.jot/index.json` for fast `jot list`
- Lazy-parse: only read full file when needed

### Git Integration

- `.gitignore` should include:
  - `.jot/index.json` (cache, regenerable)
  - `*.meta.yaml` embeddings (optional: may want to track relations but not embeddings)
- Pre-commit hook (optional): `jot lint --format=json` to validate before commit

---

## Future Considerations (Out of Scope for v1)

- **Web UI**: Optional read-only web view for browsing
- **Sync daemon**: Background process for live sync
- **Encryption**: Encrypted entries for sensitive content
- **Templates**: Entry templates for common patterns
- **Plugins**: Extensible command system
- **Collaboration**: Multi-user journals with merge conflict resolution

These are explicitly deferred to maintain v1 simplicity.

---

## Appendix: Example Session

```bash
# Initialize new journal
$ jot init ~/ideas
Initialized jot journal in /home/user/ideas

# Quick capture
$ jot add "Consider using GraphQL for API v2"
Created: 20240102-consider-using-graphql-for-api-v2

# Create detailed task
$ jot new --type=task --tags=api,urgent
# (editor opens, user writes, saves)
Created: 20240102-implement-rate-limiting

# List recent tasks
$ jot list --type=task --since=7d -v
SLUG                              TITLE                      TYPE  STATUS    PRIORITY  DUE         CREATED
20240102-implement-rate-limiting  Implement rate limiting    task  open      high      2024-01-10  2024-01-02
20240101-fix-auth-bug             Fix authentication bug     task  open      critical  2024-01-03  2024-01-01

# Search for related content
$ jot search "authentication"
20240101-fix-auth-bug: Fix authentication bug
  ...implement proper JWT validation for the **authentication** flow...

20231220-security-review: Security review notes
  ...noted issues with **authentication** token expiry...

# Mark task done
$ jot status 20240101-fix-auth-bug done
Status changed: 20240101-fix-auth-bug → done

# Export for backup
$ jot export > ~/backup/ideas-20240102.json
Exported 47 entries
```

---

*Generated: 2024-01-02*
*Version: 1.0-draft*
