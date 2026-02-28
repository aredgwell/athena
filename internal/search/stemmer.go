package search

import "strings"

// porterStem applies the Porter stemming algorithm to a single word.
// This replaces the naive suffix stripping with linguistically-aware stemming.
// Reference: M.F. Porter, "An algorithm for suffix stripping", 1980.
func porterStem(word string) string {
	if len(word) <= 2 {
		return word
	}

	w := word

	// Step 1a: plurals
	w = step1a(w)
	// Step 1b: -ed, -ing
	w = step1b(w)
	// Step 1c: y -> i
	w = step1c(w)
	// Step 2: double suffix removal
	w = step2(w)
	// Step 3: further suffix removal
	w = step3(w)
	// Step 4: -ent, -ance, etc.
	w = step4(w)
	// Step 5: final cleanup
	w = step5(w)

	return w
}

// measure counts the number of VC (vowel-consonant) sequences in the stem.
func measure(s string) int {
	n := 0
	i := 0
	// Skip leading consonants.
	for i < len(s) && !isVowelAt(s, i) {
		i++
	}
	for i < len(s) {
		// Skip vowels.
		for i < len(s) && isVowelAt(s, i) {
			i++
		}
		if i < len(s) {
			n++
			// Skip consonants.
			for i < len(s) && !isVowelAt(s, i) {
				i++
			}
		}
	}
	return n
}

// isVowelAt returns true if s[i] is a vowel. 'y' is a vowel if preceded by a consonant.
func isVowelAt(s string, i int) bool {
	switch s[i] {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	case 'y':
		return i > 0 && !isVowelAt(s, i-1)
	}
	return false
}

// containsVowel returns true if s contains at least one vowel.
func containsVowel(s string) bool {
	for i := range s {
		if isVowelAt(s, i) {
			return true
		}
	}
	return false
}

// endsWithDouble returns true if s ends with a double consonant.
func endsWithDouble(s string) bool {
	if len(s) < 2 {
		return false
	}
	return s[len(s)-1] == s[len(s)-2] && !isVowelAt(s, len(s)-1)
}

// endsCVC returns true if s ends consonant-vowel-consonant where the
// final consonant is not w, x, or y.
func endsCVC(s string) bool {
	if len(s) < 3 {
		return false
	}
	l := len(s)
	c := s[l-1]
	return !isVowelAt(s, l-1) && isVowelAt(s, l-2) && !isVowelAt(s, l-3) &&
		c != 'w' && c != 'x' && c != 'y'
}

func step1a(w string) string {
	if strings.HasSuffix(w, "sses") {
		return w[:len(w)-2]
	}
	if strings.HasSuffix(w, "ies") {
		return w[:len(w)-2]
	}
	if strings.HasSuffix(w, "ss") {
		return w
	}
	if strings.HasSuffix(w, "s") {
		return w[:len(w)-1]
	}
	return w
}

func step1b(w string) string {
	if strings.HasSuffix(w, "eed") {
		stem := w[:len(w)-3]
		if measure(stem) > 0 {
			return w[:len(w)-1] // -> ee
		}
		return w
	}

	var stemFound string
	changed := false
	if strings.HasSuffix(w, "ed") {
		stemFound = w[:len(w)-2]
		if containsVowel(stemFound) {
			w = stemFound
			changed = true
		}
	} else if strings.HasSuffix(w, "ing") {
		stemFound = w[:len(w)-3]
		if containsVowel(stemFound) {
			w = stemFound
			changed = true
		}
	}

	if changed {
		if strings.HasSuffix(w, "at") || strings.HasSuffix(w, "bl") || strings.HasSuffix(w, "iz") {
			return w + "e"
		}
		if endsWithDouble(w) {
			c := w[len(w)-1]
			if c != 'l' && c != 's' && c != 'z' {
				return w[:len(w)-1]
			}
		}
		if measure(w) == 1 && endsCVC(w) {
			return w + "e"
		}
	}
	return w
}

func step1c(w string) string {
	if strings.HasSuffix(w, "y") && containsVowel(w[:len(w)-1]) {
		return w[:len(w)-1] + "i"
	}
	return w
}

func step2(w string) string {
	replacements := []struct {
		suffix, repl string
	}{
		{"ational", "ate"},
		{"tional", "tion"},
		{"enci", "ence"},
		{"anci", "ance"},
		{"izer", "ize"},
		{"abli", "able"},
		{"alli", "al"},
		{"entli", "ent"},
		{"eli", "e"},
		{"ousli", "ous"},
		{"ization", "ize"},
		{"ation", "ate"},
		{"ator", "ate"},
		{"alism", "al"},
		{"iveness", "ive"},
		{"fulness", "ful"},
		{"ousness", "ous"},
		{"aliti", "al"},
		{"iviti", "ive"},
		{"biliti", "ble"},
	}
	for _, r := range replacements {
		if strings.HasSuffix(w, r.suffix) {
			stem := w[:len(w)-len(r.suffix)]
			if measure(stem) > 0 {
				return stem + r.repl
			}
			return w
		}
	}
	return w
}

func step3(w string) string {
	replacements := []struct {
		suffix, repl string
	}{
		{"icate", "ic"},
		{"ative", ""},
		{"alize", "al"},
		{"iciti", "ic"},
		{"ical", "ic"},
		{"ful", ""},
		{"ness", ""},
	}
	for _, r := range replacements {
		if strings.HasSuffix(w, r.suffix) {
			stem := w[:len(w)-len(r.suffix)]
			if measure(stem) > 0 {
				return stem + r.repl
			}
			return w
		}
	}
	return w
}

func step4(w string) string {
	suffixes := []string{
		"al", "ance", "ence", "er", "ic", "able", "ible", "ant",
		"ement", "ment", "ent", "ion", "ou", "ism", "ate", "iti",
		"ous", "ive", "ize",
	}
	for _, s := range suffixes {
		if strings.HasSuffix(w, s) {
			stem := w[:len(w)-len(s)]
			if s == "ion" {
				// Special: stem must end in s or t.
				if len(stem) > 0 && (stem[len(stem)-1] == 's' || stem[len(stem)-1] == 't') {
					if measure(stem) > 1 {
						return stem
					}
				}
				return w
			}
			if measure(stem) > 1 {
				return stem
			}
			return w
		}
	}
	return w
}

func step5(w string) string {
	if strings.HasSuffix(w, "e") {
		stem := w[:len(w)-1]
		m := measure(stem)
		if m > 1 {
			return stem
		}
		if m == 1 && !endsCVC(stem) {
			return stem
		}
	}
	if strings.HasSuffix(w, "ll") && measure(w[:len(w)-1]) > 1 {
		return w[:len(w)-1]
	}
	return w
}
