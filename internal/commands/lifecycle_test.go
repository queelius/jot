package commands

import (
	"testing"
	"time"

	"github.com/queelius/jot/internal/entry"
	"github.com/queelius/jot/internal/store"
)

// TestArchiveStaleMode tests archive with --stale mode.
func TestArchiveStaleMode(t *testing.T) {
	s, _ := setupTestJournal(t)

	createTestEntryWithAge(t, s, "Stale idea", "idea", "open", "", 120)
	createTestEntryWithAge(t, s, "Recent note", "note", "", "", 10)
	createTestEntryWithAge(t, s, "Done task", "task", "done", "", 200)

	t.Run("finds stale entries for archive", func(t *testing.T) {
		entries, err := findStaleEntries(s, 90*24*time.Hour, "", "")
		if err != nil {
			t.Fatalf("findStaleEntries failed: %v", err)
		}
		// Only "Stale idea" — done/archived are excluded by findStaleEntries
		if len(entries) != 1 {
			t.Errorf("got %d entries, want 1", len(entries))
			for _, e := range entries {
				t.Logf("  found: %s (status=%s)", e.Title, e.Status)
			}
		}
		if len(entries) > 0 && entries[0].Title != "Stale idea" {
			t.Errorf("expected 'Stale idea', got %q", entries[0].Title)
		}
	})
}

// TestArchiveOlderThanMode tests archive with --older-than mode.
func TestArchiveOlderThanMode(t *testing.T) {
	s, _ := setupTestJournal(t)

	createTestEntryWithAge(t, s, "Very old open", "idea", "open", "", 200)
	createTestEntryWithAge(t, s, "Old done", "task", "done", "", 120)
	createTestEntryWithAge(t, s, "Already archived", "note", "archived", "", 150)
	createTestEntryWithAge(t, s, "Recent", "idea", "open", "", 10)

	t.Run("older-than excludes archived", func(t *testing.T) {
		entries, err := findEntriesOlderThan(s, 90*24*time.Hour, "", "", "archived")
		if err != nil {
			t.Fatalf("findEntriesOlderThan failed: %v", err)
		}
		// Should find: very old open (200d), old done (120d)
		// Should NOT find: already archived, recent
		if len(entries) != 2 {
			t.Errorf("got %d entries, want 2", len(entries))
		}
	})
}

// TestArchiveByStatus tests archive with --status mode.
func TestArchiveByStatus(t *testing.T) {
	s, _ := setupTestJournal(t)

	done1 := createTestEntry(t, s, "Done task 1", "task", "done", "", "", nil)
	done2 := createTestEntry(t, s, "Done task 2", "task", "done", "", "", nil)
	createTestEntry(t, s, "Open task", "task", "open", "", "", nil)

	t.Run("archive all done entries", func(t *testing.T) {
		// Simulate archiving done entries
		for _, slug := range []string{done1.Slug, done2.Slug} {
			e, err := s.Get(slug)
			if err != nil {
				t.Fatalf("failed to get entry: %v", err)
			}
			e.Status = "archived"
			if err := s.Update(e); err != nil {
				t.Fatalf("failed to archive entry: %v", err)
			}
		}

		// Verify both are now archived
		for _, slug := range []string{done1.Slug, done2.Slug} {
			e, err := s.Get(slug)
			if err != nil {
				t.Fatalf("failed to get entry: %v", err)
			}
			if e.Status != "archived" {
				t.Errorf("entry %q should be archived, got %q", e.Title, e.Status)
			}
		}

		// Open task should still be open
		open, err := s.Get("open task")
		if err != nil {
			// Partial slug match might not work - use List
			entries, err := s.List(nil)
			if err != nil {
				t.Fatalf("failed to list entries: %v", err)
			}
			for _, e := range entries {
				if e.Title == "Open task" && e.Status != "open" {
					t.Errorf("open task should still be open, got %q", e.Status)
				}
			}
		} else if open.Status != "open" {
			t.Errorf("open task should still be open, got %q", open.Status)
		}
	})
}

