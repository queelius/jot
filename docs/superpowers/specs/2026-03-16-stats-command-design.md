# Design: `jot stats` Command — Journal Snapshot Data

## Context

jot stores structured journal entries (tasks, ideas, notes, plans, logs) with rich metadata (status, priority, tags, due dates). The LLM layer (Claude Code, journal-analyst agent) provides intelligence, but jot currently offers no way to get a snapshot of the journal's state. With 100+ entries, the LLM has to call `jot list --json` and compute aggregates itself — wasteful and slow.

`jot stats` provides pre-computed snapshot data as JSON, designed for LLM consumption. The intelligence stays in the LLM; jot provides the data primitives.

## Design Principles

- **LLM-first**: Always outputs JSON. No table/markdown rendering. The LLM interprets the data.
- **Snapshot-oriented**: Current state, not historical trends.
- **Filter-compatible**: Reuses existing `store.Filter` flags to scope the data.
- **Sectioned**: Output is a JSON object with one key per section. Caller chooses which sections to compute.

## Command Interface

```bash
jot stats                                    # default: summary + overdue + blocked
jot stats --all                              # all sections
jot stats --section=health                   # specific section
jot stats --section=summary,overdue          # multiple sections
jot stats --tags=ctk                         # scoped to entries with tag
jot stats --type=task --status=open          # scoped by type/status
jot stats --since=30d                        # scoped to recent entries
jot stats --stale-days=30                    # stale threshold for health (default: 30)
```

- **GroupID**: `query` (alongside list, show, search, tags)
- **Output**: Always JSON. The command does not call `getOutputFormat()` and always marshals `StatsResult` as indented JSON to stdout. Passing `--table` or `--markdown` has no effect (these are global flags on rootCmd and cannot be selectively disabled without Cobra hacks; the command simply ignores them).
- **Filters**: Same `store.Filter` flags as `list` — `--tags`, `--type`, `--status`, `--priority`, `--since`, `--until`. Note: `list`-specific flags (`--due`, `--search`/`-q`, `--sort`, `--reverse`) are not available on `stats` since they are implemented as post-filter operations in `list.go`, not in `store.Filter`. The `--limit` flag from `store.Filter` applies to the initial entry set, not to individual section outputs. The `--fuzzy` global flag works for tag matching as with other commands.
- **`--section`**: Comma-separated section names. Default: `summary,overdue,blocked`. Case-insensitive. Unknown section names return an error.
- **`--all`**: Shorthand for all five sections.
- **`--stale-days`**: Only used by the `health` section. Silently ignored when `health` is not requested.

## Output Sections

### `summary` — Aggregate counts

```json
{
  "summary": {
    "total": 87,
    "by_type": {"task": 42, "idea": 25, "note": 15, "plan": 3, "log": 2},
    "by_status": {"open": 31, "in_progress": 8, "done": 35, "blocked": 4, "archived": 9},
    "by_priority": {"critical": 2, "high": 7, "medium": 15, "low": 12, "unset": 51},
    "tags_count": 23
  }
}
```

Counts entries in the filtered set by type, status, and priority. `tags_count` is the number of distinct tags across the filtered entries. Zero-count keys are omitted from the maps (e.g., if no entries have `priority: critical`, that key is absent from `by_priority`).

### `overdue` — Past-due active entries

```json
{
  "overdue": [
    {"slug": "20260310-api-deadline", "title": "API Deadline", "type": "task", "status": "open", "priority": "high", "due": "2026-03-10", "tags": ["backend"], "created": "2026-03-01T...", "modified": "2026-03-08T..."}
  ]
}
```

Entries where `due` is before today AND status is not `done` or `archived`. Sorted by due date ascending (most overdue first). Entries without a `due` date are excluded.

### `blocked` — Blocked entries

```json
{
  "blocked": [
    {"slug": "20260301-auth-rework", "title": "Auth Rework", "type": "task", "status": "blocked", "priority": "high", "blocked_by": "waiting on security review", "tags": ["auth"], "created": "2026-02-15T...", "modified": "2026-03-01T..."}
  ]
}
```

Entries with `status: blocked`. Includes `blocked_by` field if set. Sorted by modified date ascending (least recently touched first — surfaces stale blocked items for triage).

### `health` — Per-tag project health

```json
{
  "health": [
    {
      "tag": "ctk",
      "total": 12,
      "open": 5,
      "in_progress": 2,
      "done": 4,
      "blocked": 1,
      "archived": 0,
      "overdue": 1,
      "stale": 2
    }
  ]
}
```

For each tag present in the filtered entries: full status breakdown (including `archived`), count of overdue entries (same logic as overdue section), count of stale entries (active entries — not done/archived — not modified in `--stale-days` days, default 30). Sorted by tag name. Zero counts are included in health (unlike summary maps) to give a complete per-project picture.

### `recent` — Recently modified entries

```json
{
  "recent": [
    {"slug": "20260315-tag-command", "title": "Tag Command", "type": "task", "status": "done", "tags": ["jot"], "created": "2026-03-14T...", "modified": "2026-03-16T..."}
  ]
}
```

Entries modified in the last 7 days. Sorted by modified date descending (most recent first).

Note: the `--since` flag filters the global entry set, not the `recent` window. If `--since=30d` is provided, the entry set is already narrowed to 30 days, and `recent` still applies its 7-day window within that set. This avoids semantic overload — `--since` always means "restrict to entries created after this date."

## Data Types

### Entry projection

