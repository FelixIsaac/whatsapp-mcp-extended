#!/bin/bash

echo "WhatsApp Message Processing & Webhook Integration Test"
echo "====================================================="
echo "Testing the integration between message handling and webhook triggers"

BASE_URL="http://localhost:8080"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print test results
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✅ PASS${NC}: $2"
    else
        echo -e "${RED}❌ FAIL${NC}: $2"
    fi
}

echo -e "\n📋 MANUAL INTEGRATION TESTS"
echo "============================"
echo "These tests require manual interaction with WhatsApp"
echo "to verify that webhook triggers work with actual messages."

# Create comprehensive test webhooks
echo -e "\n🔧 Setting up test webhooks..."

# Webhook 1: Keyword trigger
echo "Creating keyword trigger webhook..."
KEYWORD_WEBHOOK='{
  "name": "Manual Test - Keywords",
  "webhook_url": "https://httpbin.org/post",
  "secret_token": "manual-test-secret",
  "enabled": true,
  "triggers": [
    {
      "trigger_type": "keyword",
      "trigger_value": "webhook",
      "match_type": "contains",
      "enabled": true
    }
  ]
}'

RESPONSE=$(curl -s -X POST "$BASE_URL/api/webhooks" \
  -H "Content-Type: application/json" \
  -d "$KEYWORD_WEBHOOK")

KEYWORD_ID=$(echo "$RESPONSE" | jq -r '.data.id // "null"')
echo "Keyword webhook ID: $KEYWORD_ID"

# Webhook 2: Regex trigger
echo "Creating regex trigger webhook..."
REGEX_WEBHOOK='{
  "name": "Manual Test - Regex",
  "webhook_url": "https://httpbin.org/post",
  "secret_token": "manual-test-secret",
  "enabled": true,
  "triggers": [
    {
      "trigger_type": "keyword",
      "trigger_value": "test|demo|example",
      "match_type": "regex",
      "enabled": true
    }
  ]
}'

RESPONSE=$(curl -s -X POST "$BASE_URL/api/webhooks" \
  -H "Content-Type: application/json" \
  -d "$REGEX_WEBHOOK")

REGEX_ID=$(echo "$RESPONSE" | jq -r '.data.id // "null"')
echo "Regex webhook ID: $REGEX_ID"

# Webhook 3: All messages
echo "Creating all messages webhook..."
ALL_WEBHOOK='{
  "name": "Manual Test - All Messages",
  "webhook_url": "https://httpbin.org/post",
  "enabled": true,
  "triggers": [
    {
      "trigger_type": "all",
      "trigger_value": "",
      "match_type": "exact",
      "enabled": true
    }
  ]
}'

RESPONSE=$(curl -s -X POST "$BASE_URL/api/webhooks" \
  -H "Content-Type: application/json" \
  -d "$ALL_WEBHOOK")

ALL_ID=$(echo "$RESPONSE" | jq -r '.data.id // "null"')
echo "All messages webhook ID: $ALL_ID"

echo -e "\n🔍 MANUAL TEST INSTRUCTIONS"
echo "============================"
echo "Please perform the following tests manually:"
echo ""

echo -e "${BLUE}Test 1: Keyword Trigger Test${NC}"
echo "1. Send a WhatsApp message containing the word 'webhook'"
echo "2. Check webhook logs after sending:"
echo "   curl -s \"$BASE_URL/api/webhooks/$KEYWORD_ID/logs\" | jq '.'"
echo ""

echo -e "${BLUE}Test 2: Regex Trigger Test${NC}"
echo "1. Send a WhatsApp message containing 'test', 'demo', or 'example'"
echo "2. Check webhook logs after sending:"
echo "   curl -s \"$BASE_URL/api/webhooks/$REGEX_ID/logs\" | jq '.'"
echo ""

echo -e "${BLUE}Test 3: All Messages Test${NC}"
echo "1. Send any WhatsApp message"
echo "2. Check webhook logs after sending:"
echo "   curl -s \"$BASE_URL/api/webhooks/$ALL_ID/logs\" | jq '.'"
echo ""

echo -e "${BLUE}Test 4: Media Message Test${NC}"
echo "First, create a media webhook:"
echo "curl -s -X POST \"$BASE_URL/api/webhooks\" \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  -d '{"
echo "    \"name\": \"Media Test\","
echo "    \"webhook_url\": \"https://httpbin.org/post\","
echo "    \"enabled\": true,"
echo "    \"triggers\": ["
echo "      {"
echo "        \"trigger_type\": \"media_type\","
echo "        \"trigger_value\": \"image\","
echo "        \"match_type\": \"exact\","
echo "        \"enabled\": true"
echo "      }"
echo "    ]"
echo "  }'"
echo ""
echo "Then send an image via WhatsApp and check the logs."
echo ""

echo -e "${BLUE}Test 5: Specific Sender Test${NC}"
echo "Create a sender-specific webhook (replace SENDER_JID with actual JID):"
echo "curl -s -X POST \"$BASE_URL/api/webhooks\" \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  -d '{"
echo "    \"name\": \"Sender Test\","
echo "    \"webhook_url\": \"https://httpbin.org/post\","
echo "    \"enabled\": true,"
echo "    \"triggers\": ["
echo "      {"
echo "        \"trigger_type\": \"sender\","
echo "        \"trigger_value\": \"SENDER_JID@s.whatsapp.net\","
echo "        \"match_type\": \"exact\","
echo "        \"enabled\": true"
echo "      }"
echo "    ]"
echo "  }'"
echo ""

