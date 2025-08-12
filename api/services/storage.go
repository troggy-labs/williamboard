package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	config_pkg "github.com/lincolngreen/williamboard/api/config"
)

type StorageService struct {
	uploadDir string
	baseURL   string
}

type UploadURLResult struct {
	SubmissionID string `json:"submissionId"`
	URL          string `json:"url"`
	MaxSizeMB    int    `json:"maxSizeMB"`
}

func NewStorageService(cfg *config_pkg.Config) *StorageService {
	uploadDir := cfg.UploadDir
	if uploadDir == "" {
		uploadDir = "/data/uploads" // Render persistent disk mount point
	}

	// Create upload directory if it doesn't exist
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		panic(fmt.Sprintf("unable to create upload directory: %v", err))
	}

	return &StorageService{
		uploadDir: uploadDir,
		baseURL:   cfg.PublicBaseURL,
	}
}

// GenerateUploadURL creates an upload endpoint URL for direct file uploads
func (s *StorageService) GenerateUploadURL(submissionID uuid.UUID) *UploadURLResult {
	return &UploadURLResult{
		SubmissionID: submissionID.String(),
		URL:          fmt.Sprintf("%s/v1/uploads/%s", s.baseURL, submissionID.String()),
		MaxSizeMB:    12,
	}
}

// SaveFile saves uploaded file data to disk
func (s *StorageService) SaveFile(submissionID uuid.UUID, filename string, data io.Reader) error {
	submissionDir := filepath.Join(s.uploadDir, submissionID.String())
	if err := os.MkdirAll(submissionDir, 0755); err != nil {
		return fmt.Errorf("failed to create submission directory: %w", err)
	}

	filePath := filepath.Join(submissionDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, data)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

// GetPublicURL returns the public URL for a file
func (s *StorageService) GetPublicURL(submissionID uuid.UUID, filename string) string {
	return fmt.Sprintf("%s/files/%s/%s", s.baseURL, submissionID.String(), filename)
}

// GetOriginalImageURL returns the public URL for an original image
func (s *StorageService) GetOriginalImageURL(submissionID uuid.UUID) string {
	return s.GetPublicURL(submissionID, "original.jpg")
}

// GetDerivativeImageURL returns the public URL for a derivative image
func (s *StorageService) GetDerivativeImageURL(submissionID uuid.UUID) string {
	return s.GetPublicURL(submissionID, "derivative.jpg")
}

// GetCropImageURL returns the public URL for a flyer crop
func (s *StorageService) GetCropImageURL(submissionID uuid.UUID, regionID string) string {
	filename := fmt.Sprintf("crop_%s.jpg", regionID)
	return s.GetPublicURL(submissionID, filename)
}

// GetFilePath returns the local file system path for a file
func (s *StorageService) GetFilePath(submissionID uuid.UUID, filename string) string {
	return filepath.Join(s.uploadDir, submissionID.String(), filename)
}

// GetUploadDir returns the upload directory path
func (s *StorageService) GetUploadDir() string {
	return s.uploadDir
}