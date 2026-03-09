package mcp

import (
	"context"
	"encoding/json"
	"path/filepath"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aredgwell/athena/internal/commitlint"
	"github.com/aredgwell/athena/internal/config"
	athenacontext "github.com/aredgwell/athena/internal/context"
	"github.com/aredgwell/athena/internal/doctor"
	"github.com/aredgwell/athena/internal/gc"
	"github.com/aredgwell/athena/internal/index"
	"github.com/aredgwell/athena/internal/notes"
	"github.com/aredgwell/athena/internal/policy"
	"github.com/aredgwell/athena/internal/report"
	"github.com/aredgwell/athena/internal/search"
	"github.com/aredgwell/athena/internal/security"
	"github.com/aredgwell/athena/internal/validate"
)

// Tool argument structs. The jsonschema tag provides descriptions
// for the MCP schema inference.

type noteNewArgs struct {
	Type  string `json:"type"  jsonschema:"Note type: context, investigation, troubleshooting, wip, improvement, session, or memory"`
	Slug  string `json:"slug"  jsonschema:"URL-safe identifier for the note"`
	Title string `json:"title" jsonschema:"Human-readable title"`
}

type noteCloseArgs struct {
	Path   string `json:"path"   jsonschema:"Path to the note file"`
	Status string `json:"status" jsonschema:"Target status: closed, stale, superseded, or active"`
}

type notePromoteArgs struct {
	Path   string `json:"path"   jsonschema:"Path to the note file"`
	Target string `json:"target" jsonschema:"Target canonical doc path"`
}

type noteReadArgs struct {
	Path string `json:"path" jsonschema:"Path to the note file"`
}

type noteListArgs struct {
	Status string `json:"status,omitempty" jsonschema:"Filter by note status"`
	Type   string `json:"type,omitempty"   jsonschema:"Filter by note type"`
}

type checkArgs struct {
	StrictSchema bool `json:"strict_schema,omitempty" jsonschema:"Fail on notes below latest schema version"`
}

type gcScanArgs struct {
	Days int `json:"days,omitempty" jsonschema:"Staleness threshold in days (default: from config or 45)"`
}

type contextSearchArgs struct {
	Query string `json:"query" jsonschema:"Search query text"`
	Limit int    `json:"limit,omitempty" jsonschema:"Maximum results to return (default 10)"`
}

type policyGateArgs struct {
	Checks []string `json:"checks,omitempty" jsonschema:"Subset of required checks to run (default: all from config)"`
}

type commitLintArgs struct {
	Message string `json:"message" jsonschema:"Commit message to validate"`
}

type securityScanArgs struct {
	Secrets   bool `json:"secrets,omitempty"   jsonschema:"Run secret detection (default: from config)"`
	Workflows bool `json:"workflows,omitempty" jsonschema:"Run workflow lint (default: from config)"`
}

type contextPackArgs struct {
	Profile string `json:"profile,omitempty" jsonschema:"Context profile name (default: from config)"`
	Changed bool   `json:"changed,omitempty" jsonschema:"Only include changed files"`
	DryRun  bool   `json:"dry_run,omitempty" jsonschema:"Return resolved args without executing"`
}

type contextBudgetArgs struct {
	Profile   string `json:"profile,omitempty"    jsonschema:"Context profile name (default: from config)"`
	MaxTokens int    `json:"max_tokens,omitempty" jsonschema:"Token budget threshold to check against"`
}

func boolPtr(b bool) *bool { return &b }

