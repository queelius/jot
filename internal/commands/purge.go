package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/queelius/jot/internal/entry"
	"github.com/queelius/jot/internal/store"
)

var purgeCmd = &cobra.Command{
	Use:     "purge",
	Short:   "Permanently delete archived entries",
	GroupID: "lifecycle",
	Long: `Permanently remove archived entries from disk.

Safety: Only targets entries with status "archived". You must archive
entries before purging them.

Requires one mode flag: --all or --older-than.
Requires --force flag AND interactive confirmation to execute.

Modes:
  --all                Purge all archived entries
  --older-than 180d    Purge archived entries older than the given duration

Narrowing flags (combine with any mode):
  --type, --tags

Examples:
  jot purge --all                          # Preview what would be purged
  jot purge --all --force                  # Purge with interactive confirmation
  jot purge --all --force --yes            # Purge without interactive prompt
  jot purge --older-than 6m --force        # Purge old archived entries
  jot purge --all --type=note --force      # Purge only archived notes`,
	RunE: runPurge,
}

var (
	purgeAll      bool
	purgeOlderStr string
	purgeType     string
	purgeTag      string
	purgeForce bool
	purgeYes   bool
)

func init() {
	purgeCmd.Flags().BoolVar(&purgeAll, "all", false, "purge all archived entries")
	purgeCmd.Flags().StringVar(&purgeOlderStr, "older-than", "", "purge archived entries older than duration (e.g., 180d, 1y)")
	purgeCmd.Flags().StringVarP(&purgeType, "type", "t", "", "filter by type")
	purgeCmd.Flags().StringVar(&purgeTag, "tags", "", "filter by tag")
	purgeCmd.Flags().BoolVar(&purgeForce, "force", false, "required to actually delete (with interactive confirmation)")
	purgeCmd.Flags().BoolVarP(&purgeYes, "yes", "y", false, "skip interactive confirmation (use with --force)")

	rootCmd.AddCommand(purgeCmd)
}

func runPurge(cmd *cobra.Command, args []string) error {
	// Require exactly one mode
	modeCount := 0
	if purgeAll {
		modeCount++
	}
	if purgeOlderStr != "" {
		modeCount++
	}
	if modeCount == 0 {
		return fmt.Errorf("one mode required: --all or --older-than")
	}
	if modeCount > 1 {
		return fmt.Errorf("only one mode allowed: --all or --older-than")
	}

	s, err := getStore()
	if err != nil {
		return err
	}

	var candidates []*entry.Entry

	switch {
	case purgeAll:
		filter := &store.Filter{
			Status: "archived",
			Type:   purgeType,
			Tag:    purgeTag,
		}
		candidates, err = s.List(filter)
		if err != nil {
			return err
		}

	case purgeOlderStr != "":
		dur, err := store.ParseDuration(purgeOlderStr)
		if err != nil || dur <= 0 {
			return fmt.Errorf("invalid duration: %s", purgeOlderStr)
		}
		filter := &store.Filter{
			Status: "archived",
			Type:   purgeType,
			Tag:    purgeTag,
		}
		entries, err := s.List(filter)
		if err != nil {
			return err
		}
		cutoff := time.Now().Add(-dur)
		for _, e := range entries {
			if e.Modified.Before(cutoff) {
				candidates = append(candidates, e)
			}
		}
	}

	if len(candidates) == 0 {
		fmt.Println("No archived entries to purge.")
		return nil
	}

	// Always show what would be purged
	fmt.Printf("Found %d archived entries to purge:\n\n", len(candidates))
	for _, e := range candidates {
		fmt.Printf("  %s  %s\n", truncateSlug(e.Slug), truncateTitle(e.Title))
	}
	fmt.Println()

	if !purgeForce {
		fmt.Println("Use --force to permanently delete these entries.")
		return nil
	}

	// Interactive confirmation (skip with --yes)
	if !purgeYes {
		fmt.Printf("This will PERMANENTLY DELETE %d entries. Type YES to confirm: ", len(candidates))
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)

		if response != "YES" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Execute deletion
	deleted := 0
	for _, e := range candidates {
		if err := s.Delete(e.Slug); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to delete %s: %v\n", e.Slug, err)
			continue
		}
		deleted++
	}

	fmt.Printf("Purged %d entries.\n", deleted)
	return nil
}