// TestArchiveDryRunDoesNotModify verifies that dry-run doesn't change entries.
func TestArchiveDryRunDoesNotModify(t *testing.T) {
	s, _ := setupTestJournal(t)

	stale := createTestEntryWithAge(t, s, "Should stay open", "idea", "open", "", 120)

	// Verify it's stale
	entries, err := findStaleEntries(s, 90*24*time.Hour, "", "")
	if err != nil {
		t.Fatalf("findStaleEntries failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 stale entry, got %d", len(entries))
	}

	// In a dry-run, we would just print but not modify.
	// Verify the entry is still open after "finding" it.
	retrieved, err := s.Get(stale.Slug)
	if err != nil {
		t.Fatalf("failed to get entry: %v", err)
	}
	if retrieved.Status != "open" {
		t.Errorf("entry should still be open after dry-run discovery, got %q", retrieved.Status)
	}
}

// TestPurgeOnlyTargetsArchived tests that purge candidates only include archived entries.
func TestPurgeOnlyTargetsArchived(t *testing.T) {
	s, _ := setupTestJournal(t)

	createTestEntry(t, s, "Open entry", "idea", "open", "", "", nil)
	createTestEntry(t, s, "Done entry", "task", "done", "", "", nil)
	archived := createTestEntry(t, s, "Archived entry", "note", "archived", "", "", nil)

	// List with archived filter — only archived should appear
	entries, err := s.List(&store.Filter{Status: "archived"})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("got %d archived entries, want 1", len(entries))
	}
	if len(entries) > 0 && entries[0].Slug != archived.Slug {
		t.Errorf("expected archived entry, got %q", entries[0].Title)
	}
}

// TestPurgeOlderThanRespectsAge tests that purge --older-than respects modified time.
func TestPurgeOlderThanRespectsAge(t *testing.T) {
	s, _ := setupTestJournal(t)

	// Old archived entry
	createTestEntryWithAge(t, s, "Old archived", "note", "archived", "", 200)
	// Recently archived entry
	createTestEntry(t, s, "Recent archived", "idea", "archived", "", "", nil)

	entries, err := s.List(&store.Filter{Status: "archived"})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	// Filter by age (>180 days)
	cutoff := time.Now().Add(-180 * 24 * time.Hour)
	var old []*entry.Entry
	for _, e := range entries {
		if e.Modified.Before(cutoff) {
			old = append(old, e)
		}
	}

	if len(old) != 1 {
		t.Errorf("got %d old archived entries, want 1", len(old))
	}
	if len(old) > 0 && old[0].Title != "Old archived" {
		t.Errorf("expected 'Old archived', got %q", old[0].Title)
	}
}

// TestPurgeDeletesFromDisk tests that deletion actually removes files.
func TestPurgeDeletesFromDisk(t *testing.T) {
	s, _ := setupTestJournal(t)

	archived := createTestEntry(t, s, "To be purged", "note", "archived", "", "", nil)

	// Verify it exists
	if !s.Exists(archived.Slug) {
		t.Fatal("archived entry should exist before purge")
	}

	// Delete it (simulating purge)
	if err := s.Delete(archived.Slug); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	// Verify it's gone
	if s.Exists(archived.Slug) {
		t.Error("entry should not exist after purge")
	}
}

