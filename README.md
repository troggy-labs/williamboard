# WilliamBoard

A bulletin board photo → event extraction → auto-publish system using GPT-4o Structured Outputs, deployed entirely on Render.

## Overview

WilliamBoard processes photos of bulletin boards and flyers to automatically extract event information and publish structured event data. The system uses computer vision and LLM processing to identify flyers, extract event details, geocode locations, and publish events with confidence scoring.

## Truly Synchronous Architecture (No AWS, No Workers)

- **API Server Only** (`/api`) - Single service with **direct GPT-4o processing**
- **PostgreSQL** - Event data, flyers, venues with PostGIS for location  
- **Render Persistent Disk** - File storage for uploaded images
- **GPT-4o Vision** - Direct API calls for flyer detection and event extraction

### Key UX Innovation: Synchronous Processing

Unlike typical async upload systems, WilliamBoard processes images **immediately** during upload:
- ✅ **Instant feedback**: "Found 3 events!" or "No events detected"
- ✅ **No waiting**: Users know immediately if their photo worked
- ✅ **Clear success**: No confusing "uploaded but processing" states
- ✅ **Better errors**: Failed processing = clear retry action

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 14+ with PostGIS extension (for local development)
- OpenAI API key (for Stage 2+)

### 1. Environment Setup

```bash
# Copy environment template
cp .env.example .env

# Edit .env with your credentials
vi .env
```

Required environment variables for local development:
```bash
# Application
APP_NAME=WilliamBoard
PUBLIC_BASE_URL=http://localhost:8080
PORT=8080
ENVIRONMENT=development

# Database (for local development)
DATABASE_URL=postgres://user:pass@localhost:5432/williamboard?sslmode=disable

# File storage (local development)
UPLOAD_DIR=./uploads

# OpenAI (required for Stage 2+ - real flyer detection)
OPENAI_API_KEY=your-openai-api-key-here

# Geocoding (optional for Stage 3+)
GEOCODER_API_KEY=your-mapbox-api-key
```

### 2. Database Setup (Local Development)

```bash
# Create PostgreSQL database
createdb williamboard

# Enable required extensions
psql williamboard -c "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";"
psql williamboard -c "CREATE EXTENSION IF NOT EXISTS \"postgis\";"

# Run migrations
psql williamboard < migrations/001_initial_schema.sql
```

### 3. Render Deployment (Production)

**No AWS setup needed!** Just:

1. Fork this repository
2. Connect to Render and deploy using `render.yaml`
3. Set `OPENAI_API_KEY` in Render dashboard
4. Database and persistent disk are automatically provisioned

### 4. Build and Run (Local Development)

```bash
# Install dependencies
go mod tidy

# Build API server
go build -o bin/api ./api/main.go

# Start API server (processes uploads synchronously)
./bin/api
```

## Testing WilliamBoard

### Stage 1: Basic Infrastructure ✅
### Stage 2: GPT-4o Vision Analysis ✅

**Current Status**: Both stages implemented and ready for testing!

### 1. Health Check

Test that the API server starts correctly:

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "ok",
  "app": "WilliamBoard"
}
```

### 2. Database Connection Test

The API should start without database errors. Check logs for:
```
Starting WilliamBoard API server on port 8080
```

### 3. Configuration Validation Test

Test configuration loading:

```bash
go run -c 'package main

import (
    "fmt"
    "github.com/joho/godotenv"
    "github.com/lincolngreen/williamboard/api/config"
)

func main() {
    godotenv.Load()
    cfg, err := config.Load()
    if err != nil {
        panic(err)
    }
    fmt.Printf("✓ Config loaded: %s\n", cfg.AppName)
}' /dev/stdin
```

### 4. Upload Flow Test

Test the complete upload workflow:

```bash
# 1. Request upload URL
curl -X POST http://localhost:8080/v1/uploads/signed-url \
  -H "Content-Type: application/json" \
  -d '{"contentType": "image/jpeg"}'
```

Expected response:
```json
{
  "submissionId": "uuid-here",
  "url": "http://localhost:8080/v1/uploads/uuid-here",
  "maxSizeMB": 12
}
```

```bash
# 2. Upload file directly (processes synchronously)
curl -X PUT http://localhost:8080/v1/uploads/{submission-id} \
  -F "file=@/path/to/your/image.jpg"
