package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lincolngreen/williamboard/api/config"
	"github.com/lincolngreen/williamboard/api/models"
	"gorm.io/gorm"
)

type AdminHandler struct {
	config *config.Config
	db     *gorm.DB
}

type AdminEventCandidate struct {
	ID               string                 `json:"id"`
	SubmissionID     string                 `json:"submission_id"`
	FlyerID          string                 `json:"flyer_id"`
	EventID          string                 `json:"event_id"`
	Fields           map[string]interface{} `json:"fields"`
	Confidences      map[string]interface{} `json:"confidences"`
	SourceExcerpt    *string                `json:"source_excerpt"`
	Geocode          *map[string]interface{} `json:"geocode"`
	CompositeScore   *float64               `json:"composite_score"`
	PublishResult    *string                `json:"publish_result"`
	PublicationReason *string               `json:"publication_reason"`
	CreatedAt        time.Time              `json:"created_at"`
	
	// Derived fields for display
	Title            string     `json:"title"`
	Date             string     `json:"date"`
	Venue            string     `json:"venue"`
	Address          string     `json:"address"`
	Confidence       float64    `json:"confidence"`
	QualityScore     float64    `json:"quality_score"`  // Convert pointer to value for template
	Status           string     `json:"status"`
	StatusColor      string     `json:"status_color"`
	OriginalImageURL string     `json:"original_image_url"`
	ThumbnailURL     string     `json:"thumbnail_url"`
	PublishedEventStartTime *time.Time `json:"published_event_start_time"` // When the published event is scheduled
}

func NewAdminHandler(cfg *config.Config, db *gorm.DB) *AdminHandler {
	return &AdminHandler{
		config: cfg,
		db:     db,
	}
}

// AdminDashboard shows all event candidates in a table
// GET /admin
func (h *AdminHandler) AdminDashboard(c *gin.Context) {
	// Get all event candidates with related data including submission for images
	var candidates []models.EventCandidate
	if err := h.db.Preload("Flyer.Submission").Order("created_at DESC").Find(&candidates).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "admin.html", gin.H{
			"error": "Failed to load event candidates",
		})
		return
	}

	// Transform for display
	adminCandidates := make([]AdminEventCandidate, len(candidates))
	for i, candidate := range candidates {
		adminCandidate := h.transformEventCandidate(&candidate)
		adminCandidates[i] = adminCandidate
	}

	// Get summary stats
	stats := h.getAdminStats()

	c.HTML(http.StatusOK, "admin.html", gin.H{
		"candidates": adminCandidates,
		"stats":      stats,
		"title":      "WilliamBoard Admin",
	})
}

