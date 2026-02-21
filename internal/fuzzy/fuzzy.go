// Package fuzzy provides normalize+Levenshtein fuzzy string matching.
package fuzzy

import (
	"sort"
	"strings"
)

// Levenshtein computes the edit distance between two strings using a
// two-row optimization that requires O(min(m,n)) space.
func Levenshtein(a, b string) int {
	// Swap so that a is the shorter string.
	if len(a) > len(b) {
		a, b = b, a
	}

	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}

	// Two-row approach: prev and curr, each of length la+1.
	prev := make([]int, la+1)
	curr := make([]int, la+1)

	for i := 0; i <= la; i++ {
		prev[i] = i
	}

	for j := 1; j <= lb; j++ {
		curr[0] = j
		for i := 1; i <= la; i++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			ins := curr[i-1] + 1
			del := prev[i] + 1
			sub := prev[i-1] + cost

			m := ins
			if del < m {
				m = del
			}
			if sub < m {
				m = sub
			}
			curr[i] = m
		}
		prev, curr = curr, prev
	}

	return prev[la]
}

// Normalize lowercases the string and replaces [-._/ ] with '-'.
func Normalize(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '-', '.', '_', '/', ' ':
			b.WriteByte('-')
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// Threshold returns max(1, len(Normalize(query))/4).
func Threshold(query string) int {
	n := len(Normalize(query)) / 4
	if n < 1 {
		return 1
	}
	return n
}

// Match returns true if the normalized query fuzzy-matches the normalized
// candidate within maxDist edits.
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

// Result holds a fuzzy match result.
type Result struct {
	Value    string `json:"value"`
	Distance int    `json:"distance"`
}

// RankMatches returns candidates that fuzzy-match the query, sorted by
// distance ascending then value ascending.
func RankMatches(query string, candidates []string) []Result {
	maxDist := Threshold(query)
	nq := Normalize(query)

	var results []Result
	for _, c := range candidates {
		nc := Normalize(c)
		var dist int
		if nq == "" {
			dist = 0
		} else if nc == "" {
			continue
		} else if nq == nc {
			dist = 0
		} else {
			dist = Levenshtein(nq, nc)
			if dist > maxDist {
				continue
			}
		}
		results = append(results, Result{Value: c, Distance: dist})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Distance != results[j].Distance {
			return results[i].Distance < results[j].Distance
		}
		return results[i].Value < results[j].Value
	})

	return results
}