```

Expected response (after processing):
```json
{
  "message": "Image processed successfully", 
  "submissionId": "uuid-here",
  "status": "parsed",
  "eventsFound": 3,
  "flyersFound": 2
}
```

Or if no events found:
```json
{
  "message": "Image processed successfully",
  "submissionId": "uuid-here", 
  "status": "parsed",
  "eventsFound": 0,
  "flyersFound": 0
}
```

### ⚡ Real-Time Processing Notes

**With OpenAI API Key (Stage 2 Active):**
- Upload → GPT-4o Vision analysis → Structured event extraction
- Processing time: 2-15 seconds depending on image complexity  
- Returns actual event counts and confidence scores
- Stores detailed flyer/event data in database

**Without API Key (Stage 1 Only):**
- Upload → File storage only
- Processing time: <1 second
- Always returns 0 events/flyers found
- Still demonstrates synchronous processing UX

```bash
# 3. Check detailed results (optional)
curl http://localhost:8080/v1/submissions/{submission-id}/status
```

Expected response:
```json
{
  "status": "parsed",
  "step": "done",
  "flyers": [...],
  "candidates": [...]
}
```

```bash
# 5. View uploaded file
curl http://localhost:8080/files/{submission-id}/original.jpg
# Should return the uploaded image
```

### 5. Processing Test

With the API server running, check logs for synchronous processing messages:

**Without API Key:**
```
Starting WilliamBoard API server on port 8080
Processing submission uuid-here synchronously
Successfully processed submission uuid-here: 0 flyers, 0 events
```

**With OpenAI API Key:**
```
Starting WilliamBoard API server on port 8080
Processing submission uuid-here synchronously  
Vision analysis completed for uuid-here: found 2 flyers, 3 total events
Successfully processed submission uuid-here: 2 flyers, 3 events
```

### 6. Database Inspection

Check what was stored in the database:

```bash
# Connect to your database
psql $DATABASE_URL

# Check submissions
SELECT id, status, created_at FROM submissions ORDER BY created_at DESC LIMIT 5;

# Check detected flyers (Stage 2 only)
SELECT id, region_id, detection_confidence FROM flyers ORDER BY created_at DESC LIMIT 5;

# Check extracted events (Stage 2 only)  
SELECT id, event_id, fields->>'title' as title FROM event_candidates ORDER BY created_at DESC LIMIT 5;
```

### 7. Events API Test

Test the events endpoint:

```bash
curl http://localhost:8080/v1/events
```

Expected response (empty initially, events appear after Stage 4):
```json
{
  "type": "FeatureCollection", 
  "features": []
}
```

## Stage 2 Testing with Real Images

### Quick Test Setup

1. **Get an OpenAI API key** from https://platform.openai.com/api-keys

2. **Update your .env file:**
```bash
OPENAI_API_KEY=sk-your-real-api-key-here
```

3. **Find a test image:** 
   - Bulletin board photo with visible event flyers
   - Community center announcements
   - Coffee shop event boards
   - University campus bulletin boards

4. **Test the upload:**
```bash
# Get upload URL
RESPONSE=$(curl -s -X POST http://localhost:8080/v1/uploads/signed-url \
  -H "Content-Type: application/json" \
  -d '{"contentType": "image/jpeg"}')

# Extract submission ID
SUBMISSION_ID=$(echo $RESPONSE | grep -o '"submissionId":"[^"]*"' | cut -d'"' -f4)

# Upload your test image (replace with your image path)
curl -X PUT "http://localhost:8080/v1/uploads/$SUBMISSION_ID" \
  -F "file=@/path/to/your/bulletin-board.jpg"
```

### Expected Results

With a good bulletin board image, you should see:
```json
{
  "message": "Image processed successfully",
  "submissionId": "abc-123-def", 
  "status": "parsed",
  "eventsFound": 2,
  "flyersFound": 1
}
```

### Troubleshooting

**"eventsFound": 0** - This could mean:
- Image doesn't contain clear event flyers
- Text is too small/blurry to read
- Flyers are non-event content (ads, notices, etc.)
- Try a different image with clearer event flyers

**Processing timeout** - This could mean:
- GPT-4o API is slow (try again)
- Image is too large (>18MB)
- Network connectivity issues

**API errors** - Check:
- OpenAI API key is valid and has credits
- Image format is supported (JPEG/PNG/WebP/GIF)
- Worker logs for detailed error messages

## API Documentation

### Upload Flow

1. **Get Signed URL**: `POST /v1/uploads/signed-url`
   - Request: `{"contentType": "image/jpeg"}`
   - Returns presigned S3 URL for direct upload

2. **Complete Upload**: `POST /v1/submissions/{id}/complete`
   - Marks upload complete and triggers processing

3. **Check Status**: `GET /v1/submissions/{id}/status`
   - Returns processing status and results

### Events API

- **List Events**: `GET /v1/events`
  - Query params: `bbox`, `start_date`, `end_date`, `keyword`, `limit`, `offset`
  - Returns GeoJSON FeatureCollection

- **Get Event**: `GET /v1/events/{id}`
  - Returns single event details

- **Calendar Export**: `GET /v1/events/{id}/ics`
  - Returns event in ICS calendar format

- **Unpublish Event**: `POST /v1/events/{id}/unpublish`
  - Request: `{"reason": "spam|duplicate|bad_location|inappropriate"}`

## Database Schema

Key tables:
- `submissions` - Uploaded images and processing status
- `flyers` - Detected flyer regions within images  
- `venues` - Locations with geocoding and PostGIS points
- `event_candidates` - Extracted events before publish decision
- `events` - Published events with moderation state
- `audit_logs` - System audit trail

## Development

### Adding New Endpoints

1. Add handler to `api/handlers/`
2. Register route in `api/main.go#setupRouter`
3. Add tests

