package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/queelius/jot/internal/entry"
)

func TestStore_CreateAndGet(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	// Create entries directory
	entriesDir := filepath.Join(root, "entries")
	if err := os.MkdirAll(entriesDir, 0755); err != nil {
		t.Fatalf("failed to create entries dir: %v", err)
	}

	now := time.Now()
	e := &entry.Entry{
		Title:    "Test Entry",
		Type:     "note",
		Content:  "Test content",
		Created:  now,
		Modified: now,
		Slug:     entry.GenerateSlug("Test Entry", now),
	}

	// Create entry
	if err := s.Create(e); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get entry
	got, err := s.Get(e.Slug)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Title != e.Title {
		t.Errorf("Title = %q, want %q", got.Title, e.Title)
	}
	if got.Type != e.Type {
		t.Errorf("Type = %q, want %q", got.Type, e.Type)
	}
}

func TestStore_CreateDuplicate(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	now := time.Now()
	e := &entry.Entry{
		Title:    "Test Entry",
		Type:     "note",
		Content:  "Test content",
		Created:  now,
		Modified: now,
		Slug:     entry.GenerateSlug("Test Entry", now),
	}

	// Create first entry
	if err := s.Create(e); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Try to create duplicate
	err := s.Create(e)
	if err == nil {
		t.Error("Create() should fail for duplicate entry")
	}
}

func TestStore_Update(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	now := time.Now()
	e := &entry.Entry{
		Title:    "Original Title",
		Type:     "note",
		Content:  "Original content",
		Created:  now,
		Modified: now,
		Slug:     entry.GenerateSlug("Original Title", now),
	}

	if err := s.Create(e); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update entry
	e.Title = "Updated Title"
	e.Content = "Updated content"
	if err := s.Update(e); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	got, err := s.Get(e.Slug)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", got.Title, "Updated Title")
	}
}

func TestStore_Delete(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	now := time.Now()
	e := &entry.Entry{
		Title:    "To Delete",
		Type:     "note",
		Content:  "Content",
		Created:  now,
		Modified: now,
		Slug:     entry.GenerateSlug("To Delete", now),
	}

	if err := s.Create(e); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify exists
	if !s.Exists(e.Slug) {
		t.Error("Entry should exist before delete")
	}

	// Delete
	if err := s.Delete(e.Slug); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deleted
	if s.Exists(e.Slug) {
		t.Error("Entry should not exist after delete")
	}
}

