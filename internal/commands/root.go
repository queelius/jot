// Package commands implements the jot CLI commands.
package commands

import (
	"github.com/spf13/cobra"

	"github.com/queelius/jot/internal/config"
	"github.com/queelius/jot/internal/store"
)

var (
	// Global output format flags (mutually exclusive)
	jsonFlag     bool
	tableFlag    bool
	markdownFlag bool
	fuzzyFlag    bool

	// Cached values
	cachedRoot   string
	cachedConfig *config.Config
	cachedStore  *store.Store
)

var rootCmd = &cobra.Command{
	Use:   "jot",
	Short: "A plaintext idea toolkit for the LLM era",
	Long: `jot is a CLI-first, plaintext-native toolkit for capturing and organizing
ideas, plans, tasks, and notes.

Designed for the LLM era: simple primitives, predictable structure,
machine-readable output. Intelligence lives in the LLM layer (Claude Code).`,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddGroup(
		&cobra.Group{ID: "create", Title: "Create:"},
		&cobra.Group{ID: "query", Title: "View and Query:"},
		&cobra.Group{ID: "modify", Title: "Modify:"},
		&cobra.Group{ID: "lifecycle", Title: "Lifecycle:"},
		&cobra.Group{ID: "data", Title: "Data:"},
		&cobra.Group{ID: "admin", Title: "Admin:"},
	)

	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "output as JSON")
	rootCmd.PersistentFlags().BoolVar(&tableFlag, "table", false, "output as table")
	rootCmd.PersistentFlags().BoolVar(&markdownFlag, "markdown", false, "output as markdown")
	rootCmd.PersistentFlags().BoolVar(&markdownFlag, "md", false, "output as markdown (alias)")
	rootCmd.PersistentFlags().BoolVar(&fuzzyFlag, "fuzzy", false, "use fuzzy matching for tags and search")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// getRoot returns the journal root directory, caching the result.
func getRoot() (string, error) {
	if cachedRoot != "" {
		return cachedRoot, nil
	}

	root, err := config.FindRoot()
	if err != nil {
		return "", err
	}

	cachedRoot = root
	return root, nil
}

// getConfig returns the journal configuration, caching the result.
func getConfig() (*config.Config, error) {
	if cachedConfig != nil {
		return cachedConfig, nil
	}

	root, err := getRoot()
	if err != nil {
		return nil, err
	}

	cfg, err := config.Load(root)
	if err != nil {
		return nil, err
	}

	cachedConfig = cfg
	return cfg, nil
}

// getStore returns the entry store, caching the result.
func getStore() (*store.Store, error) {
	if cachedStore != nil {
		return cachedStore, nil
	}

	root, err := getRoot()
	if err != nil {
		return nil, err
	}

	cachedStore = store.New(root)
	return cachedStore, nil
}

// getFuzzy returns whether fuzzy matching is enabled.
func getFuzzy() bool {
	return fuzzyFlag
}

// getOutputFormat returns the output format from flags or config.
func getOutputFormat() string {
	// Check individual flags first
	if jsonFlag {
		return "json"
	}
	if tableFlag {
		return "table"
	}
	if markdownFlag {
		return "markdown"
	}

	// Fall back to config
	cfg, err := getConfig()
	if err != nil {
		return "table"
	}

	if cfg.Output.Format != "" {
		return cfg.Output.Format
	}

	return "table"
}
