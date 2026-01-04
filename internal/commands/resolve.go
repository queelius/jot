package commands

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/queelius/jot/internal/entry"
	"github.com/queelius/jot/internal/store"
)

// ResolveSlug finds an entry by exact or partial slug match.
// If no exact match is found, it searches for partial matches (case-insensitive).
// For a single partial match, it returns that entry silently.
// For multiple matches, it prompts the user to select one.
func ResolveSlug(s *store.Store, slug string) (*entry.Entry, error) {
	// Try exact match first
	e, err := s.Get(slug)
	if err == nil {
		return e, nil
	}

	// Fall back to partial match
	matches, err := s.FindByPartialSlug(slug)
	if err != nil {
		return nil, err
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("entry not found: %s", slug)
	case 1:
		return matches[0], nil
	default:
		return promptForSelection(matches, slug)
	}
}

// promptForSelection displays a numbered list of matches and prompts user to select one.
func promptForSelection(matches []*entry.Entry, query string) (*entry.Entry, error) {
	fmt.Printf("Multiple matches for '%s':\n", query)
	for i, e := range matches {
		fmt.Printf("  %d. %s (%s)\n", i+1, e.Slug, e.Title)
	}
	fmt.Printf("Select [1-%d]: ", len(matches))

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}

	response = strings.TrimSpace(response)
	if response == "" {
		return nil, fmt.Errorf("cancelled")
	}

	choice, err := strconv.Atoi(response)
	if err != nil || choice < 1 || choice > len(matches) {
		return nil, fmt.Errorf("invalid selection: %s", response)
	}

	return matches[choice-1], nil
}
