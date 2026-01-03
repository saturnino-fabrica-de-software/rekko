#!/bin/bash

# ========================================
# Rekko API Endpoint Tests
# ========================================
# Collection of cURL commands to test all endpoints
# with the test tenant
#
# Usage:
#   ./scripts/test_endpoints.sh [command]
#
# Commands:
#   health      - Health check
#   register    - Register a face
#   verify      - Verify a face
#   search      - Search similar faces
#   get         - Get face details
#   delete      - Delete a face
#   list        - List verifications
#   all         - Run all tests in sequence
# ========================================

set -e

# Configuration
API_BASE_URL=${API_BASE_URL:-"http://localhost:8080"}
API_KEY=${API_KEY:-"test-api-key-rekko-dev"}
EXTERNAL_ID=${EXTERNAL_ID:-"user-test-123"}

# Sample 1x1 pixel base64 image (for testing structure)
# For real tests, replace with actual face image
SAMPLE_IMAGE="iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_header() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

print_request() {
    echo -e "${YELLOW}Request:${NC}"
    echo "$1"
    echo ""
}

print_response() {
    echo -e "${GREEN}Response:${NC}"
    echo "$1" | jq '.' 2>/dev/null || echo "$1"
    echo ""
}

# Test commands

test_health() {
    print_header "Health Check"

    CMD="curl -s ${API_BASE_URL}/health"
    print_request "$CMD"

    RESPONSE=$(eval "$CMD")
    print_response "$RESPONSE"
}

test_register() {
    print_header "Register Face"

    CMD="curl -s -X POST ${API_BASE_URL}/api/v1/faces \\
  -H \"X-API-Key: ${API_KEY}\" \\
  -H \"Content-Type: application/json\" \\
  -d '{
    \"external_id\": \"${EXTERNAL_ID}\",
    \"image_base64\": \"${SAMPLE_IMAGE}\",
    \"metadata\": {
      \"name\": \"Test User\",
      \"event\": \"Test Event 2024\",
      \"ticket_number\": \"T-12345\"
    }
  }'"

    print_request "$CMD"

    RESPONSE=$(curl -s -X POST "${API_BASE_URL}/api/v1/faces" \
      -H "X-API-Key: ${API_KEY}" \
      -H "Content-Type: application/json" \
      -d "{
        \"external_id\": \"${EXTERNAL_ID}\",
        \"image_base64\": \"${SAMPLE_IMAGE}\",
        \"metadata\": {
          \"name\": \"Test User\",
          \"event\": \"Test Event 2024\",
          \"ticket_number\": \"T-12345\"
        }
      }")

    print_response "$RESPONSE"
}

test_verify() {
    print_header "Verify Face"

    CMD="curl -s -X POST ${API_BASE_URL}/api/v1/verify \\
  -H \"X-API-Key: ${API_KEY}\" \\
  -H \"Content-Type: application/json\" \\
  -d '{
    \"external_id\": \"${EXTERNAL_ID}\",
    \"image_base64\": \"${SAMPLE_IMAGE}\"
  }'"

    print_request "$CMD"

    RESPONSE=$(curl -s -X POST "${API_BASE_URL}/api/v1/verify" \
      -H "X-API-Key: ${API_KEY}" \
      -H "Content-Type: application/json" \
      -d "{
        \"external_id\": \"${EXTERNAL_ID}\",
        \"image_base64\": \"${SAMPLE_IMAGE}\"
      }")

    print_response "$RESPONSE"
}

test_search() {
    print_header "Search Similar Faces"

    CMD="curl -s -X POST ${API_BASE_URL}/api/v1/search \\
  -H \"X-API-Key: ${API_KEY}\" \\
  -H \"Content-Type: application/json\" \\
  -d '{
    \"image_base64\": \"${SAMPLE_IMAGE}\",
    \"threshold\": 0.8,
    \"limit\": 10
  }'"

    print_request "$CMD"

    RESPONSE=$(curl -s -X POST "${API_BASE_URL}/api/v1/search" \
      -H "X-API-Key: ${API_KEY}" \
      -H "Content-Type: application/json" \
      -d "{
        \"image_base64\": \"${SAMPLE_IMAGE}\",
        \"threshold\": 0.8,
        \"limit\": 10
      }")

    print_response "$RESPONSE"
}

test_get() {
    print_header "Get Face Details"

    CMD="curl -s ${API_BASE_URL}/api/v1/faces/${EXTERNAL_ID} \\
  -H \"X-API-Key: ${API_KEY}\""

    print_request "$CMD"

    RESPONSE=$(curl -s "${API_BASE_URL}/api/v1/faces/${EXTERNAL_ID}" \
      -H "X-API-Key: ${API_KEY}")

    print_response "$RESPONSE"
}

test_delete() {
    print_header "Delete Face"

    CMD="curl -s -X DELETE ${API_BASE_URL}/api/v1/faces/${EXTERNAL_ID} \\
  -H \"X-API-Key: ${API_KEY}\""

    print_request "$CMD"

    RESPONSE=$(curl -s -X DELETE "${API_BASE_URL}/api/v1/faces/${EXTERNAL_ID}" \
      -H "X-API-Key: ${API_KEY}")

    print_response "$RESPONSE"
}

test_list() {
    print_header "List Verifications (Audit Log)"

    CMD="curl -s \"${API_BASE_URL}/api/v1/verifications?limit=10&offset=0\" \\
  -H \"X-API-Key: ${API_KEY}\""

    print_request "$CMD"

    RESPONSE=$(curl -s "${API_BASE_URL}/api/v1/verifications?limit=10&offset=0" \
      -H "X-API-Key: ${API_KEY}")

    print_response "$RESPONSE"
}

test_all() {
    echo -e "${GREEN}Running all endpoint tests...${NC}"

    test_health
    sleep 1

    test_register
    sleep 1

    test_get
    sleep 1

    test_verify
    sleep 1

    test_search
    sleep 1

    test_list
    sleep 1

    # Clean up
    test_delete

    echo ""
    echo -e "${GREEN}✅ All tests completed!${NC}"
}

# Main
case "${1:-help}" in
    health)
        test_health
        ;;
    register)
        test_register
        ;;
    verify)
        test_verify
        ;;
    search)
        test_search
        ;;
    get)
        test_get
        ;;
    delete)
        test_delete
        ;;
    list)
        test_list
        ;;
    all)
        test_all
        ;;
    *)
        echo "Rekko API Endpoint Tests"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  health      - Health check"
        echo "  register    - Register a face"
        echo "  verify      - Verify a face"
        echo "  search      - Search similar faces"
        echo "  get         - Get face details"
        echo "  delete      - Delete a face"
        echo "  list        - List verifications"
        echo "  all         - Run all tests in sequence"
        echo ""
        echo "Environment variables:"
        echo "  API_BASE_URL  - API base URL (default: http://localhost:8080)"
        echo "  API_KEY       - API key (default: test-api-key-rekko-dev)"
        echo "  EXTERNAL_ID   - External ID for testing (default: user-test-123)"
        echo ""
        echo "Example:"
        echo "  $0 health"
        echo "  $0 all"
        echo "  API_BASE_URL=http://localhost:3000 $0 register"
        ;;
esac
