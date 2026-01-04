package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <slug>",
	Short: "Display an entry",
	Long: `Display the contents of an entry.

Supports partial slug matching. If the slug doesn't match exactly,
entries containing the slug will be found.

By default, renders markdown in the terminal.

Examples:
  jot show api-redesign              # partial match OK
  jot show api-redesign --raw        # raw markdown
  jot show api-redesign --json       # JSON output`,
	Args: cobra.ExactArgs(1),
	RunE: runShow,
}

var (
	showRaw  bool
	showMeta bool
)

func init() {
	showCmd.Flags().BoolVar(&showRaw, "raw", false, "output raw markdown (no rendering)")
	showCmd.Flags().BoolVar(&showMeta, "meta", false, "include sidecar metadata (with --json)")

	rootCmd.AddCommand(showCmd)
}

func runShow(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	slug := args[0]
	e, err := ResolveSlug(s, slug)
	if err != nil {
		return err
	}

	if jsonFlag {
		return outputEntryJSON(e, showMeta)
	}

	if showRaw {
		fmt.Print(e.ToMarkdown())
		return nil
	}

	// Render markdown
	return renderMarkdown(e.ToMarkdown())
}

func outputEntryJSON(e interface{ ToJSONPretty() ([]byte, error) }, includeMeta bool) error {
	data, err := e.ToJSONPretty()
	if err != nil {
		return err
	}

	if includeMeta {
		// For now, just output the entry. Sidecar metadata would be added here.
		var obj map[string]interface{}
		if err := json.Unmarshal(data, &obj); err != nil {
			return err
		}
		obj["meta"] = nil // Placeholder for sidecar metadata
		data, err = json.MarshalIndent(obj, "", "  ")
		if err != nil {
			return err
		}
	}

	fmt.Println(string(data))
	return nil
}

func renderMarkdown(content string) error {
	// Check if stdout is a terminal
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) == 0 {
		// Not a terminal, output raw
		fmt.Print(content)
		return nil
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		// Fallback to raw output
		fmt.Print(content)
		return nil
	}

	out, err := renderer.Render(content)
	if err != nil {
		// Fallback to raw output
		fmt.Print(content)
		return nil
	}

	fmt.Print(out)
	return nil
}
