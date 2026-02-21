package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var claudeCmd = &cobra.Command{
	Use:     "claude",
	Short:   "Claude Code integration",
	GroupID: "admin",
	Long:    `Commands for integrating jot with Claude Code.`,
}

var claudeInstallCmd = &cobra.Command{
	Use:        "install",
	Short:      "Install jot skill for Claude Code",
	Deprecated: "use the jot plugin from https://github.com/queelius/alex-claude-plugins",
	RunE:       runClaudeInstall,
}

var claudeShowCmd = &cobra.Command{
	Use:        "show",
	Short:      "Print the jot skill content",
	Deprecated: "use the jot plugin from https://github.com/queelius/alex-claude-plugins",
	RunE:       runClaudeShow,
}

func init() {
	claudeCmd.AddCommand(claudeInstallCmd)
	claudeCmd.AddCommand(claudeShowCmd)
	rootCmd.AddCommand(claudeCmd)
}

func runClaudeInstall(cmd *cobra.Command, args []string) error {
	fmt.Fprintln(os.Stderr, "The jot skill has moved to a standalone Claude Code plugin.")
	fmt.Fprintln(os.Stderr, "Install from: https://github.com/queelius/alex-claude-plugins")
	return nil
}

func runClaudeShow(cmd *cobra.Command, args []string) error {
	fmt.Fprintln(os.Stderr, "The jot skill has moved to a standalone Claude Code plugin.")
	fmt.Fprintln(os.Stderr, "Install from: https://github.com/queelius/alex-claude-plugins")
	return nil
}
