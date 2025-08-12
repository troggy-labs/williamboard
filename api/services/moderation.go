package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/lincolngreen/williamboard/api/config"
	"github.com/sashabaranov/go-openai"
)

type ModerationService struct {
	client *openai.Client
	config *config.Config
}

type ModerationResult struct {
	QualityScore      float64 `json:"quality_score"`
	IsAppropriate     bool    `json:"is_appropriate"`
	ModerationReason  *string `json:"moderation_reason,omitempty"`
	ConfidenceFactors map[string]float64 `json:"confidence_factors"`
}

type QualityFactors struct {
	EventDetailsComplete float64 `json:"event_details_complete"`
	DateTimeConfidence   float64 `json:"datetime_confidence"`
	VenueConfidence      float64 `json:"venue_confidence"`
	ContactInfoPresent   float64 `json:"contact_info_present"`
	ProfessionalLookng   float64 `json:"professional_looking"`
	TextReadability      float64 `json:"text_readability"`
}

func NewModerationService(cfg *config.Config) *ModerationService {
	var client *openai.Client
	if cfg.OpenAIAPIKey != "" {
		client = openai.NewClient(cfg.OpenAIAPIKey)
	}
	
	return &ModerationService{
		client: client,
		config: cfg,
	}
}

// ModerateEventCandidate evaluates event quality and appropriateness
func (m *ModerationService) ModerateEventCandidate(ctx context.Context, eventData map[string]interface{}) (*ModerationResult, error) {
	if m.client == nil {
		return m.mockModerationResult(eventData), nil
	}

	// Extract event details for moderation
	eventJSON, _ := json.Marshal(eventData)
	
	prompt := fmt.Sprintf(`
Analyze this extracted event data for quality and appropriateness.

Event Data:
%s

Evaluate the following factors and provide scores 0.0-1.0:

1. Event Details Completeness (0.0 = missing key info, 1.0 = all details present)
2. Date/Time Confidence (0.0 = unclear/missing, 1.0 = clear specific datetime)  
3. Venue Confidence (0.0 = vague location, 1.0 = specific address/venue)
4. Contact Info Present (0.0 = no contact info, 1.0 = clear contact details)
5. Professional Looking (0.0 = low quality/spam-like, 1.0 = professional/legitimate)
6. Text Readability (0.0 = hard to read/messy, 1.0 = clear and well-formatted)

Also determine:
- Is this appropriate for a public event calendar? (true/false)
- If inappropriate, what's the reason?

Respond in this exact JSON format:
{
  "quality_factors": {
    "event_details_complete": 0.0-1.0,
    "datetime_confidence": 0.0-1.0, 
    "venue_confidence": 0.0-1.0,
    "contact_info_present": 0.0-1.0,
    "professional_looking": 0.0-1.0,
    "text_readability": 0.0-1.0
  },
  "is_appropriate": true/false,
  "moderation_reason": "reason if inappropriate, null otherwise"
}`, string(eventJSON))

	req := openai.ChatCompletionRequest{
		Model: m.config.OpenAIModel,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
		MaxTokens: 500,
	}

	resp, err := m.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("moderation API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no moderation response received")
	}

	content := resp.Choices[0].Message.Content

	// Parse moderation response
	var moderationData struct {
		QualityFactors   QualityFactors `json:"quality_factors"`
		IsAppropriate    bool           `json:"is_appropriate"`
		ModerationReason *string        `json:"moderation_reason"`
	}

	if err := json.Unmarshal([]byte(content), &moderationData); err != nil {
		log.Printf("Failed to parse moderation response: %v", err)
		log.Printf("Raw response: %s", content)
		return m.mockModerationResult(eventData), nil
	}

	// Calculate composite quality score (weighted average)
	qualityScore := calculateQualityScore(moderationData.QualityFactors)

	// Build confidence factors map
	confidenceFactors := map[string]float64{
		"event_details_complete": moderationData.QualityFactors.EventDetailsComplete,
		"datetime_confidence":    moderationData.QualityFactors.DateTimeConfidence,
		"venue_confidence":       moderationData.QualityFactors.VenueConfidence,
		"contact_info_present":   moderationData.QualityFactors.ContactInfoPresent,
		"professional_looking":   moderationData.QualityFactors.ProfessionalLookng,
		"text_readability":       moderationData.QualityFactors.TextReadability,
	}

	return &ModerationResult{
		QualityScore:      qualityScore,
		IsAppropriate:     moderationData.IsAppropriate,
		ModerationReason:  moderationData.ModerationReason,
		ConfidenceFactors: confidenceFactors,
	}, nil
}

// calculateQualityScore computes weighted composite score
func calculateQualityScore(factors QualityFactors) float64 {
	// Weighted scoring - some factors more important than others
	weights := map[string]float64{
		"event_details": 0.25,  // Essential event info
		"datetime":      0.20,  // Clear timing
		"venue":         0.20,  // Clear location
		"contact":       0.15,  // Contact info
		"professional":  0.15,  // Quality/legitimacy
		"readability":   0.05,  // Text quality
	}
	
	score := factors.EventDetailsComplete*weights["event_details"] +
		factors.DateTimeConfidence*weights["datetime"] +
		factors.VenueConfidence*weights["venue"] +
		factors.ContactInfoPresent*weights["contact"] +
		factors.ProfessionalLookng*weights["professional"] +
		factors.TextReadability*weights["readability"]
	
	return score
}

// mockModerationResult returns reasonable defaults when API unavailable
func (m *ModerationService) mockModerationResult(eventData map[string]interface{}) *ModerationResult {
	// Basic heuristics for mock scoring
	qualityScore := 0.75 // Default reasonable score
	
	// Check for key fields to adjust score
	if title, ok := eventData["title"].(string); ok && strings.TrimSpace(title) != "" {
		qualityScore += 0.1
	}
	if venue, ok := eventData["venue"].(string); ok && strings.TrimSpace(venue) != "" {
		qualityScore += 0.05
	}
	if date, ok := eventData["date"].(string); ok && strings.TrimSpace(date) != "" {
		qualityScore += 0.05
	}
	
	// Cap at 1.0
	if qualityScore > 1.0 {
		qualityScore = 1.0
	}
	
	return &ModerationResult{
		QualityScore:  qualityScore,
		IsAppropriate: true, // Default to appropriate in mock mode
		ConfidenceFactors: map[string]float64{
			"event_details_complete": 0.8,
			"datetime_confidence":    0.7,
			"venue_confidence":       0.7,
			"contact_info_present":   0.5,
			"professional_looking":   0.8,
			"text_readability":       0.8,
		},
	}
}