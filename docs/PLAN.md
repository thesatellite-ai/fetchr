# curl-mcp — Revision Plan

## Vision

A fully-featured HTTP client built on CycleTLS (TLS fingerprinting, JA3/JA4, HTTP/2 fingerprinting) exposed as:

1. **Go package** (`pkg/curl`) — importable in other Go apps
2. **CLI tool** (`cmd/curl-mcp`) — standalone binary with Cobra commands
3. **MCP server** — stdio + SSE transport for AI assistants

Every HTTP option is configurable. Every request can be printed as a curl command, logged to file/OpenTelemetry/webhook, and configured via JSONC config files with platform-aware loading.

---

## Project Structure

```
curl-mcp/
├── cmd/
│   └── curl-mcp/
│       └── main.go              # Cobra root + subcommands
├── pkg/
│   └── curl/
│       ├── client.go            # CycleTLS wrapper, main Client struct
│       ├── options.go           # RequestOptions, all configurable fields
│       ├── response.go          # Response struct, formatting
│       ├── curlcmd.go           # "Print as curl" command builder
│       └── logger.go            # Request logging (file, OTEL, webhook)
├── internal/
│   ├── config/
│   │   ├── config.go            # JSONC config struct + loader
│   │   └── paths.go             # Platform-aware config file paths
│   ├── mcp/
│   │   ├── server.go            # MCP server setup, tool + prompt registration
│   │   └── tools.go             # Tool handlers (request, batch)
│   └── version/
│       └── version.go           # Build-time version vars
├── docs/
│   ├── PLAN.md                  # This file
│   ├── TECHNICAL.md             # Architecture decisions
│   └── USAGE.md                 # User-facing docs
├── .github/
│   └── workflows/
│       └── release.yml          # GoReleaser cross-compile + publish
├── .goreleaser.yml              # Build matrix (linux/darwin/windows x amd64/arm64)
├── Taskfile.yml                 # Dev tasks (build, run, inspect, lint)
├── go.mod
├── go.sum
├── install.sh                   # One-click installer script
├── CLAUDE.md                    # AI context
└── README.md                    # Installation + quick start
```

---

## Phase 1: Core Package (`pkg/curl`)

### 1.1 — Client (`client.go`)

Wraps CycleTLS with a high-level API.

```go
type Client struct {
    cycleClient cycletls.CycleTLS
    defaults    RequestOptions   // merged from config file
    logger      Logger           // optional request logger
}

func New(opts ...ClientOption) *Client
func (c *Client) Do(ctx context.Context, opts RequestOptions) (*Response, error)
func (c *Client) Batch(ctx context.Context, requests []RequestOptions) ([]*BatchResult, error)
func (c *Client) Close()
```

**ClientOption** functional options:
- `WithDefaults(RequestOptions)` — default values for all requests
- `WithLogger(Logger)` — attach a logger
- `WithConfig(Config)` — load from parsed JSONC config

### 1.2 — Request Options (`options.go`)

Every CycleTLS option exposed and more.

```go
type RequestOptions struct {
    // Core
    URL    string            `json:"url" jsonschema:"required,description=The URL to request"`
    Method string            `json:"method" jsonschema:"description=HTTP method. Default: GET"`

    // Headers & Body
    Headers     map[string]string `json:"headers"`
    HeaderOrder []string          `json:"header_order"`
    Body        string            `json:"body"`
    Cookies     []Cookie          `json:"cookies"`

    // TLS Fingerprinting
    Ja3              string `json:"ja3"`
    Ja4r             string `json:"ja4r"`
    HTTP2Fingerprint string `json:"http2_fingerprint"`
    QUICFingerprint  string `json:"quic_fingerprint"`
    UserAgent        string `json:"user_agent"`

    // Connection
    Proxy              string `json:"proxy"`
    Timeout            int    `json:"timeout"`
    ServerName         string `json:"server_name"`
    InsecureSkipVerify bool   `json:"insecure"`

    // Protocol
    ForceHTTP1 bool   `json:"force_http1"`
    ForceHTTP3 bool   `json:"force_http3"`
    Protocol   string `json:"protocol"` // http1, http2, http3

    // Behavior
    DisableRedirect       bool `json:"disable_redirect"`
    DisableGrease         bool `json:"disable_grease"`
    TLS13AutoRetry        bool `json:"tls13_auto_retry"`
    EnableConnectionReuse bool `json:"enable_connection_reuse"`

    // Logging override per-request
    PrintCurl bool `json:"print_curl"` // print equivalent curl command
}
```

### 1.3 — Response (`response.go`)

