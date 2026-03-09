package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aredgwell/athena/internal/index"
	"github.com/aredgwell/athena/internal/notes"
	"github.com/aredgwell/athena/internal/search"
)

// setupTestDir creates a minimal repo structure for MCP handler tests.
func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	toml := `version = 2
[features]
[policy]
default = "standard"
[tools]
required = []
recommended = []
[security]
enable_secrets_scan = false
enable_workflow_lint = false
[gc]
days = 45
[telemetry]
enabled = true
path = ".athena/telemetry.jsonl"
`
	os.WriteFile(filepath.Join(dir, "athena.toml"), []byte(toml), 0644)
	os.MkdirAll(filepath.Join(dir, ".ai"), 0755)
	os.MkdirAll(filepath.Join(dir, ".athena"), 0755)

	return dir
}

func createTestNote(t *testing.T, dir, noteType, slug, title string) string {
	t.Helper()
	aiDir := filepath.Join(dir, ".ai")
	n, err := notes.NewNote(aiDir, noteType, slug, title)
	if err != nil {
		t.Fatalf("creating test note: %v", err)
	}
	return n.Path
}

// --- Resource handler tests ---

func TestCapabilitiesResource(t *testing.T) {
	handler := capabilitiesHandler()
	result, err := handler(context.Background(), &sdkmcp.ReadResourceRequest{
		Params: &sdkmcp.ReadResourceParams{URI: "athena://capabilities"},
	})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if len(result.Contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(result.Contents))
	}
	text := result.Contents[0].Text
	if !strings.Contains(text, "commands") {
		t.Errorf("expected capabilities JSON with commands, got %s", text)
	}
}

func TestConfigResource(t *testing.T) {
	dir := setupTestDir(t)
	handler := configHandler(dir)
	result, err := handler(context.Background(), &sdkmcp.ReadResourceRequest{
		Params: &sdkmcp.ReadResourceParams{URI: "athena://config"},
	})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := result.Contents[0].Text
	if !strings.Contains(text, "standard") {
		t.Errorf("expected config JSON with policy, got %s", text)
	}
}

func TestNotesResource(t *testing.T) {
	dir := setupTestDir(t)
	createTestNote(t, dir, "context", "test", "Test Note")

	handler := notesHandler(dir)
	result, err := handler(context.Background(), &sdkmcp.ReadResourceRequest{
		Params: &sdkmcp.ReadResourceParams{URI: "athena://notes"},
	})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := result.Contents[0].Text
	if !strings.Contains(text, "Test Note") {
		t.Errorf("expected note in response, got %s", text)
	}
}

func TestIndexResource(t *testing.T) {
	dir := setupTestDir(t)
	createTestNote(t, dir, "context", "idx-test", "Index Test")

	handler := indexHandler(dir)
	result, err := handler(context.Background(), &sdkmcp.ReadResourceRequest{
		Params: &sdkmcp.ReadResourceParams{URI: "athena://index"},
	})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := result.Contents[0].Text
	if !strings.Contains(text, "idx-test") {
		t.Errorf("expected index entry, got %s", text)
	}
}

func TestReportResource(t *testing.T) {
	dir := setupTestDir(t)
	handler := reportHandler(dir)
	result, err := handler(context.Background(), &sdkmcp.ReadResourceRequest{
		Params: &sdkmcp.ReadResourceParams{URI: "athena://report"},
	})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if len(result.Contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(result.Contents))
	}
}

// --- Tool handler tests ---

func TestNoteNewTool(t *testing.T) {
	dir := setupTestDir(t)
	handler := noteNewHandler(dir)

	result, _, err := handler(context.Background(), nil, noteNewArgs{
		Type:  "context",
		Slug:  "mcp-test",
		Title: "MCP Test Note",
	})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Fatal("unexpected error result")
	}

	text := result.Content[0].(*sdkmcp.TextContent).Text
	if !strings.Contains(text, "mcp-test") {
		t.Errorf("expected note ID in result, got %s", text)
	}

	// Verify file exists.
	noteList, _ := notes.ListNotes(filepath.Join(dir, ".ai"), "", "")
	if len(noteList) != 1 {
		t.Errorf("expected 1 note on disk, got %d", len(noteList))
	}
}

