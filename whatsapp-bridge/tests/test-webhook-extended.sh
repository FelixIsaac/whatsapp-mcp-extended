#!/bin/bash

echo "Extended WhatsApp Bridge Webhook API Tests"
echo "=========================================="
echo "Testing missing functionality and edge cases"

BASE_URL="http://localhost:8080"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print test results
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✅ PASS${NC}: $2"
    else
        echo -e "${RED}❌ FAIL${NC}: $2"
    fi
}

# Function to check if server is running
check_server() {
    curl -s -f "$BASE_URL/api/webhooks" > /dev/null
    return $?
}

echo -e "\n📡 Checking server connectivity..."
if ! check_server; then
    echo -e "${RED}❌ Server is not running on $BASE_URL${NC}"
    echo "Please start the WhatsApp bridge server first:"
    echo "./whatsapp-bridge"
    exit 1
fi
echo -e "${GREEN}✅ Server is running${NC}"

# Test 1: Missing Trigger Types
echo -e "\n🔍 TEST GROUP 1: Missing Trigger Types"
echo "======================================"

echo -e "\n1.1 Testing 'sender' trigger type"
SENDER_WEBHOOK='{
  "name": "Sender Trigger Test",
  "webhook_url": "https://httpbin.org/post",
  "enabled": true,
  "triggers": [
    {
      "trigger_type": "sender",
      "trigger_value": "1234567890",
      "match_type": "exact",
      "enabled": true
    }
  ]
}'

RESPONSE=$(curl -s -X POST "$BASE_URL/api/webhooks" \
  -H "Content-Type: application/json" \
  -d "$SENDER_WEBHOOK")

SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
WEBHOOK_ID_1=$(echo "$RESPONSE" | jq -r '.data.id')

if [ "$SUCCESS" = "true" ]; then
    print_result 0 "Sender trigger webhook created (ID: $WEBHOOK_ID_1)"
else
    print_result 1 "Failed to create sender trigger webhook"
fi

echo -e "\n1.2 Testing 'all' trigger type"
ALL_WEBHOOK='{
  "name": "All Messages Trigger Test",
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

SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
WEBHOOK_ID_2=$(echo "$RESPONSE" | jq -r '.data.id')

if [ "$SUCCESS" = "true" ]; then
    print_result 0 "All messages trigger webhook created (ID: $WEBHOOK_ID_2)"
else
    print_result 1 "Failed to create all messages trigger webhook"
fi

# Test 2: Missing Match Types
echo -e "\n🔍 TEST GROUP 2: Missing Match Types"
echo "==================================="

echo -e "\n2.1 Testing 'regex' match type"
REGEX_WEBHOOK='{
  "name": "Regex Match Test",
  "webhook_url": "https://httpbin.org/post",
  "enabled": true,
  "triggers": [
    {
      "trigger_type": "keyword",
      "trigger_value": "urgent|emergency|help",
      "match_type": "regex",
      "enabled": true
    }
  ]
}'

RESPONSE=$(curl -s -X POST "$BASE_URL/api/webhooks" \
  -H "Content-Type: application/json" \
  -d "$REGEX_WEBHOOK")

SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
WEBHOOK_ID_3=$(echo "$RESPONSE" | jq -r '.data.id')

if [ "$SUCCESS" = "true" ]; then
    print_result 0 "Regex match webhook created (ID: $WEBHOOK_ID_3)"
else
    print_result 1 "Failed to create regex match webhook"
fi

# Test 3: Error Handling and Edge Cases
echo -e "\n🔍 TEST GROUP 3: Error Handling & Edge Cases"
echo "============================================"

echo -e "\n3.1 Testing invalid JSON payload"
INVALID_RESPONSE=$(curl -s -X POST "$BASE_URL/api/webhooks" \
  -H "Content-Type: application/json" \
  -d '{"invalid": json}')

SUCCESS=$(echo "$INVALID_RESPONSE" | jq -r '.success // false')
if [ "$SUCCESS" = "false" ]; then
    print_result 0 "Invalid JSON properly rejected"
else
    print_result 1 "Invalid JSON should be rejected"
fi

echo -e "\n3.2 Testing missing required fields"
MISSING_FIELDS='{
  "name": "Test"
}'

MISSING_RESPONSE=$(curl -s -X POST "$BASE_URL/api/webhooks" \
  -H "Content-Type: application/json" \
  -d "$MISSING_FIELDS")

SUCCESS=$(echo "$MISSING_RESPONSE" | jq -r '.success // false')
if [ "$SUCCESS" = "false" ]; then
    print_result 0 "Missing required fields properly rejected"
else
    print_result 1 "Missing required fields should be rejected"
fi

echo -e "\n3.3 Testing invalid webhook URL"
INVALID_URL='{
  "name": "Invalid URL Test",
  "webhook_url": "not-a-valid-url",
  "enabled": true,
  "triggers": []
}'

