# fetchr

A fully-featured HTTP client with **TLS fingerprinting** (JA3/JA4, HTTP/2, QUIC) — available as a CLI tool, MCP server, and importable Go package.

## Why fetchr?

**AI assistants can't make HTTP requests.** Claude Desktop, Claude Code, and other sandboxed AI tools block outbound network calls — they can't `curl`, fetch APIs, or check if a website is up. fetchr solves this as an MCP server, giving AI assistants full HTTP capabilities.

**Websites block automated requests.** Standard HTTP clients send a default TLS fingerprint that anti-bot systems detect instantly. fetchr impersonates real browsers at the TLS level — matching their JA3 hash, HTTP/2 SETTINGS frame, header order, and more — so your requests look identical to Chrome or Firefox.

## Features

- **TLS Fingerprinting** — JA3, JA4, HTTP/2 SETTINGS, QUIC fingerprints
- **Browser Profiles** — Pre-configured Chrome and Firefox fingerprints, extensible via config
- **MCP Server** — stdio + SSE transport for Claude Desktop, Claude Code, and other AI assistants
- **CLI Tool** — Make requests, batch operations, and manage config from the terminal
- **Go Package** — Import `pkg/curl` in your own Go applications
- **Curl Export** — Print any request as an equivalent `curl` command
- **Request Logging** — File (JSONL), webhook, or both
- **JSONC Config** — JSON with comments, platform-aware auto-discovery, auto-created on first run

> **[Full Usage Guide](docs/USAGE.md)** — Complete reference for every command, flag, tool parameter, config option, TLS fingerprinting guide, and recipes.
>
> **[Contributing](docs/CONTRIBUTING.md)** — Development setup, architecture, and internals for contributors.

---

## Quick Start

### Install as AI Skill

Add fetchr to your AI coding agent (Claude Code, Cursor, Codex, Cline, etc.):

```bash
npx skills add thesatellite-ai/fetchr
```

### Install CLI Binary

**One-liner (macOS/Linux):**

```bash
curl -sL https://raw.githubusercontent.com/thesatellite-ai/fetchr/main/install.sh | sh
```

**Go install:**

```bash
go install github.com/thesatellite-ai/fetchr/cmd/fetchr@latest
```

**Manual download:**

```bash
# macOS (Apple Silicon)
curl -sL https://github.com/thesatellite-ai/fetchr/releases/latest/download/fetchr_darwin_arm64.tar.gz | tar xz
sudo mv fetchr /usr/local/bin/

# macOS (Intel)
curl -sL https://github.com/thesatellite-ai/fetchr/releases/latest/download/fetchr_darwin_amd64.tar.gz | tar xz
sudo mv fetchr /usr/local/bin/

# Linux (x86_64)
curl -sL https://github.com/thesatellite-ai/fetchr/releases/latest/download/fetchr_linux_amd64.tar.gz | tar xz
sudo mv fetchr /usr/local/bin/

# Linux (ARM64)
curl -sL https://github.com/thesatellite-ai/fetchr/releases/latest/download/fetchr_linux_arm64.tar.gz | tar xz
sudo mv fetchr /usr/local/bin/
```

**Windows (PowerShell):**

```powershell
Invoke-WebRequest -Uri https://github.com/thesatellite-ai/fetchr/releases/latest/download/fetchr_windows_amd64.zip -OutFile fetchr.zip
Expand-Archive fetchr.zip -DestinationPath .
Move-Item fetchr.exe C:\Windows\System32\
```

**Uninstall:**

```bash
sudo rm /usr/local/bin/fetchr
```

---

## CLI Usage

```
fetchr [command]

Commands:
  serve       Start the MCP server (stdio or SSE)
  request     Make a single HTTP request
  batch       Make multiple HTTP requests in parallel
  config      Manage configuration
  version     Print version and exit
  completion  Generate shell completion scripts
```

### Make a Request