```go
type Response struct {
    Status    int               `json:"status"`
    Headers   map[string]string `json:"headers"`
    Body      string            `json:"body"`
    BodyBytes []byte            `json:"-"`
    Cookies   []*http.Cookie    `json:"cookies,omitempty"`
    FinalURL  string            `json:"final_url"`
    Duration  time.Duration     `json:"duration_ms"`
}

func (r *Response) Format() string          // Human-readable output
func (r *Response) JSON() ([]byte, error)   // JSON serialization
```

### 1.4 — Curl Command Builder (`curlcmd.go`)

Convert any `RequestOptions` to an equivalent curl command string.

```go
func ToCurl(opts RequestOptions) string
```

Output example:
```
curl -X POST 'https://api.example.com/data' \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer token' \
  -d '{"key": "value"}' \
  --connect-timeout 30
```

### 1.5 — Logger (`logger.go`)

```go
type Logger interface {
    Log(ctx context.Context, entry LogEntry) error
}

type LogEntry struct {
    Timestamp time.Time       `json:"timestamp"`
    Request   RequestOptions  `json:"request"`
    Response  *Response       `json:"response,omitempty"`
    Error     string          `json:"error,omitempty"`
    Duration  time.Duration   `json:"duration"`
    CurlCmd   string          `json:"curl_cmd,omitempty"`
}

// Implementations
type FileLogger struct { ... }             // Append JSONL to file
type OTELLogger struct { ... }             // OpenTelemetry spans + attributes
type WebhookLogger struct { ... }          // POST LogEntry to a URL
type MultiLogger struct { loggers []Logger } // Fan-out to multiple
```

---

## Phase 2: Config System (`internal/config`)

### 2.1 — JSONC Config File (`config.go`)

Uses JSONC (JSON with comments) format. Parsed by stripping comments then unmarshaling.

```go
type Config struct {
    // Default request options applied to every request
    Defaults RequestOptions `json:"defaults"`

    // Logging
    Logging LoggingConfig `json:"logging"`

    // Named profiles (e.g., "chrome", "firefox", "api")
    Profiles map[string]RequestOptions `json:"profiles"`
}

type LoggingConfig struct {
    Enabled bool   `json:"enabled"`
    File    string `json:"file"`     // path to log file (JSONL)
    Webhook string `json:"webhook"`  // URL to POST log entries
    OTEL    *OTELConfig `json:"otel"`
}

type OTELConfig struct {
    Endpoint string `json:"endpoint"` // OTEL collector endpoint
    Service  string `json:"service"`  // service name
}
```

Example `config.jsonc`:
```jsonc
{
  // Default options for all requests
  "defaults": {
    "ja3": "771,4865-4867-4866-49195-49199-52393-52392-49196-49200-49162-49161-49171-49172-51-57-47-53-10,0-23-65281-10-11-35-16-5-51-43-13-45-28-21,29-23-24-25-256-257,0",
    "user_agent": "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:87.0) Gecko/20100101 Firefox/87.0",
    "timeout": 30
  },

  // Named profiles
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

  // Logging
  "logging": {
    "enabled": true,
    "file": "~/.curl-mcp/requests.jsonl"
  }
}
```

### 2.2 — Platform Config Paths (`paths.go`)

Config file search order (first found wins):

| Platform | Paths (in order) |
|---|---|
| **macOS** | `./curl-mcp.jsonc`, `~/.config/curl-mcp/config.jsonc`, `~/Library/Application Support/curl-mcp/config.jsonc` |
| **Linux** | `./curl-mcp.jsonc`, `$XDG_CONFIG_HOME/curl-mcp/config.jsonc`, `~/.config/curl-mcp/config.jsonc` |
| **Windows** | `.\curl-mcp.jsonc`, `%APPDATA%\curl-mcp\config.jsonc`, `%USERPROFILE%\.config\curl-mcp\config.jsonc` |

```go
func FindConfigFile() (string, error)           // Auto-discover
func LoadConfig(path string) (*Config, error)    // Load specific file
func LoadConfigAuto() (*Config, error)           // Find + load + auto-create
func DefaultConfigPath() string                  // Platform default path
func EnsureDefaultConfig() (string, error)       // Create if missing
func DefaultConfigContent() []byte               // Embedded default config.jsonc
```

**Auto-create behavior:** `LoadConfigAuto()` searches the platform paths. If no config file is found anywhere, it automatically creates the default `config.jsonc` at the platform default path (`DefaultConfigPath()`) with sensible defaults, comments explaining each option, and the Firefox + Chrome profiles pre-filled. This ensures first-run works seamlessly — users always have a config to edit.

The default config is embedded in the binary via `//go:embed` so it's always available.

The `--config` flag on any command overrides auto-discovery.

---

