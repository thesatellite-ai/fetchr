# Technical Architecture

> For installation and usage, see the [README](../README.md). For the full command/parameter reference, see the [Usage Guide](USAGE.md).

Architecture decisions, design rationale, and internals for contributors.

---

## Overview

fetchr is structured as three layers:

```
┌─────────────────────────────────────────┐
│  Interfaces                             │
│  ┌──────────┐ ┌────────┐ ┌───────────┐ │
│  │ CLI      │ │ MCP    │ │ Go pkg    │ │
│  │ (Cobra)  │ │ Server │ │ (import)  │ │
│  └────┬─────┘ └───┬────┘ └─────┬─────┘ │
│       │            │            │        │
│  ─────┴────────────┴────────────┴─────  │
│                                         │
│  ┌─────────────────────────────────┐    │
│  │  pkg/curl — Core HTTP Client    │    │
│  │  (CycleTLS wrapper)             │    │
│  └─────────────────────────────────┘    │
│                                         │
│  ┌──────────────┐ ┌─────────────────┐   │
│  │ Config       │ │ Logging         │   │
│  │ (JSONC)      │ │ (File/Webhook)  │   │
│  └──────────────┘ └─────────────────┘   │
└─────────────────────────────────────────┘
```

All three interfaces (CLI, MCP, Go API) use the same `pkg/curl.Client` underneath. This guarantees consistent behavior regardless of how you interact with fetchr.

---

## Package Layout

### `pkg/curl` — Public API

The only package intended for external import. Contains:

| File | Purpose |
|---|---|
| `client.go` | `Client` struct, `New()`, `Do()`, `Batch()`, `Close()` |
| `options.go` | `RequestOptions`, `Cookie`, `Merge()` |
| `response.go` | `Response`, `BatchResult`, `Format()`, `JSON()` |
| `curlcmd.go` | `ToCurl()` — convert options to curl command string |
| `logger.go` | `Logger` interface, `FileLogger`, `WebhookLogger`, `MultiLogger` |

**Design decisions:**

- `RequestOptions` uses `json` tags matching the MCP tool input schema, so `json.Unmarshal` from MCP tool arguments directly populates the struct.
- `Merge()` is a standalone function (not a method) to keep the merge logic pure and testable. It does shallow field-level merging — non-zero values from the override win.
- `Client` holds a single `cycletls.CycleTLS` instance. CycleTLS uses goroutines and channels internally, so `Client.Close()` is important.
- `Batch()` is implemented as concurrent `Do()` calls via goroutines. Each request gets its own goroutine; results are collected in a pre-allocated slice (no mutex needed since each goroutine writes to a unique index).

### `internal/config` — Configuration

| File | Purpose |
|---|---|
| `config.go` | `Config` struct, JSONC parser, `LoadConfig()`, `LoadConfigAuto()`, `EnsureDefaultConfig()` |
| `paths.go` | Platform-aware config file discovery |
| `default_config.jsonc` | Embedded default config (via `go:embed`) |

**Design decisions:**

- JSONC parsing is done in ~40 lines of custom code rather than adding a dependency. It handles `//` line comments, `/* */` block comments, and respects string literals.
- The default config is embedded in the binary via `go:embed`. This means `config init` and `LoadConfigAuto()` always have the default available, even without filesystem access.
- `LoadConfigAuto()` auto-creates the config on first run. This ensures users always have a working config file to edit — no manual setup required.
- Config path search is platform-aware (XDG on Linux, `~/Library` on macOS, `%APPDATA%` on Windows).
- `--config` flag on any command short-circuits discovery and loads a specific file.

### `internal/mcp` — MCP Server

| File | Purpose |
|---|---|
| `server.go` | `Server` struct, `NewServer()`, `RunStdio()`, `SSEHandler()` |
| `tools.go` | Tool handlers + JSON schemas for `request` and `batch` |
| `prompts.go` | MCP prompt definitions |

**Design decisions:**

- The `Server` wraps both the MCP server and the curl `Client`. It owns the client lifecycle.
- Tool JSON schemas are defined as const strings rather than generated from struct tags. This gives full control over descriptions, enums, and nested object schemas without fighting reflection.
- Profile resolution happens in `resolveOptions()`: defaults -> profile -> per-request. This is called once per request in the tool handler.
- The batch tool supports a top-level `profile` field that applies to all requests that don't have their own — this reduces repetition in batch inputs.

### `internal/version` — Build Info

Single file with three `var` declarations set via `-ldflags` at build time. The `String()` method formats them for display.

### `cmd/fetchr` — CLI Entry Point

