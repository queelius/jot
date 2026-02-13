package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:     "rm <slug>",
	Short:   "Remove an entry",
	GroupID: "lifecycle",
	Long: `Remove an entry, its sidecar metadata, and asset directory.

Supports partial slug matching. If the slug doesn't match exactly,
entries containing the slug will be found and offered for deletion.

By default, asks for confirmation. Use --yes to skip.

Examples:
  jot rm 20240102-api-redesign
  jot rm api-redesign              # partial match
  jot rm api-redesign --yes        # skip confirmation`,
	Aliases: []string{"remove", "delete"},
	Args:    cobra.ExactArgs(1),
	RunE:    runRm,
}

var rmYes bool

func init() {
	rmCmd.Flags().BoolVarP(&rmYes, "yes", "y", false, "skip confirmation")

	rootCmd.AddCommand(rmCmd)
}

func runRm(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	slug := args[0]

	// Resolve slug (supports partial matching)
	e, err := ResolveSlug(s, slug)
	if err != nil {
		return err
	}

	// Confirm deletion
	if !rmYes {
		fmt.Printf("Delete '%s' (%s)? [y/N] ", e.Title, e.Slug)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := s.Delete(e.Slug); err != nil {
		return err
	}

	fmt.Printf("Deleted: %s\n", e.Slug)
	return nil
}
