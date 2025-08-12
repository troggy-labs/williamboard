package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lincolngreen/williamboard/api/config"
	"github.com/lincolngreen/williamboard/api/models"
	"gorm.io/gorm"
)

type EventHandler struct {
	config *config.Config
	db     *gorm.DB
}

type EventGeoJSON struct {
	Type     string                 `json:"type"`
	Features []EventFeature         `json:"features"`
}

type EventFeature struct {
	Type       string                 `json:"type"`
	ID         string                 `json:"id"`
	Geometry   EventGeometry          `json:"geometry"`
	Properties EventProperties        `json:"properties"`
}

type EventGeometry struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"` // [longitude, latitude]
}

type EventProperties struct {
	Title       string     `json:"title"`
	StartTs     time.Time  `json:"start_ts"`
	EndTs       *time.Time `json:"end_ts,omitempty"`
	VenueName   *string    `json:"venue_name,omitempty"`
	Address     *string    `json:"address,omitempty"`
	URL         *string    `json:"url,omitempty"`
	Price       *string    `json:"price,omitempty"`
	Description *string    `json:"description,omitempty"`
	Organizer   *string    `json:"organizer,omitempty"`
	Source      string     `json:"source"`
}

type UnpublishRequest struct {
	Reason string `json:"reason" binding:"required"` // spam, duplicate, bad_location
}

func NewEventHandler(cfg *config.Config, db *gorm.DB) *EventHandler {
	return &EventHandler{
		config: cfg,
		db:     db,
	}
}

// List returns events in GeoJSON format with optional filtering
// GET /v1/events?bbox=w,s,e,n&start_date=2024-01-01&end_date=2024-12-31&keyword=music&include_past=true
func (h *EventHandler) List(c *gin.Context) {
	query := h.db.Model(&models.Event{}).
		Preload("Venue").
		Where("moderation_state = ?", "approved")

	// By default, only show future events unless include_past=true
	if c.Query("include_past") != "true" {
		query = query.Where("start_ts > ?", time.Now())
	}

	// Apply filters
	if bbox := c.Query("bbox"); bbox != "" {
		coords := strings.Split(bbox, ",")
		if len(coords) == 4 {
			// TODO: Add spatial filtering with PostGIS
			// For now, skip bbox filtering
		}
	}

	if startDate := c.Query("start_date"); startDate != "" {
		if start, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("start_ts >= ?", start)
		}
	}

	if endDate := c.Query("end_date"); endDate != "" {
		if end, err := time.Parse("2006-01-02", endDate); err == nil {
			query = query.Where("start_ts <= ?", end)
		}
	}

	if keyword := c.Query("keyword"); keyword != "" {
		searchTerm := "%" + keyword + "%"
		query = query.Where("title ILIKE ? OR description ILIKE ?", searchTerm, searchTerm)
	}

	// Pagination
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 500 {
			limit = parsedLimit
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	var events []models.Event
	if err := query.Limit(limit).Offset(offset).Order("start_ts ASC").Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to fetch events",
			},
		})
		return
	}

	// Convert to GeoJSON
	geoJSON := EventGeoJSON{
		Type:     "FeatureCollection",
		Features: make([]EventFeature, 0, len(events)),
	}

	for _, event := range events {
		feature := EventFeature{
			Type: "Feature",
			ID:   event.ID.String(),
			Properties: EventProperties{
				Title:       event.Title,
				StartTs:     event.StartTs,
				EndTs:       event.EndTs,
				URL:         event.URL,
				Price:       event.Price,
				Description: event.Description,
				Organizer:   event.Organizer,
				Source:      event.Source,
			},
		}

		if event.Venue != nil {
			feature.Properties.VenueName = &event.Venue.Name
			feature.Properties.Address = event.Venue.AddressLine

			// TODO: Parse PostGIS location to get coordinates
			// For now, use dummy coordinates
			feature.Geometry = EventGeometry{
				Type:        "Point",
				Coordinates: []float64{-122.4194, 37.7749}, // SF default
			}
		}

		geoJSON.Features = append(geoJSON.Features, feature)
	}

	c.JSON(http.StatusOK, geoJSON)
}

// Get returns a single event by ID
// GET /v1/events/{id}
func (h *EventHandler) Get(c *gin.Context) {
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid event ID",
			},
		})
		return
	}

	var event models.Event
	if err := h.db.Preload("Venue").First(&event, "id = ?", eventID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"message": "Event not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Database error",
			},
		})
		return
	}

	c.JSON(http.StatusOK, event)
}

// GetICS returns an event in ICS calendar format
// GET /v1/events/{id}/ics
func (h *EventHandler) GetICS(c *gin.Context) {
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid event ID",
			},
		})
		return
	}

	var event models.Event
	if err := h.db.Preload("Venue").First(&event, "id = ?", eventID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"message": "Event not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Database error",
			},
		})
		return
	}

	// Generate ICS content
	ics := fmt.Sprintf(`BEGIN:VCALENDAR
VERSION:2.0
PRODID:%s
METHOD:PUBLISH
BEGIN:VEVENT
UID:evt_%s@%s
DTSTART:%s
DTEND:%s
SUMMARY:%s
DESCRIPTION:%s
LOCATION:%s
URL:%s
STATUS:CONFIRMED
END:VEVENT
END:VCALENDAR`,
		h.config.ICSProdID,
		event.ID.String(),
		h.config.ICSUIDDomain,
		event.StartTs.UTC().Format("20060102T150405Z"),
		func() string {
			if event.EndTs != nil {
				return event.EndTs.UTC().Format("20060102T150405Z")
			}
			return event.StartTs.Add(2 * time.Hour).UTC().Format("20060102T150405Z")
		}(),
		strings.ReplaceAll(event.Title, ",", "\\,"),
		func() string {
			if event.Description != nil {
				return strings.ReplaceAll(*event.Description, ",", "\\,")
			}
			return ""
		}(),
		func() string {
			if event.Venue != nil {
				location := event.Venue.Name
				if event.Venue.AddressLine != nil {
					location += ", " + *event.Venue.AddressLine
				}
				return strings.ReplaceAll(location, ",", "\\,")
			}
			return ""
		}(),
		func() string {
			if event.URL != nil {
				return *event.URL
			}
			return ""
		}(),
	)

	c.Header("Content-Type", "text/calendar; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"event_%s.ics\"", event.ID.String()))
	c.String(http.StatusOK, ics)
}

// Unpublish removes an event from public listing
// POST /v1/events/{id}/unpublish
func (h *EventHandler) Unpublish(c *gin.Context) {
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid event ID",
			},
		})
		return
	}

	var req UnpublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request format",
				"details": err.Error(),
			},
		})
		return
	}

	// Validate reason
	validReasons := []string{"spam", "duplicate", "bad_location", "inappropriate"}
	isValidReason := false
	for _, validReason := range validReasons {
		if req.Reason == validReason {
			isValidReason = true
			break
		}
	}

	if !isValidReason {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid unpublish reason",
			},
		})
		return
	}

	// Update event moderation state
	result := h.db.Model(&models.Event{}).
		Where("id = ?", eventID).
		Updates(map[string]interface{}{
			"moderation_state": "blocked",
			"updated_at":       time.Now(),
		})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to unpublish event",
			},
		})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"message": "Event not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Event unpublished successfully",
		"reason":  req.Reason,
	})
}