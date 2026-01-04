package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/queelius/jot/internal/entry"
)

var statusCmd = &cobra.Command{
	Use:   "status <slug> <status>",
	Short: "Change entry status",
	Long: `Change the status of an entry.

Supports partial slug matching. If the slug doesn't match exactly,
entries containing the slug will be found.

Valid statuses: open, in_progress, done, blocked, archived

Examples:
  jot status 20240102-api-redesign in_progress
  jot status api-redesign blocked    # partial match`,
	Args: cobra.ExactArgs(2),
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	slug := args[0]
	newStatus := strings.ToLower(args[1])

	// Validate status
	valid := false
	for _, vs := range entry.ValidStatuses {
		if vs == newStatus {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid status %q, must be one of: %v", newStatus, entry.ValidStatuses)
	}

	e, err := ResolveSlug(s, slug)
	if err != nil {
		return err
	}

	if e.Status == newStatus {
		fmt.Printf("Status unchanged: %s (%s)\n", e.Slug, newStatus)
		return nil
	}

	oldStatus := e.Status
	e.Status = newStatus
	if err := s.Update(e); err != nil {
		return err
	}

	if oldStatus == "" {
		fmt.Printf("Status set: %s -> %s\n", e.Slug, newStatus)
	} else {
		fmt.Printf("Status changed: %s -> %s (was: %s)\n", e.Slug, newStatus, oldStatus)
	}
	return nil
}
