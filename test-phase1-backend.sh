#!/bin/bash
# test-phase1-backend.sh

BASE_URL="http://localhost:8080"
TIMESTAMP=$(date +%s)

echo "=== Test 1: Registration ==="
RESPONSE_FULL=$(curl -s -X POST "$BASE_URL/api/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"testuser_$TIMESTAMP\",\"email\":\"test_$TIMESTAMP@example.com\",\"password\":\"test12345678\",\"encryption_password\":\"encryption123\"}")

echo "Registration Response: $RESPONSE_FULL"

RESPONSE=$(echo "$RESPONSE_FULL")

TOKEN=$(echo $RESPONSE | jq -r '.access_token')
echo "Token: ${TOKEN:0:20}..."

echo "=== Test 2: Create Encrypted Note ==="
NOTE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/notes" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"TestNote","encrypted_content":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAdGVzdA==","content_encrypted":true,"wrapped_dek":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==","encryption_version":2,"encryption_metadata":"{\"version\":2,\"algorithm\":\"XChaCha20-Poly1305\",\"wrapped_dek\":\"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==\"}"}')

NOTE_ID=$(echo $NOTE_RESPONSE | jq -r '.id')
echo "Note ID: $NOTE_ID"

echo "=== Test 3: Retrieve Note ==="
curl -s -X GET "$BASE_URL/api/notes/$NOTE_ID" \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.encrypted_content' | head -c 50

echo -e "\n\n✅ Backend tests completed"
