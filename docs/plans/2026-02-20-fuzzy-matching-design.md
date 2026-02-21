# Fuzzy Matching for jot CLI

**Date**: 2026-02-20
**Status**: Approved
**Motivation**: The jot Claude Code plugin needs to detect project context from directory names and match them to journal tags. Exact matching fails when separators differ (e.g., directory `algebraic-mle` vs tag `algebraic.mle`). Adding fuzzy matching as a general CLI feature benefits all users and enables the plugin's SessionStart hook.

## Approach: Normalize + Levenshtein

Two-phase matching:
1. **Normalize** both query and candidate: lowercase, replace `[-._/ ]` with a common separator (`-`)
2. **Exact match** on normalized strings (distance 0)
3. If no exact match, **Levenshtein edit distance** with threshold `max(1, len(normalized_query)/4)`

This handles separator mismatches cleanly (normalization) and genuine typos (Levenshtein).

## New Package: `internal/fuzzy/`

### `fuzzy.go`

```go
// Levenshtein returns the edit distance between two strings.
func Levenshtein(a, b string) int

// Normalize lowercases and replaces [-._/ ] with '-'.
func Normalize(s string) string

// Threshold returns the max edit distance for a query: max(1, len(normalized)/4).
func Threshold(query string) int

// Match returns true if candidate fuzzy-matches query.
// Checks normalized exact match first, then Levenshtein within threshold.
func Match(query, candidate string, maxDist int) bool

// Result represents a fuzzy match with distance metadata.
type Result struct {
    Value    string // original (un-normalized) candidate
    Distance int    // edit distance after normalization (0 = normalized exact)
}

// RankMatches returns candidates that match query, sorted by distance asc then name asc.
func RankMatches(query string, candidates []string) []Result
```

### `fuzzy_test.go`

Test cases:
- `Levenshtein("kitten", "sitting")` = 3
- `Normalize("algebraic.mle")` = `"algebraic-mle"`
- `Match("algebraic-mle", "algebraic.mle", 1)` = true (normalized exact)
- `Match("jt", "jot", 1)` = true (distance 1)
- `Match("jot", "completely-different", 1)` = false
- `RankMatches("jot", ["jot", "jolt", "joss", "unrelated"])` = [{jot,0}, {jolt,1}, {joss,1}]

## Store Changes

### `internal/store/filter.go`

Add `Fuzzy bool` field to `Filter` struct. When `Fuzzy=true` and `Tag != ""`, the `matches()` method uses `fuzzy.Match()` instead of `entry.HasTag()`.

### `internal/store/store.go`

Add two methods:

```go
// FuzzyTags returns tags matching query, sorted by distance.
func (s *Store) FuzzyTags(query string) ([]fuzzy.Result, error)

// FuzzyTagSummaries returns enriched summaries for tags matching query.
func (s *Store) FuzzyTagSummaries(query string) ([]*TagSummary, error)
```

## Command Changes

### Global flag

Add `--fuzzy` to `rootCmd` persistent flags. Accessed via `getFuzzy()` helper.

### `tags.go`

- `jot tags` — unchanged (all tags, enriched table)
- `jot tags <query>` — currently delegates to `list --tag=<query>`. With `--fuzzy`, uses `FuzzyTagSummaries()` to show matching tags with their summaries instead.
- `jot tags --fuzzy <query>` — show tags matching query with distances

### `list.go`

- `jot list --tag=X --fuzzy` — passes `Fuzzy: true` to filter, matching entries with tags fuzzy-similar to X

### `search.go`

- `jot search --fuzzy <query>` — uses fuzzy substring matching (normalize + Levenshtein on words)

## Deprecation: `jot claude`

Replace `install` and `show` command bodies with:
```
fmt.Fprintln(os.Stderr, "jot claude is deprecated. Install the jot plugin:")
fmt.Fprintln(os.Stderr, "  https://github.com/queelius/alex-claude-plugins")
```

## Files to Create/Modify

| File | Change |
|------|--------|
| `internal/fuzzy/fuzzy.go` | **New** — Levenshtein, Normalize, Match, RankMatches |
| `internal/fuzzy/fuzzy_test.go` | **New** — comprehensive tests |
| `internal/store/filter.go` | Add `Fuzzy bool`, update `matches()` |
| `internal/store/store.go` | Add `FuzzyTags()`, `FuzzyTagSummaries()` |
| `internal/store/store_test.go` | Tests for fuzzy store methods |
| `internal/commands/root.go` | Add `--fuzzy` persistent flag + `getFuzzy()` |
| `internal/commands/tags.go` | Handle `--fuzzy` on tag query |
| `internal/commands/list.go` | Pass `Fuzzy` to filter |
| `internal/commands/search.go` | Fuzzy search mode |
| `internal/commands/claude.go` | Deprecation messages |

## Implementation Order

1. `internal/fuzzy/` package + tests
2. Store integration (filter + new methods) + tests
3. Command changes (--fuzzy flag, tags, list, search)
4. Deprecate `jot claude`
5. Build + full test suite
