package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/queelius/jot/internal/entry"
	"github.com/queelius/jot/internal/store"
)

var exportCmd = &cobra.Command{
	Use:     "export",
	Short:   "Export entries",
	GroupID: "data",
	Long: `Export entries to JSON or markdown format.

Supports filtering like the list command.

Examples:
  jot export > backup.json
  jot export --format=markdown > backup.md
  jot export --type=task --status=open > open-tasks.json`,
	RunE: runExport,
}

var (
	exportType   string
	exportTag    string
	exportStatus string
	exportSince  string
)

func init() {
	exportCmd.Flags().StringVarP(&exportType, "type", "t", "", "filter by type")
	exportCmd.Flags().StringVar(&exportTag, "tag", "", "filter by tag")
	exportCmd.Flags().StringVarP(&exportStatus, "status", "s", "", "filter by status")
	exportCmd.Flags().StringVar(&exportSince, "since", "", "entries created since")

	rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	filter := &store.Filter{
		Type:   exportType,
		Tag:    exportTag,
		Status: exportStatus,
	}

	if exportSince != "" {
		if dur, err := store.ParseDuration(exportSince); err == nil && dur > 0 {
			filter.Since = time.Now().Add(-dur)
		} else if t, err := store.ParseDate(exportSince); err == nil {
			filter.Since = t
		}
	}

	entries, err := s.List(filter)
	if err != nil {
		return err
	}

	format := getOutputFormat()
	switch format {
	case "markdown":
		return exportEntriesMarkdown(entries)
	default:
		return exportEntriesJSON(entries)
	}
}

func exportEntriesJSON(entries []*entry.Entry) error {
	// Build export object
	type exportData struct {
		Version  string         `json:"version"`
		Exported string         `json:"exported"`
		Count    int            `json:"count"`
		Entries  []*entry.Entry `json:"entries"`
	}

	export := exportData{
		Version:  "1.0",
		Exported: time.Now().Format(time.RFC3339),
		Count:    len(entries),
		Entries:  entries,
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}

func exportEntriesMarkdown(entries []*entry.Entry) error {
	for i, e := range entries {
		if i > 0 {
			fmt.Print("\n---\n\n")
		}

		// Build frontmatter
		fmt.Println("---")
		fmt.Printf("slug: %s\n", e.Slug)
		fmt.Printf("title: %s\n", e.Title)
		if e.Type != "" {
			fmt.Printf("type: %s\n", e.Type)
		}
		if len(e.Tags) > 0 {
			fmt.Printf("tags: [%s]\n", strings.Join(e.Tags, ", "))
		}
		if e.Status != "" {
			fmt.Printf("status: %s\n", e.Status)
		}
		if e.Priority != "" {
			fmt.Printf("priority: %s\n", e.Priority)
		}
		if e.Due != "" {
			fmt.Printf("due: %s\n", e.Due)
		}
		fmt.Printf("created: %s\n", e.Created.Format(time.RFC3339))
		fmt.Printf("modified: %s\n", e.Modified.Format(time.RFC3339))
		fmt.Println("---")

		if e.Content != "" {
			fmt.Printf("\n%s\n", e.Content)
		}
	}

	fmt.Fprintf(os.Stderr, "Exported %d entries\n", len(entries))
	return nil
}
