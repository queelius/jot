package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var tagStdin bool

var tagCmd = &cobra.Command{
	Use:     "tag",
	Short:   "Manage entry tags",
	GroupID: "modify",
	Long: `Add, remove, or replace tags on entries.

Subcommands:
  add    Add tags to an entry (additive, deduplicates)
  rm     Remove tags from an entry (silent if missing)
  set    Replace all tags on an entry

Use --stdin to read slugs from stdin for batch operations.

Examples:
  jot tag add api-redesign backend,v2
  jot tag rm api-redesign legacy
  jot tag set api-redesign backend,v2,production

  # Batch: tag all open tasks with sprint-5
  jot list --type=task --status=open --json | jq -r '.slug' | jot tag add --stdin sprint-5`,
}

var tagAddCmd = &cobra.Command{
	Use:   "add <slug> <tag1,tag2,...>",
	Short: "Add tags to an entry",
	Long: `Add one or more tags to an entry. Existing tags are preserved;
duplicates are silently ignored.

With --stdin, reads slugs from stdin (one per line) and adds the
specified tags to all of them.

Examples:
  jot tag add api-redesign backend,v2
  jot list --type=task --json | jq -r '.slug' | jot tag add --stdin sprint-5`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runTagAdd,
}

var tagRmCmd = &cobra.Command{
	Use:     "rm <slug> <tag1,tag2,...>",
	Aliases: []string{"remove", "del"},
	Short:   "Remove tags from an entry",
	Long: `Remove one or more tags from an entry. Tags not present on
the entry are silently ignored.

With --stdin, reads slugs from stdin (one per line) and removes
the specified tags from all of them.

Examples:
  jot tag rm api-redesign legacy
  jot list --tags=old-name --json | jq -r '.slug' | jot tag rm --stdin old-name`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runTagRm,
}

var tagSetCmd = &cobra.Command{
	Use:   "set <slug> [tag1,tag2,...]",
	Short: "Replace all tags on an entry",
	Long: `Replace all tags on an entry with the specified tags.
If no tags are provided, clears all tags.

With --stdin, reads slugs from stdin (one per line) and sets
the same tags on all of them.

Examples:
  jot tag set api-redesign backend,v2,production
  jot tag set api-redesign                          # clears all tags`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runTagSet,
}

func init() {
	tagCmd.PersistentFlags().BoolVar(&tagStdin, "stdin", false, "read slugs from stdin (one per line)")
	tagCmd.AddCommand(tagAddCmd, tagRmCmd, tagSetCmd)
	rootCmd.AddCommand(tagCmd)
}

// tagInput holds the parsed positional arguments for a tag subcommand.
type tagInput struct {
	slugs []string
	tags  []string
}

// parseTagInput parses positional args and stdin into a unified tagInput.
// Without --stdin: args = [slug, tags?]  → slugs from args[0], tags from args[1]
// With    --stdin: args = [tags?]        → slugs from stdin,   tags from args[0]
func parseTagInput(args []string, useStdin bool) (*tagInput, error) {
	input := &tagInput{}

	if useStdin {
		if len(args) >= 1 {
			input.tags = parseTags(args[0])
		}
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				input.slugs = append(input.slugs, line)
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("reading stdin: %w", err)
		}
		if len(input.slugs) == 0 {
			return nil, fmt.Errorf("no slugs provided on stdin")
		}
	} else {
		if len(args) < 1 {
			return nil, fmt.Errorf("slug required")
		}
		input.slugs = []string{args[0]}
		if len(args) >= 2 {
			input.tags = parseTags(args[1])
		}
	}

	return input, nil
}

// tagMutator transforms an entry's tag list. It receives the current tags
// and returns the new tags to set on the entry.
type tagMutator func(current []string) []string

// runTagOp is the shared implementation for tag add/rm/set.
// It handles store access, slug resolution, update, and output.
// In batch mode, errors are collected and reported after processing all slugs.
func runTagOp(input *tagInput, mutate tagMutator) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	var errs []string
	count := 0
	for _, slug := range input.slugs {
		e, err := ResolveSlug(s, slug)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", slug, err))
			continue
		}

		oldTags := make([]string, len(e.Tags))
		copy(oldTags, e.Tags)

		e.Tags = mutate(e.Tags)

		if err := s.Update(e); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", slug, err))
			continue
		}

		if jsonFlag {
			data, err := json.Marshal(e.Summary())
			if err != nil {
				return err
			}
			fmt.Println(string(data))
		} else {
			fmt.Printf("Tagged: %s -> %v (was: %v)\n", e.Slug, e.Tags, oldTags)
		}
		count++
	}

	if len(input.slugs) > 1 && !jsonFlag {
		fmt.Printf("Updated %d entries.\n", count)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed on %d entries:\n  %s", len(errs), strings.Join(errs, "\n  "))
	}
	return nil
}

// mutateAddTags returns a mutator that appends tags, deduplicating against existing ones.
func mutateAddTags(newTags []string) tagMutator {
	return func(current []string) []string {
		seen := make(map[string]bool, len(current)+len(newTags))
		result := make([]string, len(current), len(current)+len(newTags))
		copy(result, current)
		for _, t := range current {
			seen[t] = true
		}
		for _, t := range newTags {
			if !seen[t] {
				result = append(result, t)
				seen[t] = true
			}
		}
		return result
	}
}

// mutateRemoveTags returns a mutator that removes the specified tags.
func mutateRemoveTags(rmTags []string) tagMutator {
	return func(current []string) []string {
		rmSet := make(map[string]bool, len(rmTags))
		for _, t := range rmTags {
			rmSet[t] = true
		}
		var kept []string
		for _, t := range current {
			if !rmSet[t] {
				kept = append(kept, t)
			}
		}
		return kept
	}
}

// mutateSetTags returns a mutator that replaces all tags with the given set.
func mutateSetTags(newTags []string) tagMutator {
	return func(_ []string) []string {
		return newTags
	}
}

func runTagAdd(cmd *cobra.Command, args []string) error {
	input, err := parseTagInput(args, tagStdin)
	if err != nil {
		return err
	}
	if len(input.tags) == 0 {
		return fmt.Errorf("at least one tag required")
	}
	return runTagOp(input, mutateAddTags(input.tags))
}

func runTagRm(cmd *cobra.Command, args []string) error {
	input, err := parseTagInput(args, tagStdin)
	if err != nil {
		return err
	}
	if len(input.tags) == 0 {
		return fmt.Errorf("at least one tag required")
	}
	return runTagOp(input, mutateRemoveTags(input.tags))
}

func runTagSet(cmd *cobra.Command, args []string) error {
	input, err := parseTagInput(args, tagStdin)
	if err != nil {
		return err
	}
	return runTagOp(input, mutateSetTags(input.tags))
}
