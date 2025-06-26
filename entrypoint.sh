#!/bin/bash

# Start the Go WhatsApp bridge in the background
cd /app
./whatsapp-bridge &

# Start the Python MCP server with SSE transport on port 8081
cd /app/whatsapp-mcp-server
uv run sse_main.py

# Keep the container running
wait