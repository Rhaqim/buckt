package service

import (
	"crypto/sha256"
	"fmt"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rhaqim/buckt/config"
	"github.com/Rhaqim/buckt/internal/domain"
	errs "github.com/Rhaqim/buckt/internal/error"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/internal/utils"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/google/uuid"
)

type Store struct {
	ownerStore  domain.Repository[model.OwnerModel]
	bucketStore domain.Repository[model.BucketModel]
	folderStore *model.FolderRepository
	fileStore   domain.Repository[model.FileModel]
	tagStore    domain.Repository[model.TagModel]
}

type StorageService struct {
	*logger.Logger
	*config.Config
	Store
}

func NewStorageService(log *logger.Logger, cfg *config.Config, ownerStore domain.Repository[model.OwnerModel], bucketStore domain.Repository[model.BucketModel], folderStore *model.FolderRepository, fileStore domain.Repository[model.FileModel], tagStore domain.Repository[model.TagModel]) domain.StorageFileService {
	store := Store{fileStore: fileStore, bucketStore: bucketStore, ownerStore: ownerStore, tagStore: tagStore, folderStore: folderStore}

	return &StorageService{log, cfg, store}
}

func (s *StorageService) UploadFile(file_ *multipart.FileHeader, bucketName string, folderPath string) error {
	// Read file from request
	fileName, file, err := utils.ProcessFile(file_)
	if err != nil {
		return err
	}

	name := strings.Split(fileName, ".")[0]

	// Check if bucket exists
	bucket, err := s.bucketStore.GetBy("name = ?", bucketName)
	if err != nil {
		return errs.ErrBucketNotFound
	}

	currentParentID := bucket.ID.String()

	for _, folder := range utils.ValidateFolderPath(folderPath) {
		// Check if the folder exists under the current parent
		folderModel, err := s.folderStore.GetBy("name = ? AND parent_id = ?", folder, currentParentID)
		if err != nil {
			// Folder does not exist; create it
			newFolderModel := model.FolderModel{
				Name:     folder,
				ParentID: uuid.MustParse(currentParentID), // Convert currentParentID back to UUID
				BucketID: bucket.ID,
			}

			if err := s.folderStore.Create(&newFolderModel); err != nil {
				return fmt.Errorf("failed to create folder: %w", err)
			}

			// Update currentParentID to the new folder's ID
			currentParentID = newFolderModel.ID.String()
		} else {
			// Folder exists; update currentParentID to its ID
			currentParentID = folderModel.ID.String()
		}
	}

	// Check if file already exists in the bucket
	// if _, err := s.fileStore.GetBy("name = ?", name); err == nil {
	// 	return errs.ErrFileAlreadyExists
	// }

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
		Name:        name,
		Path:        filePath,
		ContentType: contentType,
		Size:        fileSize,
		Hash:        hash,
		BucketID:    bucket.ID, // Associate the file with the existing bucket
		ParentID:    uuid.MustParse(currentParentID),
	}

	// Insert the file entry into the database
	if err := s.fileStore.Create(&fileModel); err != nil {
		return fmt.Errorf("failed to save file metadata: %w", err)
	}

	return nil
}

func (s *StorageService) Serve(filename string, serve bool) (string, error) {
	// Get file from database
	file, err := s.fileStore.GetBy("name = ?", filename)
	if err != nil {
		return "", err
	}

	// Read file from storage
	filePath := file.Path
	if _, err := os.Stat(filePath); err != nil {
		return "", err
	}

	if serve {
		filePath = s.Config.Endpoint.URL + "/server/" + filename
	}

	return filePath, nil
}

func (s *StorageService) DownloadFile(filename string) ([]byte, error) {
	// Get file from database
	filePath, err := s.Serve(filename, false)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *StorageService) DeleteFile(filename string) error {
	// Get file from database
	file, err := s.fileStore.GetBy("name = ?", filename)
	if err != nil {
		return err
	}

	// Delete file from storage
	filePath := file.Path
	if err := os.Remove(filePath); err != nil {
		return err
	}

	// Delete file from database
	return s.fileStore.Delete(file.ID)
}

func (s *StorageService) CreateBucket(name, description, owner_ string) error {
	owner, err := s.ownerStore.GetBy("name = ?", owner_)
	if err != nil {
		return err
	}

	var bucket model.BucketModel = model.BucketModel{
		Name:        name,
		Description: description,
		OwnerID:     owner.ID,
	}

	return s.bucketStore.Create(&bucket)
}

func (s *StorageService) CreateOwner(name, email string) error {
	var owner model.OwnerModel = model.OwnerModel{
		Name:  name,
		Email: email,
	}

	return s.ownerStore.Create(&owner)
}

func (s *StorageService) GetBuckets() ([]model.BucketModel, error) {
	return s.bucketStore.FindAll()
}

func (s *StorageService) GetFiles(bucketName string) ([]interface{}, error) {
	var response []interface{}

	bucket, err := s.bucketStore.GetBy("name = ?", bucketName)
	if err != nil {
		return nil, err
	}

	files, err := s.fileStore.GetMany("bucket_id = ?", bucket.ID.String())
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		response = append(response, file)
	}

	return response, nil
}

func (s *StorageService) GetFilesInFolder(bucketName, folderPath string) ([]interface{}, error) {
	var response []interface{}

	files, err := s.folderStore.GetFilesFromPath(bucketName, folderPath)
	if err != nil {
		return nil, err
	}

	fmt.Println("Files returned", files)

	for _, file := range files {
		response = append(response, file)
	}

	return response, nil
}

func (s *StorageService) GetSubFolders(bucketName, folderPath string) ([]interface{}, error) {
	var response []interface{}

	folders, err := s.folderStore.GetSubfolders(bucketName, folderPath)
	if err != nil {
		return nil, err
	}

	for _, folder := range folders {
		response = append(response, folder)
	}

	return response, nil
}
