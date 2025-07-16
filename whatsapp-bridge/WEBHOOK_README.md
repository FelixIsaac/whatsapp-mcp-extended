# WhatsApp Bridge Webhook Extension

This document describes the webhook functionality added to the WhatsApp bridge in Phase 1 of the implementation.

## Overview

The webhook extension allows the WhatsApp bridge to send HTTP notifications to external systems when specific message events occur. This enables real-time integration with other applications, automation systems, or notification services.

## Features Implemented (Phase 1)

### Database Schema
- **webhook_configs**: Stores webhook configuration (URL, secret, enabled status)
- **webhook_triggers**: Defines trigger conditions for each webhook
- **webhook_logs**: Logs all webhook delivery attempts and responses

### Trigger Types
- **all**: Triggers on every message
- **chat_jid**: Triggers on messages from specific chats/groups
- **sender**: Triggers on messages from specific senders
- **keyword**: Triggers on messages containing specific keywords
- **media_type**: Triggers on specific types of media (image, video, audio, document)

### Match Types
- **exact**: Exact string match
- **contains**: Case-insensitive substring match
- **regex**: Regular expression pattern matching

### Webhook Delivery
- Asynchronous delivery using goroutines
- Exponential backoff retry (1s, 2s, 4s, 8s, 16s)
- Maximum 5 retry attempts
- HMAC-SHA256 signature authentication (optional)
- 30-second timeout per request
- Comprehensive logging of delivery attempts

## REST API Endpoints

### Webhook Management

#### List All Webhooks
```bash
GET /api/webhooks
```

#### Create New Webhook
```bash
POST /api/webhooks
Content-Type: application/json

{
  "name": "My Webhook",
  "webhook_url": "https://example.com/webhook",
  "secret_token": "optional-secret",
  "enabled": true,
  "triggers": [
    {
      "trigger_type": "keyword",
      "trigger_value": "urgent",
      "match_type": "contains",
      "enabled": true
    }
  ]
}
```

#### Get Specific Webhook
```bash
GET /api/webhooks/{id}
```

#### Update Webhook
```bash
PUT /api/webhooks/{id}
Content-Type: application/json

{
  "id": 1,
  "name": "Updated Webhook",
  "webhook_url": "https://example.com/webhook",
  "secret_token": "new-secret",
  "enabled": true,
  "triggers": [...]
}
```

#### Delete Webhook
```bash
DELETE /api/webhooks/{id}
```

#### Test Webhook
```bash
POST /api/webhooks/{id}/test
```

#### Enable/Disable Webhook
```bash
POST /api/webhooks/{id}/enable
Content-Type: application/json

{
  "enabled": true
}
```

#### Get Webhook Logs
```bash
GET /api/webhooks/{id}/logs
GET /api/webhook-logs  # All webhook logs
```

## Webhook Payload Structure

When a message matches a webhook trigger, the following JSON payload is sent:

```json
{
  "event_type": "message_received",
  "timestamp": "2025-07-16T12:00:00Z",
  "webhook_config": {
    "id": 1,
    "name": "My Webhook"
  },
  "trigger": {
    "type": "keyword",
    "value": "urgent",
    "match_type": "contains"
  },
  "message": {
    "id": "message-id-123",
    "chat_jid": "1234567890@s.whatsapp.net",
    "chat_name": "Contact Name",
    "sender": "1234567890",
    "sender_name": "John Doe",
    "content": "This is an urgent message",
    "timestamp": "2025-07-16T12:00:00Z",
    "is_from_me": false,
    "media_type": "",
    "filename": "",
    "media_download_url": ""
  },
  "metadata": {
    "group_info": {
      "is_group": false,
      "group_name": "",
      "participant_count": 0
    },
    "delivery_attempt": 1,
    "processing_time_ms": 15
  }
}
```

## Security

### HMAC Authentication
If a `secret_token` is configured, each webhook request includes an `X-Webhook-Signature` header with an HMAC-SHA256 signature:

```
X-Webhook-Signature: sha256=<hex-encoded-signature>
```

