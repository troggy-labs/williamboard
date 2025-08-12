WilliamBoard — MVP v0 (gpt‑4o + Structured Outputs, Render deploy)
What we’re building
Upload a bulletin‑board photo → one gpt‑4o call returns flyer regions + events as schema‑valid JSON → moderation → geocoding → dedupe → auto‑publish → show on map + list with ICS/calendar links.
MVP out of scope: organizer accounts, recurring rules, ticket checkout, translations.

Architecture (production‑ready, minimal)
[Browser / Next.js on Render]
   |                \
   | (1) direct PUT  \ (2) API calls (status/map/events)
   v                  v
[S3/GCS + CDN]    [API on Render Web Service] --(enqueue)--> [Queue (SQS/Pub/Sub)]
                                               |                                  \
                                               |                                   v
                                               |                        [Worker on Render Background Worker]
                                               |                                   |-- call gpt‑4o (Structured Outputs)
                                               |                                   |-- moderation (image + excerpts)
                                               |                                   |-- geocode + dedupe + auto‑publish
                                               |                                   |-- write Postgres + emit status
                                               v
                                         [Postgres (+PostGIS)]
Key choices
Uploads & delivery: direct‑to‑S3/GCS from the browser; serve via CDN. No images pass through your API.
Render runs API (Go) and Worker (Go or Node). No persistent disks.
Status updates: WebSocket or SSE from API to the client.
Queue: SQS (or Pub/Sub). Worker autoscaling can be manual at first (1 instance).
Data model (unchanged essentials)
submissions: id, user_id?, original_image_url, derivative_image_url, captured_at?, exif_opt_in, status, created_at
flyers: id, submission_id, region_id, polygon, rotation_deg, detection_confidence, crop_image_url, created_at
event_candidates: id, flyer_id, fields(jsonb), confidences(jsonb), source_excerpt, geocode(jsonb), composite_score, publish_result, publication_reason, created_at
events: id, canonical_key, title, start_ts, end_ts?, venue_id, url?, price?, description?, source('flyer'), published_via('auto'|'manual'), quality_score, moderation_state, created_at
venues, dedupe_links, audit_logs, flags (as previously specced)
Canonical key: normalize(title) + normalize(venue) + round(start_ts, 30m).
LLM extraction (single call)
Model: gpt-4o (vision).
Contract: Structured Outputs with the schema below; responses must match the schema (no free‑form text).
JSON Schema (copy/paste)
{
  "name": "WilliamBoardExtractionV1",
  "schema": {
    "type": "object",
    "required": ["image", "flyers", "events"],
    "additionalProperties": false,
    "properties": {
      "image": {
        "type": "object",
        "required": ["width_px", "height_px"],
        "additionalProperties": false,
        "properties": {
          "width_px": { "type": "integer", "minimum": 1 },
          "height_px": { "type": "integer", "minimum": 1 }
        }
      },
      "flyers": {
        "type": "array",
        "items": {
          "type": "object",
          "required": ["region_id", "polygon", "detection_confidence"],
          "additionalProperties": false,
          "properties": {
            "region_id": { "type": "string" },
            "polygon": {
              "type": "array", "minItems": 4, "maxItems": 4,
              "items": {
                "type": "object",
                "required": ["x", "y"],
                "additionalProperties": false,
                "properties": { "x": { "type": "number" }, "y": { "type": "number" } }
              }
            },
            "rotation_deg": { "type": "number" },
            "detection_confidence": { "type": "number", "minimum": 0, "maximum": 1 },
            "notes": { "type": "string" }
          }
        }
      },
      "events": {
        "type": "array",
        "items": {
          "type": "object",
          "required": ["event_id", "region_id", "title", "datetime"],
          "additionalProperties": false,
          "properties": {
            "event_id": { "type": "string" },
            "region_id": { "type": "string" },
            "title": { "type": "string", "minLength": 3 },
            "datetime": {
              "type": "object",
              "required": ["start_iso"],
              "additionalProperties": false,
              "properties": {
                "start_iso": { "type": "string", "format": "date-time" },
                "end_iso": { "type": "string", "format": "date-time" },
                "tz_hint": { "type": "string" },
                "confidence": { "type": "number", "minimum": 0, "maximum": 1 }
              }
            },
            "location": {
              "type": "object",
              "required": [],
              "additionalProperties": false,
              "properties": {
                "venue_name": { "type": "string" },
                "address_line": { "type": "string" },
                "city_hint": { "type": "string" },
                "confidence": { "type": "number", "minimum": 0, "maximum": 1 }
              }
            },
            "url": { "type": "string" },
            "price": { "type": "string" },
            "organizer": { "type": "string" },
            "source_excerpt": { "type": "string" },
            "confidences": {
              "type": "object",
              "additionalProperties": false,
              "properties": {
                "title": { "type": "number" },
                "url": { "type": "number" },
                "price": { "type": "number" },
                "location_name": { "type": "number" },
                "address": { "type": "number" },
                "ocr_quality": { "type": "number" },
                "layout_consistency": { "type": "number" }
              }
            },
            "notes": { "type": "string" }
          }
        }
      }
    }
  },
  "strict": true
}
Prompt (short + strict)
System: “Extract all flyer regions and every event from this bulletin‑board photo. Only use text visible. Quote the lines used for date/venue in source_excerpt. If ambiguous, omit the field; don’t guess. Output only the JSON per schema.”
User content: the image (downscaled to ≤2048 px long side).
Auto‑publish policy (v0, simple & safe)
Gates (all must pass)
Completeness: title, absolute start_iso (future: now+30m … now+180d), geocoded location present.
Safety: moderation clear (image + source_excerpt); face‑blur run on the image.
Consistency: geocode confidence ≥0.75 or exact street address; dedupe safe/merged.
Score:
0.25*datetime_conf +
0.20*title_conf +
0.20*location_name_conf +
0.20*geocode_conf +
0.10*ocr_quality
- 0.10*risk_penalty
Publish if ≥0.80 (trusted +0.05, new/high‑velocity −0.05).
Fail → needs_review. Moderation hit → blocked.
APIs (no image bytes ever hit the API)
POST /v1/uploads/signed-url
→ Return S3/GCS presigned POST (key, url, fields), enforce content‑type/size.
POST /v1/submissions/{submissionId}/complete
→ Enqueue job; return {status:"processing"}.

