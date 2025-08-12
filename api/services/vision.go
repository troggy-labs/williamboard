package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
	config_pkg "github.com/lincolngreen/williamboard/api/config"
	"github.com/lincolngreen/williamboard/api/models"
	"gorm.io/gorm"
)

type VisionService struct {
	client *openai.Client
	config *config_pkg.Config
}

// FlyerDetectionResult represents the structured output from GPT-4o
type FlyerDetectionResult struct {
	FlyersDetected []FlyerRegion `json:"flyers_detected"`
	TotalRegions   int           `json:"total_regions"`
	ImageQuality   string        `json:"image_quality"` // "excellent", "good", "fair", "poor"
	ProcessingNotes string       `json:"processing_notes"`
}

// FlyerRegion represents a detected flyer region
type FlyerRegion struct {
	RegionID    string             `json:"region_id"`
	Confidence  float64            `json:"confidence"`
	Polygon     []Point            `json:"polygon"`
	Rotation    *float64           `json:"rotation_deg,omitempty"`
	Events      []EventCandidate   `json:"events"`
	Notes       string             `json:"notes"`
}

// Point represents a coordinate point
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// EventCandidate represents an extracted event
type EventCandidate struct {
	EventID     string            `json:"event_id"`
	Fields      EventFields       `json:"fields"`
	Confidences EventConfidences  `json:"confidences"`
	Excerpt     string            `json:"source_excerpt"`
}

// EventFields contains the extracted event data
type EventFields struct {
	Title        string    `json:"title"`
	DateTime     *string   `json:"date_time,omitempty"`
	StartTime    *string   `json:"start_time,omitempty"`  
	EndTime      *string   `json:"end_time,omitempty"`
	Venue        *string   `json:"venue,omitempty"`
	Address      *string   `json:"address,omitempty"`
	Price        *string   `json:"price,omitempty"`
	Description  *string   `json:"description,omitempty"`
	Organizer    *string   `json:"organizer,omitempty"`
	URL          *string   `json:"url,omitempty"`
	ContactInfo  *string   `json:"contact_info,omitempty"`
	Category     *string   `json:"category,omitempty"`
	AgeRestriction *string `json:"age_restriction,omitempty"`
}

// EventConfidences contains confidence scores for each field
type EventConfidences struct {
	Title     float64 `json:"title"`
	DateTime  float64 `json:"date_time"`
	Location  float64 `json:"location"`
	Overall   float64 `json:"overall"`
}

func NewVisionService(cfg *config_pkg.Config) *VisionService {
	client := openai.NewClient(cfg.OpenAIAPIKey)
	
	return &VisionService{
		client: client,
		config: cfg,
	}
}

// AnalyzeImage processes an image to detect flyers and extract events
func (v *VisionService) AnalyzeImage(ctx context.Context, submissionID uuid.UUID, imagePath string) (*FlyerDetectionResult, error) {
	// Read and encode image
	imageData, err := v.prepareImage(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare image: %w", err)
	}

	// Create the prompt for structured analysis
	prompt := v.createAnalysisPrompt()

	// Call GPT-4o Vision with structured output
	req := openai.ChatCompletionRequest{
		Model: v.config.OpenAIModel,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: prompt,
					},
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL: fmt.Sprintf("data:image/jpeg;base64,%s", imageData),
						},
					},
				},
			},
		},
		MaxTokens:   2000,
		Temperature: 0.1, // Low temperature for consistent structured output
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	}

	// Set timeout context
	ctx, cancel := context.WithTimeout(ctx, time.Duration(v.config.OpenAITimeoutMS)*time.Millisecond)
	defer cancel()

	resp, err := v.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("GPT-4o API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from GPT-4o")
	}

	// Parse structured output
	var result FlyerDetectionResult
	content := resp.Choices[0].Message.Content
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse structured output: %w, content: %s", err, content)
	}

	return &result, nil
}

