package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	// Application
	AppName       string
	PublicBaseURL string
	Port          string
	Environment   string

	// Database
	DatabaseURL string

	// OpenAI
	OpenAIAPIKey      string
	OpenAIModel       string
	OpenAITimeoutMS   int
	StructuredOutput  bool
	ImageMaxLongSide  int
	ImageJPEGQuality  int

	// Storage
	UploadDir string

	// Queue (in-memory for simplicity)
	RegionTZ string

	// Geocoding
	Geocoder      string
	GeocoderAPIKey string

	// Auto-publish settings
	AutoPublishEnabled           bool
	AutoPublishThreshold         float64
	GeoConfThreshold            float64
	AutoPublishMinStartOffsetMin int
	AutoPublishMaxStartOffsetDays int
	TrustAdjust                 float64

	// ICS
	ICSUIDDomain string
	ICSProdID    string

	// Optional features
	PGVectorEnabled bool

	// Observability
	OTELEndpoint string
}

func Load() (*Config, error) {
	cfg := &Config{
		AppName:       getEnv("APP_NAME", "WilliamBoard"),
		PublicBaseURL: getEnv("PUBLIC_BASE_URL", "http://localhost:8080"),
		Port:          getEnv("PORT", "8080"),
		Environment:   getEnv("ENVIRONMENT", "development"),

		DatabaseURL: getEnv("DATABASE_URL", ""),

		OpenAIAPIKey:      getEnv("OPENAI_API_KEY", ""),
		OpenAIModel:       getEnv("OPENAI_MODEL", "gpt-4o"),
		OpenAITimeoutMS:   getEnvInt("OPENAI_TIMEOUT_MS", 15000),
		StructuredOutput:  getEnvBool("STRUCTURED_OUTPUT", true),
		ImageMaxLongSide:  getEnvInt("IMAGE_MAX_LONG_SIDE", 2048),
		ImageJPEGQuality:  getEnvInt("IMAGE_JPEG_QUALITY", 85),

		UploadDir: getEnv("UPLOAD_DIR", "/data/uploads"),

		RegionTZ: getEnv("REGION_TZ", "America/Los_Angeles"),

		Geocoder:       getEnv("GEOCODER", "mapbox"),
		GeocoderAPIKey: getEnv("GEOCODER_API_KEY", ""),

		AutoPublishEnabled:            getEnvBool("AUTO_PUBLISH_ENABLED", true),
		AutoPublishThreshold:          getEnvFloat("AUTO_PUBLISH_THRESHOLD", 0.80),
		GeoConfThreshold:             getEnvFloat("GEO_CONF_THRESHOLD", 0.75),
		AutoPublishMinStartOffsetMin: getEnvInt("AUTO_PUBLISH_MIN_START_OFFSET_MIN", 30),
		AutoPublishMaxStartOffsetDays: getEnvInt("AUTO_PUBLISH_MAX_START_OFFSET_DAYS", 180),
		TrustAdjust:                   getEnvFloat("TRUST_ADJUST", 0.05),

		ICSUIDDomain: getEnv("ICS_UID_DOMAIN", "williamboard.app"),
		ICSProdID:    getEnv("ICS_PRODID", "-//WilliamBoard//EN"),

		PGVectorEnabled: getEnvBool("PGVECTOR_ENABLED", false),
		OTELEndpoint:    getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	required := map[string]string{
		"DATABASE_URL":   c.DatabaseURL,
		"OPENAI_API_KEY": c.OpenAIAPIKey,
	}

	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("required environment variable %s is not set", name)
		}
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func (c *Config) GetLocation() (*time.Location, error) {
	return time.LoadLocation(c.RegionTZ)
}