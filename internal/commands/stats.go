package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/queelius/jot/internal/store"
)

var (
	statsSection   string
	statsAll       bool
	statsStaleDays int
	statsType      string
	statsTag       string
	statsStatus    string
	statsPriority  string
	statsSince     string
	statsUntil     string
	statsLimit     int
)

var statsCmd = &cobra.Command{
	Use:     "stats",
	Short:   "Journal snapshot data (JSON)",
	GroupID: "query",
	Long: `Compute journal snapshot data as JSON for LLM consumption.

Outputs pre-computed sections: summary, overdue, blocked, health, recent.
Default sections: summary, overdue, blocked. Use --all for everything.

Always outputs JSON regardless of --table/--markdown flags.

Examples:
  jot stats                              # default sections
  jot stats --all                        # all sections
  jot stats --section=health             # specific section
  jot stats --section=summary,health     # multiple sections
  jot stats --tags=myproject             # scoped to a tag
  jot stats --type=task --status=open    # scoped by type/status`,
	RunE: runStats,
}

func init() {
	statsCmd.Flags().StringVar(&statsSection, "section", "", "sections to include (comma-separated: summary,overdue,blocked,health,recent)")
	statsCmd.Flags().BoolVar(&statsAll, "all", false, "include all sections")
	statsCmd.Flags().IntVar(&statsStaleDays, "stale-days", 30, "days without modification to consider stale (for health section)")
	statsCmd.Flags().StringVarP(&statsType, "type", "t", "", "filter by type")
	statsCmd.Flags().StringVar(&statsTag, "tags", "", "filter by tag")
	statsCmd.Flags().StringVarP(&statsStatus, "status", "s", "", "filter by status")
	statsCmd.Flags().StringVarP(&statsPriority, "priority", "p", "", "filter by priority")
	statsCmd.Flags().StringVar(&statsSince, "since", "", "entries created since (e.g., 7d, 2w, 2024-01-01)")
	statsCmd.Flags().StringVar(&statsUntil, "until", "", "entries created until")
	statsCmd.Flags().IntVarP(&statsLimit, "limit", "n", 0, "limit initial entry set")

	rootCmd.AddCommand(statsCmd)
}

// parseSections parses the --section flag and --all into a map of section names.
func parseSections(sectionFlag string, all bool) (map[string]bool, error) {
	if all {
		sections := make(map[string]bool)
		for _, s := range store.ValidSections {
			sections[s] = true
		}
		return sections, nil
	}

	if sectionFlag == "" {
		return map[string]bool{"summary": true, "overdue": true, "blocked": true}, nil
	}

	valid := make(map[string]bool)
	for _, s := range store.ValidSections {
		valid[s] = true
	}

	sections := make(map[string]bool)
	for _, s := range strings.Split(sectionFlag, ",") {
		name := strings.TrimSpace(strings.ToLower(s))
		if name == "" {
			continue
		}
		if !valid[name] {
			return nil, fmt.Errorf("unknown section %q, valid sections: %s", name, strings.Join(store.ValidSections, ", "))
		}
		sections[name] = true
	}

	if len(sections) == 0 {
		return nil, fmt.Errorf("no valid sections specified")
	}

	return sections, nil
}

func runStats(cmd *cobra.Command, args []string) error {
	sections, err := parseSections(statsSection, statsAll)
	if err != nil {
		return err
	}

	s, err := getStore()
	if err != nil {
		return err
	}

	filter := &store.Filter{
		Type:     statsType,
		Tag:      statsTag,
		Status:   statsStatus,
		Priority: statsPriority,
		Limit:    statsLimit,
		Fuzzy:    getFuzzy(),
	}

	if statsSince != "" {
		if dur, err := store.ParseDuration(statsSince); err == nil && dur > 0 {
			filter.Since = time.Now().Add(-dur)
		} else if t, err := store.ParseDate(statsSince); err == nil {
			filter.Since = t
		}
	}
	if statsUntil != "" {
		if t, err := store.ParseDate(statsUntil); err == nil {
			filter.Until = t
		}
	}

	result, err := s.Stats(filter, sections, statsStaleDays)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
