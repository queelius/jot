package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <slug>",
	Short: "Remove an entry",
	Long: `Remove an entry, its sidecar metadata, and asset directory.

By default, asks for confirmation. Use --force to skip.

Examples:
  jot rm 20240102-api-redesign
  jot rm 20240102-api-redesign --force`,
	Aliases: []string{"remove", "delete"},
	Args:    cobra.ExactArgs(1),
	RunE:    runRm,
}

var rmForce bool

func init() {
	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "skip confirmation")

	rootCmd.AddCommand(rmCmd)
}

func runRm(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	slug := args[0]

	// Check if entry exists
	e, err := s.Get(slug)
	if err != nil {
		return fmt.Errorf("entry not found: %s", slug)
	}

	// Confirm deletion
	if !rmForce {
		fmt.Printf("Delete '%s' (%s)? [y/N] ", e.Title, slug)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := s.Delete(slug); err != nil {
		return err
	}

	fmt.Printf("Deleted: %s\n", slug)
	return nil
}
