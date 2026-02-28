package search

import (
	"os"
	"path/filepath"
	"testing"
)

// --- Tokenizer tests ---

func TestTokenize_Basic(t *testing.T) {
	tokens := Tokenize("Hello World")
	want := []string{"hello", "world"}
	assertTokens(t, tokens, want)
}

func TestTokenize_Punctuation(t *testing.T) {
	tokens := Tokenize("auth-middleware (v2)")
	want := []string{"auth", "middlewar", "v2"} // "middleware" stems to "middlewar" (-e suffix not stripped, but no — actually suffix "re" doesn't match)
	// Actually let's check: "middleware" doesn't end in any of our suffixes, so it stays as-is
	want = []string{"auth", "middleware", "v2"}
	assertTokens(t, tokens, want)
}

func TestTokenize_Stopwords(t *testing.T) {
	tokens := Tokenize("the quick brown fox")
	want := []string{"quick", "brown", "fox"}
	assertTokens(t, tokens, want)
}

func TestTokenize_MinLength(t *testing.T) {
	tokens := Tokenize("a I go to x")
	// "a" and "I"(→"i") are <2 chars, "go" is 2 chars and not a stopword, "to" is a stopword, "x" is <2
	want := []string{"go"}
	assertTokens(t, tokens, want)
}

func TestTokenize_Stemming(t *testing.T) {
	tokens := Tokenize("authenticating authentication configured")
	// "authenticating" → strip "ing" → "authenticat" (len 11, stem 8 ≥3 ✓)
	// "authentication" → strip "tion" → "authentica" (len 14, stem 10 ≥3 ✓)
	// "configured" → strip "ed" → "configur" (len 10, stem 8 ≥3 ✓)
	want := []string{"authenticat", "authentica", "configur"}
	assertTokens(t, tokens, want)
}

func TestTokenize_Empty(t *testing.T) {
	tokens := Tokenize("")
	if len(tokens) != 0 {
		t.Errorf("expected empty, got %v", tokens)
	}
}

func TestTokenize_MarkdownSyntax(t *testing.T) {
	tokens := Tokenize("## Header\n- list item\n```code block```")
	// "##"/"```" split away; "header" stems to "head" (strip -er)
	want := []string{"head", "list", "item", "code", "block"}
	assertTokens(t, tokens, want)
}

// --- BM25 index tests ---

func TestBuildIndex_Empty(t *testing.T) {
	idx := BuildIndex(nil)
	if idx.DocCount != 0 {
		t.Errorf("doc count: got %d, want 0", idx.DocCount)
	}
	if idx.AvgDocLen != 0 {
		t.Errorf("avg doc len: got %f, want 0", idx.AvgDocLen)
	}
}

func TestBuildIndex_SingleDoc(t *testing.T) {
	docs := []IndexableDoc{
		{Path: "test.md", Title: "Auth Setup", Body: "Configure authentication for the API"},
	}
	idx := BuildIndex(docs)
	if idx.DocCount != 1 {
		t.Errorf("doc count: got %d, want 1", idx.DocCount)
	}
	if idx.Version != 1 {
		t.Errorf("version: got %d, want 1", idx.Version)
	}
	if len(idx.InvertedIndex) == 0 {
		t.Error("expected non-empty inverted index")
	}
	// "auth" should appear in title (3x boost) + body via "authentication" stem
	if _, ok := idx.InvertedIndex["auth"]; !ok {
		t.Error("expected 'auth' in inverted index (from title)")
	}
}

func TestBuildIndex_MultiDoc(t *testing.T) {
	docs := []IndexableDoc{
		{Path: "a.md", Title: "Auth", Body: "auth tokens and sessions"},
		{Path: "b.md", Title: "Database", Body: "postgresql queries and auth"},
		{Path: "c.md", Title: "Frontend", Body: "react components"},
	}
	idx := BuildIndex(docs)
	if idx.DocCount != 3 {
		t.Errorf("doc count: got %d, want 3", idx.DocCount)
	}
	// "auth" appears in docs 0 and 1, not 2
	postings := idx.InvertedIndex["auth"]
	if len(postings) != 2 {
		t.Errorf("'auth' postings: got %d, want 2", len(postings))
	}
}

// --- Query tests ---

func TestQuery_ExactMatch(t *testing.T) {
	idx := buildTestIndex()
	results := idx.Query("authentication", 10)
	if len(results) == 0 {
		t.Fatal("expected results for 'authentication'")
	}
	// Doc with "Auth" in title should be first
	if results[0].Title != "Auth Setup" {
		t.Errorf("top result: got %q, want 'Auth Setup'", results[0].Title)
	}
}

func TestQuery_MultiTerm(t *testing.T) {
	idx := buildTestIndex()
	results := idx.Query("database queries", 10)
	if len(results) == 0 {
		t.Fatal("expected results for 'database queries'")
	}
	if results[0].Title != "Database Migration" {
		t.Errorf("top result: got %q, want 'Database Migration'", results[0].Title)
	}
}