```bash
# Simple GET (returns body only by default)
fetchr request https://httpbin.org/get

# POST with JSON
fetchr request https://httpbin.org/post \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"name": "John"}'

# Full response (status, headers, body, duration)
fetchr request https://httpbin.org/get -v

# Structured JSON output (body auto-parsed when Content-Type is JSON)
fetchr request https://httpbin.org/get --json
fetchr request https://httpbin.org/get --json | jq '.body.headers'

# Use Chrome fingerprint profile
fetchr request https://tls.peet.ws/api/all --profile chrome

# Print as curl command (does not execute the request)
fetchr request https://httpbin.org/post -X POST \
  -H "Content-Type: application/json" -d '{"key":"value"}' --print-curl

# Save response to file
fetchr request https://httpbin.org/get -o response.txt

# Through a proxy
fetchr request https://httpbin.org/ip --proxy socks5://127.0.0.1:1080

# Skip TLS verification / don't follow redirects / force HTTP/1.1
fetchr request https://self-signed.local/api --insecure
fetchr request https://httpbin.org/redirect/3 --no-redirect
fetchr request https://httpbin.org/get --http1
```

### Batch Requests

```bash
# From a JSON file
fetchr batch -f requests.json

# From stdin
echo '[{"url":"https://httpbin.org/get"},{"url":"https://httpbin.org/ip"}]' | fetchr batch --stdin

# Apply profile to all
fetchr batch -f requests.json --profile chrome

# JSON output
fetchr batch -f requests.json --json
```

Example `requests.json`:

```json
[
  {"url": "https://httpbin.org/get"},
  {
    "url": "https://httpbin.org/post",
    "method": "POST",
    "headers": {"Content-Type": "application/json"},
    "body": "{\"action\": \"create\"}"
  },
  {"url": "https://httpbin.org/headers", "headers": {"X-Custom": "hello"}}
]
```

### Configuration

```bash
# Show where config lives
fetchr config path

# Create default config
fetchr config init

# Show resolved config (defaults + profiles)
fetchr config show

# Validate config syntax
fetchr config validate

# Use a specific config file
fetchr request https://example.com --config ./my-config.jsonc
```

### Start MCP Server

```bash
# Stdio (default, for Claude Desktop / Claude Code)
fetchr serve

# SSE transport
fetchr serve --transport sse --port :3000
```

---

## MCP Integration

### Claude Code

Add to `~/.claude/settings.json` or project `.claude/settings.json`:

```json
{
  "mcpServers": {
    "curl": {
      "command": "fetchr",
      "args": ["serve"]
    }
  }
}
```

### Claude Desktop

Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "curl": {
      "command": "fetchr",
      "args": ["serve"]
    }
  }
}
```

### SSE Client

```bash
fetchr serve --transport sse --port :3000
# Connect any MCP-compatible client to http://localhost:3000
```

### MCP Inspector (Development)

```bash
# Stdio mode
task inspect

