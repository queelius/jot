// Package entry provides the core Entry type and operations for jot.
package entry

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Entry represents a single jot entry with frontmatter and content.
type Entry struct {
	// Core fields (validated)
	Slug     string    `json:"slug" yaml:"-"`
	Title    string    `json:"title" yaml:"title"`
	Type     string    `json:"type,omitempty" yaml:"type,omitempty"`
	Tags     []string  `json:"tags,omitempty" yaml:"tags,omitempty"`
	Status   string    `json:"status,omitempty" yaml:"status,omitempty"`
	Priority string    `json:"priority,omitempty" yaml:"priority,omitempty"`
	Due      string    `json:"due,omitempty" yaml:"due,omitempty"`
	Created  time.Time `json:"created" yaml:"created"`
	Modified time.Time `json:"modified" yaml:"modified"`

	// Task-specific fields
	BlockedBy string   `json:"blocked_by,omitempty" yaml:"blocked_by,omitempty"`
	DependsOn []string `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`

	// Content (not in frontmatter)
	Content string `json:"content,omitempty" yaml:"-"`

	// Extension fields (arbitrary YAML)
	Extensions map[string]interface{} `json:"extensions,omitempty" yaml:"-"`

	// File path (internal use)
	Path string `json:"-" yaml:"-"`
}

// frontmatter is used for YAML parsing with extensions support.
type frontmatter struct {
	Title     string                 `yaml:"title"`
	Type      string                 `yaml:"type,omitempty"`
	Tags      []string               `yaml:"tags,omitempty"`
	Status    string                 `yaml:"status,omitempty"`
	Priority  string                 `yaml:"priority,omitempty"`
	Due       string                 `yaml:"due,omitempty"`
	Created   time.Time              `yaml:"created"`
	Modified  time.Time              `yaml:"modified"`
	BlockedBy string                 `yaml:"blocked_by,omitempty"`
	DependsOn []string               `yaml:"depends_on,omitempty"`
	Extra     map[string]interface{} `yaml:",inline"`
}

// ValidTypes are the allowed entry types.
var ValidTypes = []string{"idea", "task", "note", "plan", "log"}

// ValidStatuses are the allowed status values.
var ValidStatuses = []string{"open", "in_progress", "done", "blocked", "archived"}

// ValidPriorities are the allowed priority values.
var ValidPriorities = []string{"low", "medium", "high", "critical"}

// ParseFile reads an entry from a markdown file.
func ParseFile(path string) (*Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	entry, err := Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	entry.Path = path
	entry.Slug = SlugFromPath(path)

	return entry, nil
}

// Parse parses an entry from markdown content with optional frontmatter.
func Parse(content string) (*Entry, error) {
	entry := &Entry{
		Created:  time.Now(),
		Modified: time.Now(),
	}

	// Check for frontmatter
	if strings.HasPrefix(content, "---\n") {
		parts := strings.SplitN(content[4:], "\n---", 2)
		if len(parts) == 2 {
			// Parse frontmatter
			var fm frontmatter
			if err := yaml.Unmarshal([]byte(parts[0]), &fm); err != nil {
				return nil, fmt.Errorf("parsing frontmatter: %w", err)
			}

			entry.Title = fm.Title
			entry.Type = fm.Type
			entry.Tags = fm.Tags
			entry.Status = fm.Status
			entry.Priority = fm.Priority
			entry.Due = fm.Due
			entry.BlockedBy = fm.BlockedBy
			entry.DependsOn = fm.DependsOn

			if !fm.Created.IsZero() {
				entry.Created = fm.Created
			}
			if !fm.Modified.IsZero() {
				entry.Modified = fm.Modified
			}

			// Remove known fields from Extra to get extensions
			delete(fm.Extra, "title")
			delete(fm.Extra, "type")
			delete(fm.Extra, "tags")
			delete(fm.Extra, "status")
			delete(fm.Extra, "priority")
			delete(fm.Extra, "due")
			delete(fm.Extra, "created")
			delete(fm.Extra, "modified")
			delete(fm.Extra, "blocked_by")
			delete(fm.Extra, "depends_on")

			if len(fm.Extra) > 0 {
				entry.Extensions = fm.Extra
			}

			// Content is after the closing ---
			entry.Content = strings.TrimPrefix(parts[1], "\n")
		} else {
			// No closing ---, treat as content
			entry.Content = content
		}
	} else {
		entry.Content = content
	}

	// If no title, use first line of content
	if entry.Title == "" && entry.Content != "" {
		scanner := bufio.NewScanner(strings.NewReader(entry.Content))
		if scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			// Remove markdown heading prefix
			line = strings.TrimPrefix(line, "# ")
			line = strings.TrimPrefix(line, "## ")
			entry.Title = line
		}
	}

	return entry, nil
}

