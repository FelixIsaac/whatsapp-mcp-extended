BRIDGE := http://127.0.0.1:8180
WEBUI  := http://127.0.0.1:8090

# Load API_KEY from .env if present
-include .env
export API_KEY

.PHONY: setup up down restart logs status pair reconnect check sync-venv open-ui

## One-shot host setup (Docker auto-start, Task Scheduler jobs, uv sync)
setup:
	powershell.exe -ExecutionPolicy Bypass -File setup.ps1

## Start all containers in the background
up:
	docker-compose up -d

## Stop all containers
down:
	docker-compose down

## Restart bridge only (e.g. after a code change)
restart:
	docker-compose restart whatsapp-bridge

## Tail bridge logs
logs:
	docker-compose logs -f whatsapp-bridge

## Show current connection state (needs_pairing, connected, jid, uptime)
status:
	@curl -sf -H "X-API-Key: $(API_KEY)" $(BRIDGE)/api/connection 2>/dev/null \
	  | python3 -m json.tool 2>/dev/null \
	  || curl -sf -H "X-API-Key: $(API_KEY)" $(BRIDGE)/api/connection \
	  || echo "Bridge unreachable at $(BRIDGE)"

## Force reconnect (no re-pairing needed, uses existing session)
reconnect:
	@curl -sf -X POST -H "X-API-Key: $(API_KEY)" $(BRIDGE)/api/reconnect \
	  | python3 -m json.tool 2>/dev/null \
	  || echo "Reconnect triggered (or bridge unreachable)"

## Pair via phone number code — no QR scan needed.
## Usage: make pair PHONE=+60123456789
pair:
	@test -n "$(PHONE)" || (echo "Usage: make pair PHONE=+60123456789" && exit 1)
	@echo "Requesting pairing code for $(PHONE)..."
	@curl -sf -X POST \
	  -H "X-API-Key: $(API_KEY)" \
	  -H "Content-Type: application/json" \
	  -d '{"phone":"$(PHONE)"}' \
	  $(BRIDGE)/api/pair \
	  | python3 -m json.tool 2>/dev/null \
	  || curl -sf -X POST \
	       -H "X-API-Key: $(API_KEY)" \
	       -H "Content-Type: application/json" \
	       -d '{"phone":"$(PHONE)"}' \
	       $(BRIDGE)/api/pair
	@echo ""
	@echo "Enter the 8-character code in WhatsApp > Settings > Linked Devices > Link a Device"

## Check pairing status
pairing-status:
	@curl -sf -H "X-API-Key: $(API_KEY)" $(BRIDGE)/api/pairing \
	  | python3 -m json.tool 2>/dev/null \
	  || curl -sf -H "X-API-Key: $(API_KEY)" $(BRIDGE)/api/pairing

## Run pre-flight checks on the Python MCP server
check:
	cd whatsapp-mcp-server && uv run python check.py

## Sync the Python venv (run this if MCP server fails to import)
sync-venv:
	cd whatsapp-mcp-server && uv sync

## Open the web UI in the default browser (QR scan, webhooks, contacts)
open-ui:
	start $(WEBUI) 2>/dev/null || open $(WEBUI) 2>/dev/null || xdg-open $(WEBUI) || echo "Open $(WEBUI) in your browser"

help:
	@echo ""
	@echo "whatsapp-mcp-extended — available targets:"
	@echo ""
	@echo "  setup          One-shot Windows host setup (run once after cloning)"
	@echo "  up             docker-compose up -d"
	@echo "  down           docker-compose down"
	@echo "  restart        Restart bridge container only"
	@echo "  logs           Tail bridge logs"
	@echo "  status         Show connection state (connected, needs_pairing, jid)"
	@echo "  pair           Pair via phone code: make pair PHONE=+60123456789"
	@echo "  pairing-status Check pairing code progress"
	@echo "  reconnect      Force reconnect (no re-pairing)"
	@echo "  check          Run Python MCP server pre-flight checks"
	@echo "  sync-venv      Re-sync Python venv (fix missing module errors)"
	@echo "  open-ui        Open web UI in browser"
	@echo ""
