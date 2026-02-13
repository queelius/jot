package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/queelius/jot/internal/store"
)

var archiveCmd = &cobra.Command{
	Use:     "archive",
	Short:   "Bulk archive entries",
	GroupID: "lifecycle",
	Long: `Set status to "archived" for multiple entries at once.

Requires one mode flag: --stale, --older-than, or --status.
Dry-run by default — shows what would change. Use --confirm to execute.

Modes:
  --stale              Archive everything 'jot stale' would find
  --older-than 90d     Archive entries not modified in the given duration
  --status done        Archive all entries with the given status

Narrowing flags (combine with any mode):
  --type, --tags

Examples:
  jot archive --stale                      # Preview stale entries to archive
  jot archive --stale --confirm            # Actually archive them
  jot archive --stale --days 30 --confirm  # Archive entries stale for 30+ days
  jot archive --older-than 6m              # Preview entries older than 6 months
  jot archive --status done --confirm      # Archive all done entries
  jot archive --stale --type=idea          # Only stale ideas`,
	RunE: runArchive,
}

var (
	archiveStale    bool
	archiveOlderStr string
	archiveStatus   string
	archiveDays     int
	archiveType     string
	archiveTag      string
	archiveConfirm  bool
)

func init() {
	archiveCmd.Flags().BoolVar(&archiveStale, "stale", false, "archive stale entries (active, not modified recently)")
	archiveCmd.Flags().StringVar(&archiveOlderStr, "older-than", "", "archive entries older than duration (e.g., 90d, 6m, 1y)")
	archiveCmd.Flags().StringVar(&archiveStatus, "status", "", "archive entries with this status (e.g., done)")
	archiveCmd.Flags().IntVar(&archiveDays, "days", 90, "staleness threshold in days (used with --stale)")
	archiveCmd.Flags().StringVarP(&archiveType, "type", "t", "", "filter by type")
	archiveCmd.Flags().StringVar(&archiveTag, "tags", "", "filter by tag")
	archiveCmd.Flags().BoolVar(&archiveConfirm, "confirm", false, "actually perform the archive (default is dry-run)")

	rootCmd.AddCommand(archiveCmd)
}

func runArchive(cmd *cobra.Command, args []string) error {
	// Require exactly one mode
	modeCount := 0
	if archiveStale {
		modeCount++
	}
	if archiveOlderStr != "" {
		modeCount++
	}
	if archiveStatus != "" {
		modeCount++
	}
	if modeCount == 0 {
		return fmt.Errorf("one mode required: --stale, --older-than, or --status")
	}
	if modeCount > 1 {
		return fmt.Errorf("only one mode allowed: --stale, --older-than, or --status")
	}

	s, err := getStore()
	if err != nil {
		return err
	}

	var candidates []*entryAction

	switch {
	case archiveStale:
		threshold := time.Duration(archiveDays) * 24 * time.Hour
		entries, err := findStaleEntries(s, threshold, archiveType, archiveTag)
		if err != nil {
			return err
		}
		for _, e := range entries {
			candidates = append(candidates, &entryAction{slug: e.Slug, title: e.Title, reason: "stale"})
		}

	case archiveOlderStr != "":
		dur, err := store.ParseDuration(archiveOlderStr)
		if err != nil || dur <= 0 {
			return fmt.Errorf("invalid duration: %s", archiveOlderStr)
		}
		// Exclude already-archived entries
		entries, err := findEntriesOlderThan(s, dur, archiveType, archiveTag, "archived")
		if err != nil {
			return err
		}
		for _, e := range entries {
			candidates = append(candidates, &entryAction{slug: e.Slug, title: e.Title, reason: "older-than " + archiveOlderStr})
		}

	case archiveStatus != "":
		if archiveStatus == "archived" {
			return fmt.Errorf("cannot archive entries that are already archived")
		}
		filter := &store.Filter{
			Status: archiveStatus,
			Type:   archiveType,
			Tag:    archiveTag,
		}
		entries, err := s.List(filter)
		if err != nil {
			return err
		}
		for _, e := range entries {
			candidates = append(candidates, &entryAction{slug: e.Slug, title: e.Title, reason: "status=" + archiveStatus})
		}
	}

	if len(candidates) == 0 {
		fmt.Println("No entries to archive.")
		return nil
	}

	// Dry-run: show what would be archived
	if !archiveConfirm {
		fmt.Printf("Would archive %d entries (use --confirm to execute):\n\n", len(candidates))
		for _, c := range candidates {
			fmt.Printf("  %s  %s  (%s)\n", truncateSlug(c.slug), truncateTitle(c.title), c.reason)
		}
		return nil
	}

	// Execute: archive each entry
	archived := 0
	for _, c := range candidates {
		e, err := s.Get(c.slug)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", c.slug, err)
			continue
		}
		e.Status = "archived"
		if err := s.Update(e); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to archive %s: %v\n", c.slug, err)
			continue
		}
		archived++
	}

	fmt.Printf("Archived %d entries.\n", archived)
	return nil
}

// entryAction tracks an entry targeted by a bulk operation.
type entryAction struct {
	slug   string
	title  string
	reason string
}
