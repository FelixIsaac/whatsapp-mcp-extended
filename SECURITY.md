# Security Policy

## Reporting a Vulnerability

To report a security vulnerability privately, use [GitHub's private vulnerability reporting](https://github.com/FelixIsaac/whatsapp-mcp-extended/security/advisories/new).

Alternatively email: `felix@noversetech.com` — include a description, reproduction steps, and impact.

Expected response within **48 hours**. Public disclosure after fix is shipped.

## Security Fixes vs Upstream

This fork (`FelixIsaac/whatsapp-mcp-extended`) fixes all known security issues from the abandoned upstream `lharries/whatsapp-mcp`:

| CVE / Issue | Upstream | This fork |
|-------------|----------|-----------|
| **Path traversal on `/api/send`** — `mediaPath` passed unsanitized to `os.ReadFile()` (Issue #241) | Vulnerable | Fixed — path validated against allowed directories |
| **Bridge binds to `0.0.0.0` by default** — unauthenticated `/api/send` and `/api/download` exposed on all LAN interfaces (Issue #215) | Vulnerable | Fixed — defaults to `127.0.0.1`; set `BRIDGE_HOST=0.0.0.0` to opt in |
| **No API key authentication** — any process on localhost can send WhatsApp messages | Vulnerable | Fixed — `API_KEY` required; all bridge endpoints enforce `X-API-Key` header |
| **MCPSafe grade D (67/100), 13 high-severity findings** (Issue #247) | Unfixed | Addressed — above fixes resolve the top findings |

## Threat Model

whatsapp-mcp-extended is designed for **local/personal use** (single user, trusted machine). It is **not** designed for multi-tenant or public internet deployments without additional hardening.

### What is protected
- Bridge API bound to `127.0.0.1` — not reachable from network by default
- All bridge endpoints require `X-API-Key` header — prevents unauthorized tool calls from other local processes
- Docker services communicate over an isolated internal network (`whatsapp_internal`)
- All Docker ports explicitly mapped to `127.0.0.1:PORT` — no accidental LAN exposure

### What is out of scope
- **Prompt injection via incoming messages** — an attacker who can send you WhatsApp messages may craft a message that influences Claude's behavior. Mitigate at the agent layer (human approval before destructive actions).
- **Malicious media files** — downloaded media is written to the local filesystem. Scan before opening untrusted files.
- **WhatsApp account bans** — the anti-ban interceptor reduces risk but does not eliminate it. Do not use for mass messaging.

## Supported Versions

Only the latest release is supported. Upgrade to the latest version before reporting.

| Version | Supported |
|---------|-----------|
| 0.3.x   | ✅ Yes |
| < 0.3   | ❌ No |
