package commands

import (
	"encoding/json"
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

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List and filter entries",
	GroupID: "query",
	Long: `List entries with optional filtering.

Filters can be combined. All filters are AND-combined.

Examples:
  jot list
  jot list --type=task --status=open
  jot list --tags=api --since=7d
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
	listCmd.Flags().StringVar(&listTag, "tags", "", "filter by tag")
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
		Fuzzy:    getFuzzy(),
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
	// Default is descending (newest first); reverse flag inverts to ascending
	sort.Slice(entries, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "modified":
			less = entries[i].Modified.Before(entries[j].Modified)
		case "title":
			less = entries[i].Title < entries[j].Title
		case "priority":
			less = priorityOrder(entries[i].Priority) < priorityOrder(entries[j].Priority)
		default: // created
			less = entries[i].Created.Before(entries[j].Created)
		}
		if reverse {
			return less
		}
		return !less
	})
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

// truncateSlug strips the date prefix and truncates for table display.
func truncateSlug(slug string) string {
	if len(slug) > 9 && slug[8] == '-' {
		slug = slug[9:]
	}
	if len(slug) > 35 {
		slug = slug[:32] + "..."
	}
	return slug
}

// truncateTitle truncates a title for table display.
func truncateTitle(title string) string {
	if len(title) > 40 {
		return title[:37] + "..."
	}
	return title
}

func outputTable(entries []*entry.Entry) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if listVerbose {
		fmt.Fprintln(w, "SLUG\tTITLE\tTYPE\tSTATUS\tPRIORITY\tDUE\tCREATED")
	} else {
		fmt.Fprintln(w, "SLUG\tTITLE\tTYPE\tCREATED")
	}

	for _, e := range entries {
		displaySlug := truncateSlug(e.Slug)
		title := truncateTitle(e.Title)

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
// Matches against content, title, tags, type, status, priority, due, and slug.
func searchEntries(entries []*entry.Entry, query string) []*entry.Entry {
	query = strings.ToLower(query)
	var result []*entry.Entry

	for _, e := range entries {
		if entryMatchesQuery(e, query) {
			result = append(result, e)
		}
	}

	return result
}

func entryMatchesQuery(e *entry.Entry, query string) bool {
	if strings.Contains(strings.ToLower(e.Title), query) {
		return true
	}
	if strings.Contains(strings.ToLower(e.Content), query) {
		return true
	}
	for _, tag := range e.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	if strings.Contains(strings.ToLower(e.Type), query) ||
		strings.Contains(strings.ToLower(e.Status), query) ||
		strings.Contains(strings.ToLower(e.Priority), query) ||
		strings.Contains(strings.ToLower(e.Due), query) ||
		strings.Contains(strings.ToLower(e.Slug), query) {
		return true
	}
	return false
}

// filterByDue filters entries by due date (today, week, overdue, or a specific date).
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

// formatRelativeDue formats a due date as a relative string (e.g., "2d", "overdue").
func formatRelativeDue(due string) string {
	if due == "" {
		return ""
	}

	now := time.Now()
	dueDate, err := time.ParseInLocation("2006-01-02", due, now.Location())
	if err != nil {
		return due
	}

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
