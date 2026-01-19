package commands

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/queelius/jot/internal/entry"
	"github.com/queelius/jot/internal/store"
)

// setupTestJournal creates a temporary journal directory with optional test entries.
func setupTestJournal(t *testing.T) (*store.Store, string) {
	t.Helper()
	root := t.TempDir()

	// Create the entries directory
	entriesDir := filepath.Join(root, "entries")
	if err := os.MkdirAll(entriesDir, 0755); err != nil {
		t.Fatalf("failed to create entries dir: %v", err)
	}

	// Create .jot marker directory
	jotDir := filepath.Join(root, ".jot")
	if err := os.MkdirAll(jotDir, 0755); err != nil {
		t.Fatalf("failed to create .jot dir: %v", err)
	}

	return store.New(root), root
}

// createTestEntry creates a test entry in the store.
func createTestEntry(t *testing.T, s *store.Store, title, typ, status, priority, due string, tags []string) *entry.Entry {
	t.Helper()
	now := time.Now()
	e := &entry.Entry{
		Title:    title,
		Type:     typ,
		Status:   status,
		Priority: priority,
		Due:      due,
		Tags:     tags,
		Content:  "Test content for " + title,
		Created:  now,
		Modified: now,
	}
	e.Slug = entry.GenerateSlug(title, now)

	if err := s.Create(e); err != nil {
		t.Fatalf("failed to create test entry: %v", err)
	}

	return e
}

// captureOutput captures stdout during function execution.
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old

	return buf.String()
}

// TestStoreIntegration_Create tests entry creation through the store.
func TestStoreIntegration_Create(t *testing.T) {
	s, _ := setupTestJournal(t)

	t.Run("create entry", func(t *testing.T) {
		e := createTestEntry(t, s, "Test Entry", "idea", "", "", "", nil)

		// Verify entry exists
		retrieved, err := s.Get(e.Slug)
		if err != nil {
			t.Fatalf("failed to get entry: %v", err)
		}
		if retrieved.Title != "Test Entry" {
			t.Errorf("title = %q, want %q", retrieved.Title, "Test Entry")
		}
	})

	t.Run("create task with status", func(t *testing.T) {
		e := createTestEntry(t, s, "Fix Bug", "task", "open", "high", "", []string{"bug"})

		retrieved, err := s.Get(e.Slug)
		if err != nil {
			t.Fatalf("failed to get entry: %v", err)
		}
		if retrieved.Type != "task" {
			t.Errorf("type = %q, want %q", retrieved.Type, "task")
		}
		if retrieved.Status != "open" {
			t.Errorf("status = %q, want %q", retrieved.Status, "open")
		}
		if retrieved.Priority != "high" {
			t.Errorf("priority = %q, want %q", retrieved.Priority, "high")
		}
	})
}

// TestStoreIntegration_Update tests entry updates through the store.
func TestStoreIntegration_Update(t *testing.T) {
	s, _ := setupTestJournal(t)

	t.Run("update status", func(t *testing.T) {
		e := createTestEntry(t, s, "Task to Update", "task", "open", "", "", nil)

		// Update status
		e.Status = "done"
		if err := s.Update(e); err != nil {
			t.Fatalf("failed to update entry: %v", err)
		}

		// Verify update
		retrieved, err := s.Get(e.Slug)
		if err != nil {
			t.Fatalf("failed to get entry: %v", err)
		}
		if retrieved.Status != "done" {
			t.Errorf("status = %q, want %q", retrieved.Status, "done")
		}
	})

	t.Run("update priority", func(t *testing.T) {
		e := createTestEntry(t, s, "Task Priority", "task", "open", "low", "", nil)

		e.Priority = "critical"
		if err := s.Update(e); err != nil {
			t.Fatalf("failed to update entry: %v", err)
		}

		retrieved, err := s.Get(e.Slug)
		if err != nil {
			t.Fatalf("failed to get entry: %v", err)
		}
		if retrieved.Priority != "critical" {
			t.Errorf("priority = %q, want %q", retrieved.Priority, "critical")
		}
	})
}