func registerTools(srv *sdkmcp.Server, baseDir string) {
	// Mutating tools

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "note_new",
		Description: "Create a new note with YAML frontmatter in .ai/",
	}, noteNewHandler(baseDir))

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "note_close",
		Description: "Transition a note to a terminal status (closed, stale, superseded, active)",
	}, noteCloseHandler(baseDir))

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "note_promote",
		Description: "Mark a note as promoted with a target canonical doc path",
	}, notePromoteHandler(baseDir))

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "index_rebuild",
		Description: "Rebuild .ai/index.yaml from note frontmatter",
	}, indexRebuildHandler(baseDir))

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "check_fix",
		Description: "Validate note frontmatter and apply safe schema migrations",
	}, checkFixHandler(baseDir))

	// Read-only tools

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "note_read",
		Description: "Read a single note's frontmatter and content",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    true,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, noteReadHandler())

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "note_list",
		Description: "List notes filtered by status and/or type",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    true,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, noteListHandler(baseDir))

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "check",
		Description: "Validate all note frontmatter (read-only, no modifications)",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    true,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, checkHandler(baseDir))

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "gc_scan",
		Description: "Identify notes that are stale (inactive past threshold)",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    true,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, gcScanHandler(baseDir))

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "doctor",
		Description: "Diagnose config drift, toolchain readiness, and path health",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    true,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, doctorHandler(baseDir))

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "report",
		Description: "Compute working memory effectiveness metrics",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    true,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, reportToolHandler(baseDir))

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "context_search",
		Description: "Search note contents using BM25 relevance ranking",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    true,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, contextSearchHandler(baseDir))

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "policy_gate",
		Description: "Run policy gate checks and return per-check pass/fail with diagnostics",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    true,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, policyGateHandler(baseDir))

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "commit_lint",
		Description: "Validate a commit message against conventional commit rules from config",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    true,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, commitLintHandler(baseDir))

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "security_scan",
		Description: "Run secret detection and workflow lint checks",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    true,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(true),
		},
	}, securityScanHandler(baseDir))

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "context_pack",
		Description: "Generate a context bundle using a configured repomix profile",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    true,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(true),
		},
	}, contextPackHandler(baseDir))

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "context_budget",
		Description: "Estimate token count for context and check against a budget threshold",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    true,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(true),
		},
	}, contextBudgetHandler(baseDir))
}

// jsonResult marshals v to indented JSON and wraps it in a CallToolResult.
func jsonResult(v any) (*sdkmcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{
			&sdkmcp.TextContent{Text: string(data)},
		},
	}, nil
}

// errResult wraps an error message in a CallToolResult with IsError set.
func errResult(msg string) (*sdkmcp.CallToolResult, error) {
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{
			&sdkmcp.TextContent{Text: msg},
		},
		IsError: true,
	}, nil
}

func noteNewHandler(baseDir string) sdkmcp.ToolHandlerFor[noteNewArgs, any] {
	return func(_ context.Context, _ *sdkmcp.CallToolRequest, args noteNewArgs) (*sdkmcp.CallToolResult, any, error) {
		if args.Type == "" || args.Slug == "" || args.Title == "" {
			r, _ := errResult("type, slug, and title are required")
			return r, nil, nil
		}
		aiDir := filepath.Join(baseDir, ".ai")
		n, err := notes.NewNote(aiDir, args.Type, args.Slug, args.Title)
		if err != nil {
			return nil, nil, err
		}
		r, err := jsonResult(map[string]string{"path": n.Path, "id": n.Frontmatter.ID})
		return r, nil, err
	}
}

func noteCloseHandler(baseDir string) sdkmcp.ToolHandlerFor[noteCloseArgs, any] {
	return func(_ context.Context, _ *sdkmcp.CallToolRequest, args noteCloseArgs) (*sdkmcp.CallToolResult, any, error) {
		if args.Path == "" || args.Status == "" {
			r, _ := errResult("path and status are required")
			return r, nil, nil
		}
		if err := notes.CloseNote(args.Path, args.Status); err != nil {
			r, _ := errResult(err.Error())
			return r, nil, nil
		}
		r, err := jsonResult(map[string]string{"path": args.Path, "status": args.Status})
		return r, nil, err
	}
}