func TestQuery_Ranking(t *testing.T) {
	idx := buildTestIndex()
	results := idx.Query("auth", 10)
	if len(results) < 2 {
		t.Fatal("expected at least 2 results")
	}
	// Doc with "Auth" in title (3x boost) should outrank body-only mention
	if results[0].Title != "Auth Setup" {
		t.Errorf("expected title-match to rank first, got %q", results[0].Title)
	}
}

func TestQuery_Limit(t *testing.T) {
	idx := buildTestIndex()
	results := idx.Query("auth", 1)
	if len(results) != 1 {
		t.Errorf("expected 1 result with limit=1, got %d", len(results))
	}
}

func TestQuery_NoMatch(t *testing.T) {
	idx := buildTestIndex()
	results := idx.Query("zyxwvut", 10)
	if len(results) != 0 {
		t.Errorf("expected no results, got %d", len(results))
	}
}

func TestQuery_TitleBoost(t *testing.T) {
	docs := []IndexableDoc{
		{Path: "a.md", Title: "Middleware Config", Body: "some generic body text here"},
		{Path: "b.md", Title: "Generic Note", Body: "this mentions middleware in the body only"},
	}
	idx := BuildIndex(docs)
	results := idx.Query("middleware", 10)
	if len(results) < 2 {
		t.Fatal("expected 2 results")
	}
	if results[0].Path != "a.md" {
		t.Errorf("expected title-match doc first, got %s", results[0].Path)
	}
	if results[0].Score <= results[1].Score {
		t.Error("expected title-match to score higher")
	}
}

func TestQuery_TagBoost(t *testing.T) {
	docs := []IndexableDoc{
		{Path: "a.md", Title: "Note One", Tags: []string{"security"}, Body: "generic content"},
		{Path: "b.md", Title: "Note Two", Body: "generic content about security"},
	}
	idx := BuildIndex(docs)
	results := idx.Query("security", 10)
	if len(results) < 2 {
		t.Fatal("expected 2 results")
	}
	if results[0].Path != "a.md" {
		t.Errorf("expected tag-match doc first, got %s", results[0].Path)
	}
}

func TestQuery_EmptyQuery(t *testing.T) {
	idx := buildTestIndex()
	results := idx.Query("", 10)
	if len(results) != 0 {
		t.Errorf("expected no results for empty query, got %d", len(results))
	}
}

func TestQuery_StopwordsOnly(t *testing.T) {
	idx := buildTestIndex()
	results := idx.Query("the and or", 10)
	if len(results) != 0 {
		t.Errorf("expected no results for stopwords-only query, got %d", len(results))
	}
}

// --- Serialization tests ---

func TestWriteReadIndex_RoundTrip(t *testing.T) {
	idx := buildTestIndex()
	path := filepath.Join(t.TempDir(), "search-index.json")

	if err := WriteIndex(idx, path); err != nil {
		t.Fatalf("WriteIndex: %v", err)
	}

	loaded, err := ReadIndex(path)
	if err != nil {
		t.Fatalf("ReadIndex: %v", err)
	}

	if loaded.DocCount != idx.DocCount {
		t.Errorf("doc count: got %d, want %d", loaded.DocCount, idx.DocCount)
	}
	if len(loaded.Documents) != len(idx.Documents) {
		t.Errorf("documents: got %d, want %d", len(loaded.Documents), len(idx.Documents))
	}
	if len(loaded.InvertedIndex) != len(idx.InvertedIndex) {
		t.Errorf("inverted index terms: got %d, want %d", len(loaded.InvertedIndex), len(idx.InvertedIndex))
	}
}

func TestReadIndex_Missing(t *testing.T) {
	_, err := ReadIndex(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestReadIndex_Corrupt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(path, []byte("not json"), 0o644)
	_, err := ReadIndex(path)
	if err == nil {
		t.Error("expected error for corrupt file")
	}
}

// --- helpers ---

func buildTestIndex() *SearchIndex {
	docs := []IndexableDoc{
		{
			Path:  ".ai/context/auth-setup.md",
			Title: "Auth Setup",
			Type:  "context",
			Tags:  []string{"auth", "security"},
			Body:  "Configure authentication tokens and API keys for the service.",
		},
		{
			Path:  ".ai/investigation/db-migration.md",
			Title: "Database Migration",
			Type:  "investigation",
			Tags:  []string{"database"},
			Body:  "Investigated the PostgreSQL migration path. Auth tokens need rotation.",
		},
		{
			Path:  ".ai/context/frontend-arch.md",
			Title: "Frontend Architecture",
			Type:  "context",
			Body:  "React components with TypeScript and Tailwind CSS.",
		},
	}
	return BuildIndex(docs)
}

func assertTokens(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("token count: got %d %v, want %d %v", len(got), got, len(want), want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("token[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}