// transformEventCandidate converts model to display format
func (h *AdminHandler) transformEventCandidate(candidate *models.EventCandidate) AdminEventCandidate {
	admin := AdminEventCandidate{
		ID:                candidate.ID.String(),
		FlyerID:           candidate.FlyerID.String(),
		EventID:           candidate.EventID,
		SourceExcerpt:     candidate.SourceExcerpt,
		CompositeScore:    candidate.CompositeScore,
		PublishResult:     candidate.PublishResult,
		PublicationReason: candidate.PublicationReason,
		CreatedAt:         candidate.CreatedAt,
		Confidence:        0.0, // Initialize to valid float64
		QualityScore:      0.0, // Initialize to valid float64
	}
	
	// Convert CompositeScore pointer to value for template display
	if candidate.CompositeScore != nil {
		admin.QualityScore = *candidate.CompositeScore
	}
	
	// Set image URLs from the submission
	if candidate.Flyer.Submission.OriginalImageURL != "" {
		admin.OriginalImageURL = candidate.Flyer.Submission.OriginalImageURL
		// Use the original image as thumbnail for now
		admin.ThumbnailURL = candidate.Flyer.Submission.OriginalImageURL
		
		// If there's a derivative image, use that as thumbnail instead
		if candidate.Flyer.Submission.DerivativeImageURL != nil && *candidate.Flyer.Submission.DerivativeImageURL != "" {
			admin.ThumbnailURL = *candidate.Flyer.Submission.DerivativeImageURL
		}
	}

	// Parse fields JSON
	var fields map[string]interface{}
	if err := json.Unmarshal([]byte(candidate.Fields), &fields); err == nil {
		admin.Fields = fields

		// Extract common display fields
		if title, ok := fields["title"].(string); ok {
			admin.Title = title
		}
		// Check both "date" and "date_time" fields for compatibility
		if date, ok := fields["date"].(string); ok {
			admin.Date = date
			fmt.Printf("Found 'date' field: %s for candidate %s\n", date, candidate.ID.String())
		} else if dateTime, ok := fields["date_time"].(string); ok {
			admin.Date = dateTime
			fmt.Printf("Found 'date_time' field: %s for candidate %s\n", dateTime, candidate.ID.String())
		} else {
			fmt.Printf("No date found for candidate %s, fields: %+v\n", candidate.ID.String(), fields)
		}
		if venue, ok := fields["venue"].(string); ok {
			admin.Venue = venue
		}
		if addr, ok := fields["address"].(string); ok {
			admin.Address = addr
		}
	}

	// Parse confidences JSON
	var confidences map[string]interface{}
	if err := json.Unmarshal([]byte(candidate.Confidences), &confidences); err == nil {
		admin.Confidences = confidences
		
		// Calculate average confidence safely
		total := 0.0
		count := 0
		for _, conf := range confidences {
			if confFloat, ok := conf.(float64); ok && confFloat >= 0.0 && confFloat <= 1.0 {
				total += confFloat
				count++
			}
		}
		if count > 0 {
			admin.Confidence = total / float64(count)
		} else {
			admin.Confidence = 0.0  // Explicitly set to 0.0 for template comparison
		}
	} else {
		admin.Confidence = 0.0  // Explicitly set to 0.0 when JSON parsing fails
	}

	// Parse geocoding if available
	if candidate.Geocode != nil {
		var geocode map[string]interface{}
		if err := json.Unmarshal([]byte(*candidate.Geocode), &geocode); err == nil {
			admin.Geocode = &geocode
		}
	}

	// Set status and color for display
	admin.Status, admin.StatusColor = h.getStatusDisplay(candidate.PublishResult, candidate.CompositeScore)

	// If this event is published, look up the published event timestamp
	if candidate.PublishResult != nil && *candidate.PublishResult == "published" {
		var publishedEvent models.Event
		// Look for events with matching title (case-insensitive)
		titlePattern := "%" + strings.ToLower(strings.TrimSpace(admin.Title)) + "%"
		if err := h.db.Where("LOWER(title) LIKE ? AND moderation_state = ?", titlePattern, "approved").First(&publishedEvent).Error; err == nil {
			admin.PublishedEventStartTime = &publishedEvent.StartTs
		}
	}

	return admin
}

// getStatusDisplay returns human-readable status and color
func (h *AdminHandler) getStatusDisplay(publishResult *string, score *float64) (string, string) {
	if publishResult == nil {
		return "Processing", "gray"
	}

	switch *publishResult {
	case "published":
		return "Published", "green"
	case "blocked":
		return "Blocked", "red"
	case "needs_review":
		return "Needs Review", "orange"
	default:
		if score != nil {
			return "Processed", "blue"
		}
		return "Unknown", "gray"
	}
}

// getAdminStats returns summary statistics
func (h *AdminHandler) getAdminStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// Count by status
	var published, blocked, needsReview, total int64
	h.db.Model(&models.EventCandidate{}).Where("publish_result = ?", "published").Count(&published)
	h.db.Model(&models.EventCandidate{}).Where("publish_result = ?", "blocked").Count(&blocked)
	h.db.Model(&models.EventCandidate{}).Where("publish_result = ?", "needs_review").Count(&needsReview)
	h.db.Model(&models.EventCandidate{}).Count(&total)

	stats["total"] = total
	stats["published"] = published
	stats["blocked"] = blocked
	stats["needs_review"] = needsReview
	stats["processing"] = total - published - blocked - needsReview

	// Recent activity (last 24h)
	var recent int64
	h.db.Model(&models.EventCandidate{}).Where("created_at > ?", time.Now().Add(-24*time.Hour)).Count(&recent)
	stats["recent_24h"] = recent

	return stats
}

