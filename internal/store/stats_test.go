package store

import (
	"os"
	"path/filepath"
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

func TestComputeHealth(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	staleTime := now.Add(-60 * 24 * time.Hour)

	entries := []*entry.Entry{
		makeEntry("A", "task", "open", "", today.Add(-3*24*time.Hour).Format("2006-01-02"), []string{"api"}, now, now),
		makeEntry("B", "task", "done", "", "", []string{"api"}, now, now),
		makeEntry("C", "task", "open", "", "", []string{"api", "backend"}, staleTime, staleTime),
		makeEntry("D", "idea", "archived", "", "", []string{"backend"}, now, now),
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
	if s.ByPriority == nil || len(s.ByPriority) != 0 {
		t.Errorf("by_priority should be non-nil empty map, got %v", s.ByPriority)
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
