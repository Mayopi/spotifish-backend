package service

import (
	"fmt"
	"os"
	"path/filepath"
)

// ArtStorageService manages local album art file storage.
type ArtStorageService struct {
	basePath string
}

// NewArtStorageService creates a new ArtStorageService.
func NewArtStorageService(basePath string) *ArtStorageService {
	return &ArtStorageService{basePath: basePath}
}

// EnsureDir creates the art storage directory if it doesn't exist.
func (s *ArtStorageService) EnsureDir() error {
	return os.MkdirAll(s.basePath, 0755)
}

// SaveArt saves album art data to disk.
func (s *ArtStorageService) SaveArt(key string, data []byte) error {
	path := s.GetPath(key)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create art directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write art file: %w", err)
	}
	return nil
}

// GetPath returns the filesystem path for an art key.
func (s *ArtStorageService) GetPath(key string) string {
	return filepath.Join(s.basePath, key+".img")
}

// Exists checks if art exists for a given key.
func (s *ArtStorageService) Exists(key string) bool {
	_, err := os.Stat(s.GetPath(key))
	return err == nil
}

// DeleteArt removes an art file.
func (s *ArtStorageService) DeleteArt(key string) error {
	return os.Remove(s.GetPath(key))
}
