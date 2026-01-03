package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/queelius/jot/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init [directory]",
	Short: "Initialize a new jot journal",
	Long: `Initialize a new jot journal in the specified directory.
If no directory is specified, the current directory is used.

Creates:
  .jot/config.yaml  - Journal configuration
  entries/          - Entry storage directory`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	// Resolve to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Check if already initialized
	jotDir := filepath.Join(absDir, ".jot")
	if _, err := os.Stat(jotDir); err == nil {
		return fmt.Errorf("already a jot journal: %s", absDir)
	}

	// Create .jot directory
	if err := os.MkdirAll(jotDir, 0755); err != nil {
		return fmt.Errorf("creating .jot directory: %w", err)
	}

	// Create entries directory
	entriesDir := filepath.Join(absDir, "entries")
	if err := os.MkdirAll(entriesDir, 0755); err != nil {
		return fmt.Errorf("creating entries directory: %w", err)
	}

	// Create default config
	cfg := config.DefaultConfig()
	if err := cfg.Save(absDir); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Initialized jot journal in %s\n", absDir)
	return nil
}
