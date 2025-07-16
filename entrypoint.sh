#!/bin/bash

# Start the Go WhatsApp bridge in the background
cd /app/whatsapp-bridge
./whatsapp-bridge &


# Start the webhook UI server in the background
cd /app/whatsapp-webhook-ui && python3 -m http.server 3000 &

# Start the Python MCP server with SSE transport on port 8082
cd /app/whatsapp-mcp-server
python gradio-main.py

# Keep the container running
wait