package app

import (
	"fmt"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/utils"
	"github.com/Rhaqim/buckt/pkg/response"
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
	user_id := c.GetString("owner_id")

	var req struct {
		ParentID    string `json:"parent_id"`
		FolderName  string `json:"folder_name"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(400, response.Error("invalid request", err.Error()))
		return
	}

	// create the folder
	new_folder_id, err := a.FolderService.CreateFolder(user_id, req.ParentID, req.FolderName, req.Description)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to create folder", err))
		return
	}

	c.JSON(200, response.Success("folder created, ID: "+new_folder_id))
}

// GetFolderContent implements domain.APIService.
func (a *APIService) GetFolderContent(c *gin.Context) {
	user_id := c.GetString("owner_id")

	// get the folder_id from the request
	folderID := c.Param("folder_id")
	if folderID == "" {
		c.AbortWithStatusJSON(400, response.Error("folder_id is required", ""))
		return
	}

	// get the folder content
	folderContent, err := a.FolderService.GetFolder(user_id, folderID)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to get folder content", err))
		return
	}

	c.JSON(200, response.Success(folderContent))
}

// GetFilesInFolder implements domain.APIService.
func (a *APIService) GetFilesInFolder(c *gin.Context) {
	// get the parent_id from the request
	parentID := c.Param("parent_id")
	if parentID == "" {
		c.AbortWithStatusJSON(400, response.Error("parent_id is required", ""))
		return
	}

	// get the files in the folder
	files, err := a.FileService.GetFiles(parentID)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to get files", err))
		return
	}

	c.JSON(200, response.Success(files))
}

// GetSubFolders implements domain.APIService.
func (a *APIService) GetSubFolders(c *gin.Context) {
	panic("unimplemented")
}

// GetDescendants implements domain.APIService.
func (a *APIService) GetDescendants(c *gin.Context) {
	panic("unimplemented")
}

// DeleteFolder implements domain.APIService.
func (a *APIService) DeleteFolder(c *gin.Context) {
	// get the folder_id from the request
	folderID := c.Param("folder_id")
	if folderID == "" {
		c.AbortWithStatusJSON(400, response.Error("folder_id is required", ""))
		return
	}

	// ge tthe folder with content
	_, err := a.FolderService.DeleteFolder(folderID)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to delete folder", err))
		return
	}

	c.JSON(200, response.Success("folder deleted"))
}

// DeleteFolderPermanently implements domain.APIService.
func (a *APIService) DeleteFolderPermanently(c *gin.Context) {
	user_id := c.GetString("owner_id")

	// get the folder_id from the request
	folderID := c.Param("folder_id")
	if folderID == "" {
		c.AbortWithStatusJSON(400, response.Error("folder_id is required", ""))
		return
	}

	// ge tthe folder with content
	_, err := a.FolderService.ScrubFolder(user_id, folderID)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to delete folder", err))
		return
	}

	c.JSON(200, response.Success("folder deleted"))
}

// MoveFolder implements domain.APIService.
// Subtle: this method shadows the method (FolderService).MoveFolder of APIService.FolderService.
func (a *APIService) MoveFolder(c *gin.Context) {
	var req struct {
		FolderID    string `json:"folder_id"`
		NewParentID string `json:"new_parent_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(400, response.Error("invalid request", err.Error()))
		return
	}

	// move the folder
	if err := a.FolderService.MoveFolder(req.FolderID, req.NewParentID); err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to move folder", err))
		return
	}

	c.JSON(200, response.Success("folder moved"))
}

// RenameFolder implements domain.APIService.
// Subtle: this method shadows the method (FolderService).RenameFolder of APIService.FolderService.
func (a *APIService) RenameFolder(c *gin.Context) {
	user_id := c.GetString("owner_id")

	var req struct {
		FolderID string `json:"folder_id"`
		Name     string `json:"name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(400, response.Error("invalid request", err.Error()))
		return
	}

	// rename the folder
	if err := a.FolderService.RenameFolder(user_id, req.FolderID, req.Name); err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to rename folder", err))
		return
	}

	c.JSON(200, response.Success("folder renamed"))
}

// UploadFile implements domain.APIService.
func (a *APIService) UploadFile(c *gin.Context) {
	// get the user_id from the context
	user_id := c.GetString("owner_id")

	file, err := c.FormFile("file")
	if err != nil {
		c.AbortWithStatusJSON(400, response.Error("file is required", err.Error()))
		return
	}

	// get the parent_id from the request
	parentID := c.PostForm("parent_id")
	if parentID == "" {
		c.AbortWithStatusJSON(400, response.Error("parent_id is required", ""))
		return
	}

	// Read file from request
	fileName, fileByte, err := utils.ProcessFile(file)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to process file", err))
		return
	}

	fileID, err := a.FileService.CreateFile(user_id, parentID, fileName, file.Header.Get("Content-Type"), fileByte)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to create file", err))
		return
	}

	url := a.constructURL(fileID)

	c.JSON(200, response.Success(url))
}

// DownloadFile implements domain.APIService.
func (a *APIService) DownloadFile(c *gin.Context) {
	// get the file_id from the request
	fileID := c.Param("file_id")
	if fileID == "" {
		c.AbortWithStatusJSON(400, response.Error("file_id is required", ""))
		return
	}

	file, err := a.FileService.GetFile(fileID)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to get file", err))
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+file.Name)
	c.Header("Content-Type", file.ContentType)
	c.Data(200, file.ContentType, file.Data)
}

// ServeFile implements domain.APIService.
func (a *APIService) ServeFile(c *gin.Context) {
	// get the file_id from the request
	fileID := c.Param("file_id")
	if fileID == "" {
		c.AbortWithStatusJSON(400, response.Error("file_id is required", ""))
		return
	}

	file, err := a.FileService.GetFile(fileID)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to get file", err))
		return
	}

	c.Data(200, file.ContentType, file.Data)
}

// DeleteFile implements domain.APIService.
// Subtle: this method shadows the method (FileService).DeleteFile of APIService.FileService.
func (a *APIService) DeleteFile(c *gin.Context) {
	// get the file_id from the request
	fileID := c.Param("file_id")
	if fileID == "" {
		c.AbortWithStatusJSON(400, response.Error("file_id is required", ""))
		return
	}

	// delete the file
	_, err := a.FileService.DeleteFile(fileID)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to delete file", err))
		return
	}

	c.JSON(200, response.Success("file deleted"))
}

func (a *APIService) DeleteFilePermanently(c *gin.Context) {
	// get the file_id from the request
	fileID := c.Param("file_id")
	if fileID == "" {
		c.AbortWithStatusJSON(400, response.Error("file_id is required", ""))
		return
	}

	// delete the file
	_, err := a.FileService.ScrubFile(fileID)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to delete file", err))
		return
	}

	c.JSON(200, response.Success("file deleted"))
}

/* Helper functions */

func (f *APIService) constructURL(s string) string {
	return fmt.Sprintf("/serve/%s", s)
}
