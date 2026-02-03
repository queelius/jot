package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/queelius/jot/internal/store"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search entry content",
	Long: `Search for entries containing the specified text.

Searches both titles and content. Case-insensitive.

Examples:
  jot search "GraphQL implementation"
  jot search "authentication" --type=task
  jot search "api" --context=3`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

var (
	searchType     string
	searchTag      string
	searchStatus   string
	searchPriority string
	searchContext  int
)

func init() {
	searchCmd.Flags().StringVarP(&searchType, "type", "t", "", "filter by type")
	searchCmd.Flags().StringVar(&searchTag, "tags", "", "filter by tag")
	searchCmd.Flags().StringVarP(&searchStatus, "status", "s", "", "filter by status")
	searchCmd.Flags().StringVarP(&searchPriority, "priority", "p", "", "filter by priority")
	searchCmd.Flags().IntVarP(&searchContext, "context", "C", 0, "lines of context around matches")

	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	query := args[0]

	filter := &store.Filter{
		Type:     searchType,
		Tag:      searchTag,
		Status:   searchStatus,
		Priority: searchPriority,
	}

	results, err := s.Search(query, filter)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Println("No matches found.")
		return nil
	}

	format := getOutputFormat()
	if format == "json" {
		return outputSearchJSON(results)
	}

	// Human-readable output
	for _, r := range results {
		fmt.Printf("\033[1m%s\033[0m: %s\n", r.Entry.Slug, r.Entry.Title)

		if len(r.Matches) > 0 && searchContext >= 0 {
			lines := strings.Split(r.Entry.Content, "\n")
			shownLines := make(map[int]bool)

			for _, m := range r.Matches {
				// Show context lines
				start := m.Line - 1 - searchContext
				if start < 0 {
					start = 0
				}
				end := m.Line + searchContext
				if end >= len(lines) {
					end = len(lines) - 1
				}

				for i := start; i <= end; i++ {
					if !shownLines[i] {
						shownLines[i] = true
						line := lines[i]

						// Highlight matches in the matching line
						if i == m.Line-1 {
							line = highlightMatch(line, query)
						}
						fmt.Printf("  %3d: %s\n", i+1, line)
					}
				}
			}
		}
		fmt.Println()
	}

	return nil
}

func highlightMatch(line, query string) string {
	if query == "" {
		return line
	}

	lower := strings.ToLower(line)
	queryLower := strings.ToLower(query)

	var result strings.Builder
	lastEnd := 0

	for {
		idx := strings.Index(lower[lastEnd:], queryLower)
		if idx == -1 {
			result.WriteString(line[lastEnd:])
			break
		}

		start := lastEnd + idx
		end := start + len(query)

		result.WriteString(line[lastEnd:start])
		result.WriteString("\033[1;33m") // Bold yellow
		result.WriteString(line[start:end])
		result.WriteString("\033[0m")

		lastEnd = end
	}

	return result.String()
}

type searchResultJSON struct {
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Matches int    `json:"matches"`
}

func outputSearchJSON(results []*store.SearchResult) error {
	for _, r := range results {
		obj := searchResultJSON{
			Slug:    r.Entry.Slug,
			Title:   r.Entry.Title,
			Matches: len(r.Matches),
		}

		data, err := json.Marshal(obj)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	}
	return nil
}
