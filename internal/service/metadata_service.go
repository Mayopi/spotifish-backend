package service

import (
	"io"

	"github.com/dhowden/tag"
	"github.com/rs/zerolog/log"
)

// SongMetadata holds extracted metadata from an audio file.
type SongMetadata struct {
	Title    string
	Artist   string
	Album    string
	Duration int64 // milliseconds, 0 if unknown
}

// MetadataService extracts metadata from audio files.
type MetadataService struct{}

// NewMetadataService creates a new MetadataService.
func NewMetadataService() *MetadataService {
	return &MetadataService{}
}

// Extract reads metadata and embedded picture from an audio file reader.
// Returns metadata, picture bytes (may be nil), and any error.
func (s *MetadataService) Extract(r io.ReadSeeker) (*SongMetadata, []byte, error) {
	m, err := tag.ReadFrom(r)
	if err != nil {
		log.Debug().Err(err).Msg("failed to read tags, using defaults")
		return &SongMetadata{
			Title:  "Unknown",
			Artist: "Unknown",
			Album:  "Unknown",
		}, nil, nil
	}

	meta := &SongMetadata{
		Title:  m.Title(),
		Artist: m.Artist(),
		Album:  m.Album(),
	}

	// Fallback for empty values
	if meta.Title == "" {
		meta.Title = "Unknown"
	}
	if meta.Artist == "" {
		meta.Artist = "Unknown"
	}
	if meta.Album == "" {
		meta.Album = "Unknown"
	}

	// Extract embedded picture
	var pictureData []byte
	if pic := m.Picture(); pic != nil {
		pictureData = pic.Data
	}

	return meta, pictureData, nil
}
