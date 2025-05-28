package filestorage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var ErrFileNotFound = fmt.Errorf("file not found")

// FileStorage implements file-based storage
type FileStorage struct {
	baseDir string
	mu      sync.Mutex
}

// NewFileStorage creates a new file storage
func NewFileStorage(baseDir string) (*FileStorage, error) {
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	return &FileStorage{
		baseDir: baseDir,
	}, nil
}

// Save saves the data to a file, creating the directory if it doesn't exist
func (s *FileStorage) Save(id string, data any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Marshal the data
	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create the file
	filePath := filepath.Join(s.baseDir, fmt.Sprintf("%s.json", id))
	if err := os.WriteFile(filePath, bytes, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Load loads data from a file
func (s *FileStorage) Load(id string, data interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read the file
	filePath := filepath.Join(s.baseDir, fmt.Sprintf("%s.json", id))
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrFileNotFound
		}

		return fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal the data
	if err := json.Unmarshal(bytes, data); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return nil
}

// Delete deletes a file
func (s *FileStorage) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Delete the file
	filePath := filepath.Join(s.baseDir, fmt.Sprintf("%s.json", id))
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// List lists all files in the directory. Returns an empty list if the directory doesn't exist.
func (s *FileStorage) List() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read the directory
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist, return empty list
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Extract the file IDs
	var ids []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			id := entry.Name()[:len(entry.Name())-5] // Remove the .json extension
			ids = append(ids, id)
		}
	}

	return ids, nil
}
