package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/queelius/jot/internal/entry"
	"github.com/queelius/jot/internal/store"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List entries",
	Long: `List entries with optional filtering.

Filters can be combined. All filters are AND-combined.

Examples:
  jot list
  jot list --type=task --status=open
  jot list --tag=api --since=7d
  jot list --json --limit=10`,
	Aliases: []string{"ls"},
	RunE:    runList,
}

var (
	listType     string
	listTag      string
	listStatus   string
	listPriority string
	listSince    string
	listUntil    string
	listDue      string
	listSearch   string
	listLimit    int
	listSort     string
	listReverse  bool
	listVerbose  bool
)

func init() {
	listCmd.Flags().StringVarP(&listType, "type", "t", "", "filter by type")
	listCmd.Flags().StringVar(&listTag, "tag", "", "filter by tag")
	listCmd.Flags().StringVarP(&listStatus, "status", "s", "", "filter by status")
	listCmd.Flags().StringVarP(&listPriority, "priority", "p", "", "filter by priority")
	listCmd.Flags().StringVar(&listSince, "since", "", "entries created since (e.g., 7d, 2w, 2024-01-01)")
	listCmd.Flags().StringVar(&listUntil, "until", "", "entries created until (e.g., 2024-12-31)")
	listCmd.Flags().StringVarP(&listDue, "due", "d", "", "filter by due date (today, week, overdue, or date)")
	listCmd.Flags().StringVarP(&listSearch, "search", "q", "", "search content, title, tags, and metadata")
	listCmd.Flags().IntVarP(&listLimit, "limit", "n", 0, "limit number of results")
	listCmd.Flags().StringVar(&listSort, "sort", "created", "sort by field (created, modified, title, priority)")
	listCmd.Flags().BoolVarP(&listReverse, "reverse", "r", false, "reverse sort order")
	listCmd.Flags().BoolVarP(&listVerbose, "verbose", "v", false, "show all columns (status, priority, tags)")

	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	// Build filter
	filter := &store.Filter{
		Type:     listType,
		Tag:      listTag,
		Status:   listStatus,
		Priority: listPriority,
		Limit:    listLimit,
	}

	// Parse since
	if listSince != "" {
		if dur, err := store.ParseDuration(listSince); err == nil && dur > 0 {
			filter.Since = time.Now().Add(-dur)
		} else if t, err := store.ParseDate(listSince); err == nil {
			filter.Since = t
		}
	}

	// Parse until
	if listUntil != "" {
		if t, err := store.ParseDate(listUntil); err == nil {
			filter.Until = t
		}
	}

	entries, err := s.List(filter)
	if err != nil {
		return err
	}

	// Filter by due date if specified
	if listDue != "" {
		entries = filterByDue(entries, listDue)
	}

	// Filter by search query if specified
	if listSearch != "" {
		entries = searchEntries(entries, listSearch)
	}

	// Sort entries
	sortEntries(entries, listSort, listReverse)

	// Check for empty results
	if len(entries) == 0 {
		fmt.Println("No entries found.")
		return nil
	}

	// Output based on format
	format := getOutputFormat()
	switch format {
	case "table":
		return outputTable(entries)
	case "markdown":
		return outputMarkdown(entries)
	default:
		return outputJSON(entries)
	}
}

func sortEntries(entries []*entry.Entry, sortBy string, reverse bool) {
	less := func(i, j int) bool {
		switch sortBy {
		case "modified":
			return entries[i].Modified.Before(entries[j].Modified)
		case "title":
			return entries[i].Title < entries[j].Title
		case "priority":
			return priorityOrder(entries[i].Priority) < priorityOrder(entries[j].Priority)
		default: // created
			return entries[i].Created.Before(entries[j].Created)
		}
	}

	// Sort in ascending order first
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if less(j, i) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Default is descending (newest first), reverse flag inverts this
	if !reverse {
		// Reverse to get descending order
		for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
			entries[i], entries[j] = entries[j], entries[i]
		}
	}
}

func priorityOrder(p string) int {
	switch p {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func outputJSON(entries []*entry.Entry) error {
	for _, e := range entries {
		summary := e.Summary()
		data, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	}
	return nil
}

func outputTable(entries []*entry.Entry) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if listVerbose {
		fmt.Fprintln(w, "SLUG\tTITLE\tTYPE\tSTATUS\tPRIORITY\tDUE\tCREATED")
	} else {
		fmt.Fprintln(w, "SLUG\tTITLE\tTYPE\tCREATED")
	}

	for _, e := range entries {
		// Strip date prefix from slug for display (YYYYMMDD-)
		displaySlug := e.Slug
		if len(displaySlug) > 9 && displaySlug[8] == '-' {
			displaySlug = displaySlug[9:]
		}
		if len(displaySlug) > 35 {
			displaySlug = displaySlug[:32] + "..."
		}

		title := e.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}

		if listVerbose {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				displaySlug,
				title,
				e.Type,
				e.Status,
				e.Priority,
				formatRelativeDue(e.Due),
				e.Created.Format("2006-01-02"),
			)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				displaySlug,
				title,
				e.Type,
				e.Created.Format("2006-01-02"),
			)
		}
	}

	return w.Flush()
}

func outputMarkdown(entries []*entry.Entry) error {
	for _, e := range entries {
		tags := ""
		if len(e.Tags) > 0 {
			tags = " [" + strings.Join(e.Tags, ", ") + "]"
		}
		fmt.Printf("- **%s** (%s)%s\n", e.Title, e.Slug, tags)
	}
	return nil
}

// searchEntries filters entries by a search query (case-insensitive).
// Matches against content, title, tags, type, status, priority, and due.
func searchEntries(entries []*entry.Entry, query string) []*entry.Entry {
	query = strings.ToLower(query)
	var result []*entry.Entry

	for _, e := range entries {
		// Check title
		if strings.Contains(strings.ToLower(e.Title), query) {
			result = append(result, e)
			continue
		}

		// Check content
		if strings.Contains(strings.ToLower(e.Content), query) {
			result = append(result, e)
			continue
		}

		// Check tags
		for _, tag := range e.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				result = append(result, e)
				break
			}
		}
		if len(result) > 0 && result[len(result)-1] == e {
			continue
		}

		// Check type, status, priority, due
		if strings.Contains(strings.ToLower(e.Type), query) ||
			strings.Contains(strings.ToLower(e.Status), query) ||
			strings.Contains(strings.ToLower(e.Priority), query) ||
			strings.Contains(strings.ToLower(e.Due), query) ||
			strings.Contains(strings.ToLower(e.Slug), query) {
			result = append(result, e)
		}
	}

	return result
}

// formatRelativeDue formats a due date as a relative string (e.g., "2d", "overdue").
func formatRelativeDue(due string) string {
	if due == "" {
		return ""
	}

	dueDate, err := time.Parse("2006-01-02", due)
	if err != nil {
		return due
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dueDay := time.Date(dueDate.Year(), dueDate.Month(), dueDate.Day(), 0, 0, 0, 0, now.Location())

	days := int(dueDay.Sub(today).Hours() / 24)

	switch {
	case days < -1:
		return fmt.Sprintf("\033[31m%dd overdue\033[0m", -days)
	case days == -1:
		return "\033[31myesterday\033[0m"
	case days == 0:
		return "\033[33mtoday\033[0m"
	case days == 1:
		return "tomorrow"
	case days <= 7:
		return fmt.Sprintf("%dd", days)
	case days <= 14:
		return fmt.Sprintf("%dw", days/7)
	default:
		return due
	}
}
