# Fuzzy Matching Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add normalize+Levenshtein fuzzy matching to jot CLI as a `--fuzzy` flag on tags, list, and search commands. Deprecate `jot claude` in favor of the plugin.

**Architecture:** New `internal/fuzzy/` package with pure functions (Levenshtein, Normalize, Match, RankMatches). Store layer gets `Fuzzy bool` on Filter and two new methods. Commands wire up `--fuzzy` flag.

**Tech Stack:** Go standard library only (no external dependencies for fuzzy matching).

---

### Task 1: Create `internal/fuzzy/` package — core functions

**Files:**
- Create: `internal/fuzzy/fuzzy.go`
- Create: `internal/fuzzy/fuzzy_test.go`

**Step 1: Write failing tests for Levenshtein**

```go
// internal/fuzzy/fuzzy_test.go
package fuzzy

import "testing"

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "b", 1},
		{"a", "a", 0},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"abc", "abcd", 1},
		{"abcd", "abc", 1},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			got := Levenshtein(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("Levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/fuzzy/`
Expected: FAIL — `Levenshtein` not defined

**Step 3: Implement Levenshtein**

```go
// internal/fuzzy/fuzzy.go
package fuzzy

// Levenshtein returns the edit distance between two strings.
func Levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Use two rows instead of full matrix for O(min(m,n)) space.
	if len(a) > len(b) {
		a, b = b, a
	}

	prev := make([]int, len(a)+1)
	curr := make([]int, len(a)+1)

	for i := range prev {
		prev[i] = i
	}

	for j := 1; j <= len(b); j++ {
		curr[0] = j
		for i := 1; i <= len(a); i++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[i] = min(
				curr[i-1]+1,      // insert
				prev[i]+1,        // delete
				prev[i-1]+cost,   // substitute
			)
		}
		prev, curr = curr, prev
	}

	return prev[len(a)]
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/fuzzy/`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/fuzzy/fuzzy.go internal/fuzzy/fuzzy_test.go
git commit -m "feat: add Levenshtein edit distance function"
```

---

### Task 2: Add Normalize, Threshold, Match functions

**Files:**
- Modify: `internal/fuzzy/fuzzy.go`
- Modify: `internal/fuzzy/fuzzy_test.go`

**Step 1: Write failing tests**

```go
func TestNormalize(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"algebraic.mle", "algebraic-mle"},
		{"algebraic_mle", "algebraic-mle"},
		{"algebraic-mle", "algebraic-mle"},
		{"Algebraic.MLE", "algebraic-mle"},
		{"some thing", "some-thing"},
		{"a/b/c", "a-b-c"},
		{"simple", "simple"},
		{"", ""},
		{"---", "---"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Normalize(tt.input)
			if got != tt.expected {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestThreshold(t *testing.T) {
	tests := []struct {
		query    string
		expected int
	}{
		{"jot", 1},       // len 3, 3/4=0, max(1,0)=1
		{"repoindex", 2}, // len 9, 9/4=2
		{"a", 1},         // min is always 1
		{"algebraic-mle", 3}, // len 13, 13/4=3
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := Threshold(tt.query)
			if got != tt.expected {
				t.Errorf("Threshold(%q) = %d, want %d", tt.query, got, tt.expected)
			}
		})
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		candidate string
		maxDist   int
		expected  bool
	}{
		{"normalized exact", "algebraic-mle", "algebraic.mle", 1, true},
		{"exact same", "jot", "jot", 1, true},
		{"one edit", "jt", "jot", 1, true},
		{"too far", "jot", "completely-different", 1, false},
		{"case insensitive", "JOT", "jot", 1, true},
		{"empty query", "", "anything", 1, true},
		{"empty candidate", "jot", "", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Match(tt.query, tt.candidate, tt.maxDist)
			if got != tt.expected {
				t.Errorf("Match(%q, %q, %d) = %v, want %v",
					tt.query, tt.candidate, tt.maxDist, got, tt.expected)
			}
		})
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/fuzzy/`
Expected: FAIL — functions not defined

**Step 3: Implement Normalize, Threshold, Match**

Add to `internal/fuzzy/fuzzy.go`:

```go
import "strings"

// Normalize lowercases a string and replaces separators [-._/ ] with '-'.
func Normalize(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '.', '_', '/', ' ':
			b.WriteByte('-')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Threshold returns the max edit distance for a query: max(1, len(normalized)/4).
func Threshold(query string) int {
	n := len(Normalize(query)) / 4
	if n < 1 {
		return 1
	}
	return n
}

// Match returns true if candidate fuzzy-matches query within maxDist.
// Normalizes both strings first; if normalized forms are equal, returns true
// immediately. Otherwise checks Levenshtein distance.
func Match(query, candidate string, maxDist int) bool {
	nq := Normalize(query)
	nc := Normalize(candidate)

	if nq == "" {
		return true
	}
	if nc == "" {
		return false
	}

	if nq == nc {
		return true
	}

	return Levenshtein(nq, nc) <= maxDist
}
```

**Step 4: Run tests**

Run: `go test ./internal/fuzzy/`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/fuzzy/fuzzy.go internal/fuzzy/fuzzy_test.go
git commit -m "feat: add Normalize, Threshold, and Match for fuzzy matching"
```

---

### Task 3: Add RankMatches function

**Files:**
- Modify: `internal/fuzzy/fuzzy.go`
- Modify: `internal/fuzzy/fuzzy_test.go`

**Step 1: Write failing test**

```go
func TestRankMatches(t *testing.T) {
	candidates := []string{"jot", "jolt", "joss", "unrelated", "jot-plugin"}

	results := RankMatches("jot", candidates)

	// Should include jot (exact, dist 0), jolt (dist 1), joss (dist 1), jot-plugin (contains)
	// Should NOT include unrelated
	if len(results) < 2 {
		t.Fatalf("RankMatches returned %d results, want at least 2", len(results))
	}

	// First result should be exact match
	if results[0].Value != "jot" || results[0].Distance != 0 {
		t.Errorf("first result = %+v, want {jot, 0}", results[0])
	}

	// All results should be within threshold
	threshold := Threshold("jot")
	for _, r := range results {
		if r.Distance > threshold {
			t.Errorf("result %+v exceeds threshold %d", r, threshold)
		}
	}
}

func TestRankMatchesEmpty(t *testing.T) {
	results := RankMatches("xyz", []string{"abc", "def"})
	if len(results) != 0 {
		t.Errorf("RankMatches returned %d results for no-match query, want 0", len(results))
	}
}

func TestRankMatchesSortOrder(t *testing.T) {
	// Verify sorted by distance asc, then name asc
	candidates := []string{"zoo", "zap", "zip"}
	results := RankMatches("zap", candidates)

	for i := 1; i < len(results); i++ {
		if results[i].Distance < results[i-1].Distance {
			t.Errorf("results not sorted by distance: %+v before %+v", results[i-1], results[i])
		}
		if results[i].Distance == results[i-1].Distance && results[i].Value < results[i-1].Value {
			t.Errorf("results with same distance not sorted by name: %+v before %+v", results[i-1], results[i])
		}
	}
}
```

**Step 2: Run tests — should fail**

Run: `go test ./internal/fuzzy/`

**Step 3: Implement RankMatches**

```go
import "sort"

// Result represents a fuzzy match with distance metadata.
type Result struct {
	Value    string `json:"value"`
	Distance int    `json:"distance"`
}

// RankMatches returns candidates that fuzzy-match query, sorted by distance asc then name asc.
func RankMatches(query string, candidates []string) []Result {
	maxDist := Threshold(query)
	nq := Normalize(query)

	var results []Result
	for _, c := range candidates {
		nc := Normalize(c)
		var dist int
		if nq == nc {
			dist = 0
		} else {
			dist = Levenshtein(nq, nc)
		}

		if dist <= maxDist {
			results = append(results, Result{Value: c, Distance: dist})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Distance != results[j].Distance {
			return results[i].Distance < results[j].Distance
		}
		return results[i].Value < results[j].Value
	})

	return results
}
```

**Step 4: Run tests**

Run: `go test ./internal/fuzzy/`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/fuzzy/fuzzy.go internal/fuzzy/fuzzy_test.go
git commit -m "feat: add RankMatches for ranked fuzzy matching"
```

---

### Task 4: Add Fuzzy to store Filter

**Files:**
- Modify: `internal/store/filter.go` (line 12: add Fuzzy bool to struct, line 48: update matches)
- Modify: `internal/store/store_test.go`

**Step 1: Write failing test**

Add to `internal/store/store_test.go`:

```go
func TestStore_ListFuzzyTag(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	now := time.Now()
	entries := []*entry.Entry{
		{Title: "Entry A", Type: "note", Tags: []string{"algebraic.mle"}, Created: now, Modified: now},
		{Title: "Entry B", Type: "task", Tags: []string{"repoindex"}, Created: now, Modified: now},
		{Title: "Entry C", Type: "idea", Tags: []string{"jot"}, Created: now, Modified: now},
	}
	for _, e := range entries {
		e.Slug = entry.GenerateSlug(e.Title, e.Created)
		if err := s.Create(e); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Exact tag match works
	got, err := s.List(&Filter{Tag: "algebraic.mle"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 1 {
		t.Errorf("exact tag match returned %d, want 1", len(got))
	}

	// Fuzzy: separator difference
	got, err = s.List(&Filter{Tag: "algebraic-mle", Fuzzy: true})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 1 || got[0].Title != "Entry A" {
		t.Errorf("fuzzy tag match returned %d entries, want 1 (Entry A)", len(got))
	}

	// Fuzzy: no match
	got, err = s.List(&Filter{Tag: "nonexistent", Fuzzy: true})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("fuzzy no-match returned %d, want 0", len(got))
	}
}
```

**Step 2: Run test — should fail** (Fuzzy field doesn't exist)

Run: `go test ./internal/store/ -run TestStore_ListFuzzyTag`

**Step 3: Add Fuzzy to Filter and update matches()**

In `internal/store/filter.go`, add `Fuzzy bool` field to Filter struct and update `matches()` to use `fuzzy.Match()` when `f.Fuzzy && f.Tag != ""`.

In the `matches()` function, replace the tag check:

```go
// Before:
if f.Tag != "" && !e.HasTag(f.Tag) {
    return false
}

// After:
if f.Tag != "" {
    if f.Fuzzy {
        if !hasTagFuzzy(e, f.Tag) {
            return false
        }
    } else if !e.HasTag(f.Tag) {
        return false
    }
}
```

Add helper:

```go
func hasTagFuzzy(e *entry.Entry, query string) bool {
    maxDist := fuzzy.Threshold(query)
    for _, tag := range e.Tags {
        if fuzzy.Match(query, tag, maxDist) {
            return true
        }
    }
    return false
}
```

**Step 4: Run tests**

Run: `go test ./internal/store/`
Expected: PASS (all existing tests + new fuzzy test)

**Step 5: Commit**

```bash
git add internal/store/filter.go internal/store/store_test.go
git commit -m "feat: add fuzzy tag matching to store filter"
```

---

### Task 5: Add FuzzyTags and FuzzyTagSummaries to store

**Files:**
- Modify: `internal/store/store.go`
- Modify: `internal/store/store_test.go`

**Step 1: Write failing tests**

```go
func TestStore_FuzzyTags(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	now := time.Now()
	entries := []*entry.Entry{
		{Title: "A", Type: "note", Tags: []string{"algebraic.mle", "math"}, Created: now, Modified: now},
		{Title: "B", Type: "task", Tags: []string{"repoindex"}, Created: now, Modified: now},
		{Title: "C", Type: "idea", Tags: []string{"jot"}, Created: now, Modified: now},
	}
	for _, e := range entries {
		e.Slug = entry.GenerateSlug(e.Title, e.Created)
		if err := s.Create(e); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	results, err := s.FuzzyTags("algebraic-mle")
	if err != nil {
		t.Fatalf("FuzzyTags() error = %v", err)
	}
	if len(results) == 0 {
		t.Fatal("FuzzyTags() returned no results, want at least 1")
	}
	if results[0].Value != "algebraic.mle" {
		t.Errorf("FuzzyTags() first result = %q, want 'algebraic.mle'", results[0].Value)
	}
}

func TestStore_FuzzyTagSummaries(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	now := time.Now()
	entries := []*entry.Entry{
		{Title: "A", Type: "task", Status: "open", Tags: []string{"algebraic.mle"}, Created: now, Modified: now},
		{Title: "B", Type: "idea", Tags: []string{"algebraic.mle"}, Created: now, Modified: now},
		{Title: "C", Type: "task", Tags: []string{"jot"}, Created: now, Modified: now},
	}
	for _, e := range entries {
		e.Slug = entry.GenerateSlug(e.Title, e.Created)
		if err := s.Create(e); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	summaries, err := s.FuzzyTagSummaries("algebraic-mle")
	if err != nil {
		t.Fatalf("FuzzyTagSummaries() error = %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("FuzzyTagSummaries() returned %d, want 1", len(summaries))
	}
	if summaries[0].Tag != "algebraic.mle" {
		t.Errorf("tag = %q, want 'algebraic.mle'", summaries[0].Tag)
	}
	if summaries[0].Count != 2 {
		t.Errorf("count = %d, want 2", summaries[0].Count)
	}
}
```

**Step 2: Run tests — should fail**

Run: `go test ./internal/store/ -run TestStore_Fuzzy`

**Step 3: Implement FuzzyTags and FuzzyTagSummaries**

Add to `internal/store/store.go`:

```go
import "github.com/queelis/jot/internal/fuzzy"

// FuzzyTags returns tags that fuzzy-match query, sorted by distance.
func (s *Store) FuzzyTags(query string) ([]fuzzy.Result, error) {
	tags, err := s.AllTags()
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(tags))
	for name := range tags {
		names = append(names, name)
	}

	return fuzzy.RankMatches(query, names), nil
}

// FuzzyTagSummaries returns enriched summaries for tags that fuzzy-match query.
func (s *Store) FuzzyTagSummaries(query string) ([]*TagSummary, error) {
	allSummaries, err := s.TagSummaries()
	if err != nil {
		return nil, err
	}

	maxDist := fuzzy.Threshold(query)
	var results []*TagSummary
	for _, ts := range allSummaries {
		if fuzzy.Match(query, ts.Tag, maxDist) {
			results = append(results, ts)
		}
	}

	return results, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/store/`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/store/store.go internal/store/store_test.go
git commit -m "feat: add FuzzyTags and FuzzyTagSummaries to store"
```

---

### Task 6: Add --fuzzy global flag

**Files:**
- Modify: `internal/commands/root.go` (line 14: add var, line 44-47: add flag)

**Step 1: Add flag and helper**

In `root.go`, add to the var block:

```go
fuzzyFlag bool
```

In `init()`, add:

```go
rootCmd.PersistentFlags().BoolVar(&fuzzyFlag, "fuzzy", false, "use fuzzy matching for tags and search")
```

Add helper:

```go
// getFuzzy returns whether fuzzy matching is enabled.
func getFuzzy() bool {
	return fuzzyFlag
}
```

**Step 2: Build to verify no compile errors**

Run: `go build ./...`
Expected: Success

**Step 3: Commit**

```bash
git add internal/commands/root.go
git commit -m "feat: add --fuzzy global flag to CLI"
```

---

### Task 7: Wire --fuzzy into tags command

**Files:**
- Modify: `internal/commands/tags.go`

**Step 1: Update runTags to handle fuzzy**

When `jot tags <query>` is called with `--fuzzy`, instead of delegating to `list --tag`, use `FuzzyTagSummaries()` to show matching tags with enriched table.

When `jot tags <query>` is called without `--fuzzy`, keep existing behavior (delegate to list).

**Step 2: Build and manually test**

Run: `go build -o jot ./cmd/jot && ./jot tags --fuzzy algebraic-mle`
Expected: Shows `algebraic.mle` tag summary

**Step 3: Commit**

```bash
git add internal/commands/tags.go
git commit -m "feat: wire --fuzzy into tags command"
```

---

### Task 8: Wire --fuzzy into list command

**Files:**
- Modify: `internal/commands/list.go` (line 74-80: add Fuzzy to filter)

**Step 1: Pass fuzzy flag to filter**

In `runList`, add `Fuzzy: getFuzzy()` to the filter construction.

**Step 2: Build and test**

Run: `go build -o jot ./cmd/jot && ./jot list --tag=algebraic-mle --fuzzy`
Expected: Shows entries tagged `algebraic.mle`

**Step 3: Commit**

```bash
git add internal/commands/list.go
git commit -m "feat: wire --fuzzy into list command"
```

---

### Task 9: Wire --fuzzy into search command

**Files:**
- Modify: `internal/commands/search.go`
- Modify: `internal/store/filter.go`

**Step 1: Pass fuzzy to search filter**

In `runSearch`, add `Fuzzy: getFuzzy()` to the filter construction. The tag filter in search will now support fuzzy matching.

**Step 2: Build and test**

Run: `go build -o jot ./cmd/jot && ./jot search "api" --tags=algebraic-mle --fuzzy`
Expected: Searches entries whose tags fuzzy-match `algebraic-mle`

**Step 3: Commit**

```bash
git add internal/commands/search.go
git commit -m "feat: wire --fuzzy into search command"
```

---

### Task 10: Deprecate `jot claude`

**Files:**
- Modify: `internal/commands/claude.go`

**Step 1: Replace install and show with deprecation messages**

Replace `runClaudeInstall` body with:

```go
func runClaudeInstall(cmd *cobra.Command, args []string) error {
	fmt.Fprintln(os.Stderr, "Deprecated: 'jot claude install' has been replaced by the jot plugin.")
	fmt.Fprintln(os.Stderr, "Install the plugin from: https://github.com/queelius/alex-claude-plugins")
	return nil
}
```

Replace `runClaudeShow` body with:

```go
func runClaudeShow(cmd *cobra.Command, args []string) error {
	fmt.Fprintln(os.Stderr, "Deprecated: 'jot claude show' has been replaced by the jot plugin.")
	fmt.Fprintln(os.Stderr, "Install the plugin from: https://github.com/queelius/alex-claude-plugins")
	return nil
}
```

Mark commands as deprecated using cobra:

```go
claudeInstallCmd.Deprecated = "use the jot plugin from https://github.com/queelius/alex-claude-plugins"
claudeShowCmd.Deprecated = "use the jot plugin from https://github.com/queelius/alex-claude-plugins"
```

Remove the `skillContent` const and `claudeInstallLocal` var (now unused). Clean up unused imports.

**Step 2: Build and test**

Run: `go build -o jot ./cmd/jot && ./jot claude install`
Expected: Deprecation message printed to stderr

**Step 3: Commit**

```bash
git add internal/commands/claude.go
git commit -m "deprecate: replace jot claude with plugin pointer"
```

---

### Task 11: Full build + test suite

**Files:** None (verification only)

**Step 1: Run full test suite**

Run: `go test ./...`
Expected: All tests pass

**Step 2: Run test coverage**

Run: `go test -cover ./internal/fuzzy ./internal/store ./internal/commands`
Expected: Good coverage on fuzzy and store packages

**Step 3: Manual smoke test**

```bash
go build -o jot ./cmd/jot
./jot tags                               # Enriched table (unchanged)
./jot tags --fuzzy algebraic-mle         # Fuzzy tag match
./jot list --tag=algebraic-mle --fuzzy   # Fuzzy list filter
./jot claude install                     # Deprecation message
```

**Step 4: Final commit if any fixups needed**

```bash
go test ./...
```
