# `jot stats` Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `jot stats` command that outputs pre-computed journal snapshot data as JSON for LLM consumption.

**Architecture:** A new `Stats()` method on the store computes five optional sections (summary, overdue, blocked, health, recent) from a single `s.List()` call. A thin Cobra command parses `--section`/`--all` flags and delegates to the store. Reuses existing `store.Filter` for scoping.

**Tech Stack:** Go, Cobra CLI framework, `encoding/json` for output.

**Spec:** `docs/superpowers/specs/2026-03-16-stats-command-design.md`

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/entry/entry.go` | Edit | Add `BlockedBy` to `EntrySummary` struct and `Summary()` method |
| `internal/store/stats.go` | Create | Types (`SummaryStats`, `TagHealth`, `StatsResult`), `Stats()` method, section computation helpers |
| `internal/store/stats_test.go` | Create | Unit tests for each section's computation logic |
| `internal/commands/stats.go` | Create | Cobra command, `--section`/`--all`/`--stale-days` flags, filter building, JSON output |
| `internal/commands/integration_test.go` | Edit | Round-trip integration test for stats |

---

## Chunk 1: Data Layer

### Task 1: Add `BlockedBy` to `EntrySummary`

**Files:**
- Modify: `internal/entry/entry.go:222-247`

- [ ] **Step 1: Add field to `EntrySummary` struct**

In `internal/entry/entry.go`, add `BlockedBy` after `Modified` in the `EntrySummary` struct (line 231):

```go
type EntrySummary struct {
	Slug     string   `json:"slug"`
	Title    string   `json:"title"`
	Type     string   `json:"type,omitempty"`
	Status   string   `json:"status,omitempty"`
	Priority string   `json:"priority,omitempty"`
	Due      string   `json:"due,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Created  string   `json:"created"`
	Modified string   `json:"modified"`
	BlockedBy string  `json:"blocked_by,omitempty"`
}
```

- [ ] **Step 2: Populate `BlockedBy` in `Summary()` method**

In the `Summary()` method (line 235), add `BlockedBy: e.BlockedBy` to the return struct:

```go
func (e *Entry) Summary() EntrySummary {
	return EntrySummary{
		Slug:      e.Slug,
		Title:     e.Title,
		Type:      e.Type,
		Status:    e.Status,
		Priority:  e.Priority,
		Due:       e.Due,
		Tags:      e.Tags,
		Created:   e.Created.Format(time.RFC3339),
		Modified:  e.Modified.Format(time.RFC3339),
		BlockedBy: e.BlockedBy,
	}
}
```

- [ ] **Step 3: Verify existing tests still pass**

Run: `go test ./internal/entry/ -v`
Expected: all existing tests PASS. The `omitempty` tag means existing JSON output is unchanged when `BlockedBy` is empty.

- [ ] **Step 4: Verify build**

Run: `go build ./cmd/jot`
Expected: exit 0

### Task 2: Create `internal/store/stats.go` — types and `Stats()` method

**Files:**
- Create: `internal/store/stats.go`

- [ ] **Step 1: Create stats.go with types**

Create `internal/store/stats.go` with these types:

```go
package store

import (
	"sort"
	"strings"
	"time"

	"github.com/queelius/jot/internal/entry"
)

// ValidSections lists the section names accepted by Stats().
var ValidSections = []string{"summary", "overdue", "blocked", "health", "recent"}

// SummaryStats contains aggregate counts for the filtered entry set.
type SummaryStats struct {
	Total      int            `json:"total"`
	ByType     map[string]int `json:"by_type"`
	ByStatus   map[string]int `json:"by_status"`
	ByPriority map[string]int `json:"by_priority"`
	TagsCount  int            `json:"tags_count"`
}

// TagHealth contains per-tag project health data.
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

// StatsResult contains the computed stats sections.
// Unrequested sections are nil and omitted from JSON via omitempty.
type StatsResult struct {
	Summary *SummaryStats        `json:"summary,omitempty"`
	Overdue []*entry.EntrySummary `json:"overdue,omitempty"`
	Blocked []*entry.EntrySummary `json:"blocked,omitempty"`
	Health  []*TagHealth          `json:"health,omitempty"`
	Recent  []*entry.EntrySummary `json:"recent,omitempty"`
}
```

- [ ] **Step 2: Implement `Stats()` method**

Add the `Stats()` method that calls `s.List()` once and dispatches to section helpers:

```go
// Stats computes journal snapshot data for the requested sections.
// sections is a map of section names to include (e.g., {"summary": true, "overdue": true}).
// staleDays is the threshold for "stale" entries in the health section.
func (s *Store) Stats(f *Filter, sections map[string]bool, staleDays int) (*StatsResult, error) {
	entries, err := s.List(f)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	result := &StatsResult{}

	if sections["summary"] {
		result.Summary = computeSummary(entries)
	}
	if sections["overdue"] {
		result.Overdue = computeOverdue(entries, now)
	}
	if sections["blocked"] {
		result.Blocked = computeBlocked(entries)
	}
	if sections["health"] {
		result.Health = computeHealth(entries, now, staleDays)
	}
	if sections["recent"] {
		result.Recent = computeRecent(entries, now)
	}

	return result, nil
}
```

- [ ] **Step 3: Implement `computeSummary`**

```go
func computeSummary(entries []*entry.Entry) *SummaryStats {
	s := &SummaryStats{
		ByType:     make(map[string]int),
		ByStatus:   make(map[string]int),
		ByPriority: make(map[string]int),
	}

	tags := make(map[string]bool)
	for _, e := range entries {
		s.Total++
		if e.Type != "" {
			s.ByType[e.Type]++
		}
		if e.Status != "" {
			s.ByStatus[e.Status]++
		}
		p := e.Priority
		if p == "" {
			p = "unset"
		}
		s.ByPriority[p]++
		for _, t := range e.Tags {
			tags[t] = true
		}
	}
	s.TagsCount = len(tags)

	return s
}
```

- [ ] **Step 4: Implement `computeOverdue`**

```go
func computeOverdue(entries []*entry.Entry, now time.Time) []*entry.EntrySummary {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var result []*entry.EntrySummary

	for _, e := range entries {
		if e.Due == "" || e.Status == "done" || e.Status == "archived" {
			continue
		}
		due, err := time.Parse("2006-01-02", e.Due)
		if err != nil {
			continue
		}
		if due.Before(today) {
			s := e.Summary()
			result = append(result, &s)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Due < result[j].Due
	})

	return result
}
```

- [ ] **Step 5: Implement `computeBlocked`**

```go
func computeBlocked(entries []*entry.Entry) []*entry.EntrySummary {
	var result []*entry.EntrySummary

	for _, e := range entries {
		if e.Status != "blocked" {
			continue
		}
		s := e.Summary()
		result = append(result, &s)
	}

	// Ascending by modified (least recently touched first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Modified < result[j].Modified
	})

	return result
}
```

- [ ] **Step 6: Implement `computeHealth`**

```go
func computeHealth(entries []*entry.Entry, now time.Time, staleDays int) []*TagHealth {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	staleCutoff := now.Add(-time.Duration(staleDays) * 24 * time.Hour)

	byTag := make(map[string]*TagHealth)

	for _, e := range entries {
		for _, tag := range e.Tags {
			th, ok := byTag[tag]
			if !ok {
				th = &TagHealth{Tag: tag}
				byTag[tag] = th
			}
			th.Total++

			switch e.Status {
			case "open":
				th.Open++
			case "in_progress":
				th.InProgress++
			case "done":
				th.Done++
			case "blocked":
				th.Blocked++
			case "archived":
				th.Archived++
			}

			// Check overdue (same logic as computeOverdue)
			if e.Due != "" && e.Status != "done" && e.Status != "archived" {
				if due, err := time.Parse("2006-01-02", e.Due); err == nil && due.Before(today) {
					th.Overdue++
				}
			}

			// Check stale (active + not modified recently)
			if e.Status != "done" && e.Status != "archived" && e.Modified.Before(staleCutoff) {
				th.Stale++
			}
		}
	}

	result := make([]*TagHealth, 0, len(byTag))
	for _, th := range byTag {
		result = append(result, th)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Tag < result[j].Tag
	})

	return result
}
```

- [ ] **Step 7: Implement `computeRecent`**

```go
func computeRecent(entries []*entry.Entry, now time.Time) []*entry.EntrySummary {
	cutoff := now.Add(-7 * 24 * time.Hour)
	var result []*entry.EntrySummary

	for _, e := range entries {
		if e.Modified.After(cutoff) {
			s := e.Summary()
			result = append(result, &s)
		}
	}

	// Descending by modified (most recent first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Modified > result[j].Modified
	})

	return result
}
```

- [ ] **Step 8: Verify build**

Run: `go build ./cmd/jot`
Expected: exit 0

- [ ] **Step 9: Commit**

```bash
git add internal/entry/entry.go internal/store/stats.go
git commit -m "Add Stats() data layer for jot stats command"
```

---

### Task 3: Unit tests for stats computation

**Files:**
- Create: `internal/store/stats_test.go`

- [ ] **Step 1: Create test file with test entries helper**

Create `internal/store/stats_test.go`:

```go
package store

import (
	"testing"
	"time"

	"github.com/queelius/jot/internal/entry"
)

// makeEntry creates a test entry with the given fields.
func makeEntry(title, typ, status, priority, due string, tags []string, created, modified time.Time) *entry.Entry {
	return &entry.Entry{
		Title:    title,
		Type:     typ,
		Status:   status,
		Priority: priority,
		Due:      due,
		Tags:     tags,
		Created:  created,
		Modified: modified,
		Slug:     entry.GenerateSlug(title, created),
	}
}

// makeBlockedEntry creates a blocked entry with blocked_by set.
func makeBlockedEntry(title, blockedBy string, tags []string, modified time.Time) *entry.Entry {
	e := makeEntry(title, "task", "blocked", "high", "", tags, modified, modified)
	e.BlockedBy = blockedBy
	return e
}
```

- [ ] **Step 2: Write `TestComputeSummary`**

```go
func TestComputeSummary(t *testing.T) {
	now := time.Now()
	entries := []*entry.Entry{
		makeEntry("A", "task", "open", "high", "", []string{"api"}, now, now),
		makeEntry("B", "task", "done", "low", "", []string{"api", "backend"}, now, now),
		makeEntry("C", "idea", "open", "", "", []string{"frontend"}, now, now),
		makeEntry("D", "note", "", "", "", nil, now, now),
	}

	s := computeSummary(entries)

	if s.Total != 4 {
		t.Errorf("total = %d, want 4", s.Total)
	}
	if s.ByType["task"] != 2 {
		t.Errorf("by_type[task] = %d, want 2", s.ByType["task"])
	}
	if s.ByType["idea"] != 1 {
		t.Errorf("by_type[idea] = %d, want 1", s.ByType["idea"])
	}
	if s.ByStatus["open"] != 2 {
		t.Errorf("by_status[open] = %d, want 2", s.ByStatus["open"])
	}
	if s.ByPriority["unset"] != 2 {
		t.Errorf("by_priority[unset] = %d, want 2", s.ByPriority["unset"])
	}
	if s.TagsCount != 3 {
		t.Errorf("tags_count = %d, want 3", s.TagsCount)
	}
	// Zero-count keys should be absent
	if _, ok := s.ByType["plan"]; ok {
		t.Error("by_type should not contain 'plan' with zero count")
	}
}

func TestComputeSummary_Empty(t *testing.T) {
	s := computeSummary(nil)
	if s.Total != 0 {
		t.Errorf("total = %d, want 0", s.Total)
	}
	if s.ByType == nil {
		t.Error("by_type should be empty map, not nil")
	}
	if len(s.ByType) != 0 {
		t.Errorf("by_type should be empty, got %v", s.ByType)
	}
}
```

- [ ] **Step 3: Write `TestComputeOverdue`**

```go
func TestComputeOverdue(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	entries := []*entry.Entry{
		makeEntry("Overdue", "task", "open", "", today.Add(-3*24*time.Hour).Format("2006-01-02"), nil, now, now),
		makeEntry("Due Today", "task", "open", "", today.Format("2006-01-02"), nil, now, now),
		makeEntry("Future", "task", "open", "", today.Add(5*24*time.Hour).Format("2006-01-02"), nil, now, now),
		makeEntry("Done Overdue", "task", "done", "", today.Add(-1*24*time.Hour).Format("2006-01-02"), nil, now, now),
		makeEntry("No Due", "task", "open", "", "", nil, now, now),
		makeEntry("Archived Overdue", "task", "archived", "", today.Add(-2*24*time.Hour).Format("2006-01-02"), nil, now, now),
	}

	result := computeOverdue(entries, now)

	if len(result) != 1 {
		t.Fatalf("got %d overdue, want 1", len(result))
	}
	if result[0].Title != "Overdue" {
		t.Errorf("title = %q, want %q", result[0].Title, "Overdue")
	}
}
```

- [ ] **Step 4: Write `TestComputeBlocked`**

```go
func TestComputeBlocked(t *testing.T) {
	now := time.Now()
	old := now.Add(-48 * time.Hour)

	entries := []*entry.Entry{
		makeBlockedEntry("Old Block", "waiting on review", []string{"auth"}, old),
		makeBlockedEntry("New Block", "", []string{"api"}, now),
		makeEntry("Open Task", "task", "open", "", "", nil, now, now),
	}

	result := computeBlocked(entries)

	if len(result) != 2 {
		t.Fatalf("got %d blocked, want 2", len(result))
	}
	// Should be sorted ascending by modified (old first)
	if result[0].Title != "Old Block" {
		t.Errorf("first = %q, want Old Block (least recently modified)", result[0].Title)
	}
	if result[0].BlockedBy != "waiting on review" {
		t.Errorf("blocked_by = %q, want 'waiting on review'", result[0].BlockedBy)
	}
}
```

- [ ] **Step 5: Write `TestComputeHealth`**

```go
func TestComputeHealth(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	staleTime := now.Add(-60 * 24 * time.Hour)

	entries := []*entry.Entry{
		makeEntry("A", "task", "open", "", today.Add(-3*24*time.Hour).Format("2006-01-02"), []string{"api"}, now, now),          // overdue, active
		makeEntry("B", "task", "done", "", "", []string{"api"}, now, now),                                                        // done
		makeEntry("C", "task", "open", "", "", []string{"api", "backend"}, staleTime, staleTime),                                 // stale (active, old modified)
		makeEntry("D", "idea", "archived", "", "", []string{"backend"}, now, now),                                                // archived
	}

	result := computeHealth(entries, now, 30)

	if len(result) != 2 {
		t.Fatalf("got %d tags, want 2 (api, backend)", len(result))
	}

	// Sorted by tag name: api first, backend second
	api := result[0]
	if api.Tag != "api" {
		t.Fatalf("first tag = %q, want api", api.Tag)
	}
	if api.Total != 3 {
		t.Errorf("api.total = %d, want 3", api.Total)
	}
	if api.Overdue != 1 {
		t.Errorf("api.overdue = %d, want 1", api.Overdue)
	}
	if api.Stale != 1 {
		t.Errorf("api.stale = %d, want 1", api.Stale)
	}
	if api.Done != 1 {
		t.Errorf("api.done = %d, want 1", api.Done)
	}

	backend := result[1]
	if backend.Tag != "backend" {
		t.Fatalf("second tag = %q, want backend", backend.Tag)
	}
	if backend.Archived != 1 {
		t.Errorf("backend.archived = %d, want 1", backend.Archived)
	}
}
```

- [ ] **Step 6: Write `TestComputeRecent`**

```go
func TestComputeRecent(t *testing.T) {
	now := time.Now()
	recent := now.Add(-2 * 24 * time.Hour)
	old := now.Add(-10 * 24 * time.Hour)

	entries := []*entry.Entry{
		makeEntry("Recent", "task", "open", "", "", nil, now, recent),
		makeEntry("Old", "task", "open", "", "", nil, old, old),
		makeEntry("Very Recent", "idea", "", "", "", nil, now, now),
	}

	result := computeRecent(entries, now)

	if len(result) != 2 {
		t.Fatalf("got %d recent, want 2", len(result))
	}
	// Should be sorted descending by modified (most recent first)
	if result[0].Title != "Very Recent" {
		t.Errorf("first = %q, want Very Recent", result[0].Title)
	}
}
```

- [ ] **Step 7: Write `TestStatsSections`**

This test exercises the `Stats()` method's section-selection logic through a real store.

```go
func TestStatsSections(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "entries"), 0755)
	os.MkdirAll(filepath.Join(root, ".jot"), 0755)
	s := New(root)

	now := time.Now()
	e := makeEntry("Test Entry", "task", "open", "high", "", []string{"api"}, now, now)
	e.Slug = entry.GenerateSlug(e.Title, now)
	if err := s.Create(e); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	t.Run("only summary", func(t *testing.T) {
		result, err := s.Stats(nil, map[string]bool{"summary": true}, 30)
		if err != nil {
			t.Fatalf("Stats failed: %v", err)
		}
		if result.Summary == nil {
			t.Error("summary should be populated")
		}
		if result.Overdue != nil {
			t.Error("overdue should be nil when not requested")
		}
		if result.Health != nil {
			t.Error("health should be nil when not requested")
		}
		if result.Recent != nil {
			t.Error("recent should be nil when not requested")
		}
	})

	t.Run("all sections", func(t *testing.T) {
		all := map[string]bool{"summary": true, "overdue": true, "blocked": true, "health": true, "recent": true}
		result, err := s.Stats(nil, all, 30)
		if err != nil {
			t.Fatalf("Stats failed: %v", err)
		}
		if result.Summary == nil {
			t.Error("summary should be populated with --all")
		}
		// Recent should include the entry (just created)
		if len(result.Recent) != 1 {
			t.Errorf("recent = %d, want 1", len(result.Recent))
		}
	})
}
```

- [ ] **Step 8: Write `TestStatsEmpty`**

```go
func TestStatsEmpty(t *testing.T) {
	s := computeSummary(nil)
	if s.Total != 0 {
		t.Errorf("total = %d, want 0", s.Total)
	}
	if s.ByType == nil || len(s.ByType) != 0 {
		t.Errorf("by_type should be non-nil empty map, got %v", s.ByType)
	}
	if s.ByStatus == nil || len(s.ByStatus) != 0 {
		t.Errorf("by_status should be non-nil empty map, got %v", s.ByStatus)
	}

	now := time.Now()
	overdue := computeOverdue(nil, now)
	if overdue != nil {
		t.Errorf("overdue should be nil for empty input, got %v", overdue)
	}
	blocked := computeBlocked(nil)
	if blocked != nil {
		t.Errorf("blocked should be nil for empty input, got %v", blocked)
	}
	health := computeHealth(nil, now, 30)
	if len(health) != 0 {
		t.Errorf("health should be empty for empty input, got %v", health)
	}
	recent := computeRecent(nil, now)
	if recent != nil {
		t.Errorf("recent should be nil for empty input, got %v", recent)
	}
}
```

Note: add `"os"` and `"path/filepath"` to the imports at the top of `stats_test.go`.

- [ ] **Step 9: Run all store tests**

Run: `go test ./internal/store/ -v`
Expected: all PASS

- [ ] **Step 9: Commit**

```bash
git add internal/store/stats_test.go
git commit -m "Add unit tests for stats computation"
```

---

## Chunk 2: Command Layer and Integration

### Task 4: Create `internal/commands/stats.go` — Cobra command

**Files:**
- Create: `internal/commands/stats.go`

- [ ] **Step 1: Create stats.go with command and flags**

Create `internal/commands/stats.go`:

```go
package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/queelius/jot/internal/store"
)

var (
	statsSection  string
	statsAll      bool
	statsStaleDays int
	statsType     string
	statsTag      string
	statsStatus   string
	statsPriority string
	statsSince    string
	statsUntil    string
	statsLimit    int
)

var statsCmd = &cobra.Command{
	Use:     "stats",
	Short:   "Journal snapshot data (JSON)",
	GroupID: "query",
	Long: `Compute journal snapshot data as JSON for LLM consumption.

Outputs pre-computed sections: summary, overdue, blocked, health, recent.
Default sections: summary, overdue, blocked. Use --all for everything.

Always outputs JSON regardless of --table/--markdown flags.

Examples:
  jot stats                              # default sections
  jot stats --all                        # all sections
  jot stats --section=health             # specific section
  jot stats --section=summary,health     # multiple sections
  jot stats --tags=myproject             # scoped to a tag
  jot stats --type=task --status=open    # scoped by type/status`,
	RunE: runStats,
}

