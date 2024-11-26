package service

import (
	"crypto/sha256"
	"fmt"
	"mime"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/Rhaqim/buckt/config"
	"github.com/Rhaqim/buckt/internal/domain"
	errs "github.com/Rhaqim/buckt/internal/error"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/internal/utils"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/Rhaqim/buckt/request"
	"github.com/google/uuid"
)

type BucktService struct {
	*logger.Logger
	*config.Config
	*model.BucktStore
	domain.BucktFileSystemService
}

func NewBucktService(log *logger.Logger, cfg *config.Config, store *model.BucktStore) domain.BucktService {

	bfs := NewBucktFSService(log, cfg)

	return &BucktService{log, cfg, store, bfs}
}

func (bs *BucktService) CreateOwner(name, email string) error {
	var owner model.OwnerModel = model.OwnerModel{
		Name:  name,
		Email: email,
	}

	return bs.OwnerStore.Create(&owner)
}

func (bs *BucktService) CreateBucket(name, description, ownerID string) error {
	parsedId, err := uuid.Parse(ownerID)
	if err != nil {
		return errs.ErrInvalidUUID
	}

	var bucket model.BucketModel = model.BucketModel{
		Name:        name,
		Description: description,
		OwnerID:     parsedId,
	}

	return bs.BucketStore.Create(&bucket)
}

func (bs *BucktService) DeleteBucket(bucketName string) error {
	bucket, err := bs.BucketStore.GetBy("name = ?", bucketName)
	if err != nil {
		return errs.ErrBucketNotFound
	}

	return bs.BucketStore.Delete(bucket.ID)
}

func (bs *BucktService) GetBucket(bucketName string) (interface{}, error) {
	bucket, err := bs.BucketStore.GetBy("name = ?", bucketName)
	if err != nil {
		return nil, errs.ErrBucketNotFound
	}

	return bucket, nil
}

func (bs *BucktService) GetBuckets(ownerID uuid.UUID) ([]interface{}, error) {

	buckets, err := bs.BucketStore.GetMany("owner_id = ?", ownerID)
	if err != nil {
		return nil, errs.ErrBucketNotFound
	}

	return utils.InterfaceSlice(buckets), nil
}

func (bs *BucktService) UploadFile(file_ *multipart.FileHeader, bucketName string, folderPath string) error {
	// Read file from request
	fileName, file, err := utils.ProcessFile(file_)
	if err != nil {
		return err
	}

	name := strings.Split(fileName, ".")[0]

	// Check if bucket exists
	bucket, err := bs.BucketStore.GetBy("name = ?", bucketName)
	if err != nil {
		return errs.ErrBucketNotFound
	}

	currentParentID := bucket.ID.String()

	for _, folder := range utils.ValidateFolderPath(folderPath) {
		// Check if the folder exists under the current parent
		folderModel, err := bs.FolderStore.GetBy("name = ? AND parent_id = ?", folder, currentParentID)
		if err != nil {
			// Folder does not exist; create it
			newFolderModel := model.FolderModel{
				Name:     folder,
				ParentID: uuid.MustParse(currentParentID), // Convert currentParentID back to UUID
				BucketID: bucket.ID,
			}

			if err := bs.FolderStore.Create(&newFolderModel); err != nil {
				return err
			}

			currentParentID = newFolderModel.ID.String()
		} else {
			currentParentID = folderModel.ID.String()
		}
	}

	// Check if file already exists
	_, err = bs.FileStore.GetBy("name = ? AND parent_id = ?", name, currentParentID)
	if err == nil {
		return errs.ErrFileAlreadyExists
	}

	var tagModels []model.TagModel

	// Create file tags
	tags := strings.Split(file_.Header.Get("Tags"), ",")
	for _, tag := range tags {
		tagModel := model.TagModel{
			Name: tag,
		}

		if err := bs.TagStore.Create(&tagModel); err != nil {
			return err
		}

		tagModels = append(tagModels, tagModel)
	}

	// File path
	path := filepath.Join(bucketName, folderPath, fileName)

	// Save the file to the file system
	err = bs.FSWriteFile(path, file)
	if err != nil {
		return err
	}

	// Get file extension
	ext := filepath.Ext(fileName)
	if ext == "" {
		exts, err := mime.ExtensionsByType(file_.Header.Get("Content-Type"))
		if err != nil {
			ext = ".bin"
		} else {
			ext = exts[0]
		}
	}

	// Get file content type
	contentType := mime.TypeByExtension(ext)

	// Calculate file hash (SHA-256) for uniqueness check and future retrieval
	hash := fmt.Sprintf("%x", sha256.Sum256(file))

	// File size in bytes
	fileSize := int64(len(file))

	// Create file model
	fileModel := model.FileModel{
		Name:        name,
		Path:        path,
		ContentType: contentType,
		Size:        fileSize,
		BucketID:    bucket.ID,
		ParentID:    uuid.MustParse(currentParentID),
		Hash:        hash,
		Tags:        tagModels,
	}

	// Create file
	if err := bs.FileStore.Create(&fileModel); err != nil {
		return err
	}

	return nil
}

