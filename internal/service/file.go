package service

import (
	"github.com/Rhaqim/buckt/config"
	"github.com/Rhaqim/buckt/internal/domain"
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

func (s *StorageService) UploadFile(file []byte, filename string) error {
	return nil
}

func (s *StorageService) DownloadFile(filename string) ([]byte, error) {
	return nil, nil
}

func (s *StorageService) DeleteFile(filename string) error {
	return nil
}
