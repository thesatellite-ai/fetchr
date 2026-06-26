# Security Policy

## Reporting a vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, report them privately via **[GitHub Security Advisories](https://github.com/thesatellite-ai/fetchr/security/advisories/new)**.

Please include:

- A description of the vulnerability and its impact.
- Steps to reproduce (proof-of-concept if possible).
- Affected version(s) and environment.

We will acknowledge your report within a few days and keep you updated on remediation. We ask that you give us a reasonable window to release a fix before any public disclosure, and we're happy to credit you in the advisory.

## Scope notes

fetchr makes outbound HTTP requests on your behalf and can run as a local MCP server. The most relevant areas are: the MCP server surface (stdio/SSE), proxy and TLS-verification handling (`--insecure`), credential handling in request logging (file/webhook sinks), and config parsing. Please flag anything that could leak credentials, bypass intended TLS verification, or allow SSRF beyond the user's intent.

## Supported versions

Security fixes are applied to the latest released version. Please upgrade to stay supported.

| Version | Supported |
|---|---|
| latest | ✅ |
| older | ❌ |