## Phase 3: CLI (`cmd/curl-mcp`)

### Commands

```
curl-mcp [command]

Commands:
  serve       Start the MCP server (stdio or SSE)
  request     Make a single HTTP request
  batch       Make multiple HTTP requests in parallel
  config      Manage configuration
  version     Print version and exit
  completion  Generate shell completion scripts
```

### `serve`

```
curl-mcp serve [flags]

Flags:
  --transport string   Transport mode: stdio or sse (default "stdio")
  --port string        Port to listen on, SSE mode only (default ":8080")
  --config string      Path to config.jsonc (overrides auto-discovery)
```

### `request`

```
curl-mcp request <url> [flags]

Flags:
  -X, --method string        HTTP method (default "GET")
  -H, --header stringArray   Headers (repeatable: -H "Key: Value")
  -d, --data string          Request body
      --ja3 string           JA3 fingerprint
      --ja4r string          JA4 raw fingerprint
      --h2fp string          HTTP/2 fingerprint
      --user-agent string    User-Agent string
      --proxy string         Proxy URL
      --timeout int          Timeout in seconds (default 30)
      --insecure             Skip TLS verification
      --no-redirect          Disable following redirects
      --http1                Force HTTP/1.1
      --http3                Force HTTP/3
      --profile string       Use named profile from config
      --print-curl           Print equivalent curl command
      --config string        Path to config.jsonc
  -o, --output string        Write response body to file
      --json                 Output response as JSON
```

### `batch`

```
curl-mcp batch [flags]

Flags:
  -f, --file string       JSON file with array of request objects
      --stdin             Read request array from stdin
      --profile string    Apply named profile to all requests
      --print-curl        Print curl commands for all requests
      --config string     Path to config.jsonc
      --json              Output as JSON array
```

### `config`

```
curl-mcp config [subcommand]

Subcommands:
  init        Re-create default config.jsonc (overwrites existing with --force)
  path        Print config file path (resolved or default)
  show        Print resolved config (merged defaults + profile)
  validate    Validate a config.jsonc file
```

Note: `config init` is rarely needed since `LoadConfigAuto()` auto-creates the default config on first run. Use it to reset config or re-create after deletion.

---

## Phase 4: MCP Server (`internal/mcp`)

### Tools

#### `request`
Single HTTP request. Same options as `RequestOptions` struct.

Input schema exposes all CycleTLS options: url, method, headers, header_order, body, cookies, ja3, ja4r, http2_fingerprint, user_agent, proxy, timeout, insecure, disable_redirect, force_http1, force_http3, protocol, profile, print_curl.

#### `batch`
Multiple concurrent requests.

Input: `{ "requests": [...], "profile": "chrome" }`
Output: JSON array with index, url, response/error, curl_cmd (if print_curl).

### Prompts

| Name | Description |
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

## Phase 5: Logging & Observability

### File Logger
- Appends JSONL (one JSON object per line) to a file
- Each entry: timestamp, request options, response summary, duration, curl command
- File path from config or `--log-file` flag

### OpenTelemetry Logger
- Creates spans per request with attributes: method, url, status, duration
- Configurable OTEL collector endpoint
- Service name from config

### Webhook Logger
- POSTs `LogEntry` JSON to a configured URL
- Fire-and-forget (non-blocking, errors logged to stderr)
- Useful for custom dashboards or Slack notifications

### Multi-Logger
- Combines multiple loggers (e.g., file + webhook)
- Configured via `logging` section in config.jsonc

---

## Phase 6: Build & Release

### `.goreleaser.yml`

Cross-compile for 6 targets:
- `linux/amd64`, `linux/arm64`
- `darwin/amd64`, `darwin/arm64`
- `windows/amd64`, `windows/arm64`

Inject version via ldflags:
```
-s -w -X internal/version.Version={{.Version}} -X internal/version.Commit={{.Commit}} -X internal/version.Date={{.Date}}
```

### `.github/workflows/release.yml`

On tag push `v*`:
1. Checkout + setup Go
2. GoReleaser build (`--skip=publish`)
3. Publish to public repo via `gh release create`

### `Taskfile.yml`

```yaml
tasks:
  build:        go build with ldflags
  run:          build + serve (stdio)
  run:sse:      build + serve --transport sse
  inspect:      npx @modelcontextprotocol/inspector ./curl-mcp serve
  inspect:sse:  inspector in SSE mode
  request:      CLI request shortcut
  lint:         gofmt + go vet
  tidy:         go mod tidy
  upgrade:      go get -u ./... + tidy
  clean:        rm binary
  version:      print version
```

---

## Phase 7: Installation & Public Repo

### One-Click Install

