package mocks

import (
	"net/http"

	"github.com/Rhaqim/buckt/internal/model"
)

type MockBuckt struct {
	fileService   *FileService
	folderService *FolderService
}

func NewMockBuckt() *MockBuckt {
	return &MockBuckt{
		fileService:   new(FileService),
		folderService: new(FolderService),
	}
}

func (m *MockBuckt) UploadFile(user_id, parent_id, file_name, content_type string, file_data []byte) (string, error) {
	args := m.fileService.Called(user_id, parent_id, file_name, content_type, file_data)

	return args.Get(0).(string), args.Error(1)
}

func (m *MockBuckt) GetFile(fileID string) (*model.FileModel, error) {
	args := m.fileService.Called(fileID)
	return args.Get(0).(*model.FileModel), args.Error(1)
}

func (m *MockBuckt) ListFiles(parentID string) ([]model.FileModel, error) {
	args := m.fileService.Called(parentID)
	return args.Get(0).([]model.FileModel), args.Error(1)
}

func (m *MockBuckt) MoveFile(fileID, newParentID string) error {
	args := m.fileService.Called(fileID, newParentID)
	return args.Error(0)
}

func (m *MockBuckt) RenameFile(fileID, newName string) error {
	args := m.fileService.Called(fileID, newName)
	return args.Error(0)
}

func (m *MockBuckt) DeleteFile(fileID string) (string, error) {
	args := m.fileService.Called(fileID)
	return args.String(0), args.Error(1)
}

func (m *MockBuckt) ScrubFile(fileID string) (string, error) {
	args := m.fileService.Called(fileID)
	return args.String(0), args.Error(1)
}

func (m *MockBuckt) NewFolder(userID, parentID, folderName, description string) (string, error) {
	args := m.folderService.Called(userID, parentID, folderName, description)
	return args.String(0), args.Error(1)
}

func (m *MockBuckt) ListFolders(parentID string) ([]model.FolderModel, error) {
	args := m.folderService.Called(parentID)
	return args.Get(0).([]model.FolderModel), args.Error(1)
}

func (m *MockBuckt) GetFolderWithContent(userID, folderID string) (*model.FolderModel, error) {
	args := m.folderService.Called(userID, folderID)
	return args.Get(0).(*model.FolderModel), args.Error(1)
}

func (m *MockBuckt) MoveFolder(folderID, newParentID string) error {
	args := m.folderService.Called(folderID, newParentID)
	return args.Error(0)
}

func (m *MockBuckt) RenameFolder(userID, folderID, newName string) error {
	args := m.folderService.Called(userID, folderID, newName)
	return args.Error(0)
}

func (m *MockBuckt) DeleteFolder(folderID string) (string, error) {
	args := m.folderService.Called(folderID)
	return args.String(0), args.Error(1)
}

func (m *MockBuckt) DeleteFolderPermanently(userID, folderID string) (string, error) {
	args := m.folderService.Called(userID, folderID)
	return args.String(0), args.Error(1)
}

func (m *MockBuckt) GetHandler() http.Handler {
	args := m.folderService.Called()
	return args.Get(0).(http.Handler)
}

func (m *MockBuckt) StartServer(port string) error {
	args := m.folderService.Called(port)
	return args.Error(0)
}

func (m *MockBuckt) Close() {
	m.folderService.Called()
}
