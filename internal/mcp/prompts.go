package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerPrompts() {
	s.mcpServer.AddPrompt(&mcp.Prompt{
		Name:        "usage-guide",
		Description: "Complete reference for all fetchr tools and parameters",
	}, promptUsageGuide)

	s.mcpServer.AddPrompt(&mcp.Prompt{
		Name:        "example-get",
		Description: "Example: Simple GET request",
	}, promptExampleGet)

	s.mcpServer.AddPrompt(&mcp.Prompt{
		Name:        "example-post-json",
		Description: "Example: POST request with JSON body and auth headers",
	}, promptExamplePostJSON)

	s.mcpServer.AddPrompt(&mcp.Prompt{
		Name:        "example-fingerprint",
		Description: "Example: Request with custom JA3 + User-Agent fingerprinting",
	}, promptExampleFingerprint)

	s.mcpServer.AddPrompt(&mcp.Prompt{
		Name:        "example-proxy",
		Description: "Example: Request through a proxy",
	}, promptExampleProxy)

	s.mcpServer.AddPrompt(&mcp.Prompt{
		Name:        "example-batch",
		Description: "Example: Batch requests to multiple endpoints",
	}, promptExampleBatch)

	s.mcpServer.AddPrompt(&mcp.Prompt{
		Name:        "example-batch-mixed",
		Description: "Example: Batch with mixed methods (GET + POST)",
	}, promptExampleBatchMixed)

	s.mcpServer.AddPrompt(&mcp.Prompt{
		Name:        "example-profiles",
		Description: "Example: Using named profiles from config",
	}, promptExampleProfiles)

	s.mcpServer.AddPrompt(&mcp.Prompt{
		Name:        "example-print-curl",
		Description: "Example: Print request as curl command",
	}, promptExamplePrintCurl)
}

func promptMsg(desc, text string) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: desc,
		Messages: []*mcp.PromptMessage{{
			Role:    "user",
			Content: &mcp.TextContent{Text: text},
		}},
	}, nil
}

func promptUsageGuide(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return promptMsg("Complete reference for fetchr tools", `# fetchr Usage Guide

## Tool: request
Make a single HTTP request with TLS fingerprinting.

Parameters:
- url (string, required) — The URL to request
- method (string) — GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS. Default: GET
- headers (object) — Key-value pairs of HTTP headers
- header_order (array) — Header order for fingerprint accuracy
- body (string) — Request body for POST/PUT/PATCH
- cookies (array) — Cookies as [{name, value}]
- ja3 (string) — JA3 TLS fingerprint
- ja4r (string) — JA4 raw fingerprint
- http2_fingerprint (string) — HTTP/2 SETTINGS frame fingerprint
- quic_fingerprint (string) — QUIC fingerprint
- user_agent (string) — User-Agent string
- proxy (string) — Proxy URL (http, https, socks5)
- timeout (integer) — Seconds before timeout. Default: 30
- insecure (boolean) — Skip TLS cert verification
- disable_redirect (boolean) — Don't follow 3xx redirects
- force_http1 (boolean) — Force HTTP/1.1
- force_http3 (boolean) — Force HTTP/3 (QUIC)
- protocol (string) — http1, http2, http3
- profile (string) — Named profile from config (e.g. "chrome", "firefox")
- print_curl (boolean) — Print equivalent curl command

## Tool: batch
Make multiple HTTP requests concurrently. Returns JSON array in same order.

Parameters:
- requests (array, required) — Array of request objects (same params as request tool)
- profile (string) — Apply this profile to all requests`)
}

func promptExampleGet(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return promptMsg("Simple GET request", `Use the request tool to fetch https://httpbin.org/get`)
}

func promptExamplePostJSON(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return promptMsg("POST with JSON body and auth header", `Use the request tool to POST to https://httpbin.org/post with:
- Header "Content-Type: application/json"
- Header "Authorization: Bearer my-token"
- Body: {"name": "John", "email": "john@example.com"}`)
}

func promptExampleFingerprint(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return promptMsg("Request with JA3 fingerprinting", `Use the request tool to fetch https://tls.peet.ws/api/all with:
- ja3: "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0"
- user_agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0"`)
}

func promptExampleProxy(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return promptMsg("Request through a proxy", `Use the request tool to fetch https://httpbin.org/ip with:
- proxy: "http://proxy.example.com:8080"`)
}

func promptExampleBatch(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return promptMsg("Batch fetch multiple endpoints", `Use the batch tool to fetch these 3 URLs at the same time:
1. https://httpbin.org/get
2. https://httpbin.org/ip
3. https://httpbin.org/user-agent`)
}

func promptExampleBatchMixed(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return promptMsg("Batch with mixed methods", `Use the batch tool with these requests:
1. GET https://httpbin.org/get
2. POST https://httpbin.org/post with Content-Type: application/json and body {"action": "create"}
3. GET https://httpbin.org/headers with a custom header X-Custom: hello`)
}

func promptExampleProfiles(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return promptMsg("Using named profiles", `Use the request tool to fetch https://tls.peet.ws/api/all with:
- profile: "chrome"

This uses the Chrome TLS fingerprint and User-Agent from the config file.`)
}

func promptExamplePrintCurl(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return promptMsg("Print as curl command", `Use the request tool to POST to https://httpbin.org/post with:
- Header "Content-Type: application/json"
- Body: {"key": "value"}
- print_curl: true

This will show the equivalent curl command alongside the response.`)
}
