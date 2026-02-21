package fuzzy

import (
	"testing"
)

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
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
	for _, tc := range tests {
		got := Levenshtein(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("Levenshtein(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"algebraic.mle", "algebraic-mle"},
		{"algebraic_mle", "algebraic-mle"},
		{"Algebraic.MLE", "algebraic-mle"},
		{"some thing", "some-thing"},
		{"a/b/c", "a-b-c"},
		{"simple", "simple"},
		{"", ""},
	}
	for _, tc := range tests {
		got := Normalize(tc.input)
		if got != tc.want {
			t.Errorf("Normalize(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestThreshold(t *testing.T) {
	tests := []struct {
		query string
		want  int
	}{
		{"jot", 1},          // len 3, 3/4=0, max(1,0)=1
		{"repoindex", 2},    // len 9, 9/4=2
		{"a", 1},            // len 1, 1/4=0, max(1,0)=1
		{"algebraic-mle", 3}, // len 13, 13/4=3
	}
	for _, tc := range tests {
		got := Threshold(tc.query)
		if got != tc.want {
			t.Errorf("Threshold(%q) = %d, want %d", tc.query, got, tc.want)
		}
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		query, candidate string
		maxDist          int
		want             bool
	}{
		{"algebraic-mle", "algebraic.mle", 1, true},  // normalized exact
		{"jot", "jot", 1, true},
		{"jt", "jot", 1, true},
		{"jot", "completely-different", 1, false},
		{"JOT", "jot", 1, true},
		{"", "anything", 1, true},
		{"jot", "", 1, false},
	}
	for _, tc := range tests {
		got := Match(tc.query, tc.candidate, tc.maxDist)
		if got != tc.want {
			t.Errorf("Match(%q, %q, %d) = %v, want %v", tc.query, tc.candidate, tc.maxDist, got, tc.want)
		}
	}
}

func TestRankMatches(t *testing.T) {
	candidates := []string{"jot", "jolt", "joss", "unrelated", "jot-plugin"}
	results := RankMatches("jot", candidates)

	if len(results) == 0 {
		t.Fatal("RankMatches returned no results")
	}

	// First result should be "jot" with distance 0.
	if results[0].Value != "jot" || results[0].Distance != 0 {
		t.Errorf("first result = {%q, %d}, want {%q, %d}", results[0].Value, results[0].Distance, "jot", 0)
	}

	// "unrelated" should not appear.
	for _, r := range results {
		if r.Value == "unrelated" {
			t.Error("unexpected result: \"unrelated\" should not match \"jot\"")
		}
	}
}

func TestRankMatchesEmpty(t *testing.T) {
	results := RankMatches("xyz", []string{"abc", "def"})
	if len(results) != 0 {
		t.Errorf("RankMatches(\"xyz\", ...) returned %d results, want 0", len(results))
	}
}

func TestRankMatchesSortOrder(t *testing.T) {
	// Build candidates that produce varying distances to verify sort order.
	// Query: "abc" -> threshold = max(1, 3/4) = 1
	// "abc" -> distance 0
	// "abd" -> distance 1
	// "abb" -> distance 1
	// "abx" -> distance 1
	// "xyz" -> distance 3, exceeds threshold
	candidates := []string{"abd", "abc", "abx", "abb", "xyz"}
	results := RankMatches("abc", candidates)

	// Should contain "abc" (dist 0), then "abb","abd","abx" (dist 1 each, sorted by value).
	// "xyz" should be excluded (distance 3 > threshold 1).
	expected := []Result{
		{Value: "abc", Distance: 0},
		{Value: "abb", Distance: 1},
		{Value: "abd", Distance: 1},
		{Value: "abx", Distance: 1},
	}

	if len(results) != len(expected) {
		t.Fatalf("got %d results, want %d: %+v", len(results), len(expected), results)
	}

	for i, e := range expected {
		if results[i].Value != e.Value || results[i].Distance != e.Distance {
			t.Errorf("results[%d] = {%q, %d}, want {%q, %d}", i, results[i].Value, results[i].Distance, e.Value, e.Distance)
		}
	}
}
