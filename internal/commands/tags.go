package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/queelius/jot/internal/store"
	"github.com/spf13/cobra"
)

var tagsCmd = &cobra.Command{
	Use:     "tags [tag]",
	Short:   "List all tags (or entries with a tag)",
	GroupID: "query",
	Long: `List all tags with counts, or list entries with a specific tag.

Examples:
  jot tags              # List all tags
  jot tags api          # List entries tagged 'api'`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTags,
}

func init() {
	rootCmd.AddCommand(tagsCmd)
}

func runTags(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	if len(args) > 0 {
		if getFuzzy() {
			// Fuzzy tag search: show matching tags with summaries
			summaries, err := s.FuzzyTagSummaries(args[0])
			if err != nil {
				return err
			}
			if len(summaries) == 0 {
				fmt.Printf("No tags matching %q found.\n", args[0])
				return nil
			}
			format := getOutputFormat()
			if format == "json" {
				return outputTagSummariesJSON(summaries)
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "TAG\tCOUNT\tTYPES\tOPEN\tDONE\tLATEST")
			for _, ts := range summaries {
				open, done := countOpenDone(ts.Statuses)
				fmt.Fprintf(w, "%s\t%d\t%s\t%d\t%d\t%s\n",
					ts.Tag, ts.Count, formatTypes(ts.Types), open, done,
					ts.Latest.Format("2006-01-02"))
			}
			return w.Flush()
		}
		// Non-fuzzy: delegate to list command
		listTag = args[0]
		return runList(cmd, nil)
	}

	// List all tags with summaries
	summaries, err := s.TagSummaries()
	if err != nil {
		return err
	}

	if len(summaries) == 0 {
		fmt.Println("No tags found.")
		return nil
	}

	format := getOutputFormat()
	if format == "json" {
		return outputTagSummariesJSON(summaries)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TAG\tCOUNT\tTYPES\tOPEN\tDONE\tLATEST")
	for _, ts := range summaries {
		open, done := countOpenDone(ts.Statuses)
		fmt.Fprintf(w, "%s\t%d\t%s\t%d\t%d\t%s\n",
			ts.Tag,
			ts.Count,
			formatTypes(ts.Types),
			open,
			done,
			ts.Latest.Format("2006-01-02"),
		)
	}
	return w.Flush()
}

// formatTypes renders a type map as a compact string sorted by count desc.
// e.g., map[string]int{"task": 3, "idea": 2} → "3 task, 2 idea"
func formatTypes(types map[string]int) string {
	type kv struct {
		name  string
		count int
	}
	pairs := make([]kv, 0, len(types))
	for name, count := range types {
		pairs = append(pairs, kv{name, count})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count != pairs[j].count {
			return pairs[i].count > pairs[j].count
		}
		return pairs[i].name < pairs[j].name
	})
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = fmt.Sprintf("%d %s", p.count, p.name)
	}
	return strings.Join(parts, ", ")
}

// countOpenDone returns (open, done) counts from a status map.
// "open" = everything except "done" and "archived".
func countOpenDone(statuses map[string]int) (int, int) {
	done := statuses["done"]
	archived := statuses["archived"]
	open := 0
	for _, count := range statuses {
		open += count
	}
	open -= done + archived
	return open, done
}

func outputTagSummariesJSON(summaries []*store.TagSummary) error {
	for _, ts := range summaries {
		data, err := json.Marshal(ts)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	}
	return nil
}
