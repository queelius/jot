package commands

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/queelius/jot/internal/config"
)

var whichCmd = &cobra.Command{
	Use:   "which",
	Short: "Show which journal is active",
	Long: `Show which journal is currently active.

Displays the journal root path and whether it's a local or global journal.

Examples:
  jot which
  jot which --json`,
	RunE: runWhich,
}

func init() {
	rootCmd.AddCommand(whichCmd)
}

func runWhich(cmd *cobra.Command, args []string) error {
	info, err := config.FindRootWithInfo()
	if err != nil {
		return err
	}

	cfg, err := config.Load(info.Path)
	if err != nil {
		return err
	}

	format := getOutputFormat()
	if format == "json" {
		return outputWhichJSON(info, cfg)
	}

	// Human-readable output
	scope := "local"
	if info.IsGlobal {
		scope = "global"
	}

	fmt.Printf("Journal: %s\n", cfg.Name)
	fmt.Printf("Path:    %s\n", info.Path)
	fmt.Printf("Scope:   %s\n", scope)

	return nil
}

func outputWhichJSON(info *config.RootInfo, cfg *config.Config) error {
	data := map[string]interface{}{
		"name":   cfg.Name,
		"path":   info.Path,
		"global": info.IsGlobal,
	}

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}
