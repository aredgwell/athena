package search

import (
	"strings"
	"unicode"
)

// defaultSnippetLen is the target character length for extracted snippets.
const defaultSnippetLen = 150

// ExtractSnippet returns a text window around the best-matching region of
// body for the given query. The window is approximately windowSize characters.
// Returns empty string if no query terms are found in the body.
func ExtractSnippet(body, query string, windowSize int) string {
	if windowSize <= 0 {
		windowSize = defaultSnippetLen
	}

	queryTerms := Tokenize(query)
	if len(queryTerms) == 0 || body == "" {
		return ""
	}

	// Build a set of query stems for fast lookup.
	termSet := make(map[string]struct{}, len(queryTerms))
	for _, t := range queryTerms {
		termSet[t] = struct{}{}
	}

	// Find the position of the best matching sentence/region.
	// Split body into sentences (rough: split on .\n or double newline).
	sentences := splitSentences(body)

	bestIdx := 0
	bestScore := 0
	for i, sent := range sentences {
		score := scoreSentence(sent, termSet)
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}

	if bestScore == 0 {
		// No match found — return truncated beginning.
		if len(body) > windowSize {
			return strings.TrimSpace(body[:windowSize]) + "..."
		}
		return strings.TrimSpace(body)
	}

	snippet := sentences[bestIdx]
	if len(snippet) > windowSize {
		snippet = snippet[:windowSize] + "..."
	}
	return strings.TrimSpace(snippet)
}

// splitSentences splits text into rough sentence-like segments.
func splitSentences(text string) []string {
	// Split on period+space, newlines, or double newlines.
	var sentences []string
	var current strings.Builder

	for i, r := range text {
		current.WriteRune(r)
		if r == '\n' || (r == '.' && i+1 < len(text) && text[i+1] == ' ') {
			s := strings.TrimSpace(current.String())
			if len(s) > 0 {
				sentences = append(sentences, s)
			}
			current.Reset()
		}
	}
	if s := strings.TrimSpace(current.String()); len(s) > 0 {
		sentences = append(sentences, s)
	}

	if len(sentences) == 0 {
		return []string{text}
	}
	return sentences
}

// scoreSentence counts how many distinct query terms appear in a sentence.
func scoreSentence(sentence string, termSet map[string]struct{}) int {
	words := strings.FieldsFunc(strings.ToLower(sentence), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	matched := make(map[string]struct{})
	for _, w := range words {
		stemmed := porterStem(w)
		if _, ok := termSet[stemmed]; ok {
			matched[stemmed] = struct{}{}
		}
	}
	return len(matched)
}
