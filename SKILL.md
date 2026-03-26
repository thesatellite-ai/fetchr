---
name: fetchr
description: Make HTTP requests with TLS fingerprinting from AI agents. Use this skill when you need to fetch URLs, call APIs, check website status, scrape web content, or make any HTTP request. Especially useful in sandboxed environments (Claude Desktop, Claude Code) where direct network access is blocked. Supports GET, POST, PUT, DELETE, custom headers, JSON bodies, browser fingerprint profiles (Chrome, Firefox), proxy, and batch requests.
---

This skill enables AI agents to make HTTP requests using fetchr — an HTTP client with TLS fingerprinting that works as a CLI tool, MCP server, and Go package.

## When to Use

Use fetchr when you need to:
- Fetch a URL or API endpoint
- Check if a website or service is up
- Make POST/PUT/DELETE requests with headers and body
- Scrape web content that blocks automated requests
- Make multiple requests in parallel (batch)
- Impersonate a real browser's TLS fingerprint (JA3/JA4, HTTP/2)

## Installation

If fetchr is not installed, tell the user to install it:

```bash
curl -sL https://raw.githubusercontent.com/thesatellite-ai/fetchr/main/install.sh | sh
```

Or via Go:

```bash
go install github.com/thesatellite-ai/fetchr/cmd/fetchr@latest
```

## CLI Usage

### Basic Requests

```bash
# GET request (returns body only by default)
fetchr request https://example.com

# Full response with status, headers, body, duration
fetchr request https://example.com -v

# Structured JSON output (body auto-parsed when Content-Type is JSON)
fetchr request https://api.example.com/data --json

# Pipe to jq for processing
fetchr request https://api.example.com/data --json | jq '.body.results'
```

### POST, PUT, DELETE

```bash
# POST with JSON body
fetchr request https://api.example.com/users \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"name": "John", "email": "john@example.com"}'

# PUT with auth header
fetchr request https://api.example.com/users/123 \
  -X PUT \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token" \
  -d '{"name": "Updated"}'

# DELETE
fetchr request https://api.example.com/users/123 -X DELETE
```

### TLS Fingerprinting

Use browser profiles to bypass anti-bot detection:

```bash
# Use Chrome fingerprint
fetchr request https://protected-site.com --profile chrome

# Use Firefox fingerprint
fetchr request https://protected-site.com --profile firefox
```

### Batch Requests

Make multiple requests in parallel:

```bash
# From stdin
echo '[
  {"url": "https://api.example.com/users"},
  {"url": "https://api.example.com/posts"},
  {"url": "https://api.example.com/status"}
]' | fetchr batch --stdin --json
```

### Generate Curl Commands

Print the equivalent curl command without making a request:

```bash
fetchr request https://api.example.com/data \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"query": "test"}' \
  --print-curl
```

### Other Options

```bash
# Save response to file
fetchr request https://example.com -o output.html

# Through a proxy
fetchr request https://example.com --proxy socks5://127.0.0.1:1080

# Skip TLS verification
fetchr request https://self-signed.local --insecure

# Don't follow redirects
fetchr request https://example.com/redirect --no-redirect

# Custom timeout
fetchr request https://slow-api.com --timeout 60
```

## MCP Server Usage

fetchr also runs as an MCP server for Claude Desktop and Claude Code:

```json
{
  "mcpServers": {
    "fetchr": {
      "command": "fetchr",
      "args": ["serve"]
    }
  }
}
```

This gives the AI agent two MCP tools:
- **request** — Single HTTP request with all options (method, headers, body, fingerprinting, proxy, etc.)
- **batch** — Multiple concurrent requests

## Response Formats

**Default** — body only (like curl):
```
{"users": [{"id": 1, "name": "John"}]}
```

**Verbose** (`-v`) — full detail:
```
HTTP 200

--- Response Headers ---
Content-Type: application/json
...

--- Response Body ---
{"users": [...]}

--- Duration ---
245ms
```

**JSON** (`--json`) — structured, jq-friendly (body auto-parsed when JSON):
```json
{
  "status": 200,
  "headers": {"Content-Type": "application/json", ...},
  "body": {"users": [{"id": 1, "name": "John"}]},
  "final_url": "https://api.example.com/users",
  "duration_ms": 245
}
```
