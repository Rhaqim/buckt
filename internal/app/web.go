package app

import (
	"fmt"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/utils"
	"github.com/Rhaqim/buckt/pkg/response"
	"github.com/gin-gonic/gin"
)

type WebService struct {
	domain.FolderService
	domain.FileService
}

func NewWebService(fs domain.FolderService, f domain.FileService) domain.WebService {
	return &WebService{
		FolderService: fs,
		FileService:   f,
	}
}

func (w *WebService) ViewFolder(c *gin.Context) {
	user_id := c.GetString("owner_id")

	// get the folder_id from the request
	folderID := c.Param("folder_id")

	// get the folder content
	folderContent, err := w.FolderService.GetFolder(user_id, folderID)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to get folder content", err))
		return
	}

	// Render the dashboard page with the files
	c.HTML(200, "dashboard.html", gin.H{
		"Title":   "Dashboard",
		"page":    "dashboard",
		"ID":      folderContent.ID,
		"Path":    folderContent.Path,
		"Folders": folderContent.Folders,
		"Files":   folderContent.Files,
	})
}

// NewFolder implements domain.WebService.
func (w *WebService) NewFolder(c *gin.Context) {
	user_id := c.GetString("owner_id")

	parentID := c.PostForm("parent_id")
	if parentID == "" {
		c.AbortWithStatusJSON(400, gin.H{"error": "parent_id is required"})
		return
	}

	name := c.PostForm("name")
	if name == "" {
		c.AbortWithStatusJSON(400, gin.H{"error": "name is required"})
		return
	}

	description := c.PostForm("description")

	_, err := w.FolderService.CreateFolder(user_id, parentID, name, description)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to create folder", err))
		return
	}

	// reload the page
	c.Redirect(302, "/web/folder/"+parentID)
}

// MoveFolder implements domain.WebService.
// Subtle: this method shadows the method (FolderService).MoveFolder of WebService.FolderService.
func (w *WebService) MoveFolder(c *gin.Context) {
	folder_id := c.PostForm("folder_id")
	new_parent_id := c.PostForm("new_parent_id")

	if folder_id == "" {
		c.AbortWithStatusJSON(400, gin.H{"error": "folder_id is required"})
		return
	}

	if new_parent_id == "" {
		c.AbortWithStatusJSON(400, gin.H{"error": "new_parent_id is required"})
		return
	}

	err := w.FolderService.MoveFolder(folder_id, new_parent_id)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to move folder", err))
		return
	}

	// reload the page
	c.Redirect(302, "/web/folder/"+new_parent_id)
}

// RenameFolder implements domain.WebService.
// Subtle: this method shadows the method (FolderService).RenameFolder of WebService.FolderService.
func (w *WebService) RenameFolder(c *gin.Context) {
	folder_id := c.PostForm("folder_id")
	new_name := c.PostForm("new_name")

	if folder_id == "" {
		c.AbortWithStatusJSON(400, gin.H{"error": "folder_id is required"})
		return
	}

	if new_name == "" {
		c.AbortWithStatusJSON(400, gin.H{"error": "new_name is required"})
		return
	}

	// rename the folder
	err := w.FolderService.RenameFolder(folder_id, new_name)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to rename folder", err))
		return
	}

	// reload the page
	c.Redirect(302, "/web/folder/"+folder_id)
}

// DeleteFolder implements domain.WebService.
func (w *WebService) DeleteFolder(c *gin.Context) {
	panic("unimplemented")
}

// UploadFile implements domain.WebService.
func (w *WebService) UploadFile(c *gin.Context) {
	user_id := c.GetString("owner_id")

	folderID := c.PostForm("folder_id")
	if folderID == "" {
		c.AbortWithStatusJSON(400, gin.H{"error": "folder_id is required"})
		return
	}

	// Handle file upload
	form, err := c.MultipartForm()
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": "Invalid form data"})
		return
	}

	files := form.File["files"]

	// Loop through each file
	for _, file := range files {
		// Save each file (this example just prints)
		fileName, fileByte, err := utils.ProcessFile(file)
		if err != nil {
			c.AbortWithStatusJSON(500, response.WrapError("failed to process file", err))
			return
		}

		_, err = w.FileService.CreateFile(user_id, folderID, fileName, file.Header.Get("Content-Type"), fileByte)
		if err != nil {
			c.AbortWithStatusJSON(500, response.WrapError("failed to create file", err))
			return
		}

	}

	// reload the page
	c.Redirect(302, "/web/folder/"+folderID)

	// for _, file := range files {
	// 	// Save each file (this example just prints)
	// 	c.SaveUploadedFile(file, "./uploads/"+file.Filename)
	// }
}

// DownloadFile implements domain.WebService.
func (w *WebService) DownloadFile(c *gin.Context) {
	// get the file_id from the request
	fileID := c.Param("file_id")
	if fileID == "" {
		c.AbortWithStatusJSON(400, response.Error("file_id is required", ""))
		return
	}

	// get the file
	file, err := w.FileService.GetFile(fileID)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to get file", err))
		return
	}

	// serve the file
	c.Header("Content-Disposition", "attachment; filename="+file.Name)
	c.Header("Content-Type", file.ContentType)
	c.Data(200, file.ContentType, file.Data)
}

// MoveFile implements domain.WebService.
func (w *WebService) MoveFile(c *gin.Context) {
	panic("unimplemented")
}

// DeleteFile implements domain.WebService.
// Subtle: this method shadows the method (FileService).DeleteFile of WebService.FileService.
func (w *WebService) DeleteFile(c *gin.Context) {
	// get the file_id from the request
	fileID := c.Param("file_id")
	if fileID == "" {
		c.AbortWithStatusJSON(400, response.Error("file_id is required", ""))
		return
	}

	// delete the file
	parent_id, err := w.FileService.DeleteFile(fileID)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to delete file", err))
		return
	}

	fmt.Println("parent_id", parent_id)

	c.JSON(200, response.Success("file deleted"))
}
