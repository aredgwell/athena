package search

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// BM25 parameters (Okapi BM25 defaults).
const (
	K1 = 1.2
	B  = 0.75
)

// Title and tag tokens are repeated to boost their weight in the index.
const (
	titleBoost = 3
	tagBoost   = 2
)

// SearchIndex is a serializable BM25 inverted index.
type SearchIndex struct {
	Version       int                 `json:"version"`
	Generated     string              `json:"generated"`
	DocCount      int                 `json:"doc_count"`
	AvgDocLen     float64             `json:"avg_doc_len"`
	Documents     []Document          `json:"documents"`
	InvertedIndex map[string][]Posting `json:"inverted_index"`
}

// maxSummaryLen is the maximum length of the stored body summary.
const maxSummaryLen = 500

// Document represents an indexed note's metadata.
type Document struct {
	Path    string   `json:"path"`
	Title   string   `json:"title"`
	Type    string   `json:"type"`
	Status  string   `json:"status"`
	Tags    []string `json:"tags,omitempty"`
	DocLen  int      `json:"doc_len"`
	Summary string   `json:"summary,omitempty"`
}

// Posting records a term's frequency in a specific document.
type Posting struct {
	DocIdx int `json:"doc_idx"`
	Freq   int `json:"freq"`
}

// SearchResult is a ranked search hit.
type SearchResult struct {
	Path    string   `json:"path"`
	Title   string   `json:"title"`
	Type    string   `json:"type"`
	Status  string   `json:"status"`
	Score   float64  `json:"score"`
	Tags    []string `json:"tags,omitempty"`
	Snippet string   `json:"snippet,omitempty"`
}

// IndexableDoc is the input for BuildIndex. Decoupled from the notes
// package so the search package has no internal dependencies.
type IndexableDoc struct {
	Path   string
	Title  string
	Type   string
	Status string
	Tags   []string
	Body   string
}

// BuildIndex constructs a BM25 search index from the given documents.
// Title tokens are weighted titleBoost×, tag tokens tagBoost×.
func BuildIndex(docs []IndexableDoc) *SearchIndex {
	idx := &SearchIndex{
		Version:       1,
		Generated:     time.Now().UTC().Format(time.RFC3339),
		DocCount:      len(docs),
		Documents:     make([]Document, len(docs)),
		InvertedIndex: make(map[string][]Posting),
	}

	totalLen := 0
	for i, doc := range docs {
		tokens := buildDocTokens(doc)

		summary := doc.Body
		if len(summary) > maxSummaryLen {
			summary = summary[:maxSummaryLen]
		}

		idx.Documents[i] = Document{
			Path:    doc.Path,
			Title:   doc.Title,
			Type:    doc.Type,
			Status:  doc.Status,
			Tags:    doc.Tags,
			DocLen:  len(tokens),
			Summary: summary,
		}
		totalLen += len(tokens)

		// Count term frequencies.
		tf := make(map[string]int)
		for _, t := range tokens {
			tf[t]++
		}

		for term, freq := range tf {
			idx.InvertedIndex[term] = append(idx.InvertedIndex[term], Posting{
				DocIdx: i,
				Freq:   freq,
			})
		}
	}

	if len(docs) > 0 {
		idx.AvgDocLen = float64(totalLen) / float64(len(docs))
	}
	return idx
}

// buildDocTokens produces the weighted token stream for a document.
func buildDocTokens(doc IndexableDoc) []string {
	titleTokens := Tokenize(doc.Title)
	bodyTokens := Tokenize(doc.Body)

	var tagTokens []string
	for _, tag := range doc.Tags {
		tagTokens = append(tagTokens, Tokenize(tag)...)
	}

	// Estimate capacity: title*boost + body + tags*boost.
	cap := len(titleTokens)*titleBoost + len(bodyTokens) + len(tagTokens)*tagBoost
	tokens := make([]string, 0, cap)

	for range titleBoost {
		tokens = append(tokens, titleTokens...)
	}
	tokens = append(tokens, bodyTokens...)
	for range tagBoost {
		tokens = append(tokens, tagTokens...)
	}
	return tokens
}

// FuzzyMaxDist is the maximum edit distance for fuzzy matching.
// Set to 0 to disable fuzzy matching.
const FuzzyMaxDist = 1

// Query searches the index and returns up to limit results sorted by
// descending BM25 score. Only documents with a positive score are returned.
// Fuzzy matching (edit distance 1) is used when exact terms miss.
// Snippets are extracted from stored document summaries.
func (idx *SearchIndex) Query(query string, limit int) []SearchResult {
	queryTerms := Tokenize(query)
	if len(queryTerms) == 0 {
		return nil
	}

	scores := make([]float64, idx.DocCount)
	for _, term := range queryTerms {
		postings, ok := idx.InvertedIndex[term]
		if !ok && FuzzyMaxDist > 0 {
			// Fuzzy fallback: find similar terms within edit distance.
			fuzzyTerms := fuzzyLookup(term, idx.InvertedIndex, FuzzyMaxDist)
			for _, ft := range fuzzyTerms {
				fp := idx.InvertedIndex[ft]
				// Discount fuzzy matches by 50% to prefer exact matches.
				idf := calcIDF(idx.DocCount, len(fp)) * 0.5
				for _, p := range fp {
					dl := float64(idx.Documents[p.DocIdx].DocLen)
					tf := float64(p.Freq)
					num := tf * (K1 + 1)
					denom := tf + K1*(1-B+B*dl/idx.AvgDocLen)
					scores[p.DocIdx] += idf * num / denom
				}
			}
			continue
		}
		if !ok {
			continue
		}
		idf := calcIDF(idx.DocCount, len(postings))
		for _, p := range postings {
			dl := float64(idx.Documents[p.DocIdx].DocLen)
			tf := float64(p.Freq)
			num := tf * (K1 + 1)
			denom := tf + K1*(1-B+B*dl/idx.AvgDocLen)
			scores[p.DocIdx] += idf * num / denom
		}
	}

	// Collect results with positive scores.
	var results []SearchResult
	for i, score := range scores {
		if score <= 0 {
			continue
		}
		doc := idx.Documents[i]
		snippet := ExtractSnippet(doc.Summary, query, defaultSnippetLen)
		results = append(results, SearchResult{
			Path:    doc.Path,
			Title:   doc.Title,
			Type:    doc.Type,
			Status:  doc.Status,
			Score:   math.Round(score*1000) / 1000,
			Tags:    doc.Tags,
			Snippet: snippet,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// calcIDF computes the inverse document frequency for a term.
func calcIDF(totalDocs, docFreq int) float64 {
	n := float64(totalDocs)
	df := float64(docFreq)
	return math.Log((n-df+0.5)/(df+0.5) + 1)
}

// WriteIndex serializes the index to a JSON file.
func WriteIndex(idx *SearchIndex, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(idx)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ReadIndex deserializes a search index from a JSON file.
func ReadIndex(path string) (*SearchIndex, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var idx SearchIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
}