func init() {
	statsCmd.Flags().StringVar(&statsSection, "section", "", "sections to include (comma-separated: summary,overdue,blocked,health,recent)")
	statsCmd.Flags().BoolVar(&statsAll, "all", false, "include all sections")
	statsCmd.Flags().IntVar(&statsStaleDays, "stale-days", 30, "days without modification to consider stale (for health section)")
	statsCmd.Flags().StringVarP(&statsType, "type", "t", "", "filter by type")
	statsCmd.Flags().StringVar(&statsTag, "tags", "", "filter by tag")
	statsCmd.Flags().StringVarP(&statsStatus, "status", "s", "", "filter by status")
	statsCmd.Flags().StringVarP(&statsPriority, "priority", "p", "", "filter by priority")
	statsCmd.Flags().StringVar(&statsSince, "since", "", "entries created since (e.g., 7d, 2w, 2024-01-01)")
	statsCmd.Flags().StringVar(&statsUntil, "until", "", "entries created until")
	statsCmd.Flags().IntVarP(&statsLimit, "limit", "n", 0, "limit initial entry set")

	rootCmd.AddCommand(statsCmd)
}

// parseSections parses the --section flag and --all into a map of section names.
func parseSections(sectionFlag string, all bool) (map[string]bool, error) {
	if all {
		sections := make(map[string]bool)
		for _, s := range store.ValidSections {
			sections[s] = true
		}
		return sections, nil
	}

	if sectionFlag == "" {
		return map[string]bool{"summary": true, "overdue": true, "blocked": true}, nil
	}

	valid := make(map[string]bool)
	for _, s := range store.ValidSections {
		valid[s] = true
	}

	sections := make(map[string]bool)
	for _, s := range strings.Split(sectionFlag, ",") {
		name := strings.TrimSpace(strings.ToLower(s))
		if name == "" {
			continue
		}
		if !valid[name] {
			return nil, fmt.Errorf("unknown section %q, valid sections: %s", name, strings.Join(store.ValidSections, ", "))
		}
		sections[name] = true
	}

	if len(sections) == 0 {
		return nil, fmt.Errorf("no valid sections specified")
	}

	return sections, nil
}