Stats output reuses the existing `entry.EntrySummary` type for entry lists (overdue, blocked, recent sections). `EntrySummary` already has all needed fields: `Slug`, `Title`, `Type`, `Status`, `Priority`, `Due`, `Tags`, `Created`, `Modified`.

For the `blocked` section, the `blocked_by` field is needed but not present on `EntrySummary`. Rather than creating a parallel type, we add `BlockedBy` to `EntrySummary`:

```go
type EntrySummary struct {
    Slug      string   `json:"slug"`
    Title     string   `json:"title"`
    Type      string   `json:"type,omitempty"`
    Status    string   `json:"status,omitempty"`
    Priority  string   `json:"priority,omitempty"`
    Due       string   `json:"due,omitempty"`
    Tags      []string `json:"tags,omitempty"`
    Created   string   `json:"created"`
    Modified  string   `json:"modified"`
    BlockedBy string   `json:"blocked_by,omitempty"`
}
```

This is backward-compatible — `omitempty` means `blocked_by` only appears in JSON when set. Existing consumers of `EntrySummary` (list, tag, show commands) are unaffected.

### `SummaryStats`

```go
type SummaryStats struct {
    Total      int            `json:"total"`
    ByType     map[string]int `json:"by_type"`
    ByStatus   map[string]int `json:"by_status"`
    ByPriority map[string]int `json:"by_priority"`
    TagsCount  int            `json:"tags_count"`
}
```

All map fields are initialized to empty maps (not nil) so they serialize as `{}` rather than `null` when no entries match.

### `TagHealth`

```go
type TagHealth struct {
    Tag        string `json:"tag"`
    Total      int    `json:"total"`
    Open       int    `json:"open"`
    InProgress int    `json:"in_progress"`
    Done       int    `json:"done"`
    Blocked    int    `json:"blocked"`
    Archived   int    `json:"archived"`
    Overdue    int    `json:"overdue"`
    Stale      int    `json:"stale"`
}
```

All status counts are always present (zero included) to give a complete per-project picture.

### `StatsResult`

```go
type StatsResult struct {
    Summary *SummaryStats         `json:"summary,omitempty"`
    Overdue []*entry.EntrySummary `json:"overdue,omitempty"`
    Blocked []*entry.EntrySummary `json:"blocked,omitempty"`
    Health  []*TagHealth          `json:"health,omitempty"`
    Recent  []*entry.EntrySummary `json:"recent,omitempty"`
}
```

`omitempty` ensures unrequested sections are absent from the JSON, not present as null.

### `Stats()` method signature

```go
func (s *Store) Stats(f *Filter, sections map[string]bool, staleDays int) (*StatsResult, error)
```

## Architecture

### Files

| File | Action | Purpose |
|------|--------|---------|
| `internal/store/stats.go` | Create | `StatsResult`, `SummaryStats`, `TagHealth` types; `Stats()` method; section computation helpers |
| `internal/entry/entry.go` | Edit | Add `BlockedBy` field to `EntrySummary`; populate it in `Summary()` |
| `internal/commands/stats.go` | Create | Cobra command, flag parsing, delegates to `s.Stats()` |
| `internal/store/stats_test.go` | Create | Unit tests for each section's computation logic |
| `internal/commands/integration_test.go` | Edit | Round-trip integration test |

### Data Flow

1. `statsCmd.RunE` parses `--section`/`--all` into `map[string]bool`
2. Builds `store.Filter` from existing global flags
3. Calls `s.Stats(filter, sections, staleDays)`
4. `Stats()` calls `s.List(filter)` once to get all matching entries
5. Computes each requested section from that entry list
6. Returns `StatsResult` — only requested sections are populated
7. Command marshals to indented JSON (`json.MarshalIndent`) and prints

Single filesystem walk regardless of how many sections are requested. Note: unlike `list` and `tags` which emit JSONL (one object per line), `stats` emits a single JSON object since the output is one aggregate structure, not a stream of entries.

## Testing

### Unit tests (`internal/store/stats_test.go`)

Test computation logic with synthetic entry slices:

- `TestStatsSummary` — counts by type/status/priority; total; tags_count; zero-count keys omitted from maps
- `TestStatsOverdue` — past due + active only; done/archived excluded; no-due excluded; sorted ascending by due
- `TestStatsBlocked` — blocked status entries; blocked_by included; sorted ascending by modified
- `TestStatsHealth` — per-tag aggregation; multi-tag entries counted in each; stale threshold; overdue-per-tag; archived included; sorted by tag
- `TestStatsRecent` — modified within 7-day window; older excluded; sorted descending by modified
- `TestStatsSections` — requesting specific sections populates only those; omitempty works; `--all` populates all
- `TestStatsEmpty` — no entries: summary has total 0 and empty maps; list sections are empty arrays

### Integration test (`internal/commands/integration_test.go`)

One test: create entries with various types/statuses/tags/due dates, compute stats, verify JSON structure and counts.

## Behavior Details

- `jot stats` with no entries: summary has `total: 0` with empty maps `{}`; list sections are omitted (empty slices serialize to null, omitted by `omitempty`).
- Filters narrow the entry set before any section computation. `jot stats --tags=ctk` computes summary/overdue/blocked only for entries tagged `ctk`.
- `--section` values are case-insensitive. Unknown section names return an error listing valid sections.
- The `health` section respects filters: `jot stats --section=health --type=task` shows per-tag health for tasks only.
- `--stale-days` is silently ignored when `health` is not requested.
- `--since` filters the global entry set by created date; it does not affect the `recent` section's 7-day modified window.