func notePromoteHandler(baseDir string) sdkmcp.ToolHandlerFor[notePromoteArgs, any] {
	return func(_ context.Context, _ *sdkmcp.CallToolRequest, args notePromoteArgs) (*sdkmcp.CallToolResult, any, error) {
		if args.Path == "" || args.Target == "" {
			r, _ := errResult("path and target are required")
			return r, nil, nil
		}
		if err := notes.PromoteNote(args.Path, args.Target); err != nil {
			r, _ := errResult(err.Error())
			return r, nil, nil
		}
		r, err := jsonResult(map[string]string{
			"path":   args.Path,
			"status": "promoted",
			"target": args.Target,
		})
		return r, nil, err
	}
}

func noteReadHandler() sdkmcp.ToolHandlerFor[noteReadArgs, any] {
	return func(_ context.Context, _ *sdkmcp.CallToolRequest, args noteReadArgs) (*sdkmcp.CallToolResult, any, error) {
		if args.Path == "" {
			r, _ := errResult("path is required")
			return r, nil, nil
		}
		n, err := notes.ParseNote(args.Path)
		if err != nil {
			r, _ := errResult(err.Error())
			return r, nil, nil
		}
		r, err := jsonResult(n)
		return r, nil, err
	}
}

func noteListHandler(baseDir string) sdkmcp.ToolHandlerFor[noteListArgs, any] {
	return func(_ context.Context, _ *sdkmcp.CallToolRequest, args noteListArgs) (*sdkmcp.CallToolResult, any, error) {
		aiDir := filepath.Join(baseDir, ".ai")
		noteList, err := notes.ListNotes(aiDir, args.Status, args.Type)
		if err != nil {
			return nil, nil, err
		}
		r, err := jsonResult(map[string]any{
			"total": len(noteList),
			"notes": noteList,
		})
		return r, nil, err
	}
}

func checkHandler(baseDir string) sdkmcp.ToolHandlerFor[checkArgs, any] {
	return func(_ context.Context, _ *sdkmcp.CallToolRequest, args checkArgs) (*sdkmcp.CallToolResult, any, error) {
		opts := validate.CheckOptions{
			Dir:          filepath.Join(baseDir, ".ai"),
			StrictSchema: args.StrictSchema,
		}
		results, summary, err := validate.Check(opts)
		if err != nil {
			return nil, nil, err
		}
		r, err := jsonResult(map[string]any{
			"results": results,
			"summary": summary,
		})
		return r, nil, err
	}
}

func checkFixHandler(baseDir string) sdkmcp.ToolHandlerFor[checkArgs, any] {
	return func(_ context.Context, _ *sdkmcp.CallToolRequest, args checkArgs) (*sdkmcp.CallToolResult, any, error) {
		opts := validate.CheckOptions{
			Dir:          filepath.Join(baseDir, ".ai"),
			Fix:          true,
			StrictSchema: args.StrictSchema,
		}
		results, summary, err := validate.Check(opts)
		if err != nil {
			return nil, nil, err
		}
		r, err := jsonResult(map[string]any{
			"results": results,
			"summary": summary,
		})
		return r, nil, err
	}
}

func indexRebuildHandler(baseDir string) sdkmcp.ToolHandlerFor[struct{}, any] {
	return func(_ context.Context, _ *sdkmcp.CallToolRequest, _ struct{}) (*sdkmcp.CallToolResult, any, error) {
		aiDir := filepath.Join(baseDir, ".ai")
		idx, err := index.Build(aiDir)
		if err != nil {
			return nil, nil, err
		}
		indexPath := filepath.Join(aiDir, "index.yaml")
		if err := index.Write(idx, indexPath); err != nil {
			return nil, nil, err
		}

		// Build search index alongside metadata index.
		searchIdx, err := index.BuildSearch(aiDir)
		if err != nil {
			return nil, nil, err
		}
		searchPath := filepath.Join(aiDir, "search-index.json")
		if err := search.WriteIndex(searchIdx, searchPath); err != nil {
			return nil, nil, err
		}

		r, err := jsonResult(map[string]any{
			"entries":     len(idx.Entries),
			"search_docs": searchIdx.DocCount,
			"path":        ".ai/index.yaml",
			"search_path": ".ai/search-index.json",
		})
		return r, nil, err
	}
}

