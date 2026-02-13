package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/queelius/jot/internal/entry"
	"github.com/queelius/jot/internal/store"
)

var addCmd = &cobra.Command{
	Use:     "add <content>",
	Short:   "Quick capture a new entry",
	GroupID: "create",
	Long: `Quickly capture a new entry with a one-liner.

The first argument becomes the entry content and title.
Use flags to set type, tags, priority, and due date.

Examples:
  jot add "Quick thought about API caching"
  jot add "Fix the login bug" --type=task --priority=high
  jot add "Review PR" --type=task --due=3d --tags=work,urgent`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

var (
	addType     string
	addTags     string
	addPriority string
	addDue      string
	addStatus   string
)

func init() {
	addCmd.Flags().StringVarP(&addType, "type", "t", "", "entry type (idea, task, note, plan, log)")
	addCmd.Flags().StringVar(&addTags, "tags", "", "comma-separated tags")
	addCmd.Flags().StringVarP(&addPriority, "priority", "p", "", "priority (low, medium, high, critical)")
	addCmd.Flags().StringVarP(&addDue, "due", "d", "", "due date (YYYY-MM-DD, 3d, 1w, today, tomorrow)")
	addCmd.Flags().StringVarP(&addStatus, "status", "s", "", "status (open, in_progress, done, blocked)")

	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	cfg, err := getConfig()
	if err != nil {
		return err
	}

	content := args[0]
	now := time.Now()

	// Create entry
	e := &entry.Entry{
		Title:    content,
		Content:  content,
		Created:  now,
		Modified: now,
	}

	// Set type from flag or default
	if addType != "" {
		e.Type = addType
	} else if cfg.Defaults.Type != "" {
		e.Type = cfg.Defaults.Type
	}

	// Set tags
	if addTags != "" {
		e.Tags = parseTags(addTags)
	} else if len(cfg.Defaults.Tags) > 0 {
		e.Tags = cfg.Defaults.Tags
	}

	// Set other fields
	if addPriority != "" {
		e.Priority = addPriority
	}
	if addDue != "" {
		e.Due = store.ParseRelativeDate(addDue)
	}
	if addStatus != "" {
		e.Status = addStatus
	} else if e.Type == "task" {
		e.Status = "open" // Default status for tasks
	}

	// Validate
	if errs := e.Validate(); len(errs) > 0 {
		return fmt.Errorf("validation failed: %v", errs[0])
	}

	// Generate slug and save
	e.Slug = entry.GenerateSlug(e.Title, now)

	if err := s.Create(e); err != nil {
		return err
	}

	fmt.Printf("Created: %s\n", e.Slug)
	return nil
}

func parseTags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		tag := strings.TrimSpace(p)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}
