package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/queelius/jot/internal/entry"
)

var editCmd = &cobra.Command{
	Use:     "edit <slug>",
	Short:   "Edit an entry in your editor",
	GroupID: "modify",
	Long: `Open an entry in your editor for modification.

Supports partial slug matching. If the slug doesn't match exactly,
entries containing the slug will be found.

The entry's modified timestamp is updated when saved.

Examples:
  jot edit 20240102-api-redesign
  jot edit api-redesign              # partial match`,
	Args: cobra.ExactArgs(1),
	RunE: runEdit,
}

func init() {
	rootCmd.AddCommand(editCmd)
}

func runEdit(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	cfg, err := getConfig()
	if err != nil {
		return err
	}

	slug := args[0]
	e, err := ResolveSlug(s, slug)
	if err != nil {
		return err
	}

	// Get file info before editing
	beforeInfo, err := os.Stat(e.Path)
	if err != nil {
		return fmt.Errorf("checking file: %w", err)
	}

	// Open editor
	editor := cfg.GetEditor()
	editorCmd := exec.Command(editor, e.Path)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("running editor: %w", err)
	}

	// Check if file was modified
	afterInfo, err := os.Stat(e.Path)
	if err != nil {
		return fmt.Errorf("checking file: %w", err)
	}

	if afterInfo.ModTime().Equal(beforeInfo.ModTime()) {
		fmt.Println("No changes made.")
		return nil
	}

	// Re-parse the entry to update modified timestamp
	edited, err := entry.ParseFile(e.Path)
	if err != nil {
		return fmt.Errorf("parsing edited file: %w", err)
	}

	// Update modified timestamp and save
	if err := s.Update(edited); err != nil {
		return fmt.Errorf("saving entry: %w", err)
	}

	fmt.Printf("Updated: %s\n", e.Slug)
	return nil
}
