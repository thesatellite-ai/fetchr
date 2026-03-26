package mcp

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/thesatellite-ai/fetchr/pkg/curl"
)

func (s *Server) registerTools() {
	s.mcpServer.AddTool(&mcp.Tool{
		Name:        "request",
		Description: "Make HTTP requests with TLS fingerprinting (JA3/JA4, HTTP/2). Supports all methods, custom headers, body, cookies, proxy, timeout, redirect control, and browser profiles.",
		InputSchema: json.RawMessage(requestSchema),
	}, s.handleRequest)

	s.mcpServer.AddTool(&mcp.Tool{
		Name:        "batch",
		Description: "Make multiple HTTP requests in parallel with TLS fingerprinting. Returns an array of responses in the same order as the requests.",
		InputSchema: json.RawMessage(batchSchema),
	}, s.handleBatch)
}

type requestResult struct {
	*curl.Response
	CurlCmd string `json:"curl_cmd,omitempty"`
}

func (s *Server) handleRequest(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var opts curl.RequestOptions
	if err := json.Unmarshal(req.Params.Arguments, &opts); err != nil {
		return errorResult("Invalid arguments: " + err.Error()), nil
	}

	resolved := s.resolveOptions(opts)

	resp, err := s.client.Do(context.Background(), resolved)
	if err != nil {
		return errorResult(err.Error()), nil
	}

	result := requestResult{Response: resp}
	if resolved.PrintCurl {
		result.CurlCmd = curl.ToCurl(resolved)
	}

	return jsonResult(result), nil
}

func (s *Server) handleBatch(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		Requests []curl.RequestOptions `json:"requests"`
		Profile  string                `json:"profile"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return errorResult("Invalid arguments: " + err.Error()), nil
	}

	if len(params.Requests) == 0 {
		return errorResult("requests array is empty"), nil
	}

	// Apply batch-level profile to all requests that don't have their own
	if params.Profile != "" {
		for i := range params.Requests {
			if params.Requests[i].Profile == "" {
				params.Requests[i].Profile = params.Profile
			}
		}
	}

	results, err := s.client.Batch(context.Background(), params.Requests)
	if err != nil {
		return errorResult(err.Error()), nil
	}

	return jsonResult(results), nil
}

const requestSchema = `{
	"type": "object",
	"required": ["url"],
	"properties": {
		"url": {
			"type": "string",
			"description": "The URL to request"
		},
		"method": {
			"type": "string",
			"description": "HTTP method. Default: GET",
			"enum": ["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"]
		},
		"headers": {
			"type": "object",
			"description": "HTTP headers as key-value pairs",
			"additionalProperties": { "type": "string" }
		},
		"header_order": {
			"type": "array",
			"description": "Order of headers (critical for fingerprint accuracy)",
			"items": { "type": "string" }
		},
		"body": {
			"type": "string",
			"description": "Request body (for POST, PUT, PATCH)"
		},
		"cookies": {
			"type": "array",
			"description": "Cookies to send",
			"items": {
				"type": "object",
				"properties": {
					"name": { "type": "string" },
					"value": { "type": "string" }
				},
				"required": ["name", "value"]
			}
		},
		"ja3": {
			"type": "string",
			"description": "JA3 TLS fingerprint string"
		},
		"ja4r": {
			"type": "string",
			"description": "JA4 raw fingerprint string"
		},
		"http2_fingerprint": {
			"type": "string",
			"description": "HTTP/2 fingerprint (SETTINGS frame + priorities)"
		},
		"quic_fingerprint": {
			"type": "string",
			"description": "QUIC fingerprint string"
		},
		"user_agent": {
			"type": "string",
			"description": "User-Agent header value"
		},
		"proxy": {
			"type": "string",
			"description": "Proxy URL (http, https, socks5)"
		},
		"timeout": {
			"type": "integer",
			"description": "Timeout in seconds. Default: 30"
		},
		"insecure": {
			"type": "boolean",
			"description": "Skip TLS certificate verification"
		},
		"disable_redirect": {
			"type": "boolean",
			"description": "Disable following HTTP redirects"
		},
		"force_http1": {
			"type": "boolean",
			"description": "Force HTTP/1.1"
		},
		"force_http3": {
			"type": "boolean",
			"description": "Force HTTP/3 (QUIC)"
		},
		"protocol": {
			"type": "string",
			"description": "Protocol: http1, http2, http3"
		},
		"profile": {
			"type": "string",
			"description": "Named profile from config (e.g. chrome, firefox)"
		},
		"print_curl": {
			"type": "boolean",
			"description": "Print the equivalent curl command"
		}
	}
}`

const batchSchema = `{
	"type": "object",
	"required": ["requests"],
	"properties": {
		"requests": {
			"type": "array",
			"description": "Array of request objects to execute in parallel",
			"items": {
				"type": "object",
				"required": ["url"],
				"properties": {
					"url": { "type": "string", "description": "The URL to request" },
					"method": { "type": "string", "description": "HTTP method. Default: GET" },
					"headers": { "type": "object", "additionalProperties": { "type": "string" } },
					"body": { "type": "string", "description": "Request body" },
					"ja3": { "type": "string" },
					"ja4r": { "type": "string" },
					"http2_fingerprint": { "type": "string" },
					"user_agent": { "type": "string" },
					"proxy": { "type": "string" },
					"timeout": { "type": "integer" },
					"insecure": { "type": "boolean" },
					"disable_redirect": { "type": "boolean" },
					"profile": { "type": "string" },
					"print_curl": { "type": "boolean" }
				}
			}
		},
		"profile": {
			"type": "string",
			"description": "Apply this profile to all requests that don't have their own"
		}
	}
}`