INVALID_URL_RESPONSE=$(curl -s -X POST "$BASE_URL/api/webhooks" \
  -H "Content-Type: application/json" \
  -d "$INVALID_URL")

SUCCESS=$(echo "$INVALID_URL_RESPONSE" | jq -r '.success // false')
if [ "$SUCCESS" = "false" ]; then
    print_result 0 "Invalid URL properly rejected"
else
    print_result 1 "Invalid URL should be rejected"
fi

echo -e "\n3.4 Testing non-existent webhook retrieval"
NOT_FOUND_RESPONSE=$(curl -s -X GET "$BASE_URL/api/webhooks/9999")
SUCCESS=$(echo "$NOT_FOUND_RESPONSE" | jq -r '.success // false')

if [ "$SUCCESS" = "false" ]; then
    print_result 0 "Non-existent webhook properly returns error"
else
    print_result 1 "Non-existent webhook should return error"
fi

echo -e "\n3.5 Testing non-existent webhook deletion"
DELETE_NOT_FOUND=$(curl -s -X DELETE "$BASE_URL/api/webhooks/9999")
SUCCESS=$(echo "$DELETE_NOT_FOUND" | jq -r '.success // false')

if [ "$SUCCESS" = "false" ]; then
    print_result 0 "Non-existent webhook deletion properly returns error"
else
    print_result 1 "Non-existent webhook deletion should return error"
fi

# Test 4: Enable/Disable Functionality
echo -e "\n🔍 TEST GROUP 4: Enable/Disable Functionality"
echo "============================================="

if [ "$WEBHOOK_ID_1" != "null" ] && [ "$WEBHOOK_ID_1" != "" ]; then
    echo -e "\n4.1 Testing webhook disable"
    DISABLE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/webhooks/$WEBHOOK_ID_1/enable" \
      -H "Content-Type: application/json" \
      -d '{"enabled": false}')
    
    SUCCESS=$(echo "$DISABLE_RESPONSE" | jq -r '.success // false')
    if [ "$SUCCESS" = "true" ]; then
        print_result 0 "Webhook disabled successfully"
    else
        print_result 1 "Failed to disable webhook"
    fi
    
    echo -e "\n4.2 Testing webhook re-enable"
    ENABLE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/webhooks/$WEBHOOK_ID_1/enable" \
      -H "Content-Type: application/json" \
      -d '{"enabled": true}')
    
    SUCCESS=$(echo "$ENABLE_RESPONSE" | jq -r '.success // false')
    if [ "$SUCCESS" = "true" ]; then
        print_result 0 "Webhook enabled successfully"
    else
        print_result 1 "Failed to enable webhook"
    fi
fi

# Test 5: Multiple Triggers per Webhook
echo -e "\n🔍 TEST GROUP 5: Multiple Triggers per Webhook"
echo "=============================================="

echo -e "\n5.1 Testing webhook with multiple different triggers"
MULTI_TRIGGER='{
  "name": "Multi-Trigger Test",
  "webhook_url": "https://httpbin.org/post",
  "enabled": true,
  "triggers": [
    {
      "trigger_type": "keyword",
      "trigger_value": "important",
      "match_type": "contains",
      "enabled": true
    },
    {
      "trigger_type": "media_type",
      "trigger_value": "document",
      "match_type": "exact",
      "enabled": true
    },
    {
      "trigger_type": "sender",
      "trigger_value": "boss@company.com",
      "match_type": "exact",
      "enabled": true
    }
  ]
}'

RESPONSE=$(curl -s -X POST "$BASE_URL/api/webhooks" \
  -H "Content-Type: application/json" \
  -d "$MULTI_TRIGGER")

SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
WEBHOOK_ID_4=$(echo "$RESPONSE" | jq -r '.data.id')
TRIGGER_COUNT=$(echo "$RESPONSE" | jq -r '.data.triggers | length')

if [ "$SUCCESS" = "true" ] && [ "$TRIGGER_COUNT" = "3" ]; then
    print_result 0 "Multi-trigger webhook created with $TRIGGER_COUNT triggers (ID: $WEBHOOK_ID_4)"
else
    print_result 1 "Failed to create multi-trigger webhook"
fi

# Test 6: HMAC Security Testing
echo -e "\n🔍 TEST GROUP 6: HMAC Security Testing"
echo "======================================"

echo -e "\n6.1 Testing webhook with secret token"
SECRET_WEBHOOK='{
  "name": "Secret Token Test",
  "webhook_url": "https://httpbin.org/post",
  "secret_token": "super-secret-key-123",
  "enabled": true,
  "triggers": [
    {
      "trigger_type": "keyword",
      "trigger_value": "secure",
      "match_type": "exact",
      "enabled": true
    }
  ]
}'

RESPONSE=$(curl -s -X POST "$BASE_URL/api/webhooks" \
  -H "Content-Type: application/json" \
  -d "$SECRET_WEBHOOK")

SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
WEBHOOK_ID_5=$(echo "$RESPONSE" | jq -r '.data.id')
HAS_SECRET=$(echo "$RESPONSE" | jq -r '.data.secret_token != ""')

