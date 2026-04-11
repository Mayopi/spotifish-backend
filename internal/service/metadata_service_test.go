package service

import (
	"strings"
	"testing"
)

func TestMetadataService_Extract_NoTags(t *testing.T) {
	svc := NewMetadataService()

	// Create a reader with non-audio data (will fail to parse tags)
	r := strings.NewReader("this is not audio data")
	meta, pic, err := svc.Extract(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return defaults
	if meta == nil {
		t.Fatal("meta should not be nil")
	}
	if meta.Title != "Unknown" {
		t.Errorf("Title = %q, want %q", meta.Title, "Unknown")
	}
	if meta.Artist != "Unknown" {
		t.Errorf("Artist = %q, want %q", meta.Artist, "Unknown")
	}
	if meta.Album != "Unknown" {
		t.Errorf("Album = %q, want %q", meta.Album, "Unknown")
	}
	if pic != nil {
		t.Error("picture should be nil for non-audio data")
	}
}
