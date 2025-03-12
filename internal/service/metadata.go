package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"maps"

	"github.com/Rhaqim/buckt/internal/model"
)

type MetadataService interface {
	GetMetadata(filePath string) (map[string]interface{}, error)
	InsertMetadata(filePath string, data map[string]interface{}) error
	SyncMetadata(filePath string, data map[string]interface{}) error
	RestoreFromMetadata() error
}

type metadataService struct {
	mu       sync.Mutex
	metaPath string
}

func NewMetadataService(metaPath string) MetadataService {
	return &metadataService{
		metaPath: metaPath,
	}
}

func (s *metadataService) GetMetadata(filePath string) (map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	metaFile := s.getMetadataFilePath(filePath)
	data, err := os.ReadFile(metaFile)
	if err != nil {
		return nil, err
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	return metadata, nil
}

func (s *metadataService) InsertMetadata(filePath string, data map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metaFile := s.getMetadataFilePath(filePath)
	fileData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metaFile, fileData, 0644)
}

func (s *metadataService) SyncMetadata(filePath string, data map[string]interface{}) error {
	existingData, err := s.GetMetadata(filePath)
	if err != nil {
		existingData = make(map[string]interface{}) // Create new if not found
	}

	maps.Copy(existingData, data)

	return s.InsertMetadata(filePath, existingData)
}

func (s *metadataService) RestoreFromMetadata() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	files, err := filepath.Glob(filepath.Join(s.metaPath, "*.json"))
	if err != nil {
		return err
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		var metadata map[string]interface{}
		if err := json.Unmarshal(data, &metadata); err != nil {
			return err
		}

		// Implement database restoration logic here
		// Example: db.InsertMetadata(metadata)
	}

	return nil
}

func (s *metadataService) GenerateMetadata(file *model.FileModel) ([]byte, error) {
	metadata := map[string]interface{}{
		"filename":    file.Name,
		"size":        file.Size,
		"uploaded_at": file.CreatedAt.Format(time.RFC3339),
		"checksum":    file.Hash,
		"owner":       "file.UserID",
	}

	return json.MarshalIndent(metadata, "", "  ")
}

func (s *metadataService) getMetadataFilePath(filePath string) string {
	return filepath.Join(s.metaPath, filepath.Base(filePath)+".json")
}
