package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var doneCmd = &cobra.Command{
	Use:   "done <slug>",
	Short: "Mark a task as done",
	Long: `Mark a task as completed.

This is a shortcut for: jot status <slug> done

Examples:
  jot done 20240102-fix-login-bug`,
	Args: cobra.ExactArgs(1),
	RunE: runDone,
}

func init() {
	rootCmd.AddCommand(doneCmd)
}

func runDone(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	slug := args[0]
	e, err := s.Get(slug)
	if err != nil {
		return fmt.Errorf("entry not found: %s", slug)
	}

	if e.Type != "task" {
		return fmt.Errorf("entry is not a task: %s (type: %s)", slug, e.Type)
	}

	if e.Status == "done" {
		fmt.Printf("Already done: %s\n", slug)
		return nil
	}

	e.Status = "done"
	if err := s.Update(e); err != nil {
		return err
	}

	fmt.Printf("Marked done: %s\n", slug)
	return nil
}