```bash
curl -sL https://raw.githubusercontent.com/thesatellite-ai/curl-mcp/main/install.sh | sh
```

The `install.sh` lives in the repo root. It:
1. Detects OS (linux, darwin) and arch (amd64, arm64)
2. Fetches latest release tag from GitHub API
3. Downloads the matching `.tar.gz` archive
4. Extracts and installs to `/usr/local/bin`

### Go Install

```bash
go install github.com/thesatellite-ai/curl-mcp/cmd/curl-mcp@latest
```

### Manual Binary Download

```bash
# macOS (Apple Silicon)
curl -sL https://github.com/thesatellite-ai/curl-mcp/releases/latest/download/curl-mcp_darwin_arm64.tar.gz | tar xz
sudo mv curl-mcp /usr/local/bin/

# Linux
curl -sL https://github.com/thesatellite-ai/curl-mcp/releases/latest/download/curl-mcp_linux_amd64.tar.gz | tar xz
sudo mv curl-mcp /usr/local/bin/

# Windows (PowerShell)
Invoke-WebRequest -Uri https://github.com/thesatellite-ai/curl-mcp/releases/latest/download/curl-mcp_windows_amd64.zip -OutFile curl-mcp.zip
Expand-Archive curl-mcp.zip -DestinationPath .
```

### Uninstall

```bash
sudo rm /usr/local/bin/curl-mcp
```

### Claude Desktop

```json
{
  "mcpServers": {
    "curl": {
      "command": "curl-mcp",
      "args": ["serve"]
    }
  }
}
```

### Claude Code

```json
{
  "mcpServers": {
    "curl": {
      "command": "curl-mcp",
      "args": ["serve"],
      "env": {}
    }
  }
}
```

### GitHub Release Workflow

On tag push `v*`:
1. GoReleaser builds 6 binaries and publishes the release directly
2. Uses `GITHUB_TOKEN` (automatic, no extra secrets needed)

---

## Implementation Order

| Step | What | Depends On |
|---|---|---|
| 1 | Scaffold project structure, go.mod, deps | - |
| 2 | `pkg/curl/options.go` + `response.go` | - |
| 3 | `pkg/curl/client.go` (CycleTLS wrapper) | Step 2 |
| 4 | `pkg/curl/curlcmd.go` (print as curl) | Step 2 |
| 5 | `pkg/curl/logger.go` (file logger first) | Step 2, 3 |
| 6 | `internal/config/paths.go` + `config.go` | - |
| 7 | `internal/version/version.go` | - |
| 8 | `cmd/curl-mcp/main.go` (Cobra: serve, request, batch, config, version) | Steps 2-7 |
| 9 | `internal/mcp/server.go` + `tools.go` (MCP tools + prompts) | Steps 2-3 |
| 10 | OTEL logger + webhook logger | Step 5 |
| 11 | `.goreleaser.yml` + `.github/workflows/release.yml` + `Taskfile.yml` | Step 8 |
| 12 | `README.md` + `docs/TECHNICAL.md` + `docs/USAGE.md` | All |

---

## Key Dependencies

| Package | Purpose |
|---|---|
| `github.com/Danny-Dasilva/CycleTLS/cycletls` | TLS fingerprinting HTTP client |
| `github.com/modelcontextprotocol/go-sdk/mcp` | MCP server (stdio + SSE) |
| `github.com/spf13/cobra` | CLI framework |
| `go.opentelemetry.io/otel` | OpenTelemetry tracing (optional) |

JSONC parsing: strip `//` and `/* */` comments manually before `json.Unmarshal` — no external dependency needed (~20 lines of code).

---

## Things You Didn't Mention But Should Have

1. **Named profiles** — Preset browser fingerprints (Chrome, Firefox, Safari) in config. Switch with `--profile chrome`.
2. **Connection reuse** — CycleTLS pools connections by config hash. Expose `enable_connection_reuse` option.
3. **Cookie jar persistence** — Optionally save/load cookies across requests (useful for session-based scraping).
4. **Response body size limit** — Cap at 10MB by default, configurable.
5. **Binary response handling** — Use `BodyBytes` for images/downloads, auto-detect from Content-Type.
6. **Retry with backoff** — Optional retry count + exponential backoff for transient failures.
7. **Header order preservation** — CycleTLS supports `HeaderOrder` which is critical for fingerprint accuracy.
8. **CLAUDE.md** — AI context file describing the project for Claude Code sessions.
9. **`--dry-run`** — Print what would be sent without making the request (combines with `--print-curl`).
10. **Environment variable overrides** — `CURL_MCP_CONFIG`, `CURL_MCP_JA3`, `CURL_MCP_PROXY` etc. for CI/scripts.