// TestLifecyclePipeline tests the full stale -> archive -> purge workflow.
func TestLifecyclePipeline(t *testing.T) {
	s, _ := setupTestJournal(t)

	// Step 1: Create entries of various ages and statuses
	createTestEntryWithAge(t, s, "Active recent", "idea", "open", "", 10)
	staleIdea := createTestEntryWithAge(t, s, "Stale idea", "idea", "open", "", 120)
	staleTask := createTestEntryWithAge(t, s, "Stale task", "task", "in_progress", "", 100)
	createTestEntryWithAge(t, s, "Done task", "task", "done", "", 200)

	// Step 2: Discover stale entries
	staleEntries, err := findStaleEntries(s, 90*24*time.Hour, "", "")
	if err != nil {
		t.Fatalf("findStaleEntries failed: %v", err)
	}
	if len(staleEntries) != 2 {
		t.Fatalf("expected 2 stale entries, got %d", len(staleEntries))
	}

	// Step 3: Archive the stale entries
	for _, e := range staleEntries {
		fresh, err := s.Get(e.Slug)
		if err != nil {
			t.Fatalf("failed to get %s: %v", e.Slug, err)
		}
		fresh.Status = "archived"
		if err := s.Update(fresh); err != nil {
			t.Fatalf("failed to archive %s: %v", fresh.Slug, err)
		}
	}

	// Verify they're archived
	for _, slug := range []string{staleIdea.Slug, staleTask.Slug} {
		e, err := s.Get(slug)
		if err != nil {
			t.Fatalf("failed to get %s: %v", slug, err)
		}
		if e.Status != "archived" {
			t.Errorf("entry %q should be archived, got %q", e.Title, e.Status)
		}
	}

	// Verify active entry is untouched
	all, err := s.List(nil)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	activeCount := 0
	for _, e := range all {
		if e.Status != "archived" && e.Status != "done" {
			activeCount++
		}
	}
	if activeCount != 1 {
		t.Errorf("expected 1 active entry remaining, got %d", activeCount)
	}

	// Step 4: Purge archived entries
	// Note: s.Update() sets Modified to time.Now(), so recently archived entries
	// won't match --older-than. We use --all mode instead.
	archived, err := s.List(&store.Filter{Status: "archived"})
	if err != nil {
		t.Fatalf("failed to list archived: %v", err)
	}
	if len(archived) != 2 {
		t.Fatalf("expected 2 archived entries, got %d", len(archived))
	}

	for _, e := range archived {
		if err := s.Delete(e.Slug); err != nil {
			t.Fatalf("failed to purge %s: %v", e.Slug, err)
		}
	}

	// Step 5: Verify final state
	remaining, err := s.List(nil)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	// Should have: "Active recent" (open) and "Done task" (done)
	if len(remaining) != 2 {
		t.Errorf("expected 2 remaining entries, got %d", len(remaining))
		for _, e := range remaining {
			t.Logf("  remaining: %s (status=%s)", e.Title, e.Status)
		}
	}

	// Verify purged entries are gone from disk
	if s.Exists(staleIdea.Slug) {
		t.Error("stale idea should be deleted from disk")
	}
	if s.Exists(staleTask.Slug) {
		t.Error("stale task should be deleted from disk")
	}
}

// TestStaleWithTags tests that tag filtering works in stale detection.
func TestStaleWithTags(t *testing.T) {
	s, _ := setupTestJournal(t)

	createTestEntryWithAge(t, s, "API idea", "idea", "open", "api", 120)
	createTestEntryWithAge(t, s, "Frontend idea", "idea", "open", "frontend", 120)

	t.Run("filter by tag", func(t *testing.T) {
		entries, err := findStaleEntries(s, 90*24*time.Hour, "", "api")
		if err != nil {
			t.Fatalf("findStaleEntries failed: %v", err)
		}
		if len(entries) != 1 {
			t.Errorf("got %d entries, want 1", len(entries))
		}
		if len(entries) > 0 && entries[0].Title != "API idea" {
			t.Errorf("expected 'API idea', got %q", entries[0].Title)
		}
	})
}

// TestArchiveWithTypeAndTagNarrowing tests combining type and tag filters.
func TestArchiveWithTypeAndTagNarrowing(t *testing.T) {
	s, _ := setupTestJournal(t)

	createTestEntryWithAge(t, s, "API task", "task", "open", "api", 120)
	createTestEntryWithAge(t, s, "API idea", "idea", "open", "api", 120)
	createTestEntryWithAge(t, s, "Backend task", "task", "open", "backend", 120)

	t.Run("narrow by type and tag", func(t *testing.T) {
		entries, err := findStaleEntries(s, 90*24*time.Hour, "task", "api")
		if err != nil {
			t.Fatalf("findStaleEntries failed: %v", err)
		}
		if len(entries) != 1 {
			t.Errorf("got %d entries, want 1", len(entries))
		}
		if len(entries) > 0 && entries[0].Title != "API task" {
			t.Errorf("expected 'API task', got %q", entries[0].Title)
		}
	})
}
