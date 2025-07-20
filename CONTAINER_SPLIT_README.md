# Container Split Documentation

## Overview
The WhatsApp MCP system has been split into two separate containers:

1. **whatsapp-bridge**: Handles WhatsApp connectivity and API
2. **whatsapp-mcp**: Handles the MCP server and Gradio UI

## Container Architecture

### WhatsApp Bridge Container (`whatsapp-bridge`)
- **Dockerfile**: `Dockerfile.bridge`
- **Ports**: 8080 (WhatsApp Bridge API)
- **Responsibilities**: 
  - WhatsApp client connection
  - Message handling
  - API endpoints for WhatsApp operations
- **Storage**: `/app/store` (shared with MCP container)

### WhatsApp MCP Container (`whatsapp-mcp`)
- **Dockerfile**: `dockerfile` (updated)
- **Ports**: 8081 (MCP server), 8082 (Gradio UI)
- **Responsibilities**:
  - MCP server functionality
  - Gradio web interface
  - Communication with WhatsApp Bridge via HTTP API
- **Storage**: `/app/store` (shared with Bridge container)

### Webhook UI Container (`webhook-ui`)
- **Dockerfile**: `Dockerfile.ui` (unchanged)
- **Ports**: 8089 (mapped to container port 8080)
- **Responsibilities**: Web interface for webhook management

## Key Changes Made

### 1. New Dockerfile.bridge
- Minimal Go runtime container
- Only builds and runs the WhatsApp Bridge
- Exposes port 8080

### 2. Updated dockerfile (for MCP)
- Removed Go build stage
- Only handles Python MCP server and Gradio UI
- Exposes ports 8081 and 8082

### 3. Updated docker-compose.yaml
- Split into separate services: `whatsapp-bridge` and `whatsapp-mcp`
- Added internal network (`whatsapp_internal`) for inter-container communication
- Set `BRIDGE_HOST=whatsapp-bridge` environment variable for MCP container
- Shared storage volume between bridge and MCP containers

### 4. Updated whatsapp.py
- Added environment variable support for bridge host
- Uses `BRIDGE_HOST` environment variable to connect to bridge container
- Falls back to localhost for development

## Network Configuration

### Internal Communication
- `whatsapp_internal` network for container-to-container communication
- MCP server connects to bridge using hostname `whatsapp-bridge`

### External Access
- Bridge API: `localhost:8080`
- MCP Server: `localhost:8081`
- Gradio UI: `localhost:8082`
- Webhook UI: `localhost:8089`

## Deployment

To deploy the split containers:

```bash
docker-compose up --build
```

The containers will start in the correct order:
1. `whatsapp-bridge` (first)
2. `whatsapp-mcp` (depends on bridge)
3. `webhook-ui` (depends on both)

## Shared Storage

Both containers share the same storage volume (`./store:/app/store`) ensuring:
- WhatsApp session data is persistent
- Database files are accessible to both containers
- No data duplication or synchronization issues

## Environment Variables

### whatsapp-mcp container
- `BRIDGE_HOST`: Set to `whatsapp-bridge` for container communication
- `TZ`: Timezone setting

### whatsapp-bridge container
- `TZ`: Timezone setting
