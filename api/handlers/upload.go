package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lincolngreen/williamboard/api/config"
	"github.com/lincolngreen/williamboard/api/models"
	"github.com/lincolngreen/williamboard/api/services"
	"gorm.io/gorm"
)

type UploadHandler struct {
	config     *config.Config
	db         *gorm.DB
	storage    *services.StorageService
	vision     *services.VisionService
	moderation *services.ModerationService
	geocoding  *services.GeocodingService
}

type SignedURLRequest struct {
	ContentType  string     `json:"contentType" binding:"required"`
	SubmissionID *uuid.UUID `json:"submissionId"`
}

func NewUploadHandler(cfg *config.Config, db *gorm.DB, storage *services.StorageService) *UploadHandler {
	vision := services.NewVisionService(cfg)
	moderation := services.NewModerationService(cfg)
	geocoding := services.NewGeocodingService(cfg)
	
	return &UploadHandler{
		config:     cfg,
		db:         db,
		storage:    storage,
		vision:     vision,
		moderation: moderation,
		geocoding:  geocoding,
	}
}

// GetSignedURL generates an upload URL for direct file upload
// POST /v1/uploads/signed-url
func (h *UploadHandler) GetSignedURL(c *gin.Context) {
	var req SignedURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request format",
				"details": err.Error(),
			},
		})
		return
	}

	// Validate content type
	allowedTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/webp"}
	isValidType := false
	for _, allowedType := range allowedTypes {
		if req.ContentType == allowedType {
			isValidType = true
			break
		}
	}
	
	if !isValidType {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid content type. Allowed: jpeg, jpg, png, webp",
			},
		})
		return
	}

	// Generate submission ID if not provided
	submissionID := uuid.New()
	if req.SubmissionID != nil {
		submissionID = *req.SubmissionID
	}

	// Create submission record
	submission := models.Submission{
		ID:               submissionID,
		OriginalImageURL: h.storage.GetOriginalImageURL(submissionID),
		Status:           "uploaded",
	}

	if err := h.db.Create(&submission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to create submission record",
			},
		})
		return
	}

	// Generate upload URL
	result := h.storage.GenerateUploadURL(submissionID)
	c.JSON(http.StatusOK, result)
}

// UploadFile handles direct file upload
// PUT /v1/uploads/{id}
func (h *UploadHandler) UploadFile(c *gin.Context) {
	submissionIDStr := c.Param("id")
	submissionID, err := uuid.Parse(submissionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid submission ID",
			},
		})
		return
	}

	// Check if submission exists
	var submission models.Submission
	if err := h.db.First(&submission, "id = ?", submissionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"message": "Submission not found",
			},
		})
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "No file uploaded",
				"details": err.Error(),
			},
		})
		return
	}
	defer file.Close()

	// Validate file size (12MB max)
	if header.Size > 12*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "File too large. Maximum size is 12MB",
			},
		})
		return
	}

	// Save file
	if err := h.storage.SaveFile(submissionID, "original.jpg", file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to save file",
			},
		})
		return
	}

	// Process immediately (synchronous)
	if err := h.processUploadSync(submissionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to process image",
				"details": err.Error(),
			},
		})
		return
	}

	// Get results after processing
	if err := h.db.Preload("Flyers.EventCandidates").First(&submission, "id = ?", submissionID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to retrieve results",
			},
		})
		return
	}

	// Count found events
	eventCount := 0
	for _, flyer := range submission.Flyers {
		eventCount += len(flyer.EventCandidates)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Image processed successfully",
		"submissionId":  submissionID.String(),
		"status":        submission.Status,
		"eventsFound":   eventCount,
		"flyersFound":   len(submission.Flyers),
	})
}

