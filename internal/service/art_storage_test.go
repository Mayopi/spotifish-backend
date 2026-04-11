package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArtStorageService_SaveAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewArtStorageService(tmpDir)

	key := "test-uuid-123"
	data := []byte("fake image data")

	if err := svc.SaveArt(key, data); err != nil {
		t.Fatalf("SaveArt failed: %v", err)
	}

	// Verify file exists
	expectedPath := filepath.Join(tmpDir, key+".img")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatal("art file should exist")
	}

	// Read back
	readData, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(readData) != string(data) {
		t.Fatalf("data mismatch: got %q, want %q", readData, data)
	}

	// Test Exists
	if !svc.Exists(key) {
		t.Fatal("Exists should return true")
	}

	// Test GetPath
	if svc.GetPath(key) != expectedPath {
		t.Fatalf("GetPath mismatch: got %q, want %q", svc.GetPath(key), expectedPath)
	}

	// Test Delete
	if err := svc.DeleteArt(key); err != nil {
		t.Fatalf("DeleteArt failed: %v", err)
	}
	if svc.Exists(key) {
		t.Fatal("art should not exist after delete")
	}
}

func TestArtStorageService_EnsureDir(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "nested", "art")
	svc := NewArtStorageService(tmpDir)

	if err := svc.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	info, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("should be a directory")
	}
}
