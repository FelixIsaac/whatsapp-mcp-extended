#!/bin/bash

echo "Testing WhatsApp Bridge Webhook API"
echo "==================================="

BASE_URL="http://localhost:8080"

# Test 1: List all webhooks (should be empty initially)
echo -e "\n1. Testing GET /api/webhooks (list all webhooks)"
curl -s -X GET "$BASE_URL/api/webhooks" | jq '.'

# Test 2: Create a test webhook
echo -e "\n2. Testing POST /api/webhooks (create new webhook)"
WEBHOOK_CONFIG='{
  "name": "Test Webhook",
  "webhook_url": "https://httpbin.org/post",
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
}'

RESPONSE=$(curl -s -X POST "$BASE_URL/api/webhooks" \
  -H "Content-Type: application/json" \
  -d "$WEBHOOK_CONFIG")

echo "$RESPONSE" | jq '.'

# Extract webhook ID from response
WEBHOOK_ID=$(echo "$RESPONSE" | jq -r '.data.id')

if [ "$WEBHOOK_ID" != "null" ] && [ "$WEBHOOK_ID" != "" ]; then
  echo -e "\nWebhook created with ID: $WEBHOOK_ID"
  
  # Test 3: Get specific webhook
  echo -e "\n3. Testing GET /api/webhooks/$WEBHOOK_ID (get specific webhook)"
  curl -s -X GET "$BASE_URL/api/webhooks/$WEBHOOK_ID" | jq '.'
  
  # Test 4: Test webhook connectivity
  echo -e "\n4. Testing POST /api/webhooks/$WEBHOOK_ID/test (test webhook)"
  curl -s -X POST "$BASE_URL/api/webhooks/$WEBHOOK_ID/test" | jq '.'
  
  # Test 5: Get webhook logs
  echo -e "\n5. Testing GET /api/webhooks/$WEBHOOK_ID/logs (get webhook logs)"
  curl -s -X GET "$BASE_URL/api/webhooks/$WEBHOOK_ID/logs" | jq '.'
  
  # Test 6: Update webhook (disable it)
  echo -e "\n6. Testing PUT /api/webhooks/$WEBHOOK_ID (update webhook)"
  UPDATE_CONFIG='{
    "id": '$WEBHOOK_ID',
    "name": "Test Webhook (Updated)",
    "webhook_url": "https://httpbin.org/post",
    "secret_token": "test-secret-updated",
    "enabled": false,
    "triggers": [
      {
        "trigger_type": "keyword",
        "trigger_value": "updated",
        "match_type": "contains",
        "enabled": true
      }
    ]
  }'
  
  curl -s -X PUT "$BASE_URL/api/webhooks/$WEBHOOK_ID" \
    -H "Content-Type: application/json" \
    -d "$UPDATE_CONFIG" | jq '.'
  
  # Test 7: Delete webhook
  echo -e "\n7. Testing DELETE /api/webhooks/$WEBHOOK_ID (delete webhook)"
  curl -s -X DELETE "$BASE_URL/api/webhooks/$WEBHOOK_ID" | jq '.'
  
else
  echo "Failed to create webhook, skipping subsequent tests"
fi

# Test 8: List webhooks again (should be empty if delete worked)
echo -e "\n8. Testing GET /api/webhooks (list all webhooks after delete)"
curl -s -X GET "$BASE_URL/api/webhooks" | jq '.'

echo -e "\n\nTest complete!"