Single `main.go` with Cobra commands. Each command is a factory function returning `*cobra.Command`. This keeps the file scannable and avoids global state.

---

## CycleTLS Integration

[CycleTLS](https://github.com/Danny-Dasilva/CycleTLS) provides HTTP requests with custom TLS fingerprints. Key API points:

```go
// Initialize (creates goroutine pool)
client := cycletls.Init()

// Make request
response, err := client.Do(url, options, method)

// Response fields
response.Status    // int
response.Body      // string
response.BodyBytes // []byte
response.Headers   // map[string]string
response.Cookies   // []*http.Cookie
response.FinalUrl  // string (after redirects)

// Cleanup
client.Close()
```

Our `pkg/curl.Client` wraps this and adds:
- Default option merging
- Profile resolution
- Logging
- Duration tracking
- Typed Response struct with formatting

The `toCycleOptions()` function maps our `RequestOptions` to `cycletls.Options`. Field names differ slightly (e.g., our `http2_fingerprint` vs CycleTLS's `HTTP2Fingerprint`) — the mapping is explicit rather than relying on reflection.

---

## Option Merging Strategy

Options are merged in layers:

```
Config Defaults -> Profile -> Per-request Options
```

`Merge(base, override)` does field-level comparison:
- String fields: override wins if non-empty
- Int fields: override wins if non-zero
- Bool fields: override wins if `true`
- Map fields: keys are merged (override keys win, base keys preserved)
- Slice fields: override replaces entirely

This means you can't explicitly set a timeout to `0` or disable a bool that defaults to `true` in a lower layer. This is an acceptable trade-off — these edge cases are rare and the simplicity benefits outweigh them.

---

## JSONC Parser

The JSONC parser (`stripJSONC`) is a single-pass character scanner:

1. Track whether we're inside a string literal (respecting escape sequences)
2. Outside strings: skip `//` to end of line, skip `/* ... */` blocks
3. Copy everything else to output

This is intentionally simple. It doesn't handle every JSONC edge case (e.g., comments inside already-invalid JSON), but it handles all real-world config files correctly.

---

## Logging Architecture

```
Logger interface
    ├── FileLogger      — append JSONL to file (mutex-protected)
    ├── WebhookLogger   — POST JSON to URL (fire-and-forget goroutine)
    └── MultiLogger     — fan-out to N loggers
```

- `FileLogger` opens/closes the file on each write. This is intentional — it avoids file handle leaks and works correctly with log rotation.
- `WebhookLogger` fires a goroutine per log entry and swallows errors (stderr only). This prevents slow webhooks from blocking requests.
- `MultiLogger` iterates loggers sequentially. Individual logger errors are logged to stderr but don't propagate.
- Logging is called from `Client.Do()` after the request completes. The `LogEntry` includes both request and response, plus duration and optional curl command.

---

## Build & Release

### Version Injection

Three variables in `internal/version` are set via ldflags:

```
-X internal/version.Version={{.Version}}
-X internal/version.Commit={{.Commit}}
-X internal/version.Date={{.Date}}
```

`Taskfile.yml` extracts these from git. GoReleaser provides them automatically.

### Cross-Compilation

GoReleaser builds 6 targets:

| OS | Arch |
|---|---|
| linux | amd64, arm64 |
| darwin | amd64, arm64 |
| windows | amd64, arm64 |

`CGO_ENABLED=0` ensures static binaries with no system library dependencies.

Archives are `.tar.gz` for linux/darwin and `.zip` for windows.

### Release Workflow

1. Tag a commit: `git tag v1.2.0`
2. Push: `git push origin v1.2.0`
3. GitHub Actions triggers `.github/workflows/release.yml`
4. GoReleaser builds all 6 binaries and creates a GitHub Release
5. `install.sh` automatically picks up the latest release

---

## Error Handling

- `Client.Do()` returns errors from CycleTLS as-is, wrapped with context.
- MCP tool handlers return errors as `CallToolResult` with `IsError: true` — they never return Go errors to the MCP framework.
- CLI commands return errors to Cobra, which prints them and exits with code 1.
- Logger errors are swallowed (stderr) to prevent logging failures from breaking requests.

---

## Security Considerations

- **TLS verification** is enabled by default. `insecure` must be explicitly set per-request.
- **Proxy credentials** in the proxy URL are passed to CycleTLS as-is. Be cautious with logging — proxy URLs with embedded credentials will appear in log entries.
- **Config files** may contain proxy URLs or webhook endpoints. The default config path has mode 0644.
- **Response body** is not size-limited at the `pkg/curl` level (CycleTLS reads the full response). The CLI and MCP layers should impose limits if needed.