func (bs *BucktService) RenameFile(request request.RenameFileRequest) error {
	path := filepath.Join(request.BucketName, request.FolderPath, request.Filename)

	file, err := bs.FileStore.GetBy("path = ?", path)
	if err != nil {
		return errs.ErrFileNotFound
	}

	newPath := filepath.Join(request.BucketName, request.FolderPath, request.NewFilename)

	err = bs.FSUpdateFile(path, newPath)
	if err != nil {
		return err
	}

	file.Name = request.NewFilename
	file.Path = newPath

	return bs.FileStore.Update(&file)
}

func (bs *BucktService) MoveFile(request.MoveFileRequest) error {
	panic("implement me")
}

func (bs *BucktService) DownloadFile(request request.FileRequest) ([]byte, error) {
	path := filepath.Join(request.BucketName, request.FolderPath, request.Filename)

	file, err := bs.FileStore.GetBy("path = ?", path)
	if err != nil {
		return nil, errs.ErrFileNotFound
	}

	return bs.FSGetFile(file.Path)
}

func (bs *BucktService) ServeFile(filepath string) (string, error) {
	fullPath, err := bs.FSValidatePath(filepath)
	if err != nil {
		return "", err
	}

	return fullPath, nil
}

func (bs *BucktService) DeleteFile(request.FileRequest) error {
	panic("implement me")
}

func (bs *BucktService) CreateFolder(bucketName, folderPath string) error {
	panic("implement me")
}

func (bs *BucktService) RenameFolder(request.RenameFolderRequest) error {
	panic("implement me")
}

func (bs *BucktService) MoveFolder(request.MoveFolderRequest) error {
	panic("implement me")
}

func (bs *BucktService) DeleteFolder(request.BaseFileRequest) error {
	panic("implement me")
}

func (bs *BucktService) GetFilesInFolder(request request.BaseFileRequest) ([]interface{}, error) {
	// Split the folderPath into individual folder names
	folderNames := strings.Split(request.FolderPath, "/")
	if len(folderNames) < 2 {
		return nil, errs.ErrMinParentMinChild
	}

	var files []model.FileModel

	query := `
        WITH RECURSIVE folder_hierarchy AS (
            SELECT id, name, parent_id
            FROM folder_models
            WHERE name = ? AND parent_id = (SELECT id FROM bucket_models WHERE name = ?)
            UNION ALL
            SELECT fm.id, fm.name, fm.parent_id
            FROM folder_models fm
            INNER JOIN folder_hierarchy fh ON fm.parent_id = fh.id
        )
        SELECT f.* 
        FROM folder_hierarchy fh
        JOIN file_models f ON f.parent_id = fh.id
        WHERE fh.name = ?;
    `

	// Execute query using the first folder name and the bucket name
	if err := bs.FolderStore.RawQuery(query, folderNames[0], request.BucketName, folderNames[len(folderNames)-1]).Scan(&files).Error; err != nil {
		return nil, err
	}

	return utils.InterfaceSlice(files), nil
}

func (bs *BucktService) GetSubFolders(request request.BaseFileRequest) ([]interface{}, error) {
	if request.FolderPath == "" {
		bucket, err := bs.BucketStore.GetBy("name = ?", request.BucketName)
		if err != nil {
			return nil, errs.ErrBucketNotFound
		}

		subfolders, err := bs.GetDescendants(bucket.ID)
		if err != nil {
			return nil, errs.ErrFolderNotFound
		}

		return subfolders, nil
	}

	// Split the folderPath into individual folder names
	folderNames := strings.Split(request.FolderPath, "/")
	if len(folderNames) < 1 {
		return nil, errs.ErrFolderNotInPath
	}

	// Define the slice to store the result
	var subfolders []model.FolderModel

	query := `
        WITH RECURSIVE folder_hierarchy AS (
            -- Get the first folder under the specified bucket
            SELECT id, name, parent_id
            FROM folder_models
            WHERE name = ? AND parent_id = (SELECT id FROM bucket_models WHERE name = ?)
            UNION ALL
            SELECT fm.id, fm.name, fm.parent_id
            FROM folder_models fm
            INNER JOIN folder_hierarchy fh ON fm.parent_id = fh.id
        )
        -- Fetch the immediate subfolders of the target folder (the last folder in the path)
        SELECT * 
        FROM folder_models
        WHERE parent_id = (
            SELECT id 
            FROM folder_hierarchy 
            WHERE name = ? 
            -- ORDER BY created_at DESC
            LIMIT 1
        );
    `
	// Run the query with the bucket name, first folder, and last folder name in the path
	if err := bs.FolderStore.RawQuery(query, folderNames[0], request.BucketName, folderNames[len(folderNames)-1]).Scan(&subfolders).Error; err != nil {
		return nil, err
	}

	return utils.InterfaceSlice(subfolders), nil
}

func (bs *BucktService) GetDescendants(ID uuid.UUID) ([]interface{}, error) {
	var descendants []model.FolderModel

	descendants, err := bs.FolderStore.GetMany("parent_id = ?", ID)
	if err != nil {
		return nil, errs.ErrFolderNotFound
	}

	return utils.InterfaceSlice(descendants), nil
}
