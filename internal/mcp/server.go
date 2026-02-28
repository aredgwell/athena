// Package mcp implements an MCP (Model Context Protocol) server that
// exposes Athena commands as tools and resources over stdio transport.
package mcp

import (
	"context"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Version is injected from the CLI package at wiring time.
var Version = "dev"

// NewServer creates a configured MCP server with all Athena tools and
// resources registered. baseDir is the repository root.
func NewServer(baseDir string) *sdkmcp.Server {
	srv := sdkmcp.NewServer(
		&sdkmcp.Implementation{
			Name:    "athena",
			Version: Version,
		},
		nil,
	)

	registerResources(srv, baseDir)
	registerTools(srv, baseDir)

	return srv
}

// Run starts the MCP server on stdio transport, blocking until the
// context is cancelled or the transport closes.
func Run(ctx context.Context, baseDir string) error {
	srv := NewServer(baseDir)
	return srv.Run(ctx, &sdkmcp.StdioTransport{})
}
