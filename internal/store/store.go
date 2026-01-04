// Package store provides file-based storage for jot entries.
package store

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/queelius/jot/internal/entry"
)

// Store manages entries in a jot journal directory.
type Store struct {
	Root string
}

// New creates a new store for the given journal root directory.
func New(root string) *Store {
	return &Store{Root: root}
}

// EntriesDir returns the entries directory path.
func (s *Store) EntriesDir() string {
	return filepath.Join(s.Root, "entries")
}

// Create saves a new entry to the store.
func (s *Store) Create(e *entry.Entry) error {
	if e.Slug == "" {
		e.Slug = entry.GenerateSlug(e.Title, e.Created)
	}

	path, err := entry.PathForSlug(e.Slug)
	if err != nil {
		return fmt.Errorf("generating path: %w", err)
	}

	fullPath := filepath.Join(s.Root, path)

	// Check if file already exists
	if _, err := os.Stat(fullPath); err == nil {
		return fmt.Errorf("entry already exists: %s", e.Slug)
	}

	// Create directory structure
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Write entry
	content := e.ToMarkdown()
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing entry: %w", err)
	}

	e.Path = fullPath
	return nil
}

// Get retrieves an entry by slug.
func (s *Store) Get(slug string) (*entry.Entry, error) {
	path, err := entry.PathForSlug(slug)
	if err != nil {
		return nil, err
	}

	fullPath := filepath.Join(s.Root, path)
	return entry.ParseFile(fullPath)
}

// Update saves changes to an existing entry.
func (s *Store) Update(e *entry.Entry) error {
	if e.Path == "" {
		path, err := entry.PathForSlug(e.Slug)
		if err != nil {
			return err
		}
		e.Path = filepath.Join(s.Root, path)
	}

	e.Modified = time.Now()

	content := e.ToMarkdown()
	if err := os.WriteFile(e.Path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing entry: %w", err)
	}

	return nil
}

// Delete removes an entry, its sidecar, and asset directory.
func (s *Store) Delete(slug string) error {
	path, err := entry.PathForSlug(slug)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(s.Root, path)

	// Delete main entry file
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing entry: %w", err)
	}

	// Delete sidecar metadata file
	sidecarPath := entry.SidecarPath(fullPath)
	if err := os.Remove(sidecarPath); err != nil && !os.IsNotExist(err) {
		// Ignore error, sidecar may not exist
	}

	// Delete asset directory
	assetDir := entry.AssetDir(fullPath)
	if info, err := os.Stat(assetDir); err == nil && info.IsDir() {
		if err := os.RemoveAll(assetDir); err != nil {
			return fmt.Errorf("removing asset directory: %w", err)
		}
	}

	return nil
}

// List returns all entries matching the given filter.
func (s *Store) List(f *Filter) ([]*entry.Entry, error) {
	var entries []*entry.Entry

	entriesDir := s.EntriesDir()
	if _, err := os.Stat(entriesDir); os.IsNotExist(err) {
		return entries, nil
	}

	err := filepath.WalkDir(entriesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-markdown files
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		// Skip sidecar files
		if strings.HasSuffix(path, ".meta.yaml") {
			return nil
		}

		e, err := entry.ParseFile(path)
		if err != nil {
			// Log warning but continue
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", path, err)
			return nil
		}

		entries = append(entries, e)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking entries: %w", err)
	}

	// Apply filter
	if f != nil {
		entries = f.Apply(entries)
	}

	// Sort by created date descending (newest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Created.After(entries[j].Created)
	})

	return entries, nil
}

// Exists checks if an entry with the given slug exists.
func (s *Store) Exists(slug string) bool {
	path, err := entry.PathForSlug(slug)
	if err != nil {
		return false
	}

	fullPath := filepath.Join(s.Root, path)
	_, err = os.Stat(fullPath)
	return err == nil
}

// Search performs a full-text search across all entries.
func (s *Store) Search(query string, f *Filter) ([]*SearchResult, error) {
	entries, err := s.List(f)
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var results []*SearchResult

	for _, e := range entries {
		content := strings.ToLower(e.Title + "\n" + e.Content)
		if strings.Contains(content, query) {
			results = append(results, &SearchResult{
				Entry:   e,
				Matches: findMatches(e.Content, query),
			})
		}
	}

	return results, nil
}

// SearchResult contains an entry and matching content snippets.
type SearchResult struct {
	Entry   *entry.Entry
	Matches []Match
}

// Match represents a match with context.
type Match struct {
	Line       int
	Content    string
	MatchStart int
	MatchEnd   int
}

// findMatches finds all occurrences of query in content with line context.
func findMatches(content, query string) []Match {
	var matches []Match
	lines := strings.Split(content, "\n")
	query = strings.ToLower(query)

	for i, line := range lines {
		lower := strings.ToLower(line)
		start := 0
		for {
			idx := strings.Index(lower[start:], query)
			if idx == -1 {
				break
			}
			matches = append(matches, Match{
				Line:       i + 1,
				Content:    line,
				MatchStart: start + idx,
				MatchEnd:   start + idx + len(query),
			})
			start += idx + len(query)
		}
	}

	return matches
}

// AllTags returns all unique tags with their counts.
func (s *Store) AllTags() (map[string]int, error) {
	entries, err := s.List(nil)
	if err != nil {
		return nil, err
	}

	tags := make(map[string]int)
	for _, e := range entries {
		for _, tag := range e.Tags {
			tags[tag]++
		}
	}

	return tags, nil
}

// FindByPartialSlug returns entries whose slugs contain the query (case-insensitive).
func (s *Store) FindByPartialSlug(query string) ([]*entry.Entry, error) {
	entries, err := s.List(nil)
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var matches []*entry.Entry

	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Slug), query) {
			matches = append(matches, e)
		}
	}

	return matches, nil
}