// prepareImage reads, processes, and encodes image file for optimal GPT-4o Vision analysis
func (v *VisionService) prepareImage(imagePath string) (string, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	// Check file size - GPT-4o has a 20MB limit
	maxSize := 18 * 1024 * 1024 // 18MB to be safe
	if len(data) > maxSize {
		// For now, we'll just truncate to avoid issues
		// TODO: Implement proper image resizing with image/jpeg or similar
		return "", fmt.Errorf("image too large: %d bytes (max %d bytes)", len(data), maxSize)
	}

	// Validate it's a supported image format by checking headers
	if !v.isValidImageFormat(data) {
		return "", fmt.Errorf("unsupported image format")
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

// isValidImageFormat checks if the data represents a valid image format
func (v *VisionService) isValidImageFormat(data []byte) bool {
	if len(data) < 8 {
		return false
	}

	// Check for JPEG
	if data[0] == 0xFF && data[1] == 0xD8 {
		return true
	}

	// Check for PNG
	if len(data) >= 8 && 
		data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 &&
		data[4] == 0x0D && data[5] == 0x0A && data[6] == 0x1A && data[7] == 0x0A {
		return true
	}

	// Check for WebP
	if len(data) >= 12 &&
		data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
		return true
	}

	// Check for GIF
	if len(data) >= 6 &&
		((data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 && data[4] == 0x37 && data[5] == 0x61) ||
		 (data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 && data[4] == 0x39 && data[5] == 0x61)) {
		return true
	}

	return false
}

// createAnalysisPrompt creates the detailed prompt for flyer analysis
func (v *VisionService) createAnalysisPrompt() string {
	return `You are an expert at analyzing bulletin board photos to detect and extract event information from flyers and posters.

Analyze this image and identify all event flyers/posters. For each flyer detected, extract the event details.

Return your analysis in this EXACT JSON format:

{
  "flyers_detected": [
    {
      "region_id": "flyer_1",
      "confidence": 0.95,
      "polygon": [
        {"x": 100, "y": 50},
        {"x": 300, "y": 50}, 
        {"x": 300, "y": 400},
        {"x": 100, "y": 400}
      ],
      "rotation_deg": 0,
      "events": [
        {
          "event_id": "event_1_1",
          "fields": {
            "title": "Summer Music Festival",
            "date_time": "2024-07-15T19:00:00",
            "venue": "Central Park",
            "address": "123 Main St, City, ST 12345",
            "price": "$25",
            "description": "Live music and food trucks",
            "organizer": "Music Society",
            "category": "music"
          },
          "confidences": {
            "title": 0.98,
            "date_time": 0.85,
            "location": 0.90,
            "overall": 0.91
          },
          "source_excerpt": "The text from the flyer that contains this event info"
        }
      ],
      "notes": "Clear, well-lit flyer with all details visible"
    }
  ],
  "total_regions": 1,
  "image_quality": "good",
  "processing_notes": "Clear image with good lighting. Detected 1 flyer containing 1 event."
}

Guidelines:
- Only detect actual event flyers/posters (not ads, notices, or other content)
- Polygon coordinates should outline the flyer boundaries (0,0 = top-left)
- Confidence scores: 0.0-1.0 (0.7+ for reliable detection)
- Parse dates into ISO format when possible, otherwise leave as text
- Extract all visible event details, use null for missing information
- Be conservative with confidence scores - only high confidence for clearly visible text
- If no flyers detected, return empty flyers_detected array

Focus on extracting: title, date/time, venue/location, price, description, organizer, contact info, category.`
}

// SaveResults stores the analysis results in the database
func (v *VisionService) SaveResults(db *gorm.DB, submissionID uuid.UUID, result *FlyerDetectionResult) error {
	// Create flyer records for each detected region
	for _, flyerRegion := range result.FlyersDetected {
		// Convert polygon to JSON
		polygonJSON, err := json.Marshal(flyerRegion.Polygon)
		if err != nil {
			return fmt.Errorf("failed to marshal polygon: %w", err)
		}

		// Create flyer record
		flyer := models.Flyer{
			SubmissionID:        submissionID,
			RegionID:           flyerRegion.RegionID,
			Polygon:            string(polygonJSON),
			RotationDeg:        flyerRegion.Rotation,
			DetectionConfidence: flyerRegion.Confidence,
			Notes:              &flyerRegion.Notes,
		}

		if err := db.Create(&flyer).Error; err != nil {
			return fmt.Errorf("failed to create flyer: %w", err)
		}

		// Create event candidate records for each extracted event
		for _, event := range flyerRegion.Events {
			// Convert fields and confidences to JSON
			fieldsJSON, err := json.Marshal(event.Fields)
			if err != nil {
				return fmt.Errorf("failed to marshal event fields: %w", err)
			}

			confidencesJSON, err := json.Marshal(event.Confidences)
			if err != nil {
				return fmt.Errorf("failed to marshal confidences: %w", err)
			}

			eventCandidate := models.EventCandidate{
				FlyerID:      flyer.ID,
				EventID:      event.EventID,
				Fields:       string(fieldsJSON),
				Confidences:  string(confidencesJSON),
				SourceExcerpt: &event.Excerpt,
				CompositeScore: &event.Confidences.Overall,
			}

			if err := db.Create(&eventCandidate).Error; err != nil {
				return fmt.Errorf("failed to create event candidate: %w", err)
			}
		}
	}

	return nil
}