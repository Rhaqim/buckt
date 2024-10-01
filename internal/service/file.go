package service

import (
	"crypto/sha256"
	"fmt"
	"mime"
	"os"
	"path/filepath"

	"github.com/Rhaqim/buckt/config"
	"github.com/Rhaqim/buckt/internal/domain"
	errs "github.com/Rhaqim/buckt/internal/error"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/logger"
)

type StorageService struct {
	*logger.Logger
	*config.Config
	Store
}

type Store struct {
	fileStore   domain.Repository[model.FileModel]
	bucketStore domain.Repository[model.BucketModel]
}

func NewStorageService(log *logger.Logger, cfg *config.Config, fileStore domain.Repository[model.FileModel], bucketStore domain.Repository[model.BucketModel]) domain.StorageFileService {
	store := Store{fileStore: fileStore, bucketStore: bucketStore}

	return &StorageService{log, cfg, store}
}

func (s *StorageService) UploadFile(file []byte, bucketName, fileName string) error {
	// Check if bucket exists
	bucket, err := s.bucketStore.GetBy("name", bucketName)
	if err != nil {
		return errs.ErrBucketNotFound
	}

	// Check if file already exists in the bucket
	if _, err := s.fileStore.GetBy("name", fileName); err == nil {
		return errs.ErrFileAlreadyExists
	}

	// Calculate file hash (SHA-256) for uniqueness check and future retrieval
	hash := fmt.Sprintf("%x", sha256.Sum256(file))

	// File size in bytes
	fileSize := int64(len(file))

	// Determine file content type (MIME type)
	contentType := mime.TypeByExtension(filepath.Ext(fileName))
	if contentType == "" {
		contentType = "application/octet-stream" // Default content type
	}

	// Create the full file path using bucket name and file name
	// filePath := filepath.Join("/www/media", bucketName, fileName)
	filePath := filepath.Join(s.Media.Dir, bucketName, fileName)

	// Save the file to the file system
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	if err := os.WriteFile(filePath, file, 0644); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	// Save file record to the database
	fileModel := model.FileModel{
		Name:        fileName,
		Path:        filePath,
		ContentType: contentType,
		Size:        fileSize,
		Hash:        hash,
		BucketID:    bucket.ID, // Associate the file with the existing bucket
	}

	// Insert the file entry into the database
	if err := s.fileStore.Create(&fileModel); err != nil {
		return fmt.Errorf("failed to save file metadata: %w", err)
	}

	return nil
}

func (s *StorageService) DownloadFile(filename string) ([]byte, error) {
	// Get file from database
	file, err := s.fileStore.GetBy("filename", filename)
	if err != nil {
		return nil, err
	}

	// Read file from storage
	filePath := file.Path
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *StorageService) DeleteFile(filename string) error {
	// Get file from database
	file, err := s.fileStore.GetBy("filename", filename)
	if err != nil {
		return err
	}

	// Delete file from storage
	filePath := file.Path
	if err := os.Remove(filePath); err != nil {
		return err
	}

	// Delete file from database
	if err := s.fileStore.Delete(file.ID); err != nil {
		return err
	}

	return nil
}