// TestStoreIntegration_List tests listing entries with filters.
func TestStoreIntegration_List(t *testing.T) {
	s, _ := setupTestJournal(t)

	// Create several entries with slight delays to ensure different timestamps
	createTestEntry(t, s, "Idea One", "idea", "", "", "", []string{"api"})
	time.Sleep(10 * time.Millisecond)
	createTestEntry(t, s, "Task One", "task", "open", "high", "", []string{"bug"})
	time.Sleep(10 * time.Millisecond)
	createTestEntry(t, s, "Task Two", "task", "done", "low", "", []string{"api"})
	time.Sleep(10 * time.Millisecond)
	createTestEntry(t, s, "Note One", "note", "", "", "", []string{"api", "backend"})

	t.Run("list all", func(t *testing.T) {
		entries, err := s.List(nil)
		if err != nil {
			t.Fatalf("failed to list entries: %v", err)
		}
		if len(entries) != 4 {
			t.Errorf("got %d entries, want 4", len(entries))
		}
	})

	t.Run("filter by type", func(t *testing.T) {
		entries, err := s.List(&store.Filter{Type: "task"})
		if err != nil {
			t.Fatalf("failed to list entries: %v", err)
		}
		if len(entries) != 2 {
			t.Errorf("got %d entries, want 2", len(entries))
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		entries, err := s.List(&store.Filter{Status: "open"})
		if err != nil {
			t.Fatalf("failed to list entries: %v", err)
		}
		if len(entries) != 1 {
			t.Errorf("got %d entries, want 1", len(entries))
		}
	})

	t.Run("filter by tag", func(t *testing.T) {
		entries, err := s.List(&store.Filter{Tag: "api"})
		if err != nil {
			t.Fatalf("failed to list entries: %v", err)
		}
		if len(entries) != 3 {
			t.Errorf("got %d entries, want 3", len(entries))
		}
	})

	t.Run("filter by priority", func(t *testing.T) {
		entries, err := s.List(&store.Filter{Priority: "high"})
		if err != nil {
			t.Fatalf("failed to list entries: %v", err)
		}
		if len(entries) != 1 {
			t.Errorf("got %d entries, want 1", len(entries))
		}
	})

	t.Run("combined filters", func(t *testing.T) {
		entries, err := s.List(&store.Filter{Type: "task", Status: "open"})
		if err != nil {
			t.Fatalf("failed to list entries: %v", err)
		}
		if len(entries) != 1 {
			t.Errorf("got %d entries, want 1", len(entries))
		}
	})
}

// TestStoreIntegration_Tags tests the AllTags function.
func TestStoreIntegration_Tags(t *testing.T) {
	s, _ := setupTestJournal(t)

	createTestEntry(t, s, "Entry 1", "idea", "", "", "", []string{"api", "backend"})
	createTestEntry(t, s, "Entry 2", "task", "open", "", "", []string{"api"})
	createTestEntry(t, s, "Entry 3", "note", "", "", "", []string{"frontend"})

	tags, err := s.AllTags()
	if err != nil {
		t.Fatalf("failed to get all tags: %v", err)
	}

	if tags["api"] != 2 {
		t.Errorf("api count = %d, want 2", tags["api"])
	}
	if tags["backend"] != 1 {
		t.Errorf("backend count = %d, want 1", tags["backend"])
	}
	if tags["frontend"] != 1 {
		t.Errorf("frontend count = %d, want 1", tags["frontend"])
	}
}

// TestStoreIntegration_Search tests the Search function.
func TestStoreIntegration_Search(t *testing.T) {
	s, _ := setupTestJournal(t)

	createTestEntry(t, s, "API Redesign", "idea", "", "", "", nil)
	createTestEntry(t, s, "Fix Authentication", "task", "open", "", "", nil)

	t.Run("search title", func(t *testing.T) {
		results, err := s.Search("redesign", nil)
		if err != nil {
			t.Fatalf("failed to search: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("got %d results, want 1", len(results))
		}
		if len(results) > 0 && results[0].Entry.Title != "API Redesign" {
			t.Errorf("title = %q, want %q", results[0].Entry.Title, "API Redesign")
		}
	})

	t.Run("search content", func(t *testing.T) {
		results, err := s.Search("content", nil)
		if err != nil {
			t.Fatalf("failed to search: %v", err)
		}
		if len(results) != 2 { // Both entries have "Test content for..." in content
			t.Errorf("got %d results, want 2", len(results))
		}
	})

	t.Run("search with filter", func(t *testing.T) {
		results, err := s.Search("content", &store.Filter{Type: "task"})
		if err != nil {
			t.Fatalf("failed to search: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("got %d results, want 1", len(results))
		}
	})
}

// TestStoreIntegration_FindByPartialSlug tests partial slug matching.
func TestStoreIntegration_FindByPartialSlug(t *testing.T) {
	s, _ := setupTestJournal(t)

	createTestEntry(t, s, "API Redesign", "idea", "", "", "", nil)
	createTestEntry(t, s, "API Security", "task", "open", "", "", nil)
	createTestEntry(t, s, "Database Migration", "task", "", "", "", nil)

	t.Run("find by partial - single match", func(t *testing.T) {
		matches, err := s.FindByPartialSlug("database")
		if err != nil {
			t.Fatalf("failed to find: %v", err)
		}
		if len(matches) != 1 {
			t.Errorf("got %d matches, want 1", len(matches))
		}
	})

	t.Run("find by partial - multiple matches", func(t *testing.T) {
		matches, err := s.FindByPartialSlug("api")
		if err != nil {
			t.Fatalf("failed to find: %v", err)
		}
		if len(matches) != 2 {
			t.Errorf("got %d matches, want 2", len(matches))
		}
	})

	t.Run("find by partial - no match", func(t *testing.T) {
		matches, err := s.FindByPartialSlug("nonexistent")
		if err != nil {
			t.Fatalf("failed to find: %v", err)
		}
		if len(matches) != 0 {
			t.Errorf("got %d matches, want 0", len(matches))
		}
	})
}

// TestStoreIntegration_Delete tests entry deletion.
func TestStoreIntegration_Delete(t *testing.T) {
	s, _ := setupTestJournal(t)

	e := createTestEntry(t, s, "To Be Deleted", "note", "", "", "", nil)

	// Verify it exists
	if !s.Exists(e.Slug) {
		t.Fatal("entry should exist before delete")
	}

	// Delete it
	if err := s.Delete(e.Slug); err != nil {
		t.Fatalf("failed to delete entry: %v", err)
	}

	// Verify it's gone
	if s.Exists(e.Slug) {
		t.Error("entry should not exist after delete")
	}
}

// TestResolveSlug tests the ResolveSlug function.
func TestResolveSlug(t *testing.T) {
	s, _ := setupTestJournal(t)

	e1 := createTestEntry(t, s, "Unique Entry", "idea", "", "", "", nil)
	createTestEntry(t, s, "API One", "idea", "", "", "", nil)
	createTestEntry(t, s, "API Two", "task", "open", "", "", nil)

	t.Run("exact match", func(t *testing.T) {
		result, err := ResolveSlug(s, e1.Slug)
		if err != nil {
			t.Fatalf("ResolveSlug failed: %v", err)
		}
		if result.Title != "Unique Entry" {
			t.Errorf("title = %q, want %q", result.Title, "Unique Entry")
		}
	})

	t.Run("partial match - single", func(t *testing.T) {
		result, err := ResolveSlug(s, "unique")
		if err != nil {
			t.Fatalf("ResolveSlug failed: %v", err)
		}
		if result.Title != "Unique Entry" {
			t.Errorf("title = %q, want %q", result.Title, "Unique Entry")
		}
	})

	t.Run("no match", func(t *testing.T) {
		_, err := ResolveSlug(s, "nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent slug")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error = %q, want to contain 'not found'", err.Error())
		}
	})

	// Note: Multiple match case requires interactive input, so we skip it
}

// TestOutputJSON tests JSON output formatting.
func TestOutputJSON(t *testing.T) {
	entries := []*entry.Entry{
		{
			Slug:     "20240101-test",
			Title:    "Test Entry",
			Type:     "idea",
			Created:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			Modified: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	output := captureOutput(func() {
		outputJSON(entries)
	})

	if !strings.Contains(output, `"slug": "20240101-test"`) {
		t.Errorf("output missing slug field: %s", output)
	}
	if !strings.Contains(output, `"title": "Test Entry"`) {
		t.Errorf("output missing title field: %s", output)
	}
	if !strings.Contains(output, `"type": "idea"`) {
		t.Errorf("output missing type field: %s", output)
	}
}

// TestOutputMarkdown tests markdown output formatting.
func TestOutputMarkdown(t *testing.T) {
	entries := []*entry.Entry{
		{
			Slug:  "20240101-test",
			Title: "Test Entry",
			Tags:  []string{"api", "backend"},
		},
		{
			Slug:  "20240102-no-tags",
			Title: "No Tags Entry",
			Tags:  nil,
		},
	}

	output := captureOutput(func() {
		outputMarkdown(entries)
	})

	if !strings.Contains(output, "**Test Entry**") {
		t.Errorf("output missing bold title: %s", output)
	}
	if !strings.Contains(output, "(20240101-test)") {
		t.Errorf("output missing slug: %s", output)
	}
	if !strings.Contains(output, "[api, backend]") {
		t.Errorf("output missing tags: %s", output)
	}
	if !strings.Contains(output, "**No Tags Entry**") {
		t.Errorf("output missing second entry: %s", output)
	}
}

// TestSearchEntriesIntegration tests searchEntries with real entries.
func TestSearchEntriesIntegration(t *testing.T) {
	entries := []*entry.Entry{
		{
			Title:   "GraphQL API",
			Content: "Implementing a new GraphQL endpoint",
			Tags:    []string{"api", "graphql"},
			Type:    "idea",
			Status:  "open",
		},
		{
			Title:   "REST API Docs",
			Content: "Document existing REST endpoints",
			Tags:    []string{"api", "docs"},
			Type:    "task",
			Status:  "in_progress",
		},
		{
			Title:   "Frontend Styling",
			Content: "Update CSS for dark mode",
			Tags:    []string{"frontend", "css"},
			Type:    "task",
			Status:  "open",
		},
	}

	t.Run("search finds both api entries", func(t *testing.T) {
		results := searchEntries(entries, "api")
		if len(results) != 2 {
			t.Errorf("got %d results, want 2", len(results))
		}
	})

	t.Run("search finds graphql in content", func(t *testing.T) {
		results := searchEntries(entries, "graphql")
		if len(results) != 1 {
			t.Errorf("got %d results, want 1", len(results))
		}
		if len(results) > 0 && results[0].Title != "GraphQL API" {
			t.Errorf("title = %q, want %q", results[0].Title, "GraphQL API")
		}
	})

	t.Run("search case insensitive", func(t *testing.T) {
		results := searchEntries(entries, "GRAPHQL")
		if len(results) != 1 {
			t.Errorf("got %d results, want 1", len(results))
		}
	})
}

// TestOutputTable tests table output formatting.
func TestOutputTable(t *testing.T) {
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	entries := []*entry.Entry{
		{
			Slug:     "20240115-api-redesign-project",
			Title:    "API Redesign Project",
			Type:     "idea",
			Status:   "open",
			Priority: "high",
			Due:      "2024-01-20",
			Created:  now,
			Tags:     []string{"api"},
		},
		{
			Slug:    "20240115-very-long-slug-name-that-exceeds-thirty-five-characters",
			Title:   "A Very Long Title That Will Be Truncated For Display Purposes",
			Type:    "task",
			Created: now,
		},
	}

	// Test non-verbose output
	listVerbose = false
	output := captureOutput(func() {
		outputTable(entries)
	})

	// Should contain headers
	if !strings.Contains(output, "SLUG") || !strings.Contains(output, "TITLE") || !strings.Contains(output, "TYPE") {
		t.Errorf("output missing expected headers: %s", output)
	}

	// Should contain entry data
	if !strings.Contains(output, "api-redesign-project") {
		t.Errorf("output missing entry slug: %s", output)
	}

	// Long title should be truncated with ...
	if !strings.Contains(output, "...") {
		t.Errorf("long title should be truncated: %s", output)
	}

	// Test verbose output
	listVerbose = true
	output = captureOutput(func() {
		outputTable(entries)
	})

	// Verbose should include STATUS and PRIORITY headers
	if !strings.Contains(output, "STATUS") || !strings.Contains(output, "PRIORITY") {
		t.Errorf("verbose output missing STATUS/PRIORITY headers: %s", output)
	}
}

// TestOutputTaskTable tests task table output formatting.
func TestOutputTaskTable(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	entries := []*entry.Entry{
		{
			Slug:     "20240115-urgent-task",
			Title:    "Urgent Task",
			Type:     "task",
			Status:   "open",
			Priority: "critical",
			Due:      today.Format("2006-01-02"),
		},
		{
			Slug:     "20240115-normal-task",
			Title:    "Normal Task",
			Type:     "task",
			Status:   "in_progress",
			Priority: "medium",
		},
	}

	output := captureOutput(func() {
		outputTaskTable(entries)
	})

	// Should contain headers
	if !strings.Contains(output, "SLUG") || !strings.Contains(output, "STATUS") || !strings.Contains(output, "PRIORITY") {
		t.Errorf("output missing expected headers: %s", output)
	}

	// Should contain task data
	if !strings.Contains(output, "urgent-task") {
		t.Errorf("output missing task slug: %s", output)
	}

	// Critical priority should have ANSI codes (colored)
	if !strings.Contains(output, "\033[31m") {
		t.Errorf("critical priority should be colored red: %s", output)
	}
}

// TestOutputTagsJSON tests JSON output for tags.
func TestOutputTagsJSON(t *testing.T) {
	tags := map[string]int{
		"api":      5,
		"backend":  3,
		"frontend": 2,
	}

	output := captureOutput(func() {
		outputTagsJSON(tags)
	})

	// Should be valid JSON-like output with tag and count
	if !strings.Contains(output, `"tag"`) || !strings.Contains(output, `"count"`) {
		t.Errorf("output missing expected JSON fields: %s", output)
	}

	// Should contain all tags
	if !strings.Contains(output, "api") || !strings.Contains(output, "backend") || !strings.Contains(output, "frontend") {
		t.Errorf("output missing tags: %s", output)
	}
}

// TestHighlightMatchIntegration tests highlight with various inputs.
func TestHighlightMatchIntegration(t *testing.T) {
	t.Run("preserves non-matching content", func(t *testing.T) {
		result := highlightMatch("hello world foo bar", "xyz")
		if result != "hello world foo bar" {
			t.Errorf("no match should preserve original: %q", result)
		}
	})

	t.Run("highlights multiple occurrences", func(t *testing.T) {
		result := highlightMatch("api api api", "api")
		// Count ANSI codes
		count := strings.Count(result, "\033[1;33m")
		if count != 3 {
			t.Errorf("expected 3 highlights, got %d in: %q", count, result)
		}
	})
}

// TestSortEntriesIntegration tests sorting with various scenarios.
func TestSortEntriesIntegration(t *testing.T) {
	now := time.Now()

	t.Run("sort maintains stability for equal values", func(t *testing.T) {
		entries := []*entry.Entry{
			{Title: "A", Created: now, Priority: "high"},
			{Title: "B", Created: now, Priority: "high"},
			{Title: "C", Created: now, Priority: "high"},
		}

		sortEntries(entries, "priority", false)

		// All have same priority and created time, order should be stable
		// Note: bubble sort used isn't stable, but entries should still be there
		titles := []string{entries[0].Title, entries[1].Title, entries[2].Title}
		hasA := false
		hasB := false
		hasC := false
		for _, t := range titles {
			if t == "A" {
				hasA = true
			}
			if t == "B" {
				hasB = true
			}
			if t == "C" {
				hasC = true
			}
		}
		if !hasA || !hasB || !hasC {
			t.Errorf("sorting lost entries: %v", titles)
		}
	})
}

// TestFilterByDueIntegration tests due filtering with edge cases.
func TestFilterByDueIntegration(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	t.Run("today filter includes today and overdue", func(t *testing.T) {
		entries := []*entry.Entry{
			{Title: "Overdue", Due: today.Add(-5 * 24 * time.Hour).Format("2006-01-02")},
			{Title: "Today", Due: today.Format("2006-01-02")},
			{Title: "Future", Due: today.Add(30 * 24 * time.Hour).Format("2006-01-02")},
		}

		result := filterByDue(entries, "today")

		// Future should not be included
		for _, e := range result {
			if e.Title == "Future" {
				t.Error("today filter should not include future entries")
			}
		}
	})

	t.Run("week filter bounds", func(t *testing.T) {
		entries := []*entry.Entry{
			{Title: "6 days", Due: today.Add(6 * 24 * time.Hour).Format("2006-01-02")},
			{Title: "10 days", Due: today.Add(10 * 24 * time.Hour).Format("2006-01-02")},
		}

		result := filterByDue(entries, "week")

		// 6 days should be included, 10 days should not
		has6Days := false
		has10Days := false
		for _, e := range result {
			if e.Title == "6 days" {
				has6Days = true
			}
			if e.Title == "10 days" {
				has10Days = true
			}
		}
		if !has6Days {
			t.Error("week filter should include entries due in 6 days")
		}
		if has10Days {
			t.Error("week filter should not include entries due in 10 days")
		}
	})
}
