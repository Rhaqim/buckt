package app

import (
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/gin-gonic/gin"
)

type APIService struct {
	domain.FolderService
	domain.FileService
}

func NewAPIService(fs domain.FolderService, f domain.FileService) domain.APIService {
	return &APIService{
		FolderService: fs,
		FileService:   f,
	}
}

// CreateFolder implements domain.APIService.
// Subtle: this method shadows the method (FolderService).CreateFolder of APIService.FolderService.
func (a *APIService) CreateFolder(c *gin.Context) {
	// get the user_id from the context
	user_id, ok := c.Get("user_id")
	if !ok {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	// convert the user_id to string
	userID, ok := user_id.(string)
	if !ok {
		c.JSON(500, gin.H{"error": "failed to parse user_id"})
		return
	}

	// get the parent_id from the request
	parentID := c.PostForm("parent_id")
	if parentID == "" {
		parentID = "00000000-0000-0000-0000-000000000000"
	}

	// get the folder name from the request
	folderName := c.PostForm("folder_name")
	if folderName == "" {
		c.JSON(400, gin.H{"error": "folder_name is required"})
		return
	}

	// get the description from the request
	description := c.PostForm("description")

	// create the folder
	if err := a.FolderService.CreateFolder(userID, parentID, folderName, description); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "folder created"})
}

// DeleteFile implements domain.APIService.
// Subtle: this method shadows the method (FileService).DeleteFile of APIService.FileService.
func (a *APIService) DeleteFile(c *gin.Context) {
	// get the file_id from the request
	fileID := c.Param("file_id")
	if fileID == "" {
		c.JSON(400, gin.H{"error": "file_id is required"})
		return
	}

	// delete the file
	if err := a.FileService.DeleteFile(fileID); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "file deleted"})
}

// DeleteFolder implements domain.APIService.
func (a *APIService) DeleteFolder(c *gin.Context) {
	panic("unimplemented")
}

// DownloadFile implements domain.APIService.
func (a *APIService) DownloadFile(c *gin.Context) {
	panic("unimplemented")
}

// GetDescendants implements domain.APIService.
func (a *APIService) GetDescendants(c *gin.Context) {
	panic("unimplemented")
}

// GetFilesInFolder implements domain.APIService.
func (a *APIService) GetFilesInFolder(c *gin.Context) {
	panic("unimplemented")
}

// GetFolderContent implements domain.APIService.
func (a *APIService) GetFolderContent(c *gin.Context) {
	panic("unimplemented")
}

// GetSubFolders implements domain.APIService.
func (a *APIService) GetSubFolders(c *gin.Context) {
	panic("unimplemented")
}

// MoveFolder implements domain.APIService.
// Subtle: this method shadows the method (FolderService).MoveFolder of APIService.FolderService.
func (a *APIService) MoveFolder(c *gin.Context) {
	panic("unimplemented")
}

// RenameFolder implements domain.APIService.
// Subtle: this method shadows the method (FolderService).RenameFolder of APIService.FolderService.
func (a *APIService) RenameFolder(c *gin.Context) {
	panic("unimplemented")
}

// UploadFile implements domain.APIService.
func (a *APIService) UploadFile(c *gin.Context) {
	panic("unimplemented")
}
