# WhatsApp MCP Server - Enhanced Enterprise Edition

A comprehensive Model Context Protocol (MCP) server for WhatsApp with advanced webhook capabilities, contact management, and enterprise-grade features.

This enhanced version provides **complete WhatsApp integration** with your personal account, featuring advanced contact management, real-time webhooks, comprehensive media handling, and a modern web interface for webhook management.

Built with a containerized microservices architecture, this system offers professional-grade reliability, scalability, and maintainability for enterprise and developer use cases.

![WhatsApp MCP](./example-use.png)

## 🚀 Key Features

### 📱 Core WhatsApp Integration
- **Direct Personal Account Connection**: Connect to your personal WhatsApp account via the official multidevice API
- **Complete Message History**: Full access to your message history with advanced search and filtering
- **Media Support**: Send and receive images, videos, documents, and audio messages
- **Voice Messages**: Send audio files as playable WhatsApp voice messages with automatic format conversion
- **Group Chat Support**: Full support for group chats with participant management
- **Real-time Synchronization**: Instant message synchronization with your WhatsApp account

### 🔗 Advanced Webhook System
- **Real-time Notifications**: Instant HTTP webhook notifications for incoming messages
- **Flexible Trigger System**: Configure webhooks based on sender, keywords, media types, or all messages
- **Pattern Matching**: Support for exact match, contains, and regex pattern matching
- **Retry Logic**: Exponential backoff retry mechanism with comprehensive logging
- **Security**: HMAC-SHA256 signature authentication for webhook endpoints
- **Web UI**: Modern responsive web interface for webhook management

### 👥 Enhanced Contact Management
- **Unified Contact Resolution**: Intelligent contact name resolution from multiple sources
- **Custom Nicknames**: Set and manage custom nicknames for contacts
- **Rich Contact Data**: Access to full contact information including business names and profile data
- **Advanced Search**: Search contacts by name, phone number, or custom attributes
- **Contact Prioritization**: Smart name resolution with customizable priority ordering

### 🏗️ Enterprise Architecture
- **Microservices Design**: Containerized architecture with separate services for different functions
- **Persistent Storage**: Reliable SQLite database storage with shared volumes
- **Scalable Deployment**: Docker Compose setup for easy deployment and scaling

## 🏛️ System Architecture

This application consists of three main containerized services:

### 1. **WhatsApp Bridge Service** (`whatsapp-bridge/`)
- **Technology**: Go application using whatsmeow library
- **Port**: 8080
- **Responsibilities**: 
  - WhatsApp API connection and authentication
  - Message processing and storage
  - Webhook delivery and management
  - RESTful API endpoints
  - Contact and chat management

### 2. **MCP Server Service** (`whatsapp-mcp-server/`)
- **Technology**: Python with Model Context Protocol
- **Ports**: 8081 (MCP), 8082 (Gradio UI)
- **Responsibilities**:
  - MCP protocol implementation for Claude/AI integration
  - Advanced message search and filtering
  - Media download and processing
  - Contact management tools
  - Gradio web interface for testing

### 3. **Webhook Management UI** (`whatsapp-webhook-ui/`)
- **Technology**: Modern HTML/CSS/JavaScript SPA
- **Port**: 8089
- **Responsibilities**:
  - Webhook configuration management
  - Real-time webhook testing
  - Delivery log monitoring
  - Responsive web interface

## 📦 Installation

### Prerequisites

- **Docker & Docker Compose**: Latest versions
- **Python 3.11+**: For MCP server
- **Go 1.24+**: For WhatsApp bridge
- **UV Package Manager**: Install with `curl -LsSf https://astral.sh/uv/install.sh | sh`
- **FFmpeg** (optional): For audio message conversion
- **Anthropic Claude Desktop** or **Cursor**: For AI integration

### Quick Start with Docker

1. **Clone the repository**

   ```bash
   git clone https://github.com/AdamRussak/whatsapp-mcp
   cd whatsapp-mcp
   ```

2. **Start all services**

   ```bash
   docker network create n8n_n8n_traefik_network
   docker-compose up -d
   ```

   This will start:
   - WhatsApp Bridge: `http://localhost:8080`
   - MCP Server: `http://localhost:8081`
   - Gradio UI: `http://localhost:8082`
   - Webhook Manager: `http://localhost:8089`

