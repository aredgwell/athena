// Package search implements BM25 lexical search over note contents.
package search

import (
	"strings"
	"unicode"
)

// stopwords is a set of common English words filtered during tokenization.
var stopwords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {},
	"be": {}, "but": {}, "by": {}, "can": {}, "could": {}, "do": {},
	"does": {}, "did": {}, "for": {}, "from": {}, "had": {}, "has": {},
	"have": {}, "he": {}, "her": {}, "his": {}, "if": {}, "in": {},
	"into": {}, "is": {}, "it": {}, "its": {}, "may": {}, "might": {},
	"my": {}, "no": {}, "not": {}, "of": {}, "on": {}, "or": {},
	"our": {}, "shall": {}, "she": {}, "should": {}, "so": {}, "than": {},
	"that": {}, "the": {}, "their": {}, "them": {}, "then": {}, "there": {},
	"these": {}, "they": {}, "this": {}, "to": {}, "too": {}, "us": {},
	"very": {}, "was": {}, "we": {}, "were": {}, "what": {}, "when": {},
	"which": {}, "who": {}, "will": {}, "with": {}, "would": {}, "you": {},
	"your": {},
}

// Tokenize splits text into normalized tokens suitable for BM25 indexing.
// It lowercases, splits on non-alphanumeric boundaries, filters stopwords,
// applies Porter stemming, and drops tokens shorter than 2 characters.
func Tokenize(text string) []string {
	lower := strings.ToLower(text)

	// Split on non-letter, non-digit boundaries.
	words := strings.FieldsFunc(lower, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	tokens := make([]string, 0, len(words))
	for _, w := range words {
		if len(w) < 2 {
			continue
		}
		if _, ok := stopwords[w]; ok {
			continue
		}
		stemmed := porterStem(w)
		if len(stemmed) < 2 {
			continue
		}
		tokens = append(tokens, stemmed)
	}
	return tokens
}