if [ "$SUCCESS" = "true" ] && [ "$HAS_SECRET" = "true" ]; then
    print_result 0 "Webhook with secret token created (ID: $WEBHOOK_ID_5)"
else
    print_result 1 "Failed to create webhook with secret token"
fi

# Test 7: Performance and Stress Testing
echo -e "\n🔍 TEST GROUP 7: Performance Testing"
echo "===================================="

echo -e "\n7.1 Testing rapid webhook creation (stress test)"
SUCCESS_COUNT=0
TOTAL_ATTEMPTS=5

for i in $(seq 1 $TOTAL_ATTEMPTS); do
    STRESS_WEBHOOK="{
      \"name\": \"Stress Test $i\",
      \"webhook_url\": \"https://httpbin.org/post\",
      \"enabled\": true,
      \"triggers\": [
        {
          \"trigger_type\": \"keyword\",
          \"trigger_value\": \"stress$i\",
          \"match_type\": \"exact\",
          \"enabled\": true
        }
      ]
    }"
    
    RESPONSE=$(curl -s -X POST "$BASE_URL/api/webhooks" \
      -H "Content-Type: application/json" \
      -d "$STRESS_WEBHOOK")
    
    SUCCESS=$(echo "$RESPONSE" | jq -r '.success // false')
    if [ "$SUCCESS" = "true" ]; then
        ((SUCCESS_COUNT++))
    fi
done

if [ $SUCCESS_COUNT -eq $TOTAL_ATTEMPTS ]; then
    print_result 0 "Rapid webhook creation test ($SUCCESS_COUNT/$TOTAL_ATTEMPTS)"
else
    print_result 1 "Rapid webhook creation test ($SUCCESS_COUNT/$TOTAL_ATTEMPTS)"
fi

# Test 8: Data Validation and Boundaries
echo -e "\n🔍 TEST GROUP 8: Data Validation & Boundaries"
echo "============================================="

echo -e "\n8.1 Testing extremely long webhook name"
LONG_NAME=$(printf 'A%.0s' {1..1000})
LONG_NAME_WEBHOOK="{
  \"name\": \"$LONG_NAME\",
  \"webhook_url\": \"https://httpbin.org/post\",
  \"enabled\": true,
  \"triggers\": []
}"

RESPONSE=$(curl -s -X POST "$BASE_URL/api/webhooks" \
  -H "Content-Type: application/json" \
  -d "$LONG_NAME_WEBHOOK")

SUCCESS=$(echo "$RESPONSE" | jq -r '.success // false')
if [ "$SUCCESS" = "false" ]; then
    print_result 0 "Extremely long name properly rejected"
else
    print_result 1 "Extremely long name should be rejected"
fi

echo -e "\n8.2 Testing empty trigger list"
EMPTY_TRIGGERS='{
  "name": "Empty Triggers Test",
  "webhook_url": "https://httpbin.org/post",
  "enabled": true,
  "triggers": []
}'

RESPONSE=$(curl -s -X POST "$BASE_URL/api/webhooks" \
  -H "Content-Type: application/json" \
  -d "$EMPTY_TRIGGERS")

SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
if [ "$SUCCESS" = "true" ]; then
    print_result 0 "Webhook with empty triggers allowed"
    EMPTY_WEBHOOK_ID=$(echo "$RESPONSE" | jq -r '.data.id')
else
    print_result 1 "Webhook with empty triggers should be allowed"
fi

# Cleanup Section
echo -e "\n🧹 CLEANUP: Removing test webhooks"
echo "=================================="

# Get all webhooks and delete them
ALL_WEBHOOKS=$(curl -s -X GET "$BASE_URL/api/webhooks")
WEBHOOK_IDS=$(echo "$ALL_WEBHOOKS" | jq -r '.data[].id')

DELETED_COUNT=0
for id in $WEBHOOK_IDS; do
    if [ "$id" != "null" ] && [ "$id" != "" ]; then
        DELETE_RESPONSE=$(curl -s -X DELETE "$BASE_URL/api/webhooks/$id")
        SUCCESS=$(echo "$DELETE_RESPONSE" | jq -r '.success // false')
        if [ "$SUCCESS" = "true" ]; then
            ((DELETED_COUNT++))
        fi
    fi
done

echo "Deleted $DELETED_COUNT test webhooks"

# Final verification
FINAL_COUNT=$(curl -s -X GET "$BASE_URL/api/webhooks" | jq -r '.data | length')
if [ "$FINAL_COUNT" = "0" ]; then
    print_result 0 "All test webhooks cleaned up successfully"
else
    print_result 1 "Cleanup incomplete: $FINAL_COUNT webhooks remain"
fi

echo -e "\n📊 EXTENDED TEST SUMMARY"
echo "========================"
echo "Extended webhook API testing completed!"
echo "Check individual test results above for detailed status."
echo -e "\n${YELLOW}Next: Review logs and implement any failing functionality${NC}"
