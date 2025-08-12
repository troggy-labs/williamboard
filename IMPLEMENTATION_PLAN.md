# WilliamBoard MVP v0 Implementation Plan

## Overview
Building a bulletin board photo → event extraction → auto-publish system using GPT-4o Structured Outputs, deployed on Render with proper microservices architecture.

## Stage 1: Core Infrastructure & Data Models
**Goal**: Set up project structure, database schema, and basic Go services
**Success Criteria**: 
- Go modules initialized with proper project structure
- PostgreSQL schema created and migrated
- Basic API server responds to health checks
- Worker service can connect to queue
**Tests**: Health endpoint returns 200, database connection works, queue connection established
**Status**: Complete

### Tasks:
- Initialize Go modules for API and worker
- Set up PostgreSQL schema with migrations
- Create basic API server with health endpoint
- Set up SQS queue integration
- Configure environment variables and validation

## Stage 2: LLM Integration & Structured Outputs
**Goal**: Implement GPT-4o vision analysis with structured outputs schema
**Success Criteria**:
- Worker can process images through GPT-4o
- Structured outputs match the defined schema
- Results persist to database (flyers, event_candidates tables)
**Tests**: Process test images and verify schema compliance, database persistence
**Status**: Complete

### Tasks:
- Implement GPT-4o client with structured outputs
- Create image preprocessing (resize, format)
- Process flyer detection and event extraction
- Store results in flyers and event_candidates tables

## Stage 3: Processing Pipeline (Moderation + Geocoding)
**Goal**: Complete the processing pipeline with moderation and geocoding
**Success Criteria**:
- Images go through moderation API
- Addresses get geocoded with confidence scores
- Auto-publish scoring logic works
**Tests**: Moderated content blocks correctly, geocoding returns valid coordinates
**Status**: Complete

### Tasks:
- Integrate OpenAI moderation API
- Implement Mapbox geocoding
- Create auto-publish scoring algorithm
- Add face-blur processing with OpenCV

## Stage 4: API Endpoints & File Upload
**Goal**: Complete REST API with direct-to-S3 uploads
**Success Criteria**:
- Presigned URL generation works ✅
- Upload completion triggers processing ✅
- Status polling returns progress ✅
- Events API returns GeoJSON ✅
**Tests**: End-to-end upload flow works, status updates correctly
**Status**: Complete

### Tasks:
- Implement presigned S3 POST URLs
- Create submission completion endpoint
- Add WebSocket/SSE for status updates  
- Build events query API with GeoJSON output
- Generate ICS calendar files

## Stage 5: Frontend & Deployment
**Goal**: Next.js frontend deployed to Render with full integration
**Success Criteria**:
- Upload form works with direct-to-S3
- Map view shows events with clustering
- Event details and calendar export work
- Deployed on Render with proper configuration
**Tests**: Full user journey from upload to published event
**Status**: Not Started

### Tasks:
- Create Next.js upload interface
- Build map view with Mapbox
- Add event detail pages and calendar export
- Configure render.yaml for deployment
- Set up S3/CDN configuration

## Development Notes
- Following Go project structure with `/api` and `/workers` directories
- Using GORM for database ORM with PostgreSQL + PostGIS
- Implementing proper error handling and observability
- All services stateless for Render deployment