To verify the signature:
```python
import hmac
import hashlib

def verify_signature(payload, signature, secret):
    expected = 'sha256=' + hmac.new(
        secret.encode('utf-8'),
        payload,
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(signature, expected)
```

### Rate Limiting
- Webhook delivery does not block message processing
- Failed deliveries are retried with exponential backoff
- Maximum 5 attempts per webhook delivery

## Configuration Examples

### Family Group Notifications
```json
{
  "name": "Family Group",
  "webhook_url": "https://api.example.com/family-notifications",
  "secret_token": "family-secret-123",
  "enabled": true,
  "triggers": [
    {
      "trigger_type": "chat_jid",
      "trigger_value": "120363123456789012@g.us",
      "match_type": "exact",
      "enabled": true
    }
  ]
}
```

### Emergency Keywords
```json
{
  "name": "Emergency Alerts",
  "webhook_url": "https://alerts.example.com/emergency",
  "secret_token": "emergency-secret-456",
  "enabled": true,
  "triggers": [
    {
      "trigger_type": "keyword",
      "trigger_value": "urgent|emergency|help|911",
      "match_type": "regex",
      "enabled": true
    }
  ]
}
```

### Media File Processing
```json
{
  "name": "Document Processor",
  "webhook_url": "https://processor.example.com/documents",
  "enabled": true,
  "triggers": [
    {
      "trigger_type": "media_type",
      "trigger_value": "document",
      "match_type": "exact",
      "enabled": true
    }
  ]
}
```

## Testing

Use the provided test script to verify webhook functionality:

```bash
./test-webhook-api.sh
```

## Database Schema Details

### webhook_configs Table
```sql
CREATE TABLE webhook_configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    webhook_url TEXT NOT NULL,
    secret_token TEXT,
    enabled BOOLEAN DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### webhook_triggers Table
```sql
CREATE TABLE webhook_triggers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    webhook_config_id INTEGER REFERENCES webhook_configs(id),
    trigger_type TEXT NOT NULL,
    trigger_value TEXT,
    match_type TEXT DEFAULT 'exact',
    enabled BOOLEAN DEFAULT 1
);
```

### webhook_logs Table
```sql
CREATE TABLE webhook_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    webhook_config_id INTEGER REFERENCES webhook_configs(id),
    message_id TEXT,
    chat_jid TEXT,
    trigger_type TEXT,
    trigger_value TEXT,
    payload TEXT,
    response_status INTEGER,
    response_body TEXT,
    attempt_count INTEGER DEFAULT 1,
    delivered_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Next Steps (Future Phases)

- **Phase 2**: Advanced trigger combinations (AND/OR logic)
- **Phase 3**: Webhook templates and payload customization
- **Phase 4**: Configuration file support and hot-reload
- **Phase 5**: Integration with popular services (Slack, Discord, Teams)
- **Phase 6**: Analytics and monitoring dashboard

## Error Handling

The webhook system includes comprehensive error handling:

- Invalid webhook configurations are rejected with detailed error messages
- Network errors are retried with exponential backoff
- All delivery attempts are logged for debugging
- Webhook failures do not affect core message processing
- Circuit breaker pattern prevents overwhelming failing endpoints

## Performance Considerations

- Webhook delivery is asynchronous and non-blocking
- Database queries are optimized with proper indexing
- Memory usage is controlled through proper resource management
- Concurrent webhook deliveries are managed through goroutine pools

## Monitoring

Monitor webhook health through:
- Webhook delivery logs (`/api/webhook-logs`)
- Success/failure rates in log responses
- Response time tracking in metadata
- Database log retention policies

## Troubleshooting

Common issues and solutions:

1. **Webhook not triggering**: Check trigger configuration and message matching
2. **Delivery failures**: Verify webhook URL accessibility and authentication
3. **Performance issues**: Monitor concurrent webhook deliveries and database performance
4. **Security concerns**: Ensure HMAC verification and HTTPS usage

For detailed debugging, check the webhook logs which include full request/response details for each delivery attempt.
