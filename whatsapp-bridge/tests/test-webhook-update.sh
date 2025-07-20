#!/bin/bash

# Test script to verify webhook update functionality
API_BASE="http://localhost:8080/api"

echo "Testing webhook update functionality..."

# Create a test webhook first
echo "1. Creating test webhook..."
CREATE_RESPONSE=$(curl -s -X POST "$API_BASE/webhooks" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Webhook",
    "webhook_url": "https://example.com/test",
    "secret_token": "test-secret",
    "enabled": true,
    "triggers": [
      {
        "trigger_type": "keyword",
        "trigger_value": "test",
        "match_type": "contains",
        "enabled": true
      }
    ]
  }')

echo "Create response: $CREATE_RESPONSE"

# Extract the webhook ID from the response
WEBHOOK_ID=$(echo "$CREATE_RESPONSE" | grep -o '"id":[0-9]*' | grep -o '[0-9]*')
echo "Created webhook with ID: $WEBHOOK_ID"

if [ -z "$WEBHOOK_ID" ]; then
    echo "Failed to create webhook"
    exit 1
fi

# Wait a moment
sleep 1

# Get the webhook to verify it was created
echo "2. Getting webhook details..."
GET_RESPONSE=$(curl -s -X GET "$API_BASE/webhooks/$WEBHOOK_ID")
echo "Get response: $GET_RESPONSE"

# Update the webhook with new triggers
echo "3. Updating webhook with new triggers..."
UPDATE_RESPONSE=$(curl -s -X PUT "$API_BASE/webhooks/$WEBHOOK_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "id": '$WEBHOOK_ID',
    "name": "Updated Test Webhook",
    "webhook_url": "https://example.com/updated",
    "secret_token": "updated-secret",
    "enabled": true,
    "triggers": [
      {
        "trigger_type": "keyword",
        "trigger_value": "urgent",
        "match_type": "contains",
        "enabled": true
      },
      {
        "trigger_type": "sender",
        "trigger_value": "123456@s.whatsapp.net",
        "match_type": "exact",
        "enabled": true
      }
    ]
  }')

echo "Update response: $UPDATE_RESPONSE"

# Wait a moment
sleep 1

# Get the webhook again to verify the update
echo "4. Getting updated webhook details..."
GET_UPDATED_RESPONSE=$(curl -s -X GET "$API_BASE/webhooks/$WEBHOOK_ID")
echo "Get updated response: $GET_UPDATED_RESPONSE"

# Check if triggers were updated
if echo "$GET_UPDATED_RESPONSE" | grep -q '"trigger_value":"urgent"'; then
    echo "✓ Webhook triggers updated successfully!"
else
    echo "✗ Webhook triggers were not updated"
fi

# Clean up - delete the test webhook
echo "5. Cleaning up..."
DELETE_RESPONSE=$(curl -s -X DELETE "$API_BASE/webhooks/$WEBHOOK_ID")
echo "Delete response: $DELETE_RESPONSE"

echo "Test completed."
