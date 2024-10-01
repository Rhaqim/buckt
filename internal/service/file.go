package service

import (
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

func (s *StorageService) UploadFile(file []byte, bucketname, filename string) error {
	// Check if file already exists
	if _, err := s.fileStore.GetBy("filename", filename); err == nil {
		return errs.ErrFileAlreadyExists
	}

	// Check if bucket exists
	_, err := s.bucketStore.GetBy("name", bucketname)
	if err != nil {
		return errs.ErrBucketNotFound
	}

	// Save file to storage
	filePath := filepath.Join(bucketname, filename)
	if err := os.WriteFile(filePath, file, 0644); err != nil {
		return err
	}

	// Save file to database
	fileModel := model.FileModel{Name: filename, Path: filePath}
	if err := s.fileStore.Create(&fileModel); err != nil {
		return err
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
