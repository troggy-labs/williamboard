-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";

-- submissions table
CREATE TABLE submissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NULL, -- for future user accounts
    original_image_url VARCHAR(500) NOT NULL,
    derivative_image_url VARCHAR(500) NULL,
    captured_at TIMESTAMP WITH TIME ZONE NULL,
    exif_opt_in BOOLEAN DEFAULT FALSE,
    status VARCHAR(50) NOT NULL DEFAULT 'uploaded', -- uploaded, processing, parsed, error, done
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- flyers table (detected flyer regions)
CREATE TABLE flyers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    region_id VARCHAR(50) NOT NULL, -- from LLM response
    polygon JSONB NOT NULL, -- array of {x, y} points
    rotation_deg FLOAT NULL,
    detection_confidence FLOAT NOT NULL CHECK (detection_confidence >= 0 AND detection_confidence <= 1),
    crop_image_url VARCHAR(500) NULL, -- generated crop URL
    notes TEXT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- venues table
CREATE TABLE venues (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(200) NOT NULL,
    address_line VARCHAR(300) NULL,
    city VARCHAR(100) NULL,
    state VARCHAR(50) NULL,
    postal_code VARCHAR(20) NULL,
    country VARCHAR(50) NULL DEFAULT 'US',
    location GEOMETRY(POINT, 4326) NULL, -- PostGIS point
    geocode_confidence FLOAT NULL,
    geocode_data JSONB NULL, -- raw geocoder response
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Indexes
    UNIQUE(name, address_line, city, state)
);

-- event_candidates table (before publish decision)
CREATE TABLE event_candidates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    flyer_id UUID NOT NULL REFERENCES flyers(id) ON DELETE CASCADE,
    event_id VARCHAR(50) NOT NULL, -- from LLM response
    fields JSONB NOT NULL, -- structured event data from LLM
    confidences JSONB NOT NULL, -- confidence scores from LLM
    source_excerpt TEXT NULL,
    geocode JSONB NULL, -- geocoding results
    composite_score FLOAT NULL,
    publish_result VARCHAR(50) NULL, -- published, blocked, needs_review
    publication_reason TEXT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- events table (published events)
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    canonical_key VARCHAR(300) NOT NULL, -- normalize(title) + normalize(venue) + round(start_ts, 30m)
    title VARCHAR(300) NOT NULL,
    start_ts TIMESTAMP WITH TIME ZONE NOT NULL,
    end_ts TIMESTAMP WITH TIME ZONE NULL,
    venue_id UUID NULL REFERENCES venues(id),
    url VARCHAR(500) NULL,
    price VARCHAR(100) NULL,
    description TEXT NULL,
    organizer VARCHAR(200) NULL,
    source VARCHAR(50) NOT NULL DEFAULT 'flyer',
    published_via VARCHAR(50) NOT NULL DEFAULT 'auto', -- auto, manual
    quality_score FLOAT NULL,
    moderation_state VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, approved, blocked
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CHECK (start_ts > created_at - INTERVAL '1 day'), -- reasonable start time
    UNIQUE(canonical_key)
);

-- dedupe_links table (for merging duplicate events)
CREATE TABLE dedupe_links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    primary_event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    duplicate_event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    similarity_score FLOAT NOT NULL,
    merge_reason VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    UNIQUE(primary_event_id, duplicate_event_id)
);

-- audit_logs table
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_type VARCHAR(50) NOT NULL, -- submission, event, etc
    entity_id UUID NOT NULL,
    action VARCHAR(100) NOT NULL,
    user_id UUID NULL,
    changes JSONB NULL,
    metadata JSONB NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- flags table (for user reporting)
CREATE TABLE flags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    flag_type VARCHAR(50) NOT NULL, -- spam, inappropriate, duplicate, wrong_location
    reason TEXT NULL,
    reporter_ip INET NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, resolved, dismissed
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_submissions_status ON submissions(status);
CREATE INDEX idx_submissions_created_at ON submissions(created_at);

CREATE INDEX idx_flyers_submission_id ON flyers(submission_id);
CREATE INDEX idx_flyers_region_id ON flyers(region_id);

CREATE INDEX idx_venues_location ON venues USING GIST(location);
CREATE INDEX idx_venues_name ON venues(name);

CREATE INDEX idx_event_candidates_flyer_id ON event_candidates(flyer_id);
CREATE INDEX idx_event_candidates_publish_result ON event_candidates(publish_result);

CREATE INDEX idx_events_start_ts ON events(start_ts);
CREATE INDEX idx_events_venue_id ON events(venue_id);
CREATE INDEX idx_events_canonical_key ON events(canonical_key);
CREATE INDEX idx_events_moderation_state ON events(moderation_state);

CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);

-- Updated at triggers
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_submissions_updated_at BEFORE UPDATE ON submissions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_events_updated_at BEFORE UPDATE ON events
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();