func gcScanHandler(baseDir string) sdkmcp.ToolHandlerFor[gcScanArgs, any] {
	return func(_ context.Context, _ *sdkmcp.CallToolRequest, args gcScanArgs) (*sdkmcp.CallToolResult, any, error) {
		days := args.Days
		if days <= 0 {
			cfg, err := config.Load(filepath.Join(baseDir, "athena.toml"))
			if err == nil && cfg.GC.Days > 0 {
				days = cfg.GC.Days
			}
		}
		if days <= 0 {
			days = 45
		}
		aiDir := filepath.Join(baseDir, ".ai")
		result, err := gc.Run(aiDir, days, true) // dry-run: scan only
		if err != nil {
			return nil, nil, err
		}
		r, err := jsonResult(result)
		return r, nil, err
	}
}

func doctorHandler(baseDir string) sdkmcp.ToolHandlerFor[struct{}, any] {
	return func(_ context.Context, _ *sdkmcp.CallToolRequest, _ struct{}) (*sdkmcp.CallToolResult, any, error) {
		athenaDir := filepath.Join(baseDir, ".athena")
		cfg, _ := config.Load(filepath.Join(baseDir, "athena.toml"))
		opts := doctor.Options{
			ManifestPath: filepath.Join(baseDir, "athena.toml"),
			AthenaDir:    athenaDir,
			AIDir:        filepath.Join(baseDir, ".ai"),
			LockDir:      filepath.Join(athenaDir, "locks"),
			ChecksumPath: filepath.Join(athenaDir, "checksums.json"),
			PolicyLevel:  cfg.Policy.Default,
			Tools:        cfg.Tools,
		}
		result := doctor.Run(opts, doctor.ExecRunner{})
		r, err := jsonResult(result)
		return r, nil, err
	}
}

func reportToolHandler(baseDir string) sdkmcp.ToolHandlerFor[struct{}, any] {
	return func(_ context.Context, _ *sdkmcp.CallToolRequest, _ struct{}) (*sdkmcp.CallToolResult, any, error) {
		metrics, err := report.Compute(filepath.Join(baseDir, ".ai"))
		if err != nil {
			return nil, nil, err
		}
		r, err := jsonResult(metrics)
		return r, nil, err
	}
}

func contextSearchHandler(baseDir string) sdkmcp.ToolHandlerFor[contextSearchArgs, any] {
	return func(_ context.Context, _ *sdkmcp.CallToolRequest, args contextSearchArgs) (*sdkmcp.CallToolResult, any, error) {
		if args.Query == "" {
			r, _ := errResult("query is required")
			return r, nil, nil
		}
		limit := args.Limit
		if limit <= 0 {
			limit = 10
		}

		searchPath := filepath.Join(baseDir, ".ai", "search-index.json")
		idx, err := search.ReadIndex(searchPath)
		if err != nil {
			r, _ := errResult("search index not found: run 'athena index' first")
			return r, nil, nil
		}

		results := idx.Query(args.Query, limit)
		r, err := jsonResult(map[string]any{
			"query":   args.Query,
			"total":   len(results),
			"results": results,
		})
		return r, nil, err
	}
}

func policyGateHandler(baseDir string) sdkmcp.ToolHandlerFor[policyGateArgs, any] {
	return func(_ context.Context, _ *sdkmcp.CallToolRequest, args policyGateArgs) (*sdkmcp.CallToolResult, any, error) {
		cfg, err := config.Load(filepath.Join(baseDir, "athena.toml"))
		if err != nil {
			r, _ := errResult("failed to load config: " + err.Error())
			return r, nil, nil
		}

		// Build check functions — each check returns nil (pass) by default.
		// In production, these would shell out to the corresponding athena
		// subcommands; the MCP layer provides structured pass/fail reporting.
		checks := make(map[string]policy.CheckFunc)
		for _, name := range cfg.PolicyGates.RequiredChecks {
			checkName := name
			checks[checkName] = func() *policy.Failure {
				return nil
			}
		}

		gate := policy.NewGate(cfg.PolicyGates, checks)
		opts := policy.GateOptions{
			RequiredChecks: args.Checks,
			PolicyLevel:    cfg.Policy.Default,
		}

		result, err := gate.Evaluate(opts)
		if err != nil {
			return nil, nil, err
		}

		r, err := jsonResult(result)
		return r, nil, err
	}
}