func TestNoteNewToolMissingArgs(t *testing.T) {
	dir := setupTestDir(t)
	handler := noteNewHandler(dir)

	result, _, err := handler(context.Background(), nil, noteNewArgs{
		Type: "context",
		// Missing slug and title.
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing args")
	}
}

func TestNoteCloseTool(t *testing.T) {
	dir := setupTestDir(t)
	path := createTestNote(t, dir, "context", "close-test", "Close Test")

	handler := noteCloseHandler(dir)
	result, _, err := handler(context.Background(), nil, noteCloseArgs{
		Path:   path,
		Status: "closed",
	})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Fatal("unexpected error result")
	}

	text := result.Content[0].(*sdkmcp.TextContent).Text
	if !strings.Contains(text, "closed") {
		t.Errorf("expected closed status in result, got %s", text)
	}
}

func TestNotePromoteTool(t *testing.T) {
	dir := setupTestDir(t)
	path := createTestNote(t, dir, "improvement", "promote-test", "Promote Test")

	handler := notePromoteHandler(dir)
	result, _, err := handler(context.Background(), nil, notePromoteArgs{
		Path:   path,
		Target: "docs/auth.md",
	})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Fatal("unexpected error result")
	}

	text := result.Content[0].(*sdkmcp.TextContent).Text
	if !strings.Contains(text, "promoted") {
		t.Errorf("expected promoted status, got %s", text)
	}
}

func TestNoteReadTool(t *testing.T) {
	dir := setupTestDir(t)
	path := createTestNote(t, dir, "context", "read-test", "Read Test")

	handler := noteReadHandler()
	result, _, err := handler(context.Background(), nil, noteReadArgs{Path: path})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := result.Content[0].(*sdkmcp.TextContent).Text
	if !strings.Contains(text, "Read Test") {
		t.Errorf("expected note content, got %s", text)
	}
}

func TestNoteListTool(t *testing.T) {
	dir := setupTestDir(t)
	createTestNote(t, dir, "context", "list-test", "List Test")

	handler := noteListHandler(dir)
	result, _, err := handler(context.Background(), nil, noteListArgs{})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := result.Content[0].(*sdkmcp.TextContent).Text
	if !strings.Contains(text, "List Test") {
		t.Errorf("expected note in list, got %s", text)
	}
}

func TestCheckTool(t *testing.T) {
	dir := setupTestDir(t)
	createTestNote(t, dir, "context", "check-test", "Check Test")

	handler := checkHandler(dir)
	result, _, err := handler(context.Background(), nil, checkArgs{})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := result.Content[0].(*sdkmcp.TextContent).Text
	if !strings.Contains(text, "files_scanned") {
		t.Errorf("expected validation result, got %s", text)
	}
}

func TestIndexRebuildTool(t *testing.T) {
	dir := setupTestDir(t)
	createTestNote(t, dir, "context", "rebuild-test", "Rebuild Test")

	handler := indexRebuildHandler(dir)
	result, _, err := handler(context.Background(), nil, struct{}{})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := result.Content[0].(*sdkmcp.TextContent).Text
	if !strings.Contains(text, "entries") {
		t.Errorf("expected entries count, got %s", text)
	}

	// Verify index.yaml was written.
	if _, err := os.Stat(filepath.Join(dir, ".ai", "index.yaml")); os.IsNotExist(err) {
		t.Error("expected .ai/index.yaml to be created")
	}
	// Verify search-index.json was written.
	if _, err := os.Stat(filepath.Join(dir, ".ai", "search-index.json")); os.IsNotExist(err) {
		t.Error("expected .ai/search-index.json to be created")
	}
}

