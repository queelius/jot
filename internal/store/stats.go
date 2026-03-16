package store

import (
	"sort"
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
	Summary *SummaryStats         `json:"summary,omitempty"`
	Overdue []*entry.EntrySummary `json:"overdue,omitempty"`
	Blocked []*entry.EntrySummary `json:"blocked,omitempty"`
	Health  []*TagHealth          `json:"health,omitempty"`
	Recent  []*entry.EntrySummary `json:"recent,omitempty"`
}

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

func computeOverdue(entries []*entry.Entry, now time.Time) []*entry.EntrySummary {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var result []*entry.EntrySummary

	for _, e := range entries {
		if e.Due == "" || e.Status == "done" || e.Status == "archived" {
			continue
		}
		due, err := time.ParseInLocation("2006-01-02", e.Due, now.Location())
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

// computeHealth computes per-tag project health.
// Entries without tags are not included in any tag's health data.
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
				if due, err := time.ParseInLocation("2006-01-02", e.Due, now.Location()); err == nil && due.Before(today) {
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
