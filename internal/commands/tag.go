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

// getSlugInputs returns slugs from positional args or stdin.
func getSlugInputs(args []string, useStdin bool) ([]string, error) {
	if useStdin {
		scanner := bufio.NewScanner(os.Stdin)
		var slugs []string
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				slugs = append(slugs, line)
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("reading stdin: %w", err)
		}
		if len(slugs) == 0 {
			return nil, fmt.Errorf("no slugs provided on stdin")
		}
		return slugs, nil
	}
	if len(args) < 1 {
		return nil, fmt.Errorf("slug required")
	}
	return []string{args[0]}, nil
}

// getTagsArg extracts the tags argument, accounting for --stdin shifting args.
func getTagsArg(args []string, useStdin bool) string {
	if useStdin {
		// With --stdin, the first positional arg is the tags
		if len(args) >= 1 {
			return args[0]
		}
		return ""
	}
	// Without --stdin, first arg is slug, second is tags
	if len(args) >= 2 {
		return args[1]
	}
	return ""
}

// tagMutator transforms an entry's tag list. It receives the current tags
// and returns the new tags to set on the entry.
type tagMutator func(current []string) []string

// runTagOp is the shared implementation for tag add/rm/set.
// It handles store access, slug resolution, update, and output.
func runTagOp(args []string, mutate tagMutator) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	slugs, err := getSlugInputs(args, tagStdin)
	if err != nil {
		return err
	}

	count := 0
	for _, slug := range slugs {
		e, err := ResolveSlug(s, slug)
		if err != nil {
			return fmt.Errorf("%s: %w", slug, err)
		}

		oldTags := make([]string, len(e.Tags))
		copy(oldTags, e.Tags)

		e.Tags = mutate(e.Tags)

		if err := s.Update(e); err != nil {
			return fmt.Errorf("%s: %w", slug, err)
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

	if len(slugs) > 1 && !jsonFlag {
		fmt.Printf("Updated %d entries.\n", count)
	}
	return nil
}

// mutateAddTags returns a mutator that appends tags, deduplicating against existing ones.
func mutateAddTags(newTags []string) tagMutator {
	return func(current []string) []string {
		seen := make(map[string]bool, len(current))
		for _, t := range current {
			seen[t] = true
		}
		result := current
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
	tags := parseTags(getTagsArg(args, tagStdin))
	if len(tags) == 0 {
		return fmt.Errorf("at least one tag required")
	}
	return runTagOp(args, mutateAddTags(tags))
}

func runTagRm(cmd *cobra.Command, args []string) error {
	tags := parseTags(getTagsArg(args, tagStdin))
	if len(tags) == 0 {
		return fmt.Errorf("at least one tag required")
	}
	return runTagOp(args, mutateRemoveTags(tags))
}

func runTagSet(cmd *cobra.Command, args []string) error {
	tags := parseTags(getTagsArg(args, tagStdin))
	return runTagOp(args, mutateSetTags(tags))
}