// ToMarkdown serializes the entry to markdown with frontmatter.
func (e *Entry) ToMarkdown() string {
	var sb strings.Builder

	// Build frontmatter
	fm := make(map[string]interface{})
	fm["title"] = e.Title
	if e.Type != "" {
		fm["type"] = e.Type
	}
	if len(e.Tags) > 0 {
		fm["tags"] = e.Tags
	}
	if e.Status != "" {
		fm["status"] = e.Status
	}
	if e.Priority != "" {
		fm["priority"] = e.Priority
	}
	if e.Due != "" {
		fm["due"] = e.Due
	}
	fm["created"] = e.Created.Format(time.RFC3339)
	fm["modified"] = e.Modified.Format(time.RFC3339)
	if e.BlockedBy != "" {
		fm["blocked_by"] = e.BlockedBy
	}
	if len(e.DependsOn) > 0 {
		fm["depends_on"] = e.DependsOn
	}

	// Add extensions
	for k, v := range e.Extensions {
		fm[k] = v
	}

	yamlBytes, _ := yaml.Marshal(fm)

	sb.WriteString("---\n")
	sb.Write(yamlBytes)
	sb.WriteString("---\n")
	if e.Content != "" {
		sb.WriteString("\n")
		sb.WriteString(e.Content)
	}

	return sb.String()
}

// ToJSON returns a JSON representation of the entry.
func (e *Entry) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// ToJSONPretty returns a pretty-printed JSON representation.
func (e *Entry) ToJSONPretty() ([]byte, error) {
	return json.MarshalIndent(e, "", "  ")
}

// ToJSONSummary returns a compact JSON summary (for list output).
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
}

// Summary returns a compact summary for list output.
func (e *Entry) Summary() EntrySummary {
	return EntrySummary{
		Slug:     e.Slug,
		Title:    e.Title,
		Type:     e.Type,
		Status:   e.Status,
		Priority: e.Priority,
		Due:      e.Due,
		Tags:     e.Tags,
		Created:  e.Created.Format(time.RFC3339),
		Modified: e.Modified.Format(time.RFC3339),
	}
}

// SlugFromPath extracts the slug from a file path.
// e.g., "entries/2024/01/20240102-api-redesign.md" -> "20240102-api-redesign"
func SlugFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// Validate checks that the entry has valid field values.
func (e *Entry) Validate() []error {
	var errs []error

	if e.Title == "" {
		errs = append(errs, fmt.Errorf("title is required"))
	}

	if e.Type != "" && !contains(ValidTypes, e.Type) {
		errs = append(errs, fmt.Errorf("invalid type %q, must be one of: %v", e.Type, ValidTypes))
	}

	if e.Status != "" && !contains(ValidStatuses, e.Status) {
		errs = append(errs, fmt.Errorf("invalid status %q, must be one of: %v", e.Status, ValidStatuses))
	}

	if e.Priority != "" && !contains(ValidPriorities, e.Priority) {
		errs = append(errs, fmt.Errorf("invalid priority %q, must be one of: %v", e.Priority, ValidPriorities))
	}

	if e.Due != "" {
		if _, err := time.Parse("2006-01-02", e.Due); err != nil {
			errs = append(errs, fmt.Errorf("invalid due date %q, must be YYYY-MM-DD", e.Due))
		}
	}

	return errs
}

// IsTask returns true if this entry is a task.
func (e *Entry) IsTask() bool {
	return e.Type == "task"
}

// HasTag returns true if the entry has the given tag.
func (e *Entry) HasTag(tag string) bool {
	for _, t := range e.Tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
