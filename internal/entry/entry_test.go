package entry

import (
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    Entry
	}{
		{
			name: "with frontmatter",
			content: `---
title: Test Entry
type: idea
tags: [api, design]
status: open
---

This is the content.`,
			want: Entry{
				Title:   "Test Entry",
				Type:    "idea",
				Tags:    []string{"api", "design"},
				Status:  "open",
				Content: "This is the content.",
			},
		},
		{
			name:    "without frontmatter",
			content: "Just some content here.",
			want: Entry{
				Title:   "Just some content here.",
				Content: "Just some content here.",
			},
		},
		{
			name: "with extensions",
			content: `---
title: Test
custom_field: custom_value
---

Content`,
			want: Entry{
				Title:      "Test",
				Content:    "Content",
				Extensions: map[string]interface{}{"custom_field": "custom_value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.content)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if got.Title != tt.want.Title {
				t.Errorf("Title = %q, want %q", got.Title, tt.want.Title)
			}
			if got.Type != tt.want.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.want.Type)
			}
			if got.Status != tt.want.Status {
				t.Errorf("Status = %q, want %q", got.Status, tt.want.Status)
			}
			if strings.TrimSpace(got.Content) != strings.TrimSpace(tt.want.Content) {
				t.Errorf("Content = %q, want %q", got.Content, tt.want.Content)
			}
		})
	}
}

func TestEntry_ToMarkdown(t *testing.T) {
	e := &Entry{
		Title:    "Test Entry",
		Type:     "task",
		Tags:     []string{"api", "design"},
		Status:   "open",
		Priority: "high",
		Created:  time.Date(2024, 1, 2, 10, 30, 0, 0, time.UTC),
		Modified: time.Date(2024, 1, 2, 10, 30, 0, 0, time.UTC),
		Content:  "This is the content.",
	}

	md := e.ToMarkdown()

	if !strings.Contains(md, "title: Test Entry") {
		t.Error("Missing title in markdown")
	}
	if !strings.Contains(md, "type: task") {
		t.Error("Missing type in markdown")
	}
	if !strings.Contains(md, "status: open") {
		t.Error("Missing status in markdown")
	}
	if !strings.Contains(md, "This is the content.") {
		t.Error("Missing content in markdown")
	}
}

func TestEntry_Validate(t *testing.T) {
	tests := []struct {
		name    string
		entry   Entry
		wantErr bool
	}{
		{
			name:    "valid entry",
			entry:   Entry{Title: "Test", Type: "idea", Status: "open"},
			wantErr: false,
		},
		{
			name:    "missing title",
			entry:   Entry{Type: "idea"},
			wantErr: true,
		},
		{
			name:    "invalid type",
			entry:   Entry{Title: "Test", Type: "invalid"},
			wantErr: true,
		},
		{
			name:    "invalid status",
			entry:   Entry{Title: "Test", Status: "invalid"},
			wantErr: true,
		},
		{
			name:    "invalid priority",
			entry:   Entry{Title: "Test", Priority: "invalid"},
			wantErr: true,
		},
		{
			name:    "invalid due date",
			entry:   Entry{Title: "Test", Due: "not-a-date"},
			wantErr: true,
		},
		{
			name:    "valid due date",
			entry:   Entry{Title: "Test", Due: "2024-01-15"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.entry.Validate()
			if (len(errs) > 0) != tt.wantErr {
				t.Errorf("Validate() errors = %v, wantErr %v", errs, tt.wantErr)
			}
		})
	}
}

func TestEntry_HasTag(t *testing.T) {
	e := &Entry{Tags: []string{"api", "Design", "v2"}}

	if !e.HasTag("api") {
		t.Error("Should have tag 'api'")
	}
	if !e.HasTag("API") {
		t.Error("Should match case-insensitively")
	}
	if !e.HasTag("design") {
		t.Error("Should match 'design' case-insensitively")
	}
	if e.HasTag("missing") {
		t.Error("Should not have tag 'missing'")
	}
}
