# Contributing to WhatsApp MCP Extended

Thanks for your interest in contributing!

## Quick Start

```bash
# Clone
git clone https://github.com/felixisaac/whatsapp-mcp-extended
cd whatsapp-mcp-extended

# Start services
docker-compose up -d

# Watch logs for QR code
docker-compose logs -f whatsapp-bridge
```

## Development Setup

### Go Bridge (whatsapp-bridge/)

```bash
cd whatsapp-bridge
go run main.go
go test ./...
```

### Python MCP Server (whatsapp-mcp-server/)

```bash
cd whatsapp-mcp-server
uv sync
uv run python main.py

# Pre-commit checks
uv run python check.py
```

### Webhook UI (whatsapp-webhook-ui/)

```bash
cd whatsapp-webhook-ui
python3 -m http.server 8089
```

## Code Style

### Go
- Run `go fmt` before committing
- Follow standard Go conventions

### Python
- Use `ruff` for linting
- Use `mypy` for type checking
- Run `uv run python check.py` before committing

## Pull Requests

1. Fork the repo
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Run tests and checks
5. Commit with clear messages
6. Push and open a PR

### PR Guidelines

- Keep PRs focused on a single feature/fix
- Update documentation if needed
- Add tests for new functionality
- Ensure all checks pass

## Reporting Issues

- Check existing issues first
- Include steps to reproduce
- Include error logs if applicable
- Specify your environment (OS, Docker version, etc.)

## Architecture

```
whatsapp-bridge/     # Go - WhatsApp connection, REST API
whatsapp-mcp-server/ # Python - MCP tools, Claude integration
whatsapp-webhook-ui/ # HTML/JS - Webhook management UI
```

## Adding New MCP Tools

1. Add Go endpoint in `whatsapp-bridge/internal/api/handlers.go`
2. Add route in `whatsapp-bridge/internal/api/server.go`
3. Add Python function in `whatsapp-mcp-server/whatsapp.py`
4. Add MCP tool in `whatsapp-mcp-server/main.py` and `gradio-main.py`
5. Update README tool list

## Questions?

Open an issue or discussion on GitHub.
