package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

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
		// List entries with this tag (delegate to list command)
		listTag = args[0]
		return runList(cmd, nil)
	}

	// List all tags
	tags, err := s.AllTags()
	if err != nil {
		return err
	}

	if len(tags) == 0 {
		fmt.Println("No tags found.")
		return nil
	}

	format := getOutputFormat()
	if format == "json" {
		return outputTagsJSON(tags)
	}

	// Sort tags by name
	names := make([]string, 0, len(tags))
	for name := range tags {
		names = append(names, name)
	}
	sort.Strings(names)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TAG\tCOUNT")
	for _, name := range names {
		fmt.Fprintf(w, "%s\t%d\n", name, tags[name])
	}
	return w.Flush()
}

func outputTagsJSON(tags map[string]int) error {
	for name, count := range tags {
		obj := map[string]interface{}{
			"tag":   name,
			"count": count,
		}
		data, err := json.Marshal(obj)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	}
	return nil
}
