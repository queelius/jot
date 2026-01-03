package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/queelius/jot/internal/entry"
)

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new entry in your editor",
	Long: `Create a new entry by opening your editor with a template.

The entry is saved when you close the editor.
If you close without saving or with empty content, the entry is discarded.

Examples:
  jot new
  jot new --type=idea --tags=api,architecture
  jot new --title="API Redesign Proposal"`,
	RunE: runNew,
}

var (
	newTitle    string
	newType     string
	newTags     string
	newPriority string
	newDue      string
)

func init() {
	newCmd.Flags().StringVar(&newTitle, "title", "", "entry title")
	newCmd.Flags().StringVarP(&newType, "type", "t", "", "entry type (idea, task, note, plan, log)")
	newCmd.Flags().StringVar(&newTags, "tags", "", "comma-separated tags")
	newCmd.Flags().StringVarP(&newPriority, "priority", "p", "", "priority (low, medium, high, critical)")
	newCmd.Flags().StringVarP(&newDue, "due", "d", "", "due date (YYYY-MM-DD)")

	rootCmd.AddCommand(newCmd)
}

func runNew(cmd *cobra.Command, args []string) error {
	root, err := getRoot()
	if err != nil {
		return err
	}

	cfg, err := getConfig()
	if err != nil {
		return err
	}

	s, err := getStore()
	if err != nil {
		return err
	}

	now := time.Now()

	// Create template entry
	e := &entry.Entry{
		Title:    newTitle,
		Created:  now,
		Modified: now,
	}

	// Set type
	if newType != "" {
		e.Type = newType
	} else if cfg.Defaults.Type != "" {
		e.Type = cfg.Defaults.Type
	}

	// Set tags
	if newTags != "" {
		e.Tags = parseTags(newTags)
	} else if len(cfg.Defaults.Tags) > 0 {
		e.Tags = cfg.Defaults.Tags
	}

	// Set other fields
	if newPriority != "" {
		e.Priority = newPriority
	}
	if newDue != "" {
		e.Due = newDue
	}
	if e.Type == "task" {
		e.Status = "open"
	}

	// Create temporary file with template
	tmpDir := filepath.Join(root, ".jot", "tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}

	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("new-%d.md", now.Unix()))
	template := e.ToMarkdown()

	// Add placeholder content if empty
	if e.Content == "" {
		template += "\n\nWrite your content here...\n"
	}

	if err := os.WriteFile(tmpFile, []byte(template), 0644); err != nil {
		return fmt.Errorf("writing template: %w", err)
	}
	defer os.Remove(tmpFile)

	// Get file info before editing
	beforeInfo, _ := os.Stat(tmpFile)

	// Open editor
	editor := cfg.GetEditor()
	editorCmd := exec.Command(editor, tmpFile)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("running editor: %w", err)
	}

	// Check if file was modified
	afterInfo, err := os.Stat(tmpFile)
	if err != nil {
		return fmt.Errorf("checking file: %w", err)
	}

	if afterInfo.ModTime().Equal(beforeInfo.ModTime()) {
		fmt.Println("No changes made, entry discarded.")
		return nil
	}

	// Read and parse the edited content
	content, err := os.ReadFile(tmpFile)
	if err != nil {
		return fmt.Errorf("reading edited file: %w", err)
	}

	// Check for empty content
	trimmed := strings.TrimSpace(string(content))
	if trimmed == "" {
		fmt.Println("Empty content, entry discarded.")
		return nil
	}

	// Parse the edited entry
	edited, err := entry.Parse(string(content))
	if err != nil {
		return fmt.Errorf("parsing edited content: %w", err)
	}

	// Remove placeholder content
	edited.Content = strings.TrimSuffix(edited.Content, "\nWrite your content here...\n")
	edited.Content = strings.TrimSuffix(edited.Content, "Write your content here...\n")
	edited.Content = strings.TrimSpace(edited.Content)

	// Preserve timestamps
	edited.Created = now
	edited.Modified = now

	// Validate
	if errs := edited.Validate(); len(errs) > 0 {
		return fmt.Errorf("validation failed: %v", errs[0])
	}

	// Generate slug from title
	edited.Slug = entry.GenerateSlug(edited.Title, now)

	// Save the entry
	if err := s.Create(edited); err != nil {
		return err
	}

	fmt.Printf("Created: %s\n", edited.Slug)
	return nil
}