GET /v1/submissions/{submissionId}/status (or WS)
→ Progress + decisions:

{
  "status":"parsed",
  "flyers":[{"flyerId":"fly_1","imageUrl":".../crop1.jpg","detectionConfidence":0.91}],
  "candidates":[{"candidateId":"ec_1","decision":"published","score":0.86,"eventId":"evt_42"}]
}
GET /v1/events?... (GeoJSON for map/list)
GET /v1/events/{id} (detail)
GET /v1/events/{id}/ics (ICS; UID: evt_{id}@williamboard.app, PRODID:-//WilliamBoard//EN)
POST /v1/events/{id}/unpublish (internal)
Frontend (Next.js on Render)
Upload → direct to S3/GCS using presigned POST; on success, call /complete and open a WS to watch status.
Map view with clustering + bottom sheet list; filters: Today/This Week, distance, keyword.
Event detail page: flyer crop, date/time (local), venue/address (Open in Maps), URL/price, Add to Calendar (Google/Apple/Outlook, ICS).
Success toast with Undo (calls unpublish API → sends to review).
Accessibility: alt text from title, high‑contrast toggle.
Deployment on Render (concrete)
Services
frontend: Static Site (Next.js static export) or Web Service (if SSR).
api: Web Service (Go) — no disk.
worker: Background Worker (Go or Node) — no disk.
render.yaml (starter)
services:
  - type: web
    name: williamboard-api
    env: go
    region: oregon
    buildCommand: go build -o api ./api
    startCommand: ./api
    envVars:
      - key: DATABASE_URL
        fromDatabase:
          name: williamboard-postgres
          property: connectionString
      - key: STORAGE_BUCKET
        sync: false
      - key: CDN_BASE_URL
        sync: false
      - key: QUEUE_URL
        sync: false
      - key: OPENAI_API_KEY
        sync: false
      - key: OPENAI_MODEL
        value: gpt-4o
      - key: MODERATION_MODEL
        value: omni-moderation-latest
      - key: PUBLIC_BASE_URL
        value: https://williamboard.app
      - key: ICS_UID_DOMAIN
        value: williamboard.app
      - key: ICS_PRODID
        value: -//WilliamBoard//EN
  - type: worker
    name: williamboard-worker
    env: go
    region: oregon
    buildCommand: go build -o worker ./workers/extractor
    startCommand: ./worker
    envVars:
      - key: DATABASE_URL
        fromDatabase:
          name: williamboard-postgres
          property: connectionString
      - key: QUEUE_URL
        sync: false
      - key: OPENAI_API_KEY
        sync: false
      - key: OPENAI_MODEL
        value: gpt-4o
      - key: GEOCODER
        value: mapbox
      - key: GEOCODER_API_KEY
        sync: false
      - key: AUTO_PUBLISH_ENABLED
        value: "true"
  - type: static
    name: williamboard-frontend
    envVars:
      - key: NEXT_PUBLIC_MAPBOX_TOKEN
        sync: false
databases:
  - name: williamboard-postgres
Keep frontend/api/worker in the same region as S3/GCS + DB.
S3/GCS (uploads & CDN)
Bucket per env: williamboard-uploads-dev|prod.
CORS (example):
[
  {
    "AllowedOrigins": ["https://williamboard.app"],
    "AllowedMethods": ["PUT", "POST", "GET"],
    "AllowedHeaders": ["*"],
    "ExposeHeaders": ["ETag"],
    "MaxAgeSeconds": 3000
  }
]
CDN domain: https://cdn.williamboard.app → origins = S3/GCS bucket.
Queue
SQS (standard).
Topics: submissions.created.
DLQ: submissions.dlq (maxReceiveCount 5).
Worker logic (sketch, Go)
Derivative: load original; create ≤2048‑px JPEG for LLM; save to storage.
Call gpt‑4o with the schema (Structured Outputs).
Persist: flyers (and generate small crops for UI using Go imaging), event_candidates.
Moderate: send image + concatenated source_excerpt to moderation API. If flagged → mark blocked.
Geocode: Mapbox/OSM; write geocode with confidence.
Dedupe: canonical key + optional pgvector similarity; merge if safe.
Auto‑publish: apply gates & scoring; create events; generate ICS; emit status via WS.
API endpoint details (copy-paste)
POST /v1/uploads/signed-url
Req: { "contentType":"image/jpeg", "submissionId": null }
Res: { "submissionId":"sub_123", "url":"...", "fields": {...}, "key":"sub_123/original.jpg", "maxSizeMB":12 }
POST /v1/submissions/{id}/complete
→ { "status":"processing" } (and enqueues message)

GET /v1/submissions/{id}/status
→ see example above (include step: uploaded|extracting|moderating|geocoding|publishing|done|error)

GET /v1/events?... (GeoJSON)
GET /v1/events/{id} (detail)
GET /v1/events/{id}/ics (ICS text)
POST /v1/events/{id}/unpublish (internal) { "reason": "spam|duplicate|bad_location" }

Config (env)
APP_NAME=WilliamBoard
PUBLIC_BASE_URL=https://williamboard.app
ICS_UID_DOMAIN=williamboard.app
ICS_PRODID=-//WilliamBoard//EN

OPENAI_MODEL=gpt-4o
OPENAI_API_KEY=...
OPENAI_TIMEOUT_MS=15000
STRUCTURED_OUTPUT=true
IMAGE_MAX_LONG_SIDE=2048
IMAGE_JPEG_QUALITY=85

STORAGE_BUCKET=williamboard-uploads-<env>
CDN_BASE_URL=https://cdn.williamboard.app
QUEUE_URL=...
REGION_TZ=America/Los_Angeles

GEOCODER=mapbox|osm
GEOCODER_API_KEY=...

AUTO_PUBLISH_ENABLED=true
AUTO_PUBLISH_THRESHOLD=0.80
GEO_CONF_THRESHOLD=0.75
AUTO_PUBLISH_MIN_START_OFFSET_MIN=30
AUTO_PUBLISH_MAX_START_OFFSET_DAYS=180
TRUST_ADJUST=0.05

PGVECTOR_ENABLED=false
OTEL_EXPORTER_OTLP_ENDPOINT=...
Security & privacy
Never proxy image bytes; browser → S3/GCS directly via presigned POST.
Strip EXIF by default; optional user opt‑in to use capture location/time for better parsing.
Face‑blur on full image (OpenCV) before storing derivative.
Rate‑limits per IP/user; burst guard; abuse reporting endpoint.
Keep originals 30 days; crops/structured data until event end + 90 days.
Observability
OpenTelemetry traces per submissionId.
Dashboards: llm_extract_latency, events_per_submission, moderation_block_rate, geocode_success_rate, auto_publish_rate, auto_pub_correction_rate, p50/p95_e2e.
Log publication_reason and store raw LLM JSON (auditable).
Acceptance tests (MVP)
Upload→publish ≤10s P50 for a clean flyer with address/time.
Multi‑event flyer returns all events; polygons good enough for thumbnails (IoU ≥0.6 on ≥80%).
Relative date + capture time resolves to absolute ISO; else low confidence → review.
Duplicate across two flyers merges; older event id retained.
Moderation hit blocks publish with reason.