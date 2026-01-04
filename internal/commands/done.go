package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var doneCmd = &cobra.Command{
	Use:   "done <slug>",
	Short: "Mark a task as done",
	Long: `Mark a task as completed.

Supports partial slug matching. If the slug doesn't match exactly,
entries containing the slug will be found.

This is a shortcut for: jot status <slug> done

Examples:
  jot done 20240102-fix-login-bug
  jot done fix-login                 # partial match`,
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
	e, err := ResolveSlug(s, slug)
	if err != nil {
		return err
	}

	if e.Type != "task" {
		return fmt.Errorf("entry is not a task: %s (type: %s)", e.Slug, e.Type)
	}

	if e.Status == "done" {
		fmt.Printf("Already done: %s\n", e.Slug)
		return nil
	}

	e.Status = "done"
	if err := s.Update(e); err != nil {
		return err
	}

	fmt.Printf("Marked done: %s\n", e.Slug)
	return nil
}