// ModerateEvent handles approval/rejection of events
// POST /admin/moderate/:id
func (h *AdminHandler) ModerateEvent(c *gin.Context) {
	eventID := c.Param("id")
	action := c.PostForm("action")
	reason := c.PostForm("reason")

	if action != "approve" && action != "reject" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action"})
		return
	}

	// Find the event candidate with related data
	var candidate models.EventCandidate
	if err := h.db.Preload("Flyer.Submission").Where("id = ?", eventID).First(&candidate).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Update publish result
	var publishResult string
	if action == "approve" {
		publishResult = "published"
	} else {
		publishResult = "blocked"
	}

	// Start a transaction to handle both candidate update and event creation
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update the candidate record
	updates := map[string]interface{}{
		"publish_result": publishResult,
	}
	if reason != "" {
		updates["publication_reason"] = reason
	}

	if err := tx.Model(&candidate).Updates(updates).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update event"})
		return
	}

	// If approved, create/update the public Event record
	if action == "approve" {
		if err := h.promoteToPublicEvent(tx, &candidate); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish event: " + err.Error()})
			return
		}
	}

	tx.Commit()

	// Return success for HTMX/AJAX requests or redirect for form submissions
	if c.GetHeader("HX-Request") == "true" || c.GetHeader("Accept") == "application/json" {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"status": publishResult,
		})
	} else {
		c.Redirect(http.StatusSeeOther, "/admin")
	}
}

// promoteToPublicEvent creates an Event record from an approved EventCandidate
func (h *AdminHandler) promoteToPublicEvent(tx *gorm.DB, candidate *models.EventCandidate) error {
	// Parse the fields JSON to extract event data
	var fields map[string]interface{}
	if err := json.Unmarshal([]byte(candidate.Fields), &fields); err != nil {
		return fmt.Errorf("failed to parse event fields: %v", err)
	}

	// Extract required title field
	title, ok := fields["title"].(string)
	if !ok || title == "" {
		return errors.New("event title is required")
	}

	// Parse start time - try different formats
	startTs := time.Now().Add(24 * time.Hour) // fallback to tomorrow to ensure future events
	
	// Check both "date" and "date_time" fields for compatibility
	var dateStr string
	if date, ok := fields["date"].(string); ok && date != "" {
		dateStr = date
	} else if dateTime, ok := fields["date_time"].(string); ok && dateTime != "" {
		dateStr = dateTime
	}
	
	if dateStr != "" {
		fmt.Printf("Parsing date string: %s for event: %s\n", dateStr, title)
		// Try parsing different date formats
		formats := []string{
			"2006-01-02T15:04:05",    // ISO format first (most common from LLM)
			"2006-01-02 15:04:05",
			"2006-01-02T15:04",
			"2006-01-02 15:04",
			"2006-01-02",
			"January 2, 2006",
			"Jan 2, 2006",
		}
		
		parsed := false
		for _, format := range formats {
			if parsedTime, err := time.Parse(format, dateStr); err == nil {
				fmt.Printf("Successfully parsed '%s' as '%s' using format '%s'\n", dateStr, parsedTime.String(), format)
				// If the parsed date is in the past, assume it's for next year
				if parsedTime.Before(time.Now()) {
					parsedTime = parsedTime.AddDate(1, 0, 0)
					fmt.Printf("Date was in past, moved to next year: %s\n", parsedTime.String())
				}
				startTs = parsedTime
				parsed = true
				break
			}
		}
		
		// If we couldn't parse the date, keep the fallback
		if !parsed {
			fmt.Printf("Failed to parse date '%s', using fallback\n", dateStr)
			startTs = time.Now().Add(24 * time.Hour)
		} else {
			fmt.Printf("Final startTs for event '%s': %s\n", title, startTs.String())
		}
	}

	// Create canonical key for deduplication (title + date)
	canonicalKey := strings.ToLower(strings.TrimSpace(title)) + "_" + startTs.Format("2006-01-02")

	// Check if this event already exists
	var existingEvent models.Event
	if err := tx.Where("canonical_key = ?", canonicalKey).First(&existingEvent).Error; err == nil {
		// Event already exists, just update moderation state if needed
		if existingEvent.ModerationState != "approved" {
			return tx.Model(&existingEvent).Update("moderation_state", "approved").Error
		}
		return nil // Already published
	}

	// Create new Event record
	event := models.Event{
		CanonicalKey:    canonicalKey,
		Title:           title,
		StartTs:         startTs,
		Source:          "flyer",
		PublishedVia:    "manual",
		QualityScore:    candidate.CompositeScore,
		ModerationState: "approved",
	}

	// Extract optional fields
	if desc, ok := fields["description"].(string); ok && desc != "" {
		event.Description = &desc
	}
	if url, ok := fields["url"].(string); ok && url != "" {
		event.URL = &url
	}
	if price, ok := fields["price"].(string); ok && price != "" {
		event.Price = &price
	}
	if organizer, ok := fields["organizer"].(string); ok && organizer != "" {
		event.Organizer = &organizer
	}
	
	// Handle end time if provided
	if endStr, ok := fields["end_date"].(string); ok && endStr != "" {
		// Try parsing end time
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
			"2006-01-02 15:04",
			"2006-01-02T15:04",
			"2006-01-02",
		}
		
		for _, format := range formats {
			if parsed, err := time.Parse(format, endStr); err == nil {
				event.EndTs = &parsed
				break
			}
		}
	}

	// Handle venue
	if venueName, ok := fields["venue"].(string); ok && venueName != "" {
		// Check if venue already exists
		var venue models.Venue
		if err := tx.Where("name ILIKE ?", venueName).First(&venue).Error; err != nil {
			// Create new venue
			venue = models.Venue{
				Name: venueName,
			}
			
			// Add address if available
			if addr, ok := fields["address"].(string); ok && addr != "" {
				venue.AddressLine = &addr
			}
			
			if err := tx.Create(&venue).Error; err != nil {
				return fmt.Errorf("failed to create venue: %v", err)
			}
		}
		event.VenueID = &venue.ID
	}

	// Create the event
	if err := tx.Create(&event).Error; err != nil {
		return fmt.Errorf("failed to create event: %v", err)
	}

	return nil
}