func TestGCScanTool(t *testing.T) {
	dir := setupTestDir(t)
	createTestNote(t, dir, "context", "gc-test", "GC Test")

	handler := gcScanHandler(dir)
	result, _, err := handler(context.Background(), nil, gcScanArgs{Days: 45})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := result.Content[0].(*sdkmcp.TextContent).Text
	if !strings.Contains(text, "scanned") {
		t.Errorf("expected scan result, got %s", text)
	}
}

func TestDoctorTool(t *testing.T) {
	dir := setupTestDir(t)
	handler := doctorHandler(dir)
	result, _, err := handler(context.Background(), nil, struct{}{})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Error("unexpected error result from doctor")
	}
}

func TestReportTool(t *testing.T) {
	dir := setupTestDir(t)
	handler := reportToolHandler(dir)
	result, _, err := handler(context.Background(), nil, struct{}{})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := result.Content[0].(*sdkmcp.TextContent).Text
	if !strings.Contains(text, "staleness_ratio") {
		t.Errorf("expected metrics in result, got %s", text)
	}
}

func TestContextSearchTool(t *testing.T) {
	dir := setupTestDir(t)
	createTestNote(t, dir, "context", "auth-test", "Authentication Analysis")

	// Build search index.
	aiDir := filepath.Join(dir, ".ai")
	searchIdx, err := index.BuildSearch(aiDir)
	if err != nil {
		t.Fatalf("BuildSearch: %v", err)
	}
	if err := search.WriteIndex(searchIdx, filepath.Join(aiDir, "search-index.json")); err != nil {
		t.Fatalf("WriteIndex: %v", err)
	}

	handler := contextSearchHandler(dir)
	result, _, err := handler(context.Background(), nil, contextSearchArgs{
		Query: "authentication",
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := result.Content[0].(*sdkmcp.TextContent).Text
	if !strings.Contains(text, "auth") {
		t.Errorf("expected auth in results, got %s", text)
	}
}

func TestContextSearchToolMissingIndex(t *testing.T) {
	dir := setupTestDir(t)
	handler := contextSearchHandler(dir)
	result, _, err := handler(context.Background(), nil, contextSearchArgs{Query: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for missing index")
	}
}

func TestContextSearchToolEmptyQuery(t *testing.T) {
	dir := setupTestDir(t)
	handler := contextSearchHandler(dir)
	result, _, err := handler(context.Background(), nil, contextSearchArgs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for empty query")
	}
}

// --- Policy gate, commit lint, security scan tool tests ---

func TestPolicyGateTool(t *testing.T) {
	dir := setupTestDir(t)
	handler := policyGateHandler(dir)

	result, _, err := handler(context.Background(), nil, policyGateArgs{})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Fatal("unexpected error result")
	}

	text := result.Content[0].(*sdkmcp.TextContent).Text
	if !strings.Contains(text, `"ok"`) {
		t.Errorf("expected ok field in result, got %s", text)
	}
}

func TestPolicyGateToolSubset(t *testing.T) {
	dir := setupTestDir(t)
	handler := policyGateHandler(dir)

	result, _, err := handler(context.Background(), nil, policyGateArgs{
		Checks: []string{"check"},
	})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := result.Content[0].(*sdkmcp.TextContent).Text
	if !strings.Contains(text, "check") {
		t.Errorf("expected check in passed list, got %s", text)
	}
}

func TestCommitLintTool(t *testing.T) {
	dir := setupTestDir(t)
	handler := commitLintHandler(dir)

	result, _, err := handler(context.Background(), nil, commitLintArgs{
		Message: "feat(mcp): add policy gate tool",
	})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Fatal("unexpected error result")
	}

	text := result.Content[0].(*sdkmcp.TextContent).Text
	if !strings.Contains(text, `"valid": true`) {
		t.Errorf("expected valid commit, got %s", text)
	}
}

func TestCommitLintToolInvalid(t *testing.T) {
	dir := setupTestDir(t)
	handler := commitLintHandler(dir)

	result, _, err := handler(context.Background(), nil, commitLintArgs{
		Message: "not a conventional commit",
	})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := result.Content[0].(*sdkmcp.TextContent).Text
	if !strings.Contains(text, `"valid": false`) {
		t.Errorf("expected invalid commit, got %s", text)
	}
}

func TestCommitLintToolMissingMessage(t *testing.T) {
	dir := setupTestDir(t)
	handler := commitLintHandler(dir)

	result, _, err := handler(context.Background(), nil, commitLintArgs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for empty message")
	}
}

func TestSecurityScanTool(t *testing.T) {
	dir := setupTestDir(t)
	handler := securityScanHandler(dir)

	result, _, err := handler(context.Background(), nil, securityScanArgs{})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Fatal("unexpected error result")
	}

	text := result.Content[0].(*sdkmcp.TextContent).Text
	if !strings.Contains(text, `"ok"`) {
		t.Errorf("expected ok field in result, got %s", text)
	}
}

