package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/queelius/jot/internal/entry"
)

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import entries",
	Long: `Import entries from a JSON export file.

The file should be in the format produced by 'jot export'.

Examples:
  jot import backup.json
  cat backup.json | jot import -`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

// entryImporter is the interface required for importing entries.
type entryImporter interface {
	Create(*entry.Entry) error
	Exists(string) bool
}

var (
	importDryRun bool
	importSkip   bool
)

func init() {
	importCmd.Flags().BoolVar(&importDryRun, "dry-run", false, "show what would be imported without importing")
	importCmd.Flags().BoolVar(&importSkip, "skip-existing", false, "skip entries that already exist")

	rootCmd.AddCommand(importCmd)
}

type importData struct {
	Version  string          `json:"version"`
	Exported string          `json:"exported"`
	Count    int             `json:"count"`
	Entries  json.RawMessage `json:"entries"`
}

func runImport(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	// Read input
	var data []byte
	if args[0] == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(args[0])
	}
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	// Try to parse as export format first
	var export importData
	if err := json.Unmarshal(data, &export); err == nil && export.Version != "" {
		return importFromExport(s, export.Entries)
	}

	// Try to parse as raw array of entries
	return importFromArray(s, data)
}

func importFromExport(s entryImporter, entriesData json.RawMessage) error {
	var entries []json.RawMessage
	if err := json.Unmarshal(entriesData, &entries); err != nil {
		return fmt.Errorf("parsing entries: %w", err)
	}

	return importEntries(s, entries)
}

func importFromArray(s entryImporter, data []byte) error {
	var entries []json.RawMessage
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("parsing entries: %w", err)
	}

	return importEntries(s, entries)
}

func importEntries(s entryImporter, entries []json.RawMessage) error {
	var imported, skipped, errors int

	for _, raw := range entries {
		var e entry.Entry
		if err := json.Unmarshal(raw, &e); err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping invalid entry: %v\n", err)
			errors++
			continue
		}

		// Check for required fields
		if e.Title == "" {
			fmt.Fprintf(os.Stderr, "warning: skipping entry without title\n")
			errors++
			continue
		}

		// Check if exists
		if e.Slug != "" && s.Exists(e.Slug) {
			if importSkip {
				skipped++
				continue
			}
			fmt.Fprintf(os.Stderr, "warning: entry already exists: %s\n", e.Slug)
			errors++
			continue
		}

		if importDryRun {
			fmt.Printf("Would import: %s (%s)\n", e.Slug, e.Title)
			imported++
			continue
		}

		// Generate slug if not present
		if e.Slug == "" {
			e.Slug = entry.GenerateSlug(e.Title, e.Created)
		}

		if err := s.Create(&e); err != nil {
			if strings.Contains(err.Error(), "already exists") && importSkip {
				skipped++
				continue
			}
			fmt.Fprintf(os.Stderr, "warning: failed to import %s: %v\n", e.Slug, err)
			errors++
			continue
		}

		fmt.Printf("Imported: %s\n", e.Slug)
		imported++
	}

	// Summary
	if importDryRun {
		fmt.Printf("\nDry run: would import %d entries\n", imported)
	} else {
		fmt.Printf("\nImported %d entries", imported)
		if skipped > 0 {
			fmt.Printf(", skipped %d", skipped)
		}
		if errors > 0 {
			fmt.Printf(", %d errors", errors)
		}
		fmt.Println()
	}

	return nil
}