3. **Initial Setup**

   On first run, you'll need to scan a QR code to authenticate with WhatsApp:
   
   ```bash
   # Watch the bridge logs for QR code
   docker-compose logs -f whatsapp-bridge
   ```

   Scan the QR code with your WhatsApp mobile app to authenticate.

### Manual Development Setup

1. **Start WhatsApp Bridge**

   ```bash
   cd whatsapp-bridge
   go run main.go
   ```

2. **Start MCP Server**

   ```bash
   cd whatsapp-mcp-server
   uv sync
   uv run python whatsapp.py
   ```

3. **Start Webhook UI**

   ```bash
   cd whatsapp-webhook-ui
   python3 -m http.server 8089
   ```

## 🛠️ Advanced Features

### Webhook System

#### Configuration Options
- **Trigger Types**: all, chat_jid, sender, keyword, media_type
- **Match Types**: exact, contains, regex
- **Delivery**: Asynchronous with exponential backoff
- **Security**: HMAC-SHA256 signature authentication
- **Logging**: Comprehensive delivery logs and status tracking

#### Example Webhook Configuration
```json
{
  "name": "Urgent Messages",
  "webhook_url": "https://your-system.com/webhook",
  "secret_token": "your-secret-key",
  "enabled": true,
  "triggers": [
    {
      "trigger_type": "keyword",
      "trigger_value": "urgent|emergency|help",
      "match_type": "regex",
      "enabled": true
    }
  ]
}
```

### Contact Management

#### Advanced Features
- **Custom Nicknames**: Set personalized names for contacts
- **Multi-source Resolution**: Combines WhatsApp contacts, chat history, and custom data
- **Smart Prioritization**: Intelligent name resolution with customizable priority
- **Bulk Operations**: Mass contact management and updates

#### Priority Order
1. Custom Nickname (highest priority)
2. Full Name (from WhatsApp contacts)
3. Push Name (WhatsApp display name)
4. First Name (from WhatsApp contacts)
5. Business Name (for business contacts)
6. Phone Number (fallback)

### Media Handling

#### Supported Media Types
- **Images**: JPEG, PNG, GIF, WebP
- **Videos**: MP4, AVI, MOV, WebM
- **Audio**: MP3, WAV, OGG, M4A (auto-converted to voice messages)
- **Documents**: PDF, DOC, DOCX, XLS, XLSX, PPT, PPTX

#### Features
- **Automatic Conversion**: Audio files converted to WhatsApp voice message format
- **Media Download**: Direct download of received media files
- **Metadata Extraction**: Comprehensive media information and thumbnails
- **Size Optimization**: Automatic compression for large files

## 🔧 MCP Tools Reference

### Core Messaging Tools
- **`send_message`**: Send text messages to contacts or groups
- **`send_file`**: Send media files with automatic type detection
- **`send_audio_message`**: Send audio as WhatsApp voice messages
- **`download_media`**: Download received media files

### Search and Discovery
- **`search_contacts`**: Advanced contact search with multiple criteria
- **`list_messages`**: Retrieve messages with filtering and pagination
- **`list_chats`**: Get chat list with metadata and last message info
- **`get_message_context`**: Get conversation context around specific messages

### Contact Management
- **`get_contact_details`**: Get comprehensive contact information
- **`list_all_contacts`**: List all contacts with rich metadata
- **`set_contact_nickname`**: Set custom nicknames for contacts
- **`get_contact_nickname`**: Retrieve custom nicknames
- **`remove_contact_nickname`**: Remove custom nicknames
- **`list_contact_nicknames`**: List all custom nicknames

### Chat Operations
- **`get_chat`**: Get detailed chat information
- **`get_direct_chat_by_contact`**: Find direct chat with specific contact
- **`get_contact_chats`**: List all chats involving a contact
- **`get_last_interaction`**: Get most recent interaction with contact

## 📊 API Reference

### WhatsApp Bridge API (`localhost:8080/api`)

#### Message Operations
```bash
# Send message
POST /api/send
Content-Type: application/json

{
  "recipient": "1234567890@s.whatsapp.net",
  "message": "Hello World",
  "media_path": "/path/to/file.jpg"
}

# Download media
GET /api/download?message_id=MESSAGE_ID&chat_jid=CHAT_JID
```

#### Webhook Management
```bash
# List webhooks
GET /api/webhooks

# Create webhook
POST /api/webhooks
{
  "name": "My Webhook",
  "webhook_url": "https://example.com/webhook",
  "secret_token": "optional-secret",
  "enabled": true,
  "triggers": [...]
}

# Test webhook
POST /api/webhooks/{id}/test

# View webhook logs
GET /api/webhook-logs?webhook_id={id}
```