// --- Server registration test ---

func TestServerRegistration(t *testing.T) {
	dir := setupTestDir(t)
	srv := NewServer(dir)

	// Connect in-memory to list tools and resources.
	ctx := context.Background()
	serverTransport, clientTransport := sdkmcp.NewInMemoryTransports()

	if _, err := srv.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "1.0"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	// Check tools.
	expectedTools := []string{
		"note_new", "note_close", "note_promote", "note_read",
		"note_list", "check", "check_fix", "index_rebuild",
		"gc_scan", "doctor", "report", "context_search",
		"policy_gate", "commit_lint", "security_scan",
	}

	toolNames := make(map[string]bool)
	for tool, err := range cs.Tools(ctx, nil) {
		if err != nil {
			t.Fatalf("listing tools: %v", err)
		}
		toolNames[tool.Name] = true
	}

	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("tool %q not registered", name)
		}
	}
	if len(toolNames) != len(expectedTools) {
		t.Errorf("expected %d tools, got %d", len(expectedTools), len(toolNames))
	}

	// Check resources.
	expectedResources := []string{
		"athena://capabilities",
		"athena://config",
		"athena://notes",
		"athena://index",
		"athena://report",
	}

	resourceURIs := make(map[string]bool)
	for resource, err := range cs.Resources(ctx, nil) {
		if err != nil {
			t.Fatalf("listing resources: %v", err)
		}
		resourceURIs[resource.URI] = true
	}

	for _, uri := range expectedResources {
		if !resourceURIs[uri] {
			t.Errorf("resource %q not registered", uri)
		}
	}
}

// --- Protocol round-trip test ---

func TestProtocolRoundTrip(t *testing.T) {
	dir := setupTestDir(t)
	srv := NewServer(dir)

	ctx := context.Background()
	serverTransport, clientTransport := sdkmcp.NewInMemoryTransports()

	if _, err := srv.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "1.0"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	// Call note_new tool via protocol.
	result, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "note_new",
		Arguments: map[string]any{
			"type":  "context",
			"slug":  "roundtrip",
			"title": "Round Trip Test",
		},
	})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	if result.IsError {
		t.Fatal("unexpected error from tool call")
	}

	text := result.Content[0].(*sdkmcp.TextContent).Text
	if !strings.Contains(text, "roundtrip") {
		t.Errorf("expected roundtrip in result, got %s", text)
	}

	// Read capabilities resource via protocol.
	resResult, err := cs.ReadResource(ctx, &sdkmcp.ReadResourceParams{
		URI: "athena://capabilities",
	})
	if err != nil {
		t.Fatalf("ReadResource error: %v", err)
	}
	if len(resResult.Contents) != 1 {
		t.Fatalf("expected 1 resource content, got %d", len(resResult.Contents))
	}
	if !strings.Contains(resResult.Contents[0].Text, "commands") {
		t.Error("expected capabilities in resource response")
	}
}
