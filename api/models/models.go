package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Submission represents an uploaded bulletin board image
type Submission struct {
	ID                 uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID             *uuid.UUID `json:"user_id" gorm:"type:uuid"`
	OriginalImageURL   string     `json:"original_image_url" gorm:"size:500;not null"`
	DerivativeImageURL *string    `json:"derivative_image_url" gorm:"size:500"`
	CapturedAt         *time.Time `json:"captured_at"`
	ExifOptIn          bool       `json:"exif_opt_in" gorm:"default:false"`
	Status             string     `json:"status" gorm:"size:50;not null;default:'uploaded'"` // uploaded, processing, parsed, error, done
	CreatedAt          time.Time  `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt          time.Time  `json:"updated_at" gorm:"not null;default:now()"`

	// Relations
	Flyers []Flyer `json:"flyers,omitempty"`
}

// Flyer represents a detected flyer region in an image
type Flyer struct {
	ID                   uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	SubmissionID         uuid.UUID `json:"submission_id" gorm:"type:uuid;not null"`
	RegionID             string    `json:"region_id" gorm:"size:50;not null"`
	Polygon              string    `json:"polygon" gorm:"type:jsonb;not null"` // JSON array of {x, y} points
	RotationDeg          *float64  `json:"rotation_deg"`
	DetectionConfidence  float64   `json:"detection_confidence" gorm:"not null"`
	CropImageURL         *string   `json:"crop_image_url" gorm:"size:500"`
	Notes                *string   `json:"notes"`
	CreatedAt            time.Time `json:"created_at" gorm:"not null;default:now()"`

	// Relations
	Submission       Submission        `json:"submission,omitempty"`
	EventCandidates  []EventCandidate  `json:"event_candidates,omitempty"`
}

// Venue represents a location where events occur
type Venue struct {
	ID                uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Name              string    `json:"name" gorm:"size:200;not null"`
	AddressLine       *string   `json:"address_line" gorm:"size:300"`
	City              *string   `json:"city" gorm:"size:100"`
	State             *string   `json:"state" gorm:"size:50"`
	PostalCode        *string   `json:"postal_code" gorm:"size:20"`
	Country           string    `json:"country" gorm:"size:50;default:'US'"`
	Location          *string   `json:"location" gorm:"type:geometry(POINT,4326)"` // PostGIS point
	GeocodeConfidence *float64  `json:"geocode_confidence"`
	GeocodeData       *string   `json:"geocode_data" gorm:"type:jsonb"` // raw geocoder response
	CreatedAt         time.Time `json:"created_at" gorm:"not null;default:now()"`

	// Relations
	Events []Event `json:"events,omitempty"`
}

// EventCandidate represents an event before publish decision
type EventCandidate struct {
	ID                 uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	FlyerID            uuid.UUID  `json:"flyer_id" gorm:"type:uuid;not null"`
	EventID            string     `json:"event_id" gorm:"size:50;not null"` // from LLM response
	Fields             string     `json:"fields" gorm:"type:jsonb;not null"` // structured event data from LLM
	Confidences        string     `json:"confidences" gorm:"type:jsonb;not null"` // confidence scores
	SourceExcerpt      *string    `json:"source_excerpt"`
	Geocode            *string    `json:"geocode" gorm:"type:jsonb"` // geocoding results
	CompositeScore     *float64   `json:"composite_score"`
	PublishResult      *string    `json:"publish_result" gorm:"size:50"` // published, blocked, needs_review
	PublicationReason  *string    `json:"publication_reason"`
	CreatedAt          time.Time  `json:"created_at" gorm:"not null;default:now()"`

	// Relations
	Flyer Flyer `json:"flyer,omitempty"`
}

// Event represents a published event
type Event struct {
	ID              uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	CanonicalKey    string     `json:"canonical_key" gorm:"size:300;not null;uniqueIndex"`
	Title           string     `json:"title" gorm:"size:300;not null"`
	StartTs         time.Time  `json:"start_ts" gorm:"not null"`
	EndTs           *time.Time `json:"end_ts"`
	VenueID         *uuid.UUID `json:"venue_id" gorm:"type:uuid"`
	URL             *string    `json:"url" gorm:"size:500"`
	Price           *string    `json:"price" gorm:"size:100"`
	Description     *string    `json:"description"`
	Organizer       *string    `json:"organizer" gorm:"size:200"`
	Source          string     `json:"source" gorm:"size:50;not null;default:'flyer'"`
	PublishedVia    string     `json:"published_via" gorm:"size:50;not null;default:'auto'"` // auto, manual
	QualityScore    *float64   `json:"quality_score"`
	ModerationState string     `json:"moderation_state" gorm:"size:50;not null;default:'pending'"` // pending, approved, blocked
	CreatedAt       time.Time  `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"not null;default:now()"`

	// Relations
	Venue *Venue `json:"venue,omitempty"`
}

// DedupeLink represents merged duplicate events
type DedupeLink struct {
	ID                 uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	PrimaryEventID     uuid.UUID `json:"primary_event_id" gorm:"type:uuid;not null"`
	DuplicateEventID   uuid.UUID `json:"duplicate_event_id" gorm:"type:uuid;not null"`
	SimilarityScore    float64   `json:"similarity_score" gorm:"not null"`
	MergeReason        string    `json:"merge_reason" gorm:"size:100;not null"`
	CreatedAt          time.Time `json:"created_at" gorm:"not null;default:now()"`

	// Relations
	PrimaryEvent   Event `json:"primary_event,omitempty"`
	DuplicateEvent Event `json:"duplicate_event,omitempty"`
}

// AuditLog represents system audit trail
type AuditLog struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	EntityType string    `json:"entity_type" gorm:"size:50;not null"`
	EntityID   uuid.UUID `json:"entity_id" gorm:"type:uuid;not null"`
	Action     string    `json:"action" gorm:"size:100;not null"`
	UserID     *uuid.UUID `json:"user_id" gorm:"type:uuid"`
	Changes    *string   `json:"changes" gorm:"type:jsonb"`
	Metadata   *string   `json:"metadata" gorm:"type:jsonb"`
	CreatedAt  time.Time `json:"created_at" gorm:"not null;default:now()"`
}

// Flag represents user-reported issues
type Flag struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	EventID    uuid.UUID `json:"event_id" gorm:"type:uuid;not null"`
	FlagType   string    `json:"flag_type" gorm:"size:50;not null"` // spam, inappropriate, duplicate, wrong_location
	Reason     *string   `json:"reason"`
	ReporterIP *string   `json:"reporter_ip" gorm:"type:inet"`
	Status     string    `json:"status" gorm:"size:50;not null;default:'pending'"` // pending, resolved, dismissed
	CreatedAt  time.Time `json:"created_at" gorm:"not null;default:now()"`

	// Relations
	Event Event `json:"event,omitempty"`
}

// BeforeCreate hooks
func (s *Submission) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

func (f *Flyer) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}

func (v *Venue) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}

func (ec *EventCandidate) BeforeCreate(tx *gorm.DB) error {
	if ec.ID == uuid.Nil {
		ec.ID = uuid.New()
	}
	return nil
}

func (e *Event) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}