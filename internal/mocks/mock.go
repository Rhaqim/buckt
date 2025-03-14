package mocks

import (
	"net/http"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockDB struct {
	mock.Mock
	*gorm.DB
	*logger.BucktLogger
}

func (m *MockDB) Close() {
	m.Called()
}

func (m *MockDB) Migrate() error {
	args := m.Called()
	return args.Error(0)
}

type MockCacheManager struct {
	mock.Mock
}

func (m *MockCacheManager) SetBucktValue(key string, value any) error {
	args := m.Called(key, value)
	return args.Error(0)
}

func (m *MockCacheManager) GetBucktValue(key string) (any, error) {
	args := m.Called(key)
	return args.Get(0), args.Error(1)
}

func (m *MockCacheManager) DeleteBucktValue(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

type MockFileRepository struct {
	mock.Mock
}

// MoveFile implements domain.FileRepository.
func (m *MockFileRepository) MoveFile(file_id uuid.UUID, new_parent_id uuid.UUID) (string, string, error) {
	args := m.Called(file_id, new_parent_id)
	return args.Get(0).(string), args.Get(1).(string), args.Error(2)
}

// RenameFile implements domain.FileRepository.
func (m *MockFileRepository) RenameFile(file_id uuid.UUID, new_name string) error {
	args := m.Called(file_id, new_name)
	return args.Error(0)
}

func (m *MockFileRepository) Create(file *model.FileModel) error {
	args := m.Called(file)
	return args.Error(0)
}

func (m *MockFileRepository) GetFile(fileID uuid.UUID) (*model.FileModel, error) {
	args := m.Called(fileID)
	return args.Get(0).(*model.FileModel), args.Error(1)
}

// RestoreFileByPath implements domain.FileRepository.
func (m *MockFileRepository) RestoreFile(parent_id uuid.UUID, name string) (*model.FileModel, error) {
	args := m.Called(parent_id, name)
	return args.Get(0).(*model.FileModel), args.Error(1)
}

func (m *MockFileRepository) GetFiles(parentID uuid.UUID) ([]*model.FileModel, error) {
	args := m.Called(parentID)
	return args.Get(0).([]*model.FileModel), args.Error(1)
}

func (m *MockFileRepository) Update(file *model.FileModel) error {
	args := m.Called(file)
	return args.Error(0)
}

func (m *MockFileRepository) DeleteFile(fileID uuid.UUID) error {
	args := m.Called(fileID)
	return args.Error(0)
}

func (m *MockFileRepository) ScrubFile(fileID uuid.UUID) error {
	args := m.Called(fileID)
	return args.Error(0)
}

// MockFolderRepository is a mock implementation of the FolderRepository interface
type MockFolderRepository struct {
	mock.Mock
}

// GetRootFolder implements domain.FolderRepository.
func (m *MockFolderRepository) GetRootFolder(user_id string) (*model.FolderModel, error) {
	args := m.Called(user_id)
	return args.Get(0).(*model.FolderModel), args.Error(1)
}

func (m *MockFolderRepository) Create(folder *model.FolderModel) (string, error) {
	args := m.Called(folder)
	return args.Get(0).(string), args.Error(1)
}

func (m *MockFolderRepository) GetFolder(id uuid.UUID) (*model.FolderModel, error) {
	args := m.Called(id)
	return args.Get(0).(*model.FolderModel), args.Error(1)
}

func (m *MockFolderRepository) GetFolders(parentID uuid.UUID) ([]model.FolderModel, error) {
	args := m.Called(parentID)
	return args.Get(0).([]model.FolderModel), args.Error(1)
}

func (m *MockFolderRepository) MoveFolder(folderID, newParentID uuid.UUID) error {
	args := m.Called(folderID, newParentID)
	return args.Error(0)
}

func (m *MockFolderRepository) RenameFolder(user_id string, folderID uuid.UUID, newName string) error {
	args := m.Called(user_id, folderID, newName)
	return args.Error(0)
}

// DeleteFolder implements domain.FolderRepository.
func (m *MockFolderRepository) DeleteFolder(folder_id uuid.UUID) (parent_id string, err error) {
	args := m.Called(folder_id)
	return args.String(0), args.Error(1)
}

// ScrubFolder implements domain.FolderRepository.
func (m *MockFolderRepository) ScrubFolder(user_id string, folder_id uuid.UUID) (parent_id string, err error) {
	args := m.Called(user_id, folder_id)
	return args.String(0), args.Error(1)
}

type MockFileSystemService struct {
	mock.Mock
}

// FSUpdateFile implements domain.FileSystemService.
func (m *MockFileSystemService) FSUpdateFile(oldPath string, newPath string) error {
	args := m.Called(oldPath, newPath)
	return args.Error(0)
}

// FSValidatePath implements domain.FileSystemService.
func (m *MockFileSystemService) FSValidatePath(path string) (string, error) {
	args := m.Called(path)
	return args.String(0), args.Error(1)
}

func (m *MockFileSystemService) FSWriteFile(path string, data []byte) error {
	args := m.Called(path, data)
	return args.Error(0)
}

func (m *MockFileSystemService) FSGetFile(path string) ([]byte, error) {
	args := m.Called(path)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockFileSystemService) FSDeleteFile(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

// FSDeleteFolder implements domain.FileSystemService.
func (m *MockFileSystemService) FSDeleteFolder(folderPath string) error {
	args := m.Called(folderPath)
	return args.Error(0)
}

type MockFolderService struct {
	mock.Mock
}

func NewMockFolderService() domain.FolderService {
	return &MockFolderService{}
}

// GetRootFolder implements domain.FolderService.
func (m *MockFolderService) GetRootFolder(user_id string) (*model.FolderModel, error) {
	args := m.Called(user_id)
	return args.Get(0).(*model.FolderModel), args.Error(1)
}

// CreateFolder implements domain.FolderService.
func (m *MockFolderService) CreateFolder(user_id string, parent_id string, folder_name string, description string) (string, error) {
	args := m.Called(user_id, parent_id, folder_name, description)
	return args.Get(0).(string), args.Error(1)
}

// GetFolders implements domain.FolderService.
func (m *MockFolderService) GetFolders(parent_id string) ([]model.FolderModel, error) {
	args := m.Called(parent_id)
	return args.Get(0).([]model.FolderModel), args.Error(1)
}

// MoveFolder implements domain.FolderService.
func (m *MockFolderService) MoveFolder(folder_id string, new_parent_id string) error {
	args := m.Called(folder_id, new_parent_id)
	return args.Error(0)
}

// RenameFolder implements domain.FolderService.
func (m *MockFolderService) RenameFolder(user_id, folder_id string, new_name string) error {
	args := m.Called(user_id, folder_id, new_name)
	return args.Error(0)
}

func (m *MockFolderService) GetFolder(user_id, folderID string) (*model.FolderModel, error) {
	args := m.Called(user_id, folderID)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*model.FolderModel), args.Error(1)
}

// DeleteFolder implements domain.FolderService.
func (m *MockFolderService) DeleteFolder(folder_id string) (string, error) {
	args := m.Called(folder_id)
	return args.String(0), args.Error(1)
}

// ScrubFolder implements domain.FolderService.
func (m *MockFolderService) ScrubFolder(user_id string, folder_id string) (string, error) {
	args := m.Called(user_id, folder_id)
	return args.String(0), args.Error(1)
}

type MockFileService struct {
	mock.Mock
}

func NewMockFileService() domain.FileService {
	return &MockFileService{}
}

// CreateFile implements domain.FileService.
func (m *MockFileService) CreateFile(user_id, parent_id, file_name, content_type string, file_data []byte) (string, error) {
	args := m.Called(user_id, parent_id, file_name, content_type, file_data)
	return args.String(0), args.Error(1)
}

// GetFile implements domain.FileService.
func (m *MockFileService) GetFile(file_id string) (*model.FileModel, error) {
	args := m.Called(file_id)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*model.FileModel), args.Error(1)
}

// GetFiles implements domain.FileService.
func (m *MockFileService) GetFiles(parent_id string) ([]model.FileModel, error) {
	args := m.Called(parent_id)
	return args.Get(0).([]model.FileModel), args.Error(1)
}

// MoveFile implements domain.FileService.
func (m *MockFileService) MoveFile(file_id, new_parent_id string) error {
	args := m.Called(file_id, new_parent_id)
	return args.Error(0)
}

// RenameFile implements domain.FileService.
func (m *MockFileService) RenameFile(file_id, new_name string) error {
	args := m.Called(file_id, new_name)
	return args.Error(0)
}

// UpdateFile implements domain.FileService.
func (m *MockFileService) UpdateFile(user_id, file_id, new_file_name string, new_file_data []byte) error {
	args := m.Called(user_id, file_id, new_file_name, new_file_data)
	return args.Error(0)
}

// DeleteFile implements domain.FileService.
func (m *MockFileService) DeleteFile(file_id string) (string, error) {
	args := m.Called(file_id)
	return args.String(0), args.Error(1)
}

// ScrubFile implements domain.FileService.
func (m *MockFileService) ScrubFile(file_id string) (string, error) {
	args := m.Called(file_id)
	return args.String(0), args.Error(1)
}

type MockBuckt struct {
	*MockFileService
	*MockFolderService
}

func NewMockBuckt() *MockBuckt {
	return &MockBuckt{
		MockFileService:   new(MockFileService),
		MockFolderService: new(MockFolderService),
	}
}

func (m *MockBuckt) UploadFile(user_id, parent_id, file_name, content_type string, file_data []byte) (string, error) {
	args := m.MockFileService.Called(user_id, parent_id, file_name, content_type, file_data)

	return args.Get(0).(string), args.Error(1)
}

func (m *MockBuckt) GetFile(fileID string) (*model.FileModel, error) {
	args := m.MockFileService.Called(fileID)
	return args.Get(0).(*model.FileModel), args.Error(1)
}

func (m *MockBuckt) ListFiles(parentID string) ([]model.FileModel, error) {
	args := m.MockFileService.Called(parentID)
	return args.Get(0).([]model.FileModel), args.Error(1)
}

func (m *MockBuckt) MoveFile(fileID, newParentID string) error {
	args := m.MockFileService.Called(fileID, newParentID)
	return args.Error(0)
}

func (m *MockBuckt) RenameFile(fileID, newName string) error {
	args := m.MockFileService.Called(fileID, newName)
	return args.Error(0)
}

func (m *MockBuckt) DeleteFile(fileID string) (string, error) {
	args := m.MockFileService.Called(fileID)
	return args.String(0), args.Error(1)
}

func (m *MockBuckt) ScrubFile(fileID string) (string, error) {
	args := m.MockFileService.Called(fileID)
	return args.String(0), args.Error(1)
}

func (m *MockBuckt) NewFolder(userID, parentID, folderName, description string) (string, error) {
	args := m.MockFolderService.Called(userID, parentID, folderName, description)
	return args.String(0), args.Error(1)
}

func (m *MockBuckt) ListFolders(parentID string) ([]model.FolderModel, error) {
	args := m.MockFolderService.Called(parentID)
	return args.Get(0).([]model.FolderModel), args.Error(1)
}

func (m *MockBuckt) GetFolderWithContent(userID, folderID string) (*model.FolderModel, error) {
	args := m.MockFolderService.Called(userID, folderID)
	return args.Get(0).(*model.FolderModel), args.Error(1)
}

func (m *MockBuckt) MoveFolder(folderID, newParentID string) error {
	args := m.MockFolderService.Called(folderID, newParentID)
	return args.Error(0)
}

func (m *MockBuckt) RenameFolder(userID, folderID, newName string) error {
	args := m.MockFolderService.Called(userID, folderID, newName)
	return args.Error(0)
}

func (m *MockBuckt) DeleteFolder(folderID string) (string, error) {
	args := m.MockFolderService.Called(folderID)
	return args.String(0), args.Error(1)
}

func (m *MockBuckt) DeleteFolderPermanently(userID, folderID string) (string, error) {
	args := m.MockFolderService.Called(userID, folderID)
	return args.String(0), args.Error(1)
}

func (m *MockBuckt) GetHandler() http.Handler {
	args := m.MockFolderService.Called()
	return args.Get(0).(http.Handler)
}

func (m *MockBuckt) StartServer(port string) error {
	args := m.MockFolderService.Called(port)
	return args.Error(0)
}

func (m *MockBuckt) Close() {
	m.MockFolderService.Called()
}

type MockCloudService struct {
	mock.Mock
}

func NewMockCloudService() domain.CloudService {
	return &MockCloudService{}
}

// UploadFile implements domain.CloudService.
func (m *MockCloudService) UploadFileToCloud(file_id string) error {
	args := m.Called(file_id)
	return args.Error(0)
}

// UploadFolder implements domain.CloudService.
func (m *MockCloudService) UploadFolderToCloud(user_id string, folder_id string) error {
	args := m.Called(user_id, folder_id)
	return args.Error(0)
}