func TestStore_List(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	now := time.Now()

	// Create multiple entries
	entries := []*entry.Entry{
		{Title: "Entry 1", Type: "note", Created: now, Modified: now},
		{Title: "Entry 2", Type: "task", Status: "open", Created: now.Add(-time.Hour), Modified: now},
		{Title: "Entry 3", Type: "task", Status: "done", Created: now.Add(-2 * time.Hour), Modified: now},
	}

	for _, e := range entries {
		e.Slug = entry.GenerateSlug(e.Title, e.Created)
		if err := s.Create(e); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// List all
	got, err := s.List(nil)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 3 {
		t.Errorf("List() returned %d entries, want 3", len(got))
	}

	// List with filter
	got, err = s.List(&Filter{Type: "task"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 2 {
		t.Errorf("List(type=task) returned %d entries, want 2", len(got))
	}

	// List with status filter
	got, err = s.List(&Filter{Status: "open"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 1 {
		t.Errorf("List(status=open) returned %d entries, want 1", len(got))
	}
}

func TestStore_ListEmpty(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	// List from empty store (no entries dir)
	got, err := s.List(nil)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("List() returned %d entries, want 0", len(got))
	}
}

func TestStore_Search(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	now := time.Now()

	entries := []*entry.Entry{
		{Title: "API Design", Type: "note", Content: "REST API implementation details", Created: now, Modified: now},
		{Title: "Database Schema", Type: "note", Content: "PostgreSQL schema design", Created: now, Modified: now},
		{Title: "API Testing", Type: "task", Content: "Write API tests", Created: now, Modified: now},
	}

	for _, e := range entries {
		e.Slug = entry.GenerateSlug(e.Title, e.Created)
		if err := s.Create(e); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Search for "API"
	results, err := s.Search("API", nil)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Search(API) returned %d results, want 2", len(results))
	}

	// Search for "postgres"
	results, err = s.Search("postgres", nil)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Search(postgres) returned %d results, want 1", len(results))
	}
}

func TestStore_FindByPartialSlug(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	now := time.Now()

	entries := []*entry.Entry{
		{Title: "Fix Login Bug", Type: "task", Created: now, Modified: now},
		{Title: "Review Login Code", Type: "task", Created: now, Modified: now},
		{Title: "Database Migration", Type: "note", Created: now, Modified: now},
	}

	for _, e := range entries {
		e.Slug = entry.GenerateSlug(e.Title, e.Created)
		if err := s.Create(e); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Search for "login"
	matches, err := s.FindByPartialSlug("login")
	if err != nil {
		t.Fatalf("FindByPartialSlug() error = %v", err)
	}
	if len(matches) != 2 {
		t.Errorf("FindByPartialSlug(login) returned %d matches, want 2", len(matches))
	}

	// Search for "database"
	matches, err = s.FindByPartialSlug("database")
	if err != nil {
		t.Fatalf("FindByPartialSlug() error = %v", err)
	}
	if len(matches) != 1 {
		t.Errorf("FindByPartialSlug(database) returned %d matches, want 1", len(matches))
	}

	// Case insensitive
	matches, err = s.FindByPartialSlug("LOGIN")
	if err != nil {
		t.Fatalf("FindByPartialSlug() error = %v", err)
	}
	if len(matches) != 2 {
		t.Errorf("FindByPartialSlug(LOGIN) returned %d matches, want 2", len(matches))
	}
}

func TestStore_AllTags(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	now := time.Now()

	entries := []*entry.Entry{
		{Title: "Entry 1", Type: "note", Tags: []string{"api", "design"}, Created: now, Modified: now},
		{Title: "Entry 2", Type: "note", Tags: []string{"api", "backend"}, Created: now, Modified: now},
		{Title: "Entry 3", Type: "note", Tags: []string{"frontend"}, Created: now, Modified: now},
	}

	for _, e := range entries {
		e.Slug = entry.GenerateSlug(e.Title, e.Created)
		if err := s.Create(e); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	tags, err := s.AllTags()
	if err != nil {
		t.Fatalf("AllTags() error = %v", err)
	}

	if tags["api"] != 2 {
		t.Errorf("tags[api] = %d, want 2", tags["api"])
	}
	if tags["design"] != 1 {
		t.Errorf("tags[design] = %d, want 1", tags["design"])
	}
	if tags["frontend"] != 1 {
		t.Errorf("tags[frontend] = %d, want 1", tags["frontend"])
	}
}

func TestFindMatches(t *testing.T) {
	content := `Line one with API call
Line two
Line three has API endpoint
Line four`

	matches := findMatches(content, "api")

	if len(matches) != 2 {
		t.Fatalf("findMatches() returned %d matches, want 2", len(matches))
	}

	if matches[0].Line != 1 {
		t.Errorf("First match line = %d, want 1", matches[0].Line)
	}
	if matches[1].Line != 3 {
		t.Errorf("Second match line = %d, want 3", matches[1].Line)
	}
}

func TestStore_Exists(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	now := time.Now()
	e := &entry.Entry{
		Title:    "Test Entry",
		Type:     "note",
		Created:  now,
		Modified: now,
		Slug:     entry.GenerateSlug("Test Entry", now),
	}

	// Should not exist before creation
	if s.Exists(e.Slug) {
		t.Error("Entry should not exist before creation")
	}

	if err := s.Create(e); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Should exist after creation
	if !s.Exists(e.Slug) {
		t.Error("Entry should exist after creation")
	}
}