// processUploadSync processes the upload synchronously with GPT-4o Vision
func (h *UploadHandler) processUploadSync(submissionID uuid.UUID) error {
	// Update status to processing
	if err := h.updateSubmissionStatus(submissionID, "processing"); err != nil {
		return err
	}

	// Get the image file path
	imagePath := h.storage.GetFilePath(submissionID, "original.jpg")
	
	// Process with GPT-4o Vision directly
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	
	result, err := h.vision.AnalyzeImage(ctx, submissionID, imagePath)
	if err != nil {
		// Update status to error
		if statusErr := h.updateSubmissionStatus(submissionID, "error"); statusErr != nil {
			return fmt.Errorf("vision analysis failed: %w, status update failed: %v", err, statusErr)
		}
		return fmt.Errorf("vision analysis failed: %w", err)
	}

	// Save vision results to database
	if err := h.vision.SaveResults(h.db, submissionID, result); err != nil {
		if statusErr := h.updateSubmissionStatus(submissionID, "error"); statusErr != nil {
			return fmt.Errorf("failed to save results: %w, status update failed: %v", err, statusErr)
		}
		return fmt.Errorf("failed to save results: %w", err)
	}

	// Update status to parsed (Stage 2 complete)
	if err := h.updateSubmissionStatus(submissionID, "parsed"); err != nil {
		return err
	}

	// *** STAGE 3: MODERATION + GEOCODING ***
	
	// Process moderation and geocoding for each event candidate
	if err := h.processStage3(ctx, submissionID); err != nil {
		if statusErr := h.updateSubmissionStatus(submissionID, "error"); statusErr != nil {
			return fmt.Errorf("Stage 3 processing failed: %w, status update failed: %v", err, statusErr)
		}
		return fmt.Errorf("Stage 3 processing failed: %w", err)
	}

	// Update final status to done
	if err := h.updateSubmissionStatus(submissionID, "done"); err != nil {
		return err
	}

	return nil
}

// processStage3 handles moderation and geocoding
func (h *UploadHandler) processStage3(ctx context.Context, submissionID uuid.UUID) error {
	// Get all event candidates for this submission
	var eventCandidates []models.EventCandidate
	if err := h.db.Joins("JOIN flyers ON flyers.id = event_candidates.flyer_id").
		Where("flyers.submission_id = ?", submissionID).
		Find(&eventCandidates).Error; err != nil {
		return fmt.Errorf("failed to fetch event candidates: %w", err)
	}

	log.Printf("Processing Stage 3 for %d event candidates", len(eventCandidates))

	// Process each event candidate
	for _, candidate := range eventCandidates {
		if err := h.processEventCandidate(ctx, &candidate); err != nil {
			log.Printf("Failed to process event candidate %s: %v", candidate.ID, err)
			// Continue processing other candidates even if one fails
			continue
		}
	}

	return nil
}

// processEventCandidate processes a single event candidate through moderation and geocoding
func (h *UploadHandler) processEventCandidate(ctx context.Context, candidate *models.EventCandidate) error {
	// Parse event fields from JSON
	var eventData map[string]interface{}
	if err := json.Unmarshal([]byte(candidate.Fields), &eventData); err != nil {
		return fmt.Errorf("failed to parse event fields: %w", err)
	}

	// *** MODERATION ***
	log.Printf("Moderating event candidate %s", candidate.ID)
	moderationResult, err := h.moderation.ModerateEventCandidate(ctx, eventData)
	if err != nil {
		log.Printf("Moderation failed for %s: %v", candidate.ID, err)
		// Use default values if moderation fails
		moderationResult = &services.ModerationResult{
			QualityScore:  0.5,
			IsAppropriate: true,
		}
	}

	// Store composite score and publish decision
	candidate.CompositeScore = &moderationResult.QualityScore
	
	if !moderationResult.IsAppropriate {
		blocked := "blocked"
		candidate.PublishResult = &blocked
		candidate.PublicationReason = moderationResult.ModerationReason
	} else if moderationResult.QualityScore >= h.config.AutoPublishThreshold {
		published := "published"
		candidate.PublishResult = &published
		reason := "auto-published (high quality score)"
		candidate.PublicationReason = &reason
	} else {
		needsReview := "needs_review"
		candidate.PublishResult = &needsReview
		reason := "requires manual review (low quality score)"
		candidate.PublicationReason = &reason
	}

	// *** GEOCODING ***
	venueAddress := extractVenueAddress(eventData)
	if venueAddress != "" {
		log.Printf("Geocoding venue address for %s: %s", candidate.ID, venueAddress)
		geocodeResult, err := h.geocoding.GeocodeAddress(ctx, venueAddress)
		if err != nil {
			log.Printf("Geocoding failed for %s: %v", candidate.ID, err)
		} else {
			// Store geocoding result
			geocodeJSON, _ := json.Marshal(geocodeResult)
			geocodeStr := string(geocodeJSON)
			candidate.Geocode = &geocodeStr
			
			// Create or update venue record if high confidence
			if geocodeResult.Confidence >= h.config.GeoConfThreshold {
				if err := h.createOrUpdateVenue(eventData, geocodeResult); err != nil {
					log.Printf("Failed to create/update venue for %s: %v", candidate.ID, err)
				}
			}
		}
	}

	// Save updated candidate
	if err := h.db.Save(candidate).Error; err != nil {
		return fmt.Errorf("failed to save moderated candidate: %w", err)
	}

	log.Printf("Completed Stage 3 for candidate %s: score=%.2f, decision=%s", 
		candidate.ID, *candidate.CompositeScore, *candidate.PublishResult)

	return nil
}