# Or manually
npx @modelcontextprotocol/inspector ./fetchr serve
```

---

## MCP Tools

### `request`

Make a single HTTP request with full TLS fingerprinting support.

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `url` | string | **yes** | — | The URL to request |
| `method` | string | no | `GET` | GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS |
| `headers` | object | no | — | HTTP headers as key-value pairs |
| `header_order` | array | no | — | Header order (critical for fingerprint accuracy) |
| `body` | string | no | — | Request body |
| `cookies` | array | no | — | Cookies as `[{name, value}]` |
| `ja3` | string | no | — | JA3 TLS fingerprint |
| `ja4r` | string | no | — | JA4 raw fingerprint |
| `http2_fingerprint` | string | no | — | HTTP/2 SETTINGS frame fingerprint |
| `quic_fingerprint` | string | no | — | QUIC fingerprint |
| `user_agent` | string | no | — | User-Agent header |
| `proxy` | string | no | — | Proxy URL (http, https, socks5) |
| `timeout` | integer | no | `30` | Timeout in seconds |
| `insecure` | boolean | no | `false` | Skip TLS certificate verification |
| `disable_redirect` | boolean | no | `false` | Don't follow redirects |
| `force_http1` | boolean | no | `false` | Force HTTP/1.1 |
| `force_http3` | boolean | no | `false` | Force HTTP/3 (QUIC) |
| `protocol` | string | no | — | `http1`, `http2`, or `http3` |
| `profile` | string | no | — | Named profile from config (`chrome`, `firefox`, etc.) |
| `print_curl` | boolean | no | `false` | Include equivalent curl command in output |

#### Example: Simple GET

```json
{
  "name": "request",
  "arguments": {
    "url": "https://httpbin.org/get"
  }
}
```

#### Example: POST with JSON

```json
{
  "name": "request",
  "arguments": {
    "url": "https://httpbin.org/post",
    "method": "POST",
    "headers": {
      "Content-Type": "application/json",
      "Authorization": "Bearer my-token"
    },
    "body": "{\"name\": \"John\", \"email\": \"john@example.com\"}"
  }
}
```

#### Example: Chrome Fingerprint

```json
{
  "name": "request",
  "arguments": {
    "url": "https://tls.peet.ws/api/all",
    "profile": "chrome"
  }
}
```

#### Example: Custom JA3 + HTTP/2 Fingerprint

```json
{
  "name": "request",
  "arguments": {
    "url": "https://tls.peet.ws/api/all",
    "ja3": "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0",
    "http2_fingerprint": "1:65536;2:0;4:6291456;6:262144|15663105|0|m,a,s,p",
    "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0"
  }
}
```

#### Example: Through a Proxy

```json
{
  "name": "request",
  "arguments": {
    "url": "https://httpbin.org/ip",
    "proxy": "socks5://127.0.0.1:1080"
  }
}
```

#### Example: Print as Curl

```json
{
  "name": "request",
  "arguments": {
    "url": "https://httpbin.org/post",
    "method": "POST",
    "headers": {"Content-Type": "application/json"},
    "body": "{\"key\": \"value\"}",
    "print_curl": true
  }
}
```

### `batch`

Make multiple HTTP requests concurrently. Returns a JSON array in the same order.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `requests` | array | **yes** | Array of request objects (same parameters as `request`) |
| `profile` | string | no | Apply this profile to all requests that don't have their own |

#### Example: Fetch Multiple Endpoints

```json
{
  "name": "batch",
  "arguments": {
    "requests": [
      {"url": "https://httpbin.org/get"},
      {"url": "https://httpbin.org/ip"},
      {"url": "https://httpbin.org/user-agent"}
    ]
  }
}
```

#### Example: Mixed Methods with Profile

```json
{
  "name": "batch",
  "arguments": {
    "profile": "chrome",
    "requests": [
      {"url": "https://httpbin.org/get"},
      {
        "url": "https://httpbin.org/post",
        "method": "POST",
        "headers": {"Content-Type": "application/json"},
        "body": "{\"action\": \"create\"}"
      }
    ]
  }
}
```

---

## Configuration

fetchr uses JSONC (JSON with comments) configuration files. On first run, a default config is auto-created.

### Config File Locations

Config is searched in order (first found wins):

| Platform | Search Order |
|---|---|
| **macOS** | `./fetchr.jsonc` > `~/.config/fetchr/config.jsonc` > `~/Library/Application Support/fetchr/config.jsonc` |
| **Linux** | `./fetchr.jsonc` > `$XDG_CONFIG_HOME/fetchr/config.jsonc` > `~/.config/fetchr/config.jsonc` |
| **Windows** | `.\fetchr.jsonc` > `%APPDATA%\fetchr\config.jsonc` > `%USERPROFILE%\.config\fetchr\config.jsonc` |

Use `--config path/to/config.jsonc` on any command to override.

### Example Config

```jsonc
{
  // Default options applied to every request
  "defaults": {
    "ja3": "771,4865-4867-4866-49195-49199-52393-52392-49196-49200-49162-49161-49171-49172-51-57-47-53-10,0-23-65281-10-11-35-16-5-51-43-13-45-28-21,29-23-24-25-256-257,0",
    "user_agent": "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:87.0) Gecko/20100101 Firefox/87.0",
    "timeout": 30
  },

  // Named browser profiles
  "profiles": {
    "chrome": {
      "ja3": "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0",
      "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0",
      "http2_fingerprint": "1:65536;2:0;4:6291456;6:262144|15663105|0|m,a,s,p"
    },
    "firefox": {
      "ja3": "771,4865-4867-4866-49195-49199-52393-52392-49196-49200-49162-49161-49171-49172-51-57-47-53-10,0-23-65281-10-11-35-16-5-51-43-13-45-28-21,29-23-24-25-256-257,0",
      "user_agent": "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:87.0) Gecko/20100101 Firefox/87.0",
      "http2_fingerprint": "1:65536;2:0;4:131072;5:16384|12517377|0|m,p,a,s"
    }
  },

  // Request logging
  "logging": {
    "enabled": true,
    "file": "~/.fetchr/requests.jsonl",
    "webhook": "https://your-webhook.example.com/log"
  }
}
```

### Profiles

Profiles are named presets that bundle TLS fingerprint + User-Agent + HTTP/2 settings to impersonate a specific browser. Use them via `--profile` (CLI) or `"profile"` (MCP tool input).

The default config ships with `chrome` and `firefox`. Add your own:

```jsonc
{
  "profiles": {
    "safari": {
      "ja3": "771,4865-4866-4867-49196-49195-52393-49200-49199-52392-49162-49161-49172-49171-157-156-53-47-49160-49170-10,0-23-65281-10-11-16-5-13-18-51-45-43-27-21,29-23-24-25,0",
      "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/605.1.15 Safari/605.1.15",
      "http2_fingerprint": "1:65536;3:100;4:65535|1048576|0|m,s,p,a"
    },
    "api": {
      "user_agent": "MyApp/1.0",
      "timeout": 60
    }
  }
}
```

### Logging

When logging is enabled, every request/response pair is recorded.

**File logging** writes JSONL (one JSON object per line) to the configured path:

```json
{"timestamp":"2026-03-26T10:00:00Z","request":{"url":"https://example.com","method":"GET"},"response":{"status":200},"duration":245000000}
```

**Webhook logging** POSTs the same JSON to a URL (fire-and-forget, non-blocking).

---

## Go Package

Import `pkg/curl` to use fetchr as a library:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/thesatellite-ai/fetchr/pkg/curl"
)

func main() {
    client := curl.New(
        curl.WithDefaults(curl.RequestOptions{
            UserAgent: "Mozilla/5.0 Chrome/120.0.0.0",
            Timeout:   30,
        }),
    )
    defer client.Close()

    // Simple GET
    resp, err := client.Do(context.Background(), curl.RequestOptions{
        URL: "https://httpbin.org/get",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(resp.Format())

    // POST with JA3 fingerprint
    resp, err = client.Do(context.Background(), curl.RequestOptions{
        URL:    "https://httpbin.org/post",
        Method: "POST",
        Headers: map[string]string{
            "Content-Type": "application/json",
        },
        Body: `{"key": "value"}`,
        Ja3:  "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(resp.Format())

    // Batch requests
    results, _ := client.Batch(context.Background(), []curl.RequestOptions{
        {URL: "https://httpbin.org/get"},
        {URL: "https://httpbin.org/ip"},
        {URL: "https://httpbin.org/user-agent"},
    })
    for _, r := range results {
        if r.Error != "" {
            fmt.Printf("[%d] %s ERROR: %s\n", r.Index, r.URL, r.Error)
        } else {
            fmt.Printf("[%d] %s -> %d\n", r.Index, r.URL, r.Response.Status)
        }
    }

    // Print as curl command
    cmd := curl.ToCurl(curl.RequestOptions{
        URL:    "https://httpbin.org/post",
        Method: "POST",
        Headers: map[string]string{
            "Content-Type":  "application/json",
            "Authorization": "Bearer token",
        },
        Body:    `{"key": "value"}`,
        Timeout: 30,
    })
    fmt.Println(cmd)
    // Output:
    // curl \
    //   -X POST \
    //   'https://httpbin.org/post' \
    //   -H 'Authorization: Bearer token' \
    //   -H 'Content-Type: application/json' \
    //   -d '{"key": "value"}' \
    //   --connect-timeout 30
}
```

---

## License

Source Available License — free to use, fork, and contribute. See [LICENSE](LICENSE) for details.