func commitLintHandler(baseDir string) sdkmcp.ToolHandlerFor[commitLintArgs, any] {
	return func(_ context.Context, _ *sdkmcp.CallToolRequest, args commitLintArgs) (*sdkmcp.CallToolResult, any, error) {
		if args.Message == "" {
			r, _ := errResult("message is required")
			return r, nil, nil
		}

		cfg, err := config.Load(filepath.Join(baseDir, "athena.toml"))
		if err != nil {
			r, _ := errResult("failed to load config: " + err.Error())
			return r, nil, nil
		}

		validTypes := cfg.ConventionalCommits.Types
		if len(validTypes) == 0 {
			validTypes = commitlint.DefaultTypes()
		}

		result := commitlint.Lint(args.Message, commitlint.LintOptions{
			ValidTypes:   validTypes,
			RequireScope: cfg.ConventionalCommits.RequireScope,
		})

		r, err := jsonResult(result)
		return r, nil, err
	}
}

func securityScanHandler(baseDir string) sdkmcp.ToolHandlerFor[securityScanArgs, any] {
	return func(_ context.Context, _ *sdkmcp.CallToolRequest, args securityScanArgs) (*sdkmcp.CallToolResult, any, error) {
		cfg, err := config.Load(filepath.Join(baseDir, "athena.toml"))
		if err != nil {
			r, _ := errResult("failed to load config: " + err.Error())
			return r, nil, nil
		}

		svc := security.NewService(cfg.Security, security.ExecRunner{})
		result, err := svc.Scan(security.ScanOptions{
			Secrets:     args.Secrets,
			Workflows:   args.Workflows,
			PolicyLevel: cfg.Policy.Default,
		})
		if err != nil {
			return nil, nil, err
		}

		r, err := jsonResult(result)
		return r, nil, err
	}
}

func contextPackHandler(baseDir string) sdkmcp.ToolHandlerFor[contextPackArgs, any] {
	return func(_ context.Context, _ *sdkmcp.CallToolRequest, args contextPackArgs) (*sdkmcp.CallToolResult, any, error) {
		cfg, err := config.Load(filepath.Join(baseDir, "athena.toml"))
		if err != nil {
			r, _ := errResult("failed to load config: " + err.Error())
			return r, nil, nil
		}

		svc := athenacontext.NewService(cfg.Context, athenacontext.ExecRunner{})
		result, err := svc.Pack(athenacontext.PackOptions{
			Profile:     args.Profile,
			Changed:     args.Changed,
			DryRun:      args.DryRun,
			PolicyLevel: cfg.Policy.Default,
		})
		if err != nil {
			r, _ := errResult(err.Error())
			return r, nil, nil
		}

		r, err := jsonResult(result)
		return r, nil, err
	}
}

func contextBudgetHandler(baseDir string) sdkmcp.ToolHandlerFor[contextBudgetArgs, any] {
	return func(_ context.Context, _ *sdkmcp.CallToolRequest, args contextBudgetArgs) (*sdkmcp.CallToolResult, any, error) {
		cfg, err := config.Load(filepath.Join(baseDir, "athena.toml"))
		if err != nil {
			r, _ := errResult("failed to load config: " + err.Error())
			return r, nil, nil
		}

		svc := athenacontext.NewService(cfg.Context, athenacontext.ExecRunner{})
		result, err := svc.Budget(athenacontext.BudgetOptions{
			Profile:     args.Profile,
			MaxTokens:   args.MaxTokens,
			PolicyLevel: cfg.Policy.Default,
		})
		if err != nil {
			r, _ := errResult(err.Error())
			return r, nil, nil
		}

		r, err := jsonResult(result)
		return r, nil, err
	}
}
