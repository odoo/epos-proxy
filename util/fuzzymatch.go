package util

import (
	"regexp"
	"strings"
)

// ---------------- NORMALIZATION ----------------

func normalize(s string) string {
	s = strings.ToLower(s)

	reg := regexp.MustCompile(`[^a-z0-9\s]+`)
	s = reg.ReplaceAllString(s, " ")

	noise := []string{"printer", "receipt", "usb", "series"}
	for _, n := range noise {
		s = strings.ReplaceAll(s, n, "")
	}

	// collapse + deduplicate tokens
	tokens := strings.Fields(s)
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, t := range tokens {
		if !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}

	return strings.Join(result, " ")
}

// ---------------- TOKEN SIMILARITY ----------------

func tokenSet(s string) map[string]bool {
	tokens := strings.Fields(s)
	set := make(map[string]bool)
	for _, t := range tokens {
		set[t] = true
	}
	return set
}

func tokenSimilarity(a, b string) float64 {
	setA := tokenSet(a)
	setB := tokenSet(b)

	intersection := 0
	for k := range setA {
		if setB[k] {
			intersection++
		}
	}

	maxLen := len(setA)
	if len(setB) > maxLen {
		maxLen = len(setB)
	}

	if maxLen == 0 {
		return 0
	}

	return float64(intersection) / float64(maxLen)
}

// ---------------- LCS (Longest Common Subsequence) ----------------

func lcsLength(a, b string) int {
	m := len(a)
	n := len(b)

	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				if dp[i-1][j] > dp[i][j-1] {
					dp[i][j] = dp[i-1][j]
				} else {
					dp[i][j] = dp[i][j-1]
				}
			}
		}
	}

	return dp[m][n]
}

func sequenceSimilarity(a, b string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	lcs := lcsLength(a, b)
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}

	return float64(lcs) / float64(maxLen)
}

// ---------------- PREFIX BONUS ----------------

func prefixBonus(a, b string) float64 {
	if strings.HasPrefix(a, b) || strings.HasPrefix(b, a) {
		return 1.0
	}
	return 0.0
}

// ---------------- FINAL MATCH FUNCTION ----------------

func matchScore(a, b string) float64 {
	na := normalize(a)
	nb := normalize(b)

	token := tokenSimilarity(na, nb)
	seq := sequenceSimilarity(na, nb)
	prefix := prefixBonus(na, nb)

	score := 0.5*token + 0.3*seq + 0.2*prefix
	return score
}

// ---------------- HELPER DECISION ----------------

func IsMatch(a, b string) bool {
	score := matchScore(a, b)

	if score > 0.75 {
		return true
	}
	return false
}