### Database Migrations

Add new migrations as `migrations/00X_description.sql`

### Processing Pipeline

Synchronous processing stages:
1. **Stage 1**: File upload and storage ✅
2. **Stage 2**: GPT-4o vision analysis ✅
3. **Stage 3**: Moderation + geocoding (next)
4. **Stage 4**: Auto-publish scoring (next)

### Stage 2: GPT-4o Vision Analysis ✅

The system now includes full GPT-4o Vision integration:

**Flyer Detection:**
- Detects multiple flyers in bulletin board photos
- Provides confidence scores for each detection
- Extracts polygon coordinates for flyer boundaries
- Handles rotation and perspective correction

**Event Extraction:**
- Structured data extraction using GPT-4o's JSON mode
- Extracts: title, date/time, venue, address, price, description, organizer
- Confidence scoring for each field
- Source text excerpts for verification

**Data Persistence:**
- Stores flyers in `flyers` table with detection metadata
- Stores events in `event_candidates` table with structured data
- Preserves raw GPT-4o responses for debugging

**Supported Image Formats:**
- JPEG, PNG, WebP, GIF
- Up to 18MB file size (GPT-4o limit)
- Automatic format validation

## Deployment on Render

### Benefits of Simplified Architecture

- ✅ **No AWS dependencies** - Eliminates AWS setup complexity and costs
- ✅ **Single platform** - Everything runs on Render with unified billing
- ✅ **Automatic scaling** - Render handles traffic spikes automatically  
- ✅ **Persistent storage** - Files survive service restarts and deployments
- ✅ **Simple queue** - In-memory queue eliminates SQS setup
- ✅ **Easy deployment** - Single `render.yaml` file handles everything

### Deployment Steps

1. **Fork this repository** to your GitHub account

2. **Connect to Render:**
   - Sign up at [render.com](https://render.com)
   - Connect your GitHub account
   - Create new service from your forked repo

3. **Configure environment:**
   - Render reads `render.yaml` automatically
   - Set `OPENAI_API_KEY` in dashboard (keep secure)
   - Database and storage are provisioned automatically

4. **Deploy:**
   - Push to main branch triggers automatic deployment
   - Monitor build logs in Render dashboard
   - Services start automatically after successful build

### Render Resources Created

- **Web Service**: API server with persistent disk (10GB) for synchronous processing
- **PostgreSQL Database**: With PostGIS for location data
- **Persistent Disk**: File storage mounted at `/data`

### Production Configuration

In production on Render:
- Files stored on persistent SSD with daily snapshots
- Database with automatic backups 
- SSL/HTTPS handled automatically
- CDN via Render's global edge network
- Zero-downtime deployments

See `render.yaml` for complete deployment configuration.

## Troubleshooting

### Common Issues

**Database connection refused (Local Dev)**
- Ensure PostgreSQL is running locally
- Check DATABASE_URL format
- Verify database exists and extensions installed

**File upload failures**
- Check UPLOAD_DIR exists and is writable
- Ensure disk space available
- Verify file size under 12MB limit

**Build failures**
- Run `go mod tidy` to ensure dependencies
- Check Go version (requires 1.21+)

**Processing timeout during upload**
- Check API server logs for errors
- Verify OpenAI API key is valid
- Ensure image size is under 18MB limit

**Render deployment issues**
- Check build logs in Render dashboard
- Verify `OPENAI_API_KEY` is set (when needed for Stage 2+)
- Ensure persistent disk is mounted at `/data`

### Logs

- API server logs show request processing, synchronous vision analysis, and errors
- Database migration errors appear during startup
- OpenAI API call logs show vision processing details

For additional help, see the implementation plan in `IMPLEMENTATION_PLAN.md`.