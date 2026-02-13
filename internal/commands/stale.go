package commands

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/queelius/jot/internal/entry"
	"github.com/queelius/jot/internal/store"
)

var staleCmd = &cobra.Command{
	Use:     "stale",
	Short:   "Find entries that haven't been touched recently",
	GroupID: "lifecycle",
	Long: `Find active entries that haven't been modified recently.

Only shows entries with status NOT done/archived (active but forgotten).
Sorted by modified date ascending (stalest first).

Examples:
  jot stale                    # Entries not modified in 90 days
  jot stale --days 30          # Entries not modified in 30 days
  jot stale --type=idea        # Only stale ideas
  jot stale --tags=api         # Only stale entries tagged "api"
  jot stale --json             # JSON output`,
	RunE: runStale,
}

var (
	staleDays int
	staleType string
	staleTag  string
)

func init() {
	staleCmd.Flags().IntVar(&staleDays, "days", 90, "number of days since last modification")
	staleCmd.Flags().StringVarP(&staleType, "type", "t", "", "filter by type")
	staleCmd.Flags().StringVar(&staleTag, "tags", "", "filter by tag")

	rootCmd.AddCommand(staleCmd)
}

func runStale(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	threshold := time.Duration(staleDays) * 24 * time.Hour
	entries, err := findStaleEntries(s, threshold, staleType, staleTag)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		fmt.Println("No stale entries found.")
		return nil
	}

	format := getOutputFormat()
	switch format {
	case "json":
		return outputJSON(entries)
	case "markdown":
		return outputMarkdown(entries)
	default:
		return outputStaleTable(entries)
	}
}

// findStaleEntries returns active entries not modified within the given threshold.
// "Active" means status is NOT done or archived.
func findStaleEntries(s *store.Store, threshold time.Duration, typ, tag string) ([]*entry.Entry, error) {
	filter := &store.Filter{
		Type: typ,
		Tag:  tag,
	}

	entries, err := s.List(filter)
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().Add(-threshold)
	entries = excludeStatus(entries, "done", "archived")

	var stale []*entry.Entry
	for _, e := range entries {
		if e.Modified.Before(cutoff) {
			stale = append(stale, e)
		}
	}

	// Sort by modified ascending (stalest first)
	sortEntries(stale, "modified", true)

	return stale, nil
}

// findEntriesOlderThan returns entries modified before the given threshold,
// excluding entries with the specified statuses.
func findEntriesOlderThan(s *store.Store, threshold time.Duration, typ, tag string, excludeStatuses ...string) ([]*entry.Entry, error) {
	filter := &store.Filter{
		Type: typ,
		Tag:  tag,
	}

	entries, err := s.List(filter)
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().Add(-threshold)
	if len(excludeStatuses) > 0 {
		entries = excludeStatus(entries, excludeStatuses...)
	}

	var old []*entry.Entry
	for _, e := range entries {
		if e.Modified.Before(cutoff) {
			old = append(old, e)
		}
	}

	// Sort by modified ascending (oldest first)
	sortEntries(old, "modified", true)

	return old, nil
}

// excludeStatus filters out entries whose status matches any of the given statuses.
func excludeStatus(entries []*entry.Entry, statuses ...string) []*entry.Entry {
	var result []*entry.Entry
	for _, e := range entries {
		excluded := false
		for _, s := range statuses {
			if e.Status == s {
				excluded = true
				break
			}
		}
		if !excluded {
			result = append(result, e)
		}
	}
	return result
}

// formatAge formats a duration as a human-readable age string.
func formatAge(d time.Duration) string {
	days := int(d.Hours() / 24)
	switch {
	case days < 1:
		return "<1d"
	case days < 7:
		return fmt.Sprintf("%dd", days)
	case days < 30:
		weeks := days / 7
		return fmt.Sprintf("%dw", weeks)
	case days < 365:
		months := days / 30
		return fmt.Sprintf("%dm", months)
	default:
		years := days / 365
		return fmt.Sprintf("%dy", years)
	}
}

func outputStaleTable(entries []*entry.Entry) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SLUG\tTITLE\tTYPE\tSTATUS\tAGE")

	now := time.Now()
	for _, e := range entries {
		displaySlug := truncateSlug(e.Slug)
		title := truncateTitle(e.Title)
		age := formatAge(now.Sub(e.Modified))

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			displaySlug,
			title,
			e.Type,
			e.Status,
			age,
		)
	}

	return w.Flush()
}
