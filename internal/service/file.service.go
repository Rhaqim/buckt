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
	"github.com/Rhaqim/buckt/request"
	"github.com/google/uuid"
)

type BucktStore struct {
	ownerStore  domain.BucktRepository[model.OwnerModel]
	bucketStore domain.BucktRepository[model.BucketModel]
	folderStore *model.FolderRepository
	fileStore   domain.BucktRepository[model.FileModel]
	tagStore    domain.BucktRepository[model.TagModel]
}

type BucktService struct {
	*logger.Logger
	*config.Config
	BucktStore
}

func NewBucktService(log *logger.Logger, cfg *config.Config, ownerStore domain.BucktRepository[model.OwnerModel],
	bucketStore domain.BucktRepository[model.BucketModel], folderStore *model.FolderRepository,
	fileStore domain.BucktRepository[model.FileModel], tagStore domain.BucktRepository[model.TagModel]) domain.BucktService {

	store := BucktStore{
		fileStore:   fileStore,
		bucketStore: bucketStore,
		ownerStore:  ownerStore,
		tagStore:    tagStore,
		folderStore: folderStore,
	}

	return &BucktService{log, cfg, store}
}

func (bs *BucktService) CreateOwner(name, email string) error {
	var owner model.OwnerModel = model.OwnerModel{
		Name:  name,
		Email: email,
	}

	return bs.ownerStore.Create(&owner)
}

func (bs *BucktService) CreateBucket(name, description, ownerID string) error {
	fmt.Printf("ownerID: %s\n", ownerID)
	parsedId, err := uuid.Parse(ownerID)
	if err != nil {
		return errs.ErrInvalidUUID
	}

	var bucket model.BucketModel = model.BucketModel{
		Name:        name,
		Description: description,
		OwnerID:     parsedId,
	}

	return bs.bucketStore.Create(&bucket)
}

func (bs *BucktService) DeleteBucket(bucketName string) error {
	bucket, err := bs.bucketStore.GetBy("name = ?", bucketName)
	if err != nil {
		return errs.ErrBucketNotFound
	}

	return bs.bucketStore.Delete(bucket.ID)
}

func (bs *BucktService) GetBuckets(ownerID string) ([]interface{}, error) {
	buckets, err := bs.bucketStore.GetMany("owner_id = ?", ownerID)
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
	bucket, err := bs.bucketStore.GetBy("name = ?", bucketName)
	if err != nil {
		return errs.ErrBucketNotFound
	}

	currentParentID := bucket.ID.String()

	for _, folder := range utils.ValidateFolderPath(folderPath) {
		// Check if the folder exists under the current parent
		folderModel, err := bs.folderStore.GetBy("name = ? AND parent_id = ?", folder, currentParentID)
		if err != nil {
			// Folder does not exist; create it
			newFolderModel := model.FolderModel{
				Name:     folder,
				ParentID: uuid.MustParse(currentParentID), // Convert currentParentID back to UUID
				BucketID: bucket.ID,
			}

			if err := bs.folderStore.Create(&newFolderModel); err != nil {
				return err
			}

			currentParentID = newFolderModel.ID.String()
		} else {
			currentParentID = folderModel.ID.String()
		}
	}

	// Check if file already exists
	_, err = bs.fileStore.GetBy("name = ? AND parent_id = ?", name, currentParentID)
	if err == nil {
		return errs.ErrFileAlreadyExists
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
		ParentID:    uuid.MustParse(currentParentID),
		BucketID:    bucket.ID,
		ContentType: contentType,
		Hash:        hash,
		Size:        fileSize,
		Path:        filepath.Join(bucketName, folderPath),
	}

	// Create file
	if err := bs.fileStore.Create(&fileModel); err != nil {
		return err
	}

	// Create file tags
	tags := strings.Split(file_.Header.Get("Tags"), ",")
	for _, tag := range tags {
		tagModel := model.TagModel{
			Name: tag,
		}

		if err := bs.tagStore.Create(&tagModel); err != nil {
			return err
		}
	}

	// Create file on disk
	filePath := filepath.Join(bs.Config.Media.Dir, bucketName, folderPath, fmt.Sprintf("%s%s", fileModel.ID, ext))
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err := f.Write(file); err != nil {
		return err
	}

	return nil
}

func (bs *BucktService) RenameFile(request.RenameFileRequest) error {
	panic("implement me")
}

func (bs *BucktService) MoveFile(request.MoveFileRequest) error {
	panic("implement me")
}

func (bs *BucktService) ServeFile(request.FileRequest, bool) (string, error) {
	panic("implement me")
}

func (bs *BucktService) DownloadFile(request.FileRequest) ([]byte, error) {
	panic("implement me")
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

func (bs *BucktService) GetFilesInFolder(request.BaseFileRequest) ([]interface{}, error) {
	panic("implement me")
}

func (bs *BucktService) GetSubFolders(request.BaseFileRequest) ([]interface{}, error) {
	panic("implement me")
}
