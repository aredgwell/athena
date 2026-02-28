package search

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Single-row DP.
	prev := make([]int, lb+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr := make([]int, lb+1)
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(
				curr[j-1]+1,    // insertion
				prev[j]+1,      // deletion
				prev[j-1]+cost, // substitution
			)
		}
		prev = curr
	}
	return prev[lb]
}

// fuzzyLookup finds terms in the inverted index within maxDist edit distance
// of the query term. Returns matching index keys. Only called when exact
// match fails, so cost is proportional to vocabulary size.
func fuzzyLookup(term string, index map[string][]Posting, maxDist int) []string {
	var matches []string
	for key := range index {
		// Skip terms with large length differences (quick filter).
		diff := len(key) - len(term)
		if diff < 0 {
			diff = -diff
		}
		if diff > maxDist {
			continue
		}
		if levenshtein(term, key) <= maxDist {
			matches = append(matches, key)
		}
	}
	return matches
}

func min(vals ...int) int {
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}
