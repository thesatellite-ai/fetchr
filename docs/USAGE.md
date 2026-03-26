# fetchr Usage Guide

> For installation, quick start, and MCP integration setup, see the [README](../README.md).

Complete reference for every command, flag, tool parameter, and configuration option.

---

## Table of Contents

- [CLI Commands](#cli-commands)
  - [serve](#serve)
  - [request](#request)
  - [batch](#batch)
  - [config](#config)
  - [version](#version)
- [MCP Tools](#mcp-tools)
  - [request tool](#request-tool)
  - [batch tool](#batch-tool)
- [MCP Prompts](#mcp-prompts)
- [Configuration Reference](#configuration-reference)
  - [Config File Format](#config-file-format)
  - [defaults](#defaults)
  - [profiles](#profiles)
  - [logging](#logging)
- [TLS Fingerprinting Guide](#tls-fingerprinting-guide)
  - [JA3](#ja3)
  - [JA4](#ja4)
  - [HTTP/2 Fingerprint](#http2-fingerprint)
  - [Header Order](#header-order)
- [Go Package API](#go-package-api)
- [Recipes](#recipes)

---

## CLI Commands

### serve

Start the MCP server for AI assistant integration.

```
fetchr serve [flags]
```

| Flag | Type | Default | Description |
|---|---|---|---|
| `--transport` | string | `stdio` | Transport mode: `stdio` or `sse` |
| `--port` | string | `:8080` | Port to listen on (SSE mode only) |
| `--config` | string | (auto) | Path to config.jsonc |

**Stdio mode** (for Claude Desktop, Claude Code, and other MCP clients that launch the server as a subprocess):

```bash
fetchr serve
```

**SSE mode** (for web-based MCP clients or when the server runs as a standalone process):

```bash
fetchr serve --transport sse --port :3000
```

The SSE server includes CORS headers for cross-origin browser access.

---

### request

Make a single HTTP request from the command line.

```
fetchr request <url> [flags]
```

| Flag | Short | Type | Default | Description |
|---|---|---|---|---|
| `--method` | `-X` | string | `GET` | HTTP method |
| `--header` | `-H` | string[] | — | Headers (repeatable: `-H "Key: Value"`) |
| `--data` | `-d` | string | — | Request body |
| `--ja3` | | string | — | JA3 TLS fingerprint string |
| `--ja4r` | | string | — | JA4 raw fingerprint string |
| `--h2fp` | | string | — | HTTP/2 fingerprint |
| `--user-agent` | | string | — | User-Agent header value |
| `--proxy` | | string | — | Proxy URL (http, https, socks5) |
| `--timeout` | | int | `30` | Timeout in seconds |
| `--insecure` | | bool | `false` | Skip TLS certificate verification |
| `--no-redirect` | | bool | `false` | Disable following redirects |
| `--http1` | | bool | `false` | Force HTTP/1.1 |
| `--http3` | | bool | `false` | Force HTTP/3 (QUIC) |
| `--profile` | | string | — | Use named profile from config |
| `--print-curl` | | bool | `false` | Print equivalent curl command |
| `--output` | `-o` | string | — | Write response body to file |
| `--json` | | bool | `false` | Output response as JSON |
| `--config` | | string | (auto) | Path to config.jsonc |

**Examples:**

```bash
# Simple GET
fetchr request https://httpbin.org/get

# POST with headers and body
fetchr request https://httpbin.org/post \
  -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer my-token" \
  -d '{"name": "John", "email": "john@example.com"}'

# Use a browser profile
fetchr request https://tls.peet.ws/api/all --profile chrome

# Custom JA3 fingerprint
fetchr request https://tls.peet.ws/api/all \
  --ja3 "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0"

# Through SOCKS5 proxy
fetchr request https://httpbin.org/ip --proxy socks5://127.0.0.1:1080

# Save response body to file
fetchr request https://example.com/data.json -o data.json

# Get JSON output (for scripting)
fetchr request https://httpbin.org/get --json

# Print curl equivalent and make the request
fetchr request https://httpbin.org/post \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"key": "value"}' \
  --print-curl

# Force HTTP/1.1
fetchr request https://httpbin.org/get --http1

# Skip TLS verification (self-signed certs)
fetchr request https://self-signed.local/api --insecure

# Don't follow redirects
fetchr request https://httpbin.org/redirect/3 --no-redirect
```

**Response format (default):**

```
HTTP 200

--- Response Headers ---
Content-Type: application/json
Date: Wed, 26 Mar 2026 10:00:00 GMT
Content-Length: 256

--- Response Body ---
{
  "origin": "1.2.3.4",
  "url": "https://httpbin.org/get"
}

--- Duration ---
245ms
```

**Response format (`--json`):**

```json
{
  "status": 200,
  "headers": {
    "Content-Type": "application/json",
    "Date": "Wed, 26 Mar 2026 10:00:00 GMT"
  },
  "body": "{\"origin\": \"1.2.3.4\"}",
  "final_url": "https://httpbin.org/get",
  "duration_ms": 245000000
}
```

---

### batch

Execute multiple HTTP requests in parallel.

```
fetchr batch [flags]
```

| Flag | Short | Type | Default | Description |
|---|---|---|---|---|
| `--file` | `-f` | string | — | JSON file with array of request objects |
| `--stdin` | | bool | `false` | Read request array from stdin |
| `--profile` | | string | — | Apply named profile to all requests |
| `--print-curl` | | bool | `false` | Print curl commands for all requests |
| `--json` | | bool | `false` | Output as JSON array |
| `--config` | | string | (auto) | Path to config.jsonc |

You must provide either `--file` or `--stdin`.

**From a file:**

```bash
fetchr batch -f requests.json
```

**From stdin (pipe):**

```bash
cat requests.json | fetchr batch --stdin

# Or inline
echo '[
  {"url": "https://httpbin.org/get"},
  {"url": "https://httpbin.org/ip"}
]' | fetchr batch --stdin --json
```

**Apply a profile to all requests:**

```bash
fetchr batch -f requests.json --profile chrome
```

**Request file format** — JSON array of request objects:

```json
[
  {
    "url": "https://httpbin.org/get"
  },
  {
    "url": "https://httpbin.org/post",
    "method": "POST",
    "headers": {"Content-Type": "application/json"},
    "body": "{\"action\": \"create\"}"
  },
  {
    "url": "https://httpbin.org/delay/2",
    "timeout": 5
  }
]
```

Each object supports the same fields as `RequestOptions`: `url`, `method`, `headers`, `header_order`, `body`, `cookies`, `ja3`, `ja4r`, `http2_fingerprint`, `user_agent`, `proxy`, `timeout`, `insecure`, `disable_redirect`, `force_http1`, `force_http3`, `profile`, `print_curl`.

---

### config

Manage configuration files.

#### config init

Create the default `config.jsonc` at the platform default path.

```bash
fetchr config init           # Fails if config already exists
fetchr config init --force   # Overwrites existing config
```

#### config path

Print the resolved config file path.

```bash
fetchr config path
# /home/user/.config/fetchr/config.jsonc
```

#### config show

Print the fully resolved config (parsed and merged).

```bash
fetchr config show
```

#### config validate

Check a config file for syntax errors.

```bash
fetchr config validate                         # Validate auto-discovered config
fetchr config validate ./custom-config.jsonc   # Validate a specific file
```

---

### version

Print version, commit, and build date.

```bash
fetchr version
# fetchr v1.2.0 (commit: abc1234, built: 2026-03-26T10:00:00Z)
```

---

## MCP Tools

These tools are available when fetchr runs as an MCP server (`fetchr serve`).

### request tool

Make a single HTTP request with TLS fingerprinting.

**Input schema:**

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `url` | string | **yes** | — | The URL to request |
| `method` | string | no | `GET` | HTTP method |
| `headers` | object | no | — | HTTP headers as key-value pairs |
| `header_order` | string[] | no | — | Header order for fingerprint accuracy |
| `body` | string | no | — | Request body |
| `cookies` | array | no | — | `[{name: string, value: string}]` |
| `ja3` | string | no | — | JA3 TLS fingerprint |
| `ja4r` | string | no | — | JA4 raw fingerprint |
| `http2_fingerprint` | string | no | — | HTTP/2 SETTINGS fingerprint |
| `quic_fingerprint` | string | no | — | QUIC fingerprint |
| `user_agent` | string | no | — | User-Agent string |
| `proxy` | string | no | — | Proxy URL |
| `timeout` | integer | no | `30` | Timeout in seconds |
| `insecure` | boolean | no | `false` | Skip TLS verification |
| `disable_redirect` | boolean | no | `false` | Don't follow redirects |
| `force_http1` | boolean | no | `false` | Force HTTP/1.1 |
| `force_http3` | boolean | no | `false` | Force HTTP/3 |
| `protocol` | string | no | — | `http1`, `http2`, `http3` |
| `profile` | string | no | — | Named profile from config |
| `print_curl` | boolean | no | `false` | Include curl command in output |

**MCP call examples:**

Simple GET:
```json
{"name": "request", "arguments": {"url": "https://httpbin.org/get"}}
```

POST with auth:
```json
{
  "name": "request",
  "arguments": {
    "url": "https://api.example.com/data",
    "method": "POST",
    "headers": {
      "Content-Type": "application/json",
      "Authorization": "Bearer token123"
    },
    "body": "{\"key\": \"value\"}"
  }
}
```

Chrome profile with cookies:
```json
{
  "name": "request",
  "arguments": {
    "url": "https://example.com/dashboard",
    "profile": "chrome",
    "cookies": [
      {"name": "session", "value": "abc123"},
      {"name": "csrf", "value": "xyz789"}
    ]
  }
}
```

### batch tool

Execute multiple requests concurrently.

**Input schema:**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `requests` | array | **yes** | Array of request objects (same params as `request` tool) |
| `profile` | string | no | Apply this profile to all requests without their own |

**Response** is a JSON array:

```json
[
  {
    "index": 0,
    "url": "https://httpbin.org/get",
    "response": {
      "status": 200,
      "headers": {"Content-Type": "application/json"},
      "body": "...",
      "final_url": "https://httpbin.org/get",
      "duration_ms": 150000000
    }
  },
  {
    "index": 1,
    "url": "https://bad.example.com",
    "error": "request failed: dial tcp: no such host"
  }
]
```

---

## MCP Prompts

The server exposes these prompts (browsable in MCP Inspector):

| Prompt | Description |
|---|---|
| `usage-guide` | Complete reference for all tools and parameters |
| `example-get` | Simple GET request |
| `example-post-json` | POST with JSON body and auth headers |
| `example-fingerprint` | Request with custom JA3 + User-Agent |
| `example-proxy` | Request through a proxy |
| `example-batch` | Batch multiple endpoints |
| `example-batch-mixed` | Batch with mixed methods |
| `example-profiles` | Using named profiles from config |
| `example-print-curl` | Print request as curl command |

---

## Configuration Reference

### Config File Format

JSONC — standard JSON with `//` line comments and `/* */` block comments.

```jsonc
{
  // This is a line comment
  "key": "value" /* This is an inline comment */
}
```

### defaults

Default options applied to **every** request. Any per-request or per-profile option overrides these.

```jsonc
{
  "defaults": {
    "ja3": "...",           // Default JA3 fingerprint
    "user_agent": "...",    // Default User-Agent
    "timeout": 30,          // Default timeout (seconds)
    "proxy": "...",         // Default proxy
    "insecure": false,      // Default TLS verification
    "disable_redirect": false
  }
}
```

### profiles

Named presets. Each profile is a partial `RequestOptions` — fields set in the profile override `defaults`.

```jsonc
{
  "profiles": {
    "chrome": {
      "ja3": "771,4865-4866-4867-...",
      "user_agent": "Mozilla/5.0 ... Chrome/120.0.0.0",
      "http2_fingerprint": "1:65536;2:0;4:6291456;6:262144|15663105|0|m,a,s,p"
    },
    "firefox": {
      "ja3": "771,4865-4867-4866-...",
      "user_agent": "Mozilla/5.0 ... Firefox/87.0",
      "http2_fingerprint": "1:65536;2:0;4:131072;5:16384|12517377|0|m,p,a,s"
    },
    "api-client": {
      "user_agent": "MyApp/1.0",
      "timeout": 60,
      "disable_redirect": true
    }
  }
}
```

**Resolution order:** `defaults` -> `profile` -> `per-request options`

Each layer overrides the previous. Only non-zero/non-empty values override.

### logging

```jsonc
{
  "logging": {
    "enabled": true,        // Master switch
    "file": "~/.fetchr/requests.jsonl",  // JSONL file path (~ expanded)
    "webhook": "https://hooks.example.com/log"  // POST log entries here
  }
}
```

When both `file` and `webhook` are set, logs are sent to both.

**JSONL log entry format:**

```json
{
  "timestamp": "2026-03-26T10:00:00.000Z",
  "request": {
    "url": "https://example.com",
    "method": "GET",
    "ja3": "771,...",
    "user_agent": "Mozilla/5.0..."
  },
  "response": {
    "status": 200,
    "headers": {"Content-Type": "text/html"},
    "body": "...",
    "duration_ms": 245000000
  },
  "duration": 245000000,
  "curl_cmd": "curl 'https://example.com' ..."
}
```

---

## TLS Fingerprinting Guide

### JA3

JA3 is a method for creating SSL/TLS client fingerprints. It hashes five fields from the Client Hello message:

```
TLSVersion,Ciphers,Extensions,EllipticCurves,EllipticCurvePointFormats
```

Example (Chrome 120):
```
771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0
```

Each section is comma-separated, values within sections are dash-separated.

- `771` = TLS 1.2
- `4865-4866-4867-...` = cipher suites
- `0-23-65281-...` = extensions
- `29-23-24` = supported groups (elliptic curves)
- `0` = EC point formats

### JA4

JA4 is the next generation of JA3. The raw format (`ja4r`) provides more granular fingerprinting including explicit cipher and extension values.

```bash
fetchr request https://tls.peet.ws/api/all --ja4r "your-ja4r-string"
```

### HTTP/2 Fingerprint

The HTTP/2 fingerprint captures the SETTINGS frame and priority information that browsers send when establishing an HTTP/2 connection.

Format: `setting1:value1;setting2:value2|window_update|priority|pseudo_header_order`

Example (Chrome):
```
1:65536;2:0;4:6291456;6:262144|15663105|0|m,a,s,p
```

- `1:65536` = HEADER_TABLE_SIZE
- `2:0` = ENABLE_PUSH
- `4:6291456` = INITIAL_WINDOW_SIZE
- `6:262144` = MAX_HEADER_LIST_SIZE
- `15663105` = WINDOW_UPDATE value
- `0` = PRIORITY weight
- `m,a,s,p` = pseudo header order (:method, :authority, :scheme, :path)

### Header Order

Many anti-bot systems check the order of HTTP headers. CycleTLS supports `header_order` to match browser behavior exactly:

```json
{
  "headers": {
    "Accept": "text/html",
    "Accept-Language": "en-US",
    "Cache-Control": "no-cache",
    "User-Agent": "Mozilla/5.0..."
  },
  "header_order": [
    "User-Agent",
    "Accept",
    "Accept-Language",
    "Cache-Control"
  ]
}
```

---

## Go Package API

### Client

```go
// Create a client with defaults
client := curl.New(
    curl.WithDefaults(curl.RequestOptions{...}),
    curl.WithLogger(curl.NewFileLogger("/var/log/requests.jsonl")),
)
defer client.Close()
```

### Single Request

```go
resp, err := client.Do(ctx, curl.RequestOptions{
    URL:    "https://example.com",
    Method: "GET",
})
// resp.Status, resp.Headers, resp.Body, resp.FinalURL, resp.Duration
```

### Batch Requests

```go
results, err := client.Batch(ctx, []curl.RequestOptions{
    {URL: "https://example.com/a"},
    {URL: "https://example.com/b"},
    {URL: "https://example.com/c"},
})
// results[i].Index, results[i].URL, results[i].Response, results[i].Error
```

### Curl Export

```go
cmd := curl.ToCurl(curl.RequestOptions{
    URL:    "https://example.com",
    Method: "POST",
    Headers: map[string]string{"Content-Type": "application/json"},
    Body:   `{"key": "value"}`,
})
fmt.Println(cmd)
```

### Option Merging

```go
base := curl.RequestOptions{UserAgent: "Default", Timeout: 30}
override := curl.RequestOptions{URL: "https://example.com", Timeout: 60}
merged := curl.Merge(base, override)
// merged.UserAgent = "Default", merged.Timeout = 60, merged.URL = "https://example.com"
```

### Logging

```go
// File logger (JSONL)
fileLogger := curl.NewFileLogger("/var/log/requests.jsonl")

// Webhook logger (fire-and-forget POST)
webhookLogger := curl.NewWebhookLogger("https://hooks.example.com/log")

// Combine multiple loggers
multiLogger := curl.NewMultiLogger(fileLogger, webhookLogger)

client := curl.New(curl.WithLogger(multiLogger))
```

---

## Recipes

### Verify your TLS fingerprint

```bash
# Check what fingerprint the server sees
fetchr request https://tls.peet.ws/api/all --profile chrome --json | jq .body
```

### Compare browser fingerprints

```bash
# Chrome vs Firefox
echo '[
  {"url": "https://tls.peet.ws/api/all", "profile": "chrome"},
  {"url": "https://tls.peet.ws/api/all", "profile": "firefox"}
]' | fetchr batch --stdin --json
```

### Export requests as curl commands

```bash
fetchr request https://api.example.com/data \
  -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token" \
  -d '{"query": "test"}' \
  --print-curl
```

Output includes:
```
curl \
  -X POST \
  'https://api.example.com/data' \
  -H 'Authorization: Bearer token' \
  -H 'Content-Type: application/json' \
  -d '{"query": "test"}' \
  --connect-timeout 30

HTTP 200
--- Response Headers ---
...
```

### Scrape behind Cloudflare

```bash
fetchr request https://protected-site.com \
  --profile chrome \
  --ja3 "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0" \
  --h2fp "1:65536;2:0;4:6291456;6:262144|15663105|0|m,a,s,p"
```

### Health check multiple services

Create `healthcheck.json`:
```json
[
  {"url": "https://api.example.com/health", "timeout": 5},
  {"url": "https://web.example.com/health", "timeout": 5},
  {"url": "https://cdn.example.com/health", "timeout": 5}
]
```

```bash
fetchr batch -f healthcheck.json --json | jq '.[].response.status'
```

### Pipe into jq for processing

```bash
fetchr request https://api.github.com/repos/golang/go --json | jq -r '.body | fromjson | .full_name'
```

### Use in shell scripts

```bash
#!/bin/bash
RESPONSE=$(fetchr request https://api.example.com/token \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"client_id": "xxx", "client_secret": "yyy"}' \
  --json)

TOKEN=$(echo "$RESPONSE" | jq -r '.body | fromjson | .access_token')

fetchr request https://api.example.com/data \
  -H "Authorization: Bearer $TOKEN" \
  --profile chrome
```
