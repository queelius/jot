// Package commands implements the jot CLI commands.
package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/queelius/jot/internal/config"
	"github.com/queelius/jot/internal/store"
)

var (
	// Global flags
	formatFlag string
	prettyFlag bool

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
	rootCmd.PersistentFlags().StringVar(&formatFlag, "format", "", "output format: json, table, markdown")
	rootCmd.PersistentFlags().BoolVar(&prettyFlag, "pretty", false, "pretty-print JSON output")
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

// mustGetStore returns the store or exits with an error.
func mustGetStore() *store.Store {
	s, err := getStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return s
}

// getOutputFormat returns the output format from flags or config.
func getOutputFormat() string {
	if formatFlag != "" {
		return formatFlag
	}

	cfg, err := getConfig()
	if err != nil {
		return "json"
	}

	if cfg.Output.Format != "" {
		return cfg.Output.Format
	}

	return "json"
}

// shouldPrettyPrint returns whether to pretty-print JSON.
func shouldPrettyPrint() bool {
	if prettyFlag {
		return true
	}

	cfg, err := getConfig()
	if err != nil {
		return false
	}

	return cfg.Output.Pretty
}
