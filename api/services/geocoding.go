package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/lincolngreen/williamboard/api/config"
)

type GeocodingService struct {
	config     *config.Config
	httpClient *http.Client
}

type GeocodeResult struct {
	Latitude         float64            `json:"latitude"`
	Longitude        float64            `json:"longitude"`
	FormattedAddress string             `json:"formatted_address"`
	Confidence       float64            `json:"confidence"`
	Components       map[string]string  `json:"components"`
	RawResponse      map[string]interface{} `json:"raw_response"`
}

type MapboxFeature struct {
	Geometry struct {
		Coordinates []float64 `json:"coordinates"`
	} `json:"geometry"`
	Properties struct {
		FullAddress string  `json:"full_address"`
		Context     []struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		} `json:"context"`
	} `json:"properties"`
	Relevance float64 `json:"relevance"`
}

type MapboxResponse struct {
	Features []MapboxFeature `json:"features"`
	Query    []string        `json:"query"`
}

func NewGeocodingService(cfg *config.Config) *GeocodingService {
	return &GeocodingService{
		config:     cfg,
		httpClient: &http.Client{},
	}
}

// GeocodeAddress converts a venue address to lat/lng coordinates
func (g *GeocodingService) GeocodeAddress(ctx context.Context, address string) (*GeocodeResult, error) {
	if g.config.GeocoderAPIKey == "" || g.config.GeocoderAPIKey == "your-mapbox-api-key" {
		return g.mockGeocodeResult(address), nil
	}

	switch g.config.Geocoder {
	case "mapbox":
		return g.geocodeWithMapbox(ctx, address)
	default:
		return nil, fmt.Errorf("unsupported geocoder: %s", g.config.Geocoder)
	}
}

// geocodeWithMapbox uses Mapbox Geocoding API
func (g *GeocodingService) geocodeWithMapbox(ctx context.Context, address string) (*GeocodeResult, error) {
	// Clean and format address
	query := strings.TrimSpace(address)
	if query == "" {
		return nil, fmt.Errorf("empty address")
	}

	// Build Mapbox API URL
	baseURL := "https://api.mapbox.com/geocoding/v5/mapbox.places/"
	encodedQuery := url.QueryEscape(query)
	requestURL := fmt.Sprintf("%s%s.json?access_token=%s&limit=1&types=address,poi",
		baseURL, encodedQuery, g.config.GeocoderAPIKey)

	// Make request
	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("geocoding request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geocoding API returned status %d", resp.StatusCode)
	}

	// Parse response
	var mapboxResp MapboxResponse
	if err := json.NewDecoder(resp.Body).Decode(&mapboxResp); err != nil {
		return nil, fmt.Errorf("failed to parse geocoding response: %w", err)
	}

	if len(mapboxResp.Features) == 0 {
		return nil, fmt.Errorf("no geocoding results found for address: %s", address)
	}

	feature := mapboxResp.Features[0]
	
	// Extract coordinates (Mapbox returns [lng, lat])
	if len(feature.Geometry.Coordinates) < 2 {
		return nil, fmt.Errorf("invalid coordinates in geocoding response")
	}

	longitude := feature.Geometry.Coordinates[0]
	latitude := feature.Geometry.Coordinates[1]

	// Extract address components
	components := make(map[string]string)
	for _, context := range feature.Properties.Context {
		if strings.HasPrefix(context.ID, "place") {
			components["city"] = context.Text
		} else if strings.HasPrefix(context.ID, "region") {
			components["state"] = context.Text
		} else if strings.HasPrefix(context.ID, "country") {
			components["country"] = context.Text
		} else if strings.HasPrefix(context.ID, "postcode") {
			components["postal_code"] = context.Text
		}
	}

	// Use relevance as confidence score
	confidence := feature.Relevance
	if confidence == 0 {
		confidence = 0.5 // Default confidence if not provided
	}

	// Get formatted address
	formattedAddress := feature.Properties.FullAddress
	if formattedAddress == "" {
		formattedAddress = address // Fall back to original
	}

	// Save raw response for debugging
	rawResponse := make(map[string]interface{})
	rawData, _ := json.Marshal(feature)
	json.Unmarshal(rawData, &rawResponse)

	return &GeocodeResult{
		Latitude:         latitude,
		Longitude:        longitude,
		FormattedAddress: formattedAddress,
		Confidence:       confidence,
		Components:       components,
		RawResponse:      rawResponse,
	}, nil
}

// mockGeocodeResult returns mock coordinates for testing
func (g *GeocodingService) mockGeocodeResult(address string) *GeocodeResult {
	// Mock coordinates for common test addresses, default to SF
	lat, lng := 37.7749, -122.4194 // San Francisco default
	confidence := 0.7

	// Simple heuristics for mock data
	addressLower := strings.ToLower(address)
	if strings.Contains(addressLower, "new york") || strings.Contains(addressLower, "ny") {
		lat, lng = 40.7128, -74.0060
	} else if strings.Contains(addressLower, "los angeles") || strings.Contains(addressLower, "la") {
		lat, lng = 34.0522, -118.2437
	} else if strings.Contains(addressLower, "chicago") {
		lat, lng = 41.8781, -87.6298
	} else if strings.Contains(addressLower, "seattle") {
		lat, lng = 47.6062, -122.3321
	}

	// Higher confidence if address looks complete
	if strings.Contains(address, ",") && len(strings.Fields(address)) > 3 {
		confidence = 0.8
	}

	return &GeocodeResult{
		Latitude:         lat,
		Longitude:        lng,
		FormattedAddress: address,
		Confidence:       confidence,
		Components: map[string]string{
			"city":    "Mock City",
			"state":   "CA",
			"country": "US",
		},
		RawResponse: map[string]interface{}{
			"mock": true,
			"original_address": address,
		},
	}
}

// BuildVenueAddress constructs a geocodable address from venue fields
func (g *GeocodingService) BuildVenueAddress(name, addressLine, city, state, postalCode, country string) string {
	var parts []string
	
	// Start with venue name if it looks like it includes address info
	if name != "" && (strings.Contains(name, "St") || strings.Contains(name, "Ave") || strings.Contains(name, "Rd")) {
		parts = append(parts, name)
	}
	
	// Add address line
	if addressLine != "" {
		parts = append(parts, addressLine)
	}
	
	// Add city, state
	if city != "" {
		if state != "" {
			parts = append(parts, fmt.Sprintf("%s, %s", city, state))
		} else {
			parts = append(parts, city)
		}
	}
	
	// Add postal code
	if postalCode != "" {
		parts = append(parts, postalCode)
	}
	
	// Add country if not US
	if country != "" && country != "US" {
		parts = append(parts, country)
	}
	
	return strings.Join(parts, ", ")
}

// ValidateCoordinates checks if lat/lng are valid
func ValidateCoordinates(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}