echo -e "${BLUE}Test 6: Group Chat Test${NC}"
echo "Create a group-specific webhook (replace GROUP_JID with actual JID):"
echo "curl -s -X POST \"$BASE_URL/api/webhooks\" \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  -d '{"
echo "    \"name\": \"Group Test\","
echo "    \"webhook_url\": \"https://httpbin.org/post\","
echo "    \"enabled\": true,"
echo "    \"triggers\": ["
echo "      {"
echo "        \"trigger_type\": \"chat_jid\","
echo "        \"trigger_value\": \"GROUP_JID@g.us\","
echo "        \"match_type\": \"exact\","
echo "        \"enabled\": true"
echo "      }"
echo "    ]"
echo "  }'"
echo ""

echo -e "\n🔧 HELPER COMMANDS"
echo "=================="
echo "View all webhooks:"
echo "curl -s \"$BASE_URL/api/webhooks\" | jq '.'"
echo ""
echo "View all webhook logs:"
echo "curl -s \"$BASE_URL/api/webhook-logs\" | jq '.'"
echo ""
echo "Delete a webhook (replace ID):"
echo "curl -s -X DELETE \"$BASE_URL/api/webhooks/ID\""
echo ""
echo "Disable a webhook temporarily:"
echo "curl -s -X POST \"$BASE_URL/api/webhooks/ID/enable\" \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  -d '{\"enabled\": false}'"
echo ""

echo -e "\n⏰ MONITORING SCRIPT"
echo "==================="
echo "Run this script to monitor webhook logs in real-time:"

cat > monitor_webhooks.sh << 'EOF'
#!/bin/bash
BASE_URL="http://localhost:8080"

echo "📊 Webhook Monitoring Started"
echo "============================="
echo "Press Ctrl+C to stop"

LAST_COUNT=0

while true; do
    # Get current log count
    CURRENT_LOGS=$(curl -s "$BASE_URL/api/webhook-logs" | jq -r '.data | length // 0')
    
    if [ "$CURRENT_LOGS" -gt "$LAST_COUNT" ]; then
        NEW_LOGS=$((CURRENT_LOGS - LAST_COUNT))
        echo -e "\n🔔 $NEW_LOGS new webhook delivery(s) detected!"
        
        # Show recent logs
        curl -s "$BASE_URL/api/webhook-logs" | jq -r '.data[-'$NEW_LOGS':][] | 
            "🕐 " + .created_at + " | Webhook: " + (.webhook_config_id | tostring) + 
            " | Status: " + (.response_status | tostring) + " | Message: " + .message_id'
        
        LAST_COUNT=$CURRENT_LOGS
    fi
    
    sleep 2
done
EOF

chmod +x monitor_webhooks.sh

echo "Created monitor_webhooks.sh - run it to watch webhook deliveries in real-time"
echo ""

echo -e "\n🧪 AUTOMATED VALIDATION TESTS"
echo "============================="

# Test that webhooks are created correctly
echo "Validating created webhooks..."

for webhook_id in $KEYWORD_ID $REGEX_ID $ALL_ID; do
    if [ "$webhook_id" != "null" ]; then
        WEBHOOK_DATA=$(curl -s "$BASE_URL/api/webhooks/$webhook_id")
        WEBHOOK_NAME=$(echo "$WEBHOOK_DATA" | jq -r '.data.name')
        TRIGGER_COUNT=$(echo "$WEBHOOK_DATA" | jq -r '.data.triggers | length')
        
        if [ "$TRIGGER_COUNT" -gt "0" ]; then
            print_result 0 "Webhook $webhook_id ($WEBHOOK_NAME) has $TRIGGER_COUNT trigger(s)"
        else
            print_result 1 "Webhook $webhook_id ($WEBHOOK_NAME) has no triggers"
        fi
    fi
done

# Test webhook connectivity
echo -e "\nTesting webhook connectivity..."
for webhook_id in $KEYWORD_ID $REGEX_ID $ALL_ID; do
    if [ "$webhook_id" != "null" ]; then
        TEST_RESPONSE=$(curl -s -X POST "$BASE_URL/api/webhooks/$webhook_id/test")
        SUCCESS=$(echo "$TEST_RESPONSE" | jq -r '.success // false')
        
        if [ "$SUCCESS" = "true" ]; then
            print_result 0 "Webhook $webhook_id connectivity test passed"
        else
            print_result 1 "Webhook $webhook_id connectivity test failed"
        fi
    fi
done

echo -e "\n📋 NEXT STEPS"
echo "============="
echo "1. Perform the manual tests above by sending WhatsApp messages"
echo "2. Run monitor_webhooks.sh in another terminal to see real-time webhook deliveries"
echo "3. Check webhook logs after each test using the provided curl commands"
echo "4. Verify that webhook payloads contain correct message data"
echo "5. Test HMAC signature verification if secret tokens are used"
echo ""
echo -e "${YELLOW}Important: Keep the WhatsApp bridge server running during these tests!${NC}"
echo ""
echo "When finished testing, clean up with:"
echo "./test-webhook-api.sh  # This will remove all test webhooks"
