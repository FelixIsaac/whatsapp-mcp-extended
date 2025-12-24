# Testing Roadmap

Current coverage target: 50%. This document tracks untested functionality for future coverage.

## Python (whatsapp-mcp-server)

### Covered (~50%)
- `lib/models.py` - All dataclasses and to_dict methods
- `lib/bridge.py` - API client functions (mocked)
- `lib/utils.py` - Logging setup, get_sender_name

### Not Yet Covered

#### lib/database.py
| Function | Priority | Complexity | Notes |
|----------|----------|------------|-------|
| `list_messages()` | High | Medium | Needs temp DB fixture, test filters |
| `get_message_context()` | High | Medium | Test before/after context |
| `list_chats()` | Medium | Low | Test query, sort_by options |
| `get_contact_by_jid()` | Medium | Low | Test nickname lookup |
| `set_contact_nickname()` | Medium | Low | Test insert/update |
| `search_contacts()` | Medium | Low | Test partial matching |

#### main.py / gradio-main.py
| Function | Priority | Complexity | Notes |
|----------|----------|------------|-------|
| MCP tool handlers | Low | High | Requires MCP test harness |
| Gradio integration | Low | High | UI testing |

## Go (whatsapp-bridge)

### Covered (~50%)
- `internal/database/webhooks_test.go` - Webhook CRUD operations

### Not Yet Covered

#### internal/api/handlers.go
| Handler | Priority | Complexity | Notes |
|---------|----------|------------|-------|
| `handleSendMessage` | High | Medium | Mock WhatsApp client |
| `handleWebhookCRUD` | High | Low | HTTP handler tests |
| `handleGetGroup` | Medium | Medium | Mock group info |

#### internal/webhook/validation.go
| Function | Priority | Complexity | Notes |
|----------|----------|------------|-------|
| `isPrivateIP()` | High | Low | Security-critical |
| `ValidateWebhookURL()` | High | Medium | SSRF prevention |

#### internal/webhook/delivery.go
| Function | Priority | Complexity | Notes |
|----------|----------|------------|-------|
| `deliverWithRetry()` | Medium | Medium | Mock HTTP server |
| `computeHMAC()` | High | Low | Verify signature |

#### internal/whatsapp/messages.go
| Function | Priority | Complexity | Notes |
|----------|----------|------------|-------|
| `validateMediaPath()` | High | Low | Security-critical |
| `SendMessage()` | Medium | High | Requires whatsmeow mock |

## Integration Tests (Future)

### End-to-End Flows
1. Message send → webhook delivery → database storage
2. Media upload → path validation → file serving
3. Group operations → participant management

### Docker Compose Tests
1. Service health checks
2. Inter-service communication
3. Database persistence

## Running Tests

```bash
# Python
cd whatsapp-mcp-server && uv run pytest --cov=lib -v

# Go
cd whatsapp-bridge && go test -v -race ./...

# Coverage report
cd whatsapp-mcp-server && uv run pytest --cov=lib --cov-report=html
cd whatsapp-bridge && go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

## CI/CD

Tests run automatically on:
- Push to `main`
- Pull requests to `main`

See `.github/workflows/` for configuration.
