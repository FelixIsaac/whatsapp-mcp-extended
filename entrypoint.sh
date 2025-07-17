#!/bin/bash

# Start the Go WhatsApp bridge in the background
cd /app/whatsapp-bridge
./whatsapp-bridge &

# Start the Python MCP server with SSE transport on port 8082
cd /app/whatsapp-mcp-server
python gradio-main.py

# Keep the container running
wait