// extractVenueAddress extracts venue address from event data
func extractVenueAddress(eventData map[string]interface{}) string {
	// Try different field names that might contain address info
	addressFields := []string{"venue", "address", "location", "where"}
	
	for _, field := range addressFields {
		if value, ok := eventData[field].(string); ok && value != "" {
			return value
		}
	}
	
	return ""
}

// createOrUpdateVenue creates or updates venue record with geocoded data
func (h *UploadHandler) createOrUpdateVenue(eventData map[string]interface{}, geocodeResult *services.GeocodeResult) error {
	venueName := ""
	if name, ok := eventData["venue"].(string); ok {
		venueName = name
	}
	
	if venueName == "" {
		return fmt.Errorf("no venue name found")
	}

	// Create PostGIS point
	locationWKT := fmt.Sprintf("POINT(%f %f)", geocodeResult.Longitude, geocodeResult.Latitude)

	// Try to find existing venue
	var venue models.Venue
	err := h.db.Where("name = ? AND city = ?", venueName, geocodeResult.Components["city"]).First(&venue).Error
	
	if err == gorm.ErrRecordNotFound {
		// Create new venue
		city := geocodeResult.Components["city"]
		state := geocodeResult.Components["state"]
		postalCode := geocodeResult.Components["postal_code"]
		country := geocodeResult.Components["country"]
		
		venue = models.Venue{
			Name:              venueName,
			AddressLine:       &geocodeResult.FormattedAddress,
			City:              &city,
			State:             &state,
			PostalCode:        &postalCode,
			Country:           country,
			Location:          &locationWKT,
			GeocodeConfidence: &geocodeResult.Confidence,
		}
		
		// Store raw geocode data
		geocodeDataJSON, _ := json.Marshal(geocodeResult.RawResponse)
		geocodeDataStr := string(geocodeDataJSON)
		venue.GeocodeData = &geocodeDataStr
		
		if err := h.db.Create(&venue).Error; err != nil {
			return fmt.Errorf("failed to create venue: %w", err)
		}
		
		log.Printf("Created new venue: %s", venueName)
	} else if err != nil {
		return fmt.Errorf("failed to query venues: %w", err)
	} else {
		// Update existing venue if confidence is higher
		if venue.GeocodeConfidence == nil || geocodeResult.Confidence > *venue.GeocodeConfidence {
			venue.Location = &locationWKT
			venue.GeocodeConfidence = &geocodeResult.Confidence
			venue.AddressLine = &geocodeResult.FormattedAddress
			
			if err := h.db.Save(&venue).Error; err != nil {
				return fmt.Errorf("failed to update venue: %w", err)
			}
			
			log.Printf("Updated existing venue: %s", venueName)
		}
	}
	
	return nil
}

// updateSubmissionStatus updates the submission status in the database
func (h *UploadHandler) updateSubmissionStatus(submissionID uuid.UUID, status string) error {
	return h.db.Model(&models.Submission{}).
		Where("id = ?", submissionID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}