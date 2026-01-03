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

Valid statuses: open, in_progress, done, blocked, archived

Examples:
  jot status 20240102-api-redesign in_progress
  jot status 20240102-api-redesign blocked`,
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

	e, err := s.Get(slug)
	if err != nil {
		return fmt.Errorf("entry not found: %s", slug)
	}

	if e.Status == newStatus {
		fmt.Printf("Status unchanged: %s (%s)\n", slug, newStatus)
		return nil
	}

	oldStatus := e.Status
	e.Status = newStatus
	if err := s.Update(e); err != nil {
		return err
	}

	if oldStatus == "" {
		fmt.Printf("Status set: %s -> %s\n", slug, newStatus)
	} else {
		fmt.Printf("Status changed: %s -> %s (was: %s)\n", slug, newStatus, oldStatus)
	}
	return nil
}
