package mocks

import (
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/stretchr/testify/mock"
)

type CloudService struct {
	mock.Mock
}

func NewCloudService() domain.CloudService {
	return &CloudService{}
}

// UploadFile implements domain.CloudService.
func (m *CloudService) UploadFileToCloud(file_id string) error {
	args := m.Called(file_id)
	return args.Error(0)
}

// UploadFolder implements domain.CloudService.
func (m *CloudService) UploadFolderToCloud(user_id string, folder_id string) error {
	args := m.Called(user_id, folder_id)
	return args.Error(0)
}
