package app

import (
	"github.com/Rhaqim/buckt"
	"github.com/Rhaqim/buckt/client/web/domain"
	"github.com/Rhaqim/buckt/pkg/fileutil"
	"github.com/Rhaqim/buckt/pkg/response"
	"github.com/gin-gonic/gin"
)

type WebService struct {
	client *buckt.Client
}

func NewWebService(client *buckt.Client) domain.WebService {
	return &WebService{
		client: client,
	}
}

func (svc *WebService) ViewFolder(c *gin.Context) {
	user_id := c.GetString("owner_id")

	// get the folder_id from the request
	folderID := c.Param("folder_id")

	// get the folder content
	folderContent, err := svc.client.GetFolderWithContent(user_id, folderID)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to get folder content", err))
		return
	}

	// Render the dashboard page with the files
	parentID := ""
	if folderContent.ParentID != nil {
		parentID = folderContent.ParentID.String()
	}
	c.HTML(200, "dashboard.html", gin.H{
		"Title":    "Dashboard",
		"page":     "dashboard",
		"ID":       folderContent.ID,
		"ParentID": parentID,
		"Name":     folderContent.Name,
		"Path":     folderContent.Path,
		"Folders":  folderContent.Folders,
		"Files":    folderContent.Files,
	})
}

// NewFolder implements domain.WebService.
func (svc *WebService) NewFolder(c *gin.Context) {
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

	_, err := svc.client.NewFolder(user_id, parentID, name, description)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to create folder", err))
		return
	}

	// reload the page
	c.Redirect(302, "/web/folder/"+parentID)
}

// MoveFolder implements domain.WebService.
// Subtle: this method shadows the method (FolderService).MoveFolder of WebService.FolderService.
func (svc *WebService) MoveFolder(c *gin.Context) {
	user_id := c.GetString("owner_id")

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

	err := svc.client.MoveFolder(user_id, folder_id, new_parent_id)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to move folder", err))
		return
	}

	// reload the page
	c.Redirect(302, "/web/folder/"+new_parent_id)
}

// RenameFolder implements domain.WebService.
// Subtle: this method shadows the method (FolderService).RenameFolder of WebService.FolderService.
func (svc *WebService) RenameFolder(c *gin.Context) {
	user_id := c.GetString("owner_id")

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
	err := svc.client.RenameFolder(user_id, folder_id, new_name)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to rename folder", err))
		return
	}

	// reload the page
	c.Redirect(302, "/web/folder/"+folder_id)
}

// DeleteFolder implements domain.WebService.
func (svc *WebService) DeleteFolder(c *gin.Context) {
	folderID := c.Param("folder_id")
	if folderID == "" {
		c.AbortWithStatusJSON(400, response.Error("folder_id is required", ""))
		return
	}

	_, err := svc.client.DeleteFolder(folderID)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to delete folder", err))
		return
	}

	// HTMX requests: return empty body so hx-swap="outerHTML" removes the element
	if c.GetHeader("HX-Request") == "true" {
		c.String(200, "")
		return
	}

	c.JSON(200, response.Success("folder deleted"))
}

// DeleteFolderPermanently implements domain.WebService.
func (svc *WebService) DeleteFolderPermanently(c *gin.Context) {
	user_id := c.GetString("owner_id")

	folderID := c.Param("folder_id")
	if folderID == "" {
		c.AbortWithStatusJSON(400, response.Error("folder_id is required", ""))
		return
	}

	_, err := svc.client.DeleteFolderPermanently(user_id, folderID)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to delete folder", err))
		return
	}

	// HTMX requests: return empty body so hx-swap="outerHTML" removes the element
	if c.GetHeader("HX-Request") == "true" {
		c.String(200, "")
		return
	}

	c.JSON(200, response.Success("folder deleted"))

}

// UploadFile implements domain.WebService.
func (svc *WebService) UploadFile(c *gin.Context) {
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
		fileName, fileByte, err := fileutil.ProcessFile(file)
		if err != nil {
			c.AbortWithStatusJSON(500, response.WrapError("failed to process file", err))
			return
		}

		_, err = svc.client.UploadFile(user_id, folderID, fileName, file.Header.Get("Content-Type"), fileByte)
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
func (svc *WebService) DownloadFile(c *gin.Context) {
	// get the file_id from the request
	fileID := c.Param("file_id")
	if fileID == "" {
		c.AbortWithStatusJSON(400, response.Error("file_id is required", ""))
		return
	}

	// get the file
	file, err := svc.client.GetFile(fileID)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to get file", err))
		return
	}

	// serve the file
	c.Header("Content-Disposition", fileutil.SafeContentDisposition("attachment", file.Name))
	c.Header("Content-Type", file.ContentType)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(200, file.ContentType, file.Data)
}

// MoveFile implements domain.WebService.
func (svc *WebService) MoveFile(c *gin.Context) {
	fileID := c.Param("file_id")
	if fileID == "" {
		c.AbortWithStatusJSON(400, response.Error("file_id is required", ""))
		return
	}

	newParentID := c.PostForm("new_parent_id")
	if newParentID == "" {
		c.AbortWithStatusJSON(400, response.Error("new_parent_id is required", ""))
		return
	}

	if err := svc.client.MoveFile(fileID, newParentID); err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to move file", err))
		return
	}

	// Redirect to the target folder after move
	c.Redirect(302, "/web/folder/"+newParentID)
}

// DeleteFile implements domain.WebService.
// Subtle: this method shadows the method (FileService).DeleteFile of WebService.FileService.
func (svc *WebService) DeleteFile(c *gin.Context) {
	fileID := c.Param("file_id")
	if fileID == "" {
		c.AbortWithStatusJSON(400, response.Error("file_id is required", ""))
		return
	}

	_, err := svc.client.DeleteFile(fileID)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to delete file", err))
		return
	}

	// HTMX: return empty body so the element is removed
	if c.GetHeader("HX-Request") == "true" {
		c.String(200, "")
		return
	}

	c.JSON(200, response.Success("file deleted"))
}

func (svc *WebService) DeleteFilePermanently(c *gin.Context) {
	fileID := c.Param("file_id")
	if fileID == "" {
		c.AbortWithStatusJSON(400, response.Error("file_id is required", ""))
		return
	}

	_, err := svc.client.DeleteFilePermanently(fileID)
	if err != nil {
		c.AbortWithStatusJSON(500, response.WrapError("failed to delete file", err))
		return
	}

	// HTMX requests: return empty body so hx-swap="outerHTML" removes the element
	if c.GetHeader("HX-Request") == "true" {
		c.String(200, "")
		return
	}

	c.JSON(200, response.Success("file deleted"))
}
