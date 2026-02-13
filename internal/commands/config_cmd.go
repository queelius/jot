package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/queelius/jot/internal/config"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:     "config [key] [value]",
	Short:   "View or modify configuration",
	GroupID: "admin",
	Long: `View or modify journal configuration.

With no arguments, shows the full configuration.
With one argument, shows the value of that key.
With two arguments, sets the key to the value.

Available keys:
  name              - Journal name
  description       - Journal description
  editor            - Editor command
  date_format       - Date format (Go time format)
  defaults.type     - Default entry type
  output.format     - Default output format (json, table, markdown)
  output.pretty     - Pretty-print JSON (true/false)
  output.color      - Color output (auto, always, never)

Examples:
  jot config                        # Show all config
  jot config editor                 # Get editor setting
  jot config set editor "code -w"  # Set editor`,
	Args: cobra.MaximumNArgs(2),
	RunE: runConfig,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE:  runConfigSet,
}

func init() {
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	root, err := getRoot()
	if err != nil {
		return err
	}

	cfg, err := config.Load(root)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		// Show full config
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		fmt.Print(string(data))
		return nil
	}

	// Get specific key
	key := args[0]
	value, err := cfg.Get(key)
	if err != nil {
		return err
	}

	fmt.Println(value)
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	root, err := getRoot()
	if err != nil {
		return err
	}

	cfg, err := config.Load(root)
	if err != nil {
		return err
	}

	key := args[0]
	value := args[1]

	if err := cfg.Set(key, value); err != nil {
		return err
	}

	if err := cfg.Save(root); err != nil {
		return err
	}

	fmt.Printf("Set %s = %s\n", key, value)
	return nil
}
