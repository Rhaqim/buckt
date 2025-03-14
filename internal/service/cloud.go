package service

import (
	"github.com/Rhaqim/buckt/internal/domain"
)

type CloudStorage interface {
	TransferFile(file_id string) error
}

type CloudStorageService struct {
	domain.CloudService
}

func NewCloudStorage(client domain.CloudService) CloudStorage {

	return &CloudStorageService{
		CloudService: client,
	}
}

// TransferFile implements CloudStorage.
func (c *CloudStorageService) TransferFile(file_id string) error {
	return c.CloudService.UploadFile(file_id)
}
