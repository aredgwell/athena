package mcp

import (
	"context"
	"encoding/json"
	"path/filepath"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aredgwell/athena/internal/capabilities"
	"github.com/aredgwell/athena/internal/config"
	"github.com/aredgwell/athena/internal/index"
	"github.com/aredgwell/athena/internal/notes"
	"github.com/aredgwell/athena/internal/report"
)

func registerResources(srv *sdkmcp.Server, baseDir string) {
	srv.AddResource(&sdkmcp.Resource{
		URI:         "athena://capabilities",
		Name:        "capabilities",
		Description: "Machine-readable command, feature, and contract inventory",
		MIMEType:    "application/json",
	}, capabilitiesHandler())

	srv.AddResource(&sdkmcp.Resource{
		URI:         "athena://config",
		Name:        "config",
		Description: "Current athena.toml configuration with defaults applied",
		MIMEType:    "application/json",
	}, configHandler(baseDir))

	srv.AddResource(&sdkmcp.Resource{
		URI:         "athena://notes",
		Name:        "notes",
		Description: "All notes in .ai/ with frontmatter metadata",
		MIMEType:    "application/json",
	}, notesHandler(baseDir))

	srv.AddResource(&sdkmcp.Resource{
		URI:         "athena://index",
		Name:        "index",
		Description: "Note index built from .ai/ frontmatter",
		MIMEType:    "application/json",
	}, indexHandler(baseDir))

	srv.AddResource(&sdkmcp.Resource{
		URI:         "athena://report",
		Name:        "report",
		Description: "Working memory effectiveness metrics",
		MIMEType:    "application/json",
	}, reportHandler(baseDir))
}

func capabilitiesHandler() sdkmcp.ResourceHandler {
	return func(_ context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
		data, err := json.Marshal(capabilities.Get())
		if err != nil {
			return nil, err
		}
		return &sdkmcp.ReadResourceResult{
			Contents: []*sdkmcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			}},
		}, nil
	}
}

func configHandler(baseDir string) sdkmcp.ResourceHandler {
	return func(_ context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
		cfg, err := config.Load(filepath.Join(baseDir, "athena.toml"))
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(cfg)
		if err != nil {
			return nil, err
		}
		return &sdkmcp.ReadResourceResult{
			Contents: []*sdkmcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			}},
		}, nil
	}
}

func notesHandler(baseDir string) sdkmcp.ResourceHandler {
	return func(_ context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
		noteList, err := notes.ListNotes(filepath.Join(baseDir, ".ai"), "", "")
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(noteList)
		if err != nil {
			return nil, err
		}
		return &sdkmcp.ReadResourceResult{
			Contents: []*sdkmcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			}},
		}, nil
	}
}

func indexHandler(baseDir string) sdkmcp.ResourceHandler {
	return func(_ context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
		idx, err := index.Build(filepath.Join(baseDir, ".ai"))
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(idx)
		if err != nil {
			return nil, err
		}
		return &sdkmcp.ReadResourceResult{
			Contents: []*sdkmcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			}},
		}, nil
	}
}

func reportHandler(baseDir string) sdkmcp.ResourceHandler {
	return func(_ context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
		r, err := report.Compute(filepath.Join(baseDir, ".ai"))
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(r)
		if err != nil {
			return nil, err
		}
		return &sdkmcp.ReadResourceResult{
			Contents: []*sdkmcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			}},
		}, nil
	}
}
