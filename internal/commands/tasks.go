package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/queelius/jot/internal/entry"
	"github.com/queelius/jot/internal/store"
)

var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "List open tasks",
	Long: `List tasks, defaulting to open tasks.

Examples:
  jot tasks                    # Open tasks
  jot tasks --status=blocked   # Blocked tasks
  jot tasks --priority=high    # High priority tasks
  jot tasks --due=today        # Due today or overdue
  jot tasks --due=week         # Due within 7 days`,
	RunE: runTasks,
}

var (
	tasksStatus   string
	tasksPriority string
	tasksDue      string
)

func init() {
	tasksCmd.Flags().StringVarP(&tasksStatus, "status", "s", "", "filter by status (default: open)")
	tasksCmd.Flags().StringVarP(&tasksPriority, "priority", "p", "", "filter by priority")
	tasksCmd.Flags().StringVarP(&tasksDue, "due", "d", "", "filter by due date (today, week, or date)")

	rootCmd.AddCommand(tasksCmd)
}

func runTasks(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	// Default to open status
	status := tasksStatus
	if status == "" {
		status = "open"
	}

	filter := &store.Filter{
		Type:     "task",
		Status:   status,
		Priority: tasksPriority,
	}

	entries, err := s.List(filter)
	if err != nil {
		return err
	}

	// Filter by due date if specified
	if tasksDue != "" {
		entries = filterByDue(entries, tasksDue)
	}

	// Sort by priority (high first), then by due date
	sortTasks(entries)

	if len(entries) == 0 {
		fmt.Println("No tasks found.")
		return nil
	}

	format := getOutputFormat()
	if format == "json" {
		return outputJSON(entries)
	}

	return outputTaskTable(entries)
}

func filterByDue(entries []*entry.Entry, dueFilter string) []*entry.Entry {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var deadline time.Time

	switch strings.ToLower(dueFilter) {
	case "today":
		deadline = today.Add(24 * time.Hour)
	case "week":
		deadline = today.Add(7 * 24 * time.Hour)
	case "overdue":
		deadline = today
	default:
		// Try to parse as date
		if t, err := store.ParseDate(dueFilter); err == nil {
			deadline = t.Add(24 * time.Hour)
		} else {
			return entries
		}
	}

	var result []*entry.Entry
	for _, e := range entries {
		if e.Due == "" {
			continue
		}

		dueDate, err := time.ParseInLocation("2006-01-02", e.Due, now.Location())
		if err != nil {
			continue
		}

		if dueFilter == "overdue" {
			if dueDate.Before(today) {
				result = append(result, e)
			}
		} else {
			if dueDate.Before(deadline) {
				result = append(result, e)
			}
		}
	}

	return result
}

func sortTasks(entries []*entry.Entry) {
	// Sort by priority (highest first), then by due date (earliest first)
	sort.Slice(entries, func(i, j int) bool {
		pi := priorityOrder(entries[i].Priority)
		pj := priorityOrder(entries[j].Priority)
		if pi != pj {
			return pi > pj
		}
		// Same priority: sort by due date (earliest first, no-due last)
		di := parseDueDate(entries[i].Due)
		dj := parseDueDate(entries[j].Due)
		if di.IsZero() && dj.IsZero() {
			return false
		}
		if di.IsZero() {
			return false
		}
		if dj.IsZero() {
			return true
		}
		return di.Before(dj)
	})
}

func parseDueDate(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func outputTaskTable(entries []*entry.Entry) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SLUG\tTITLE\tSTATUS\tPRIORITY\tDUE")

	for _, e := range entries {
		displaySlug := truncateSlug(e.Slug)
		title := truncateTitle(e.Title)
		priority := e.Priority
		switch priority {
		case "critical":
			priority = "\033[31m" + priority + "\033[0m" // Red
		case "high":
			priority = "\033[33m" + priority + "\033[0m" // Yellow
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			displaySlug,
			title,
			e.Status,
			priority,
			formatRelativeDue(e.Due),
		)
	}

	return w.Flush()
}
