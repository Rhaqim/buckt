package service

import (
	"github.com/Rhaqim/buckt/config"
	"github.com/Rhaqim/buckt/internal/database"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/pkg/logger"
)

type StorageService struct {
	*logger.Logger
	*database.DB
	*config.Config
}

func NewStorageService(log *logger.Logger, db *database.DB, cfg *config.Config) domain.StorageFileService {
	return &StorageService{log, db, cfg}
}

func (s *StorageService) UploadFile(file []byte, filename string) error {
	return nil
}

func (s *StorageService) DownloadFile(filename string) ([]byte, error) {
	return nil, nil
}

func (s *StorageService) DeleteFile(filename string) error {
	return nil
}
