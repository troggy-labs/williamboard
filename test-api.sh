#!/bin/bash

# WilliamBoard API Testing Script
# Quick tests for the development server

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

API_BASE="http://localhost:8080"

print_test() {
    echo -e "${BLUE}🧪 Testing: $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

echo "🧪 WilliamBoard API Test Suite"
echo "============================="

# Test 1: Health Check
print_test "Health endpoint"
HEALTH_RESPONSE=$(curl -s "$API_BASE/health" || echo "ERROR")
if echo "$HEALTH_RESPONSE" | grep -q '"status":"ok"'; then
    print_success "Health check passed"
    echo "   Response: $HEALTH_RESPONSE"
else
    print_error "Health check failed"
    echo "   Response: $HEALTH_RESPONSE"
    echo "   Make sure the server is running with './start-dev.sh'"
    exit 1
fi

echo ""

# Test 2: Upload Flow
print_test "Upload signed URL request"
SIGNED_URL_RESPONSE=$(curl -s -X POST "$API_BASE/v1/uploads/signed-url" \
    -H "Content-Type: application/json" \
    -d '{"contentType": "image/jpeg"}' || echo "ERROR")

if echo "$SIGNED_URL_RESPONSE" | grep -q '"submissionId"'; then
    print_success "Signed URL generated"
    
    # Extract submission ID
    SUBMISSION_ID=$(echo "$SIGNED_URL_RESPONSE" | grep -o '"submissionId":"[^"]*"' | cut -d'"' -f4)
    print_info "Submission ID: $SUBMISSION_ID"
    
    UPLOAD_URL=$(echo "$SIGNED_URL_RESPONSE" | grep -o '"url":"[^"]*"' | cut -d'"' -f4)
    print_info "Upload URL: $UPLOAD_URL"
else
    print_error "Failed to get signed URL"
    echo "   Response: $SIGNED_URL_RESPONSE"
    exit 1
fi

echo ""

# Test 3: File Upload (create a test image)
print_test "File upload with test image"

# Create a minimal test JPEG (1x1 pixel red square)
TEST_IMAGE="test_image.jpg"
# This creates a minimal valid JPEG file
printf '\xff\xd8\xff\xe0\x00\x10JFIF\x00\x01\x01\x01\x00H\x00H\x00\x00\xff\xdb\x00C\x00\x08\x06\x06\x07\x06\x05\x08\x07\x07\x07\t\t\x08\n\x0c\x14\r\x0c\x0b\x0b\x0c\x19\x12\x13\x0f\x14\x1d\x1a\x1f\x1e\x1d\x1a\x1c\x1c $.\x27 ",#\x1c\x1c(7),01444\x1f'"'"'9=82<.342\xff\xc0\x00\x11\x08\x00\x01\x00\x01\x01\x01\x11\x00\x02\x11\x01\x03\x11\x01\xff\xc4\x00\x14\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x08\xff\xc4\x00\x14\x10\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xff\xda\x00\x0c\x03\x01\x00\x02\x11\x03\x11\x00\x3f\x00\x00\xff\xd9' > "$TEST_IMAGE"

if [ -f "$TEST_IMAGE" ]; then
    UPLOAD_RESPONSE=$(curl -s -X PUT "$UPLOAD_URL" \
        -F "file=@$TEST_IMAGE" || echo "ERROR")
    
    if echo "$UPLOAD_RESPONSE" | grep -q '"message":"Image processed successfully"'; then
        print_success "File upload and processing completed"
        
        # Extract results
        EVENTS_FOUND=$(echo "$UPLOAD_RESPONSE" | grep -o '"eventsFound":[0-9]*' | cut -d':' -f2)
        FLYERS_FOUND=$(echo "$UPLOAD_RESPONSE" | grep -o '"flyersFound":[0-9]*' | cut -d':' -f2)
        
        print_info "Events found: $EVENTS_FOUND"
        print_info "Flyers found: $FLYERS_FOUND"
        
        if [ "$EVENTS_FOUND" -eq 0 ] && [ "$FLYERS_FOUND" -eq 0 ]; then
            print_info "Expected result for test image (Stage 1 mode or no events in image)"
        fi
    else
        print_error "File upload failed"
        echo "   Response: $UPLOAD_RESPONSE"
    fi
    
    # Clean up test file
    rm -f "$TEST_IMAGE"
else
    print_error "Failed to create test image"
fi

echo ""

# Test 4: Status Check
if [ ! -z "$SUBMISSION_ID" ]; then
    print_test "Submission status check"
    STATUS_RESPONSE=$(curl -s "$API_BASE/v1/submissions/$SUBMISSION_ID/status" || echo "ERROR")
    
    if echo "$STATUS_RESPONSE" | grep -q '"status"'; then
        print_success "Status check passed"
        
        STATUS=$(echo "$STATUS_RESPONSE" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
        print_info "Status: $STATUS"
    else
        print_error "Status check failed"
        echo "   Response: $STATUS_RESPONSE"
    fi
fi

echo ""

# Test 5: Events endpoint
print_test "Events API"
EVENTS_RESPONSE=$(curl -s "$API_BASE/v1/events" || echo "ERROR")

if echo "$EVENTS_RESPONSE" | grep -q '"type":"FeatureCollection"'; then
    print_success "Events API working"
    
    FEATURE_COUNT=$(echo "$EVENTS_RESPONSE" | grep -o '"features":\[[^\]]*\]' | tr ',' '\n' | grep -c '{' || echo "0")
    print_info "Published events: $FEATURE_COUNT"
else
    print_error "Events API failed"
    echo "   Response: $EVENTS_RESPONSE"
fi

echo ""
echo "🎉 API Test Summary"
echo "=================="

# Check if server has OpenAI configured
if curl -s "$API_BASE/health" | grep -q "ok"; then
    if [ -f ".env" ] && grep -q "OPENAI_API_KEY=sk-" .env 2>/dev/null; then
        print_success "Stage 2 Ready: OpenAI API key configured"
        echo "   • Try uploading a real bulletin board image!"
        echo "   • Upload URL: $API_BASE/v1/uploads/signed-url"
    else
        print_info "Stage 1 Mode: Basic infrastructure working"
        echo "   • Add OPENAI_API_KEY to .env for GPT-4o vision analysis"
        echo "   • Current functionality: file upload, storage, database"
    fi
else
    print_error "Server not responding. Start with './start-dev.sh'"
fi

echo ""
echo "📖 Next steps:"
echo "   • See README.md for detailed API documentation"
echo "   • Upload real flyer images to test Stage 2"
echo "   • Check database with: psql williamboard"