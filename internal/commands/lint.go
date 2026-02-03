package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var lintCmd = &cobra.Command{
	Use:   "lint [slug]",
	Short: "Validate entries",
	Long: `Validate entry frontmatter and structure.

Supports partial slug matching. If the slug doesn't match exactly,
entries containing the slug will be found.

Checks for:
- Valid YAML frontmatter
- Valid field values (types, statuses, priorities)
- Valid date formats
- Required fields

Examples:
  jot lint                    # Lint all entries
  jot lint 20240102-api-redesign  # Lint specific entry
  jot lint api-redesign           # Partial match`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLint,
}

func init() {
	rootCmd.AddCommand(lintCmd)
}

type lintResult struct {
	Slug   string   `json:"slug"`
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

func runLint(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	var results []lintResult

	if len(args) > 0 {
		// Lint specific entry
		slug := args[0]
		e, err := ResolveSlug(s, slug)
		if err != nil {
			return err
		}

		errs := e.Validate()
		result := lintResult{
			Slug:  e.Slug,
			Valid: len(errs) == 0,
		}
		for _, validErr := range errs {
			result.Errors = append(result.Errors, validErr.Error())
		}
		results = append(results, result)
	} else {
		// Lint all entries
		entries, err := s.List(nil)
		if err != nil {
			return err
		}

		for _, e := range entries {
			errs := e.Validate()
			result := lintResult{
				Slug:  e.Slug,
				Valid: len(errs) == 0,
			}
			for _, err := range errs {
				result.Errors = append(result.Errors, err.Error())
			}
			results = append(results, result)
		}
	}

	// Output results
	format := getOutputFormat()
	if format == "json" {
		return outputLintJSON(results)
	}

	return outputLintHuman(results)
}

func outputLintJSON(results []lintResult) error {
	for _, r := range results {
		data, err := json.Marshal(r)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	}
	return nil
}

func outputLintHuman(results []lintResult) error {
	hasErrors := false

	for _, r := range results {
		if !r.Valid {
			hasErrors = true
			fmt.Printf("\033[31m✗\033[0m %s\n", r.Slug)
			for _, e := range r.Errors {
				fmt.Printf("  - %s\n", e)
			}
		}
	}

	if !hasErrors {
		validCount := len(results)
		if validCount == 1 {
			fmt.Println("\033[32m✓\033[0m Entry is valid")
		} else {
			fmt.Printf("\033[32m✓\033[0m All %d entries are valid\n", validCount)
		}
		return nil
	}

	// Count errors
	errorCount := 0
	for _, r := range results {
		if !r.Valid {
			errorCount++
		}
	}

	fmt.Printf("\n%d of %d entries have errors\n", errorCount, len(results))
	os.Exit(1)
	return nil
}