## 🔐 Security Features

### Authentication
- **WhatsApp Authentication**: Secure QR code pairing with your personal account
- **Session Management**: Persistent session storage with automatic reconnection
- **API Security**: Rate limiting and request validation

### Webhook Security
- **HMAC Signatures**: SHA256 signature verification for webhook endpoints
- **Token-based Auth**: Secret token validation for webhook requests
- **HTTPS Support**: TLS encryption for webhook delivery
- **Request Validation**: Comprehensive payload validation

### Data Protection
- **Local Storage**: All data stored locally on your system
- **Encrypted Databases**: SQLite databases with proper access controls
- **No Cloud Dependencies**: No external services required for core functionality

## 📈 Monitoring and Logging

### Webhook Monitoring
- **Delivery Status**: Real-time webhook delivery status tracking
- **Retry Monitoring**: Exponential backoff retry attempts with full logging
- **Error Tracking**: Comprehensive error logging and status codes

## 🐳 Docker Configuration

### Container Resources
```yaml
# Default resource limits
whatsapp-bridge:
  memory: 1G
  cpus: '0.5'

whatsapp-mcp:
  memory: 1G
  cpus: '0.5'

webhook-ui:
  memory: 500M
  cpus: '0.5'
```

### Environment Variables
```bash
# Bridge Service
TZ=UTC

# MCP Service
BRIDGE_HOST=whatsapp-bridge
DEBUG=true
TZ=UTC
```

### Networking
- **Internal Network**: `whatsapp_internal` for service communication
- **External Network**: `n8n_n8n_traefik_network` for external access
- **Port Mapping**: Configurable port mappings for all services

## 🔧 Development

### Building from Source
```bash
# Build all containers
docker-compose build

# Build specific service
docker-compose build whatsapp-bridge

# Development mode with hot reload
docker-compose -f docker-compose.dev.yml up
```

### Testing
```bash
# Run bridge tests
cd whatsapp-bridge
go test ./...

# Run MCP server tests
cd whatsapp-mcp-server
uv run pytest

# Test webhook delivery
cd whatsapp-bridge
./test-webhook-delivery.sh
```

### Database Management
```bash
# Access message database
sqlite3 store/messages.db

# Access WhatsApp store
sqlite3 store/whatsapp.db

# Backup databases
docker-compose exec whatsapp-bridge sqlite3 /app/store/messages.db .backup backup.db
```

## 🚨 Troubleshooting

### Common Issues

#### Authentication Problems
- **QR Code Issues**: Ensure terminal supports QR code display
- **Session Expired**: Delete session files and re-authenticate
- **Device Limit**: Remove old devices from WhatsApp settings

#### Container Issues
- **Port Conflicts**: Check for conflicting services on ports 8080-8089
- **Memory Issues**: Increase container memory limits in docker-compose.yml
- **Storage Issues**: Ensure sufficient disk space for databases

#### Webhook Problems
- **Delivery Failures**: Check webhook endpoint accessibility and SSL certificates
- **Authentication Errors**: Verify HMAC signature implementation

### Debug Commands
```bash
# Check container logs
docker-compose logs -f whatsapp-bridge
docker-compose logs -f whatsapp-mcp

# Check container health
docker-compose ps

# Access container shell
docker-compose exec whatsapp-bridge /bin/bash
```

### Performance Optimization
- **Database Optimization**: Regular VACUUM and index maintenance
- **Memory Management**: Monitor container memory usage
- **Network Optimization**: Use internal networking for service communication
- **Storage Optimization**: Regular cleanup of old media files

## 📚 Additional Resources

- **WhatsApp API Documentation**: [whatsmeow library](https://github.com/tulir/whatsmeow)
- **MCP Protocol**: [Model Context Protocol](https://modelcontextprotocol.io/)
- **Docker Documentation**: [Docker Compose](https://docs.docker.com/compose/)
- **Claude Desktop**: [Claude Desktop Integration](https://claude.ai/desktop)

## 🤝 Contributing

Contributions are welcome! Please read our contributing guidelines and submit pull requests for any improvements.

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 📞 Support

For support and questions:
- GitHub Issues: Create an issue for bug reports or feature requests
- Documentation: Check the comprehensive documentation in the project

---