func runStats(cmd *cobra.Command, args []string) error {
	sections, err := parseSections(statsSection, statsAll)
	if err != nil {
		return err
	}

	s, err := getStore()
	if err != nil {
		return err
	}

	filter := &store.Filter{
		Type:     statsType,
		Tag:      statsTag,
		Status:   statsStatus,
		Priority: statsPriority,
		Limit:    statsLimit,
		Fuzzy:    getFuzzy(),
	}

	if statsSince != "" {
		if dur, err := store.ParseDuration(statsSince); err == nil && dur > 0 {
			filter.Since = time.Now().Add(-dur)
		} else if t, err := store.ParseDate(statsSince); err == nil {
			filter.Since = t
		}
	}
	if statsUntil != "" {
		if t, err := store.ParseDate(statsUntil); err == nil {
			filter.Until = t
		}
	}

	result, err := s.Stats(filter, sections, statsStaleDays)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./cmd/jot`
Expected: exit 0

- [ ] **Step 3: Verify `TestCommandGroups` passes**

Run: `go test -run TestCommandGroups ./internal/commands -v`
Expected: PASS (stats command has GroupID "query")

- [ ] **Step 4: Verify CLI help**

Run: `./jot stats --help`
Expected: shows usage with --section, --all, --stale-days, filter flags

- [ ] **Step 5: Commit**

```bash
git add internal/commands/stats.go
git commit -m "Add jot stats command with section selection and filtering"
```

### Task 5: Unit tests for `parseSections`

**Files:**
- Create tests in `internal/commands/stats_test.go` (new file, or append to `commands_test.go`)

- [ ] **Step 1: Write `TestParseSections`**

Create `internal/commands/stats_test.go`:

```go
package commands

import (
	"testing"
)

func TestParseSections(t *testing.T) {
	tests := []struct {
		name      string
		section   string
		all       bool
		wantCount int
		wantErr   bool
	}{
		{"defaults", "", false, 3, false},
		{"all flag", "", true, 5, false},
		{"single section", "health", false, 1, false},
		{"multiple sections", "summary,health", false, 2, false},
		{"case insensitive", "SUMMARY,Health", false, 2, false},
		{"with spaces", " summary , health ", false, 2, false},
		{"unknown section", "invalid", false, 0, true},
		{"mixed valid and invalid", "summary,bogus", false, 0, true},
		{"empty after split", ",,,", false, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSections(tt.section, tt.all)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != tt.wantCount {
				t.Errorf("got %d sections, want %d: %v", len(result), tt.wantCount, result)
			}
		})
	}

	t.Run("defaults include correct sections", func(t *testing.T) {
		result, _ := parseSections("", false)
		for _, name := range []string{"summary", "overdue", "blocked"} {
			if !result[name] {
				t.Errorf("default sections missing %q", name)
			}
		}
	})

	t.Run("all includes all sections", func(t *testing.T) {
		result, _ := parseSections("", true)
		for _, name := range []string{"summary", "overdue", "blocked", "health", "recent"} {
			if !result[name] {
				t.Errorf("--all missing %q", name)
			}
		}
	})
}
```

- [ ] **Step 2: Run tests**

Run: `go test -run TestParseSections ./internal/commands -v`
Expected: all PASS

- [ ] **Step 3: Commit**

```bash
git add internal/commands/stats_test.go
git commit -m "Add unit tests for parseSections"
```

### Task 6: Integration test

**Files:**
- Modify: `internal/commands/integration_test.go`

- [ ] **Step 1: Add `TestStoreIntegration_Stats` to integration_test.go**

Add at the end of `internal/commands/integration_test.go`:

```go
// TestStoreIntegration_Stats tests stats computation with real entries through the store.
func TestStoreIntegration_Stats(t *testing.T) {
	s, _ := setupTestJournal(t)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Create diverse entries
	createTestEntry(t, s, "Open Task", "task", "open", "high", today.Add(-2*24*time.Hour).Format("2006-01-02"), []string{"api"})
	createTestEntry(t, s, "Done Task", "task", "done", "low", "", []string{"api", "backend"})
	createTestEntry(t, s, "Blocked Task", "task", "blocked", "critical", "", []string{"api"})
	createTestEntry(t, s, "Open Idea", "idea", "open", "", "", []string{"frontend"})
	createTestEntry(t, s, "Note", "note", "", "", "", nil)

	sections := map[string]bool{
		"summary": true,
		"overdue": true,
		"blocked": true,
	}

	result, err := s.Stats(nil, sections, 30)
	if err != nil {
		t.Fatalf("Stats() failed: %v", err)
	}

	// Verify summary
	if result.Summary == nil {
		t.Fatal("summary should be populated")
	}
	if result.Summary.Total != 5 {
		t.Errorf("total = %d, want 5", result.Summary.Total)
	}
	if result.Summary.ByType["task"] != 3 {
		t.Errorf("by_type[task] = %d, want 3", result.Summary.ByType["task"])
	}

	// Verify overdue (Open Task has past due date)
	if len(result.Overdue) != 1 {
		t.Errorf("overdue count = %d, want 1", len(result.Overdue))
	}

	// Verify blocked
	if len(result.Blocked) != 1 {
		t.Errorf("blocked count = %d, want 1", len(result.Blocked))
	}

	// Verify unrequested sections are nil
	if result.Health != nil {
		t.Error("health should be nil when not requested")
	}
	if result.Recent != nil {
		t.Error("recent should be nil when not requested")
	}
}
```

- [ ] **Step 2: Run all tests**

Run: `go test -count=1 ./... `
Expected: all packages PASS

- [ ] **Step 3: Run `go vet`**

Run: `go vet ./...`
Expected: exit 0

- [ ] **Step 4: Commit**

```bash
git add internal/commands/integration_test.go
git commit -m "Add integration test for jot stats"
```

### Task 7: Final verification

- [ ] **Step 1: Full fresh test run**

Run: `go test -count=1 ./...`
Expected: all packages PASS, exit 0

- [ ] **Step 2: Build binary**

Run: `go build -o jot ./cmd/jot`
Expected: exit 0

- [ ] **Step 3: Verify CLI**

Run: `./jot stats --help`
Expected: shows usage, grouped under "View and Query"

- [ ] **Step 4: Tag, push, release**

Per user's release workflow.
