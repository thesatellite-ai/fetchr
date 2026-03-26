package mcp

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/thesatellite-ai/fetchr/internal/config"
	"github.com/thesatellite-ai/fetchr/internal/version"
	"github.com/thesatellite-ai/fetchr/pkg/curl"
)

// Server wraps the MCP server with our curl client and config.
type Server struct {
	mcpServer *mcp.Server
	client    *curl.Client
	config    *config.Config
}

// NewServer creates a configured MCP server with all tools and prompts registered.
func NewServer(cfg *config.Config) *Server {
	var clientOpts []curl.ClientOption
	clientOpts = append(clientOpts, curl.WithDefaults(cfg.Defaults))

	if logger := cfg.BuildLogger(); logger != nil {
		clientOpts = append(clientOpts, curl.WithLogger(logger))
	}

	s := &Server{
		mcpServer: mcp.NewServer(&mcp.Implementation{
			Name:    "fetchr",
			Version: version.Version,
		}, nil),
		client: curl.New(clientOpts...),
		config: cfg,
	}

	s.registerTools()
	s.registerPrompts()

	return s
}

// RunStdio starts the server in stdio mode.
func (s *Server) RunStdio(ctx context.Context) error {
	return s.mcpServer.Run(ctx, &mcp.StdioTransport{})
}

// SSEHandler returns an HTTP handler for SSE transport.
func (s *Server) SSEHandler() http.Handler {
	return mcp.NewSSEHandler(func(*http.Request) *mcp.Server { return s.mcpServer }, nil)
}

// Close shuts down the curl client.
func (s *Server) Close() {
	s.client.Close()
}

// resolveOptions merges config defaults, profile, and per-request options.
func (s *Server) resolveOptions(opts curl.RequestOptions) curl.RequestOptions {
	base := s.config.Defaults

	if opts.Profile != "" {
		if profile, err := s.config.ResolveProfile(opts.Profile); err == nil {
			base = profile
		}
	}

	return curl.Merge(base, opts)
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

func jsonResult(v any) *mcp.CallToolResult {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errorResult("failed to marshal response: " + err.Error())
	}
	return textResult(string(data))
}
