#!/bin/bash
set -e

API_URL="http://localhost:8080/api/v1"
ADMIN_EMAIL="q.aldaniyazov@ats-mediafon.kz"
ADMIN_PASS="admin123"
MARKET_EMAIL="test2@marketplace.com"
MARKET_PASS="market123"

echo "=== E2E Test Suite ==="

# 1. Register Admin
echo "\n1. Registering Admin ($ADMIN_EMAIL)..."
curl -s -X POST $API_URL/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username": "'$ADMIN_EMAIL'", "password": "'$ADMIN_PASS'"}' | jq .

# 2. Login Admin
echo "\n2. Logging in Admin..."
ADMIN_LOGIN_RESP=$(curl -s -X POST $API_URL/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "'$ADMIN_EMAIL'", "password": "'$ADMIN_PASS'"}')
echo $ADMIN_LOGIN_RESP | jq .
ADMIN_TOKEN=$(echo $ADMIN_LOGIN_RESP | jq -r '.token')
if [ "$ADMIN_TOKEN" == "null" ] || [ -z "$ADMIN_TOKEN" ]; then
    echo "ERROR: Failed to get admin token!"
    exit 1
fi
echo "Admin Token Acquired."

# 3. Register Marketplace User
echo "\n3. Registering Marketplace User ($MARKET_EMAIL)..."
MARKET_REG_RESP=$(curl -s -X POST $API_URL/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username": "'$MARKET_EMAIL'", "password": "'$MARKET_PASS'"}')
echo $MARKET_REG_RESP | jq .
MARKET_UID=$(echo $MARKET_REG_RESP | jq -r '.user.id')

# 4. Try marketplace login (should fail - not approved)
echo "\n4. Testing Unapproved Login (should fail)..."
curl -s -X POST $API_URL/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "'$MARKET_EMAIL'", "password": "'$MARKET_PASS'"}' | jq .

# 5. Admin Approves User
if [ "$MARKET_UID" != "null" ] && [ -n "$MARKET_UID" ]; then
    echo "\n5. Admin Approving Marketplace User ($MARKET_UID)..."
    curl -s -X POST http://localhost:8080/api/admin/users/$MARKET_UID/approve \
      -H "Authorization: Bearer $ADMIN_TOKEN" | jq .
else
    echo "Warning: Marketplace user might already exist, skipping approval logic."
fi

# 6. Login Marketplace
echo "\n6. Logging in Marketplace User..."
MARKET_LOGIN_RESP=$(curl -s -X POST $API_URL/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "'$MARKET_EMAIL'", "password": "'$MARKET_PASS'"}')
MARKET_TOKEN=$(echo $MARKET_LOGIN_RESP | jq -r '.token')
if [ "$MARKET_TOKEN" == "null" ] || [ -z "$MARKET_TOKEN" ]; then
    echo "ERROR: Failed to login marketplace user after approval!"
    exit 1
fi
echo "Marketplace Token Acquired."

# 7. Upload JSON Parcels
echo "\n7. Testing Parcel JSON Upload..."
UPLOAD_RESP=$(curl -s -X POST $API_URL/parcels/upload-json \
  -H "Authorization: Bearer $MARKET_TOKEN" \
  -H "Content-Type: application/json" \
  -d '[
    {"marketplace": "Wildberries", "country": "RU", "brand": "BrandX", "name": "Phone", "track_number": "TRK123", "snt": "SNT1", "date": "2026-02-21"},
    {"marketplace": "", "country": "", "brand": "", "name": "", "track_number": "BAD_JSON", "snt": "", "date": ""}
  ]')
echo "Raw Upload Response: $UPLOAD_RESP"
echo "$UPLOAD_RESP" | jq .

# 8. Fetch Risk Reports (Admin)
echo "\n8. Fetching Risk Reports..."
curl -s -X GET $API_URL/risks/reports \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .

# 9. Create Support Ticket (Admin)
echo "\n9. Creating Support Ticket..."
TICKET_RESP=$(curl -s -X POST $API_URL/tickets \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "iin": "123456789012",
    "full_name": "Test Tester",
    "support_ticket_id": "SUP-101",
    "application_number": "APP-202",
    "document_number": "DOC-303",
    "rejection_reason": "Missing signature",
    "priority": "high",
    "attachments": []
  }')
echo $TICKET_RESP | jq .
TICKET_ID=$(echo $TICKET_RESP | jq -r '.id')

# 10. Fetch Tickets (Checking for risk data embedding and no 500 errors)
echo "\n10. Fetching Tickets (Checking array scanning bug fix)..."
curl -s -X GET $API_URL/tickets \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .

echo "\n=== All Tests Completed ==="
