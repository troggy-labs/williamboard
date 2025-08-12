package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lincolngreen/williamboard/api/config"
	"github.com/lincolngreen/williamboard/api/models"
	"gorm.io/gorm"
)

type SubmissionHandler struct {
	config *config.Config
	db     *gorm.DB
}

type SubmissionStatus struct {
	Status     string                    `json:"status"`
	Step       string                    `json:"step,omitempty"`
	Flyers     []FlyerStatusResult       `json:"flyers,omitempty"`
	Candidates []CandidateStatusResult   `json:"candidates,omitempty"`
	Error      *string                   `json:"error,omitempty"`
}

type FlyerStatusResult struct {
	FlyerID              string  `json:"flyerId"`
	RegionID             string  `json:"regionId"`
	ImageURL             string  `json:"imageUrl"`
	DetectionConfidence  float64 `json:"detectionConfidence"`
}

type CandidateStatusResult struct {
	CandidateID string  `json:"candidateId"`
	Decision    string  `json:"decision"`
	Score       float64 `json:"score"`
	EventID     *string `json:"eventId,omitempty"`
	Reason      *string `json:"reason,omitempty"`
}

func NewSubmissionHandler(cfg *config.Config, db *gorm.DB) *SubmissionHandler {
	return &SubmissionHandler{
		config: cfg,
		db:     db,
	}
}


// GetStatus returns the current processing status of a submission
// GET /v1/submissions/{id}/status
func (h *SubmissionHandler) GetStatus(c *gin.Context) {
	submissionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid submission ID",
			},
		})
		return
	}

	// Find the submission with related data
	var submission models.Submission
	if err := h.db.Preload("Flyers.EventCandidates").First(&submission, "id = ?", submissionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"message": "Submission not found",
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

	status := SubmissionStatus{
		Status: submission.Status,
	}

	// Determine processing step
	switch submission.Status {
	case "uploaded":
		status.Step = "uploaded"
	case "processing":
		status.Step = "extracting"
	case "parsed":
		status.Step = "moderating"
	case "moderated":
		status.Step = "geocoding"
	case "geocoded":
		status.Step = "publishing"
	case "done":
		status.Step = "done"
	case "error":
		status.Step = "error"
		errorMsg := "Processing failed"
		status.Error = &errorMsg
	}

	// Add flyer results if available
	for _, flyer := range submission.Flyers {
		flyerResult := FlyerStatusResult{
			FlyerID:             flyer.ID.String(),
			RegionID:            flyer.RegionID,
			DetectionConfidence: flyer.DetectionConfidence,
		}
		
		if flyer.CropImageURL != nil {
			flyerResult.ImageURL = *flyer.CropImageURL
		}
		
		status.Flyers = append(status.Flyers, flyerResult)

		// Add candidate results
		for _, candidate := range flyer.EventCandidates {
			candidateResult := CandidateStatusResult{
				CandidateID: candidate.ID.String(),
			}
			
			if candidate.PublishResult != nil {
				candidateResult.Decision = *candidate.PublishResult
			}
			
			if candidate.CompositeScore != nil {
				candidateResult.Score = *candidate.CompositeScore
			}
			
			if candidate.PublicationReason != nil {
				candidateResult.Reason = candidate.PublicationReason
			}
			
			status.Candidates = append(status.Candidates, candidateResult)
		}
	}

	c.JSON(http.StatusOK, status)
}