// GetRawEventCandidate returns raw LLM response for debugging
// GET /admin/raw/:id
func (h *AdminHandler) GetRawEventCandidate(c *gin.Context) {
	candidateID := c.Param("id")

	var candidate models.EventCandidate
	if err := h.db.Preload("Flyer.Submission").Where("id = ?", candidateID).First(&candidate).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event candidate not found"})
		return
	}

	// Parse fields and confidences for easier reading
	var fields map[string]interface{}
	var confidences map[string]interface{}
	var geocode map[string]interface{}

	json.Unmarshal([]byte(candidate.Fields), &fields)
	json.Unmarshal([]byte(candidate.Confidences), &confidences)
	if candidate.Geocode != nil {
		json.Unmarshal([]byte(*candidate.Geocode), &geocode)
	}

	response := gin.H{
		"id":                candidate.ID.String(),
		"flyer_id":          candidate.FlyerID.String(),
		"event_id":          candidate.EventID,
		"fields":            fields,
		"confidences":       confidences,
		"geocode":          geocode,
		"composite_score":   candidate.CompositeScore,
		"publish_result":    candidate.PublishResult,
		"publication_reason": candidate.PublicationReason,
		"source_excerpt":    candidate.SourceExcerpt,
		"created_at":        candidate.CreatedAt,
		"submission":        candidate.Flyer.Submission,
	}

	c.JSON(http.StatusOK, response)
}

// RegisterAdminRoutes adds admin routes to the router
func RegisterAdminRoutes(router *gin.RouterGroup, handler *AdminHandler) {
	router.GET("", handler.AdminDashboard)
	router.POST("/moderate/:id", handler.ModerateEvent)
	router.GET("/raw/:id", handler.GetRawEventCandidate)
}