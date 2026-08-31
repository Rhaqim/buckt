package app

import (
	"time"

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

	// folder_id is optional here — empty means the user's root folder.
	folderID := c.Param("folder_id")

	folderContent, err := svc.client.GetFolderWithContent(user_id, folderID)
	if err != nil {
		abort500(c, "failed to get folder content", err)
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

// ViewTrash renders the user's trash folder — the files and folders that were
// moved to trash (soft-deleted) — with restore and permanent-delete actions.
func (svc *WebService) ViewTrash(c *gin.Context) {
	user_id := c.GetString("owner_id")

	trash, err := svc.client.GetTrashFolder(user_id)
	if err != nil {
		abort500(c, "failed to get trash", err)
		return
	}

	// Rendered by dashboard.html (IsTrash branch). A single page template owns
	// the "body"/"toolbar" slots — a separate trash page defining them again
	// would collide in the shared template set and hijack every page.
	c.HTML(200, "dashboard.html", gin.H{
		"Title":   "Trash",
		"page":    "trash",
		"IsTrash": true,
		"ID":      trash.ID,
		"Name":    trash.Name,
		"Folders": trash.Folders,
		"Files":   trash.Files,
	})
}

// RestoreFile moves a trashed file back to its original location.
func (svc *WebService) RestoreFile(c *gin.Context) {
	user_id := c.GetString("owner_id")

	fileID, ok := requireParam(c, "file_id")
	if !ok {
		return
	}

	if err := svc.client.RestoreFile(user_id, fileID); err != nil {
		abort500(c, "failed to restore file", err)
		return
	}

	if htmxHandled(c) {
		return
	}
	c.Redirect(302, "/web/trash")
}

// RestoreFolder moves a trashed folder back to its original location.
func (svc *WebService) RestoreFolder(c *gin.Context) {
	user_id := c.GetString("owner_id")

	folderID, ok := requireParam(c, "folder_id")
	if !ok {
		return
	}

	if err := svc.client.RestoreFolder(user_id, folderID); err != nil {
		abort500(c, "failed to restore folder", err)
		return
	}

	if htmxHandled(c) {
		return
	}
	c.Redirect(302, "/web/trash")
}

// NewFolder implements domain.WebService.
func (svc *WebService) NewFolder(c *gin.Context) {
	user_id := c.GetString("owner_id")

	parentID, ok := requireForm(c, "parent_id")
	if !ok {
		return
	}
	name, ok := requireForm(c, "name")
	if !ok {
		return
	}

	if _, err := svc.client.NewFolder(user_id, parentID, name, c.PostForm("description")); err != nil {
		abort500(c, "failed to create folder", err)
		return
	}

	c.Redirect(302, "/web/folder/"+parentID)
}

// MoveFolder implements domain.WebService.
// Subtle: this method shadows the method (FolderService).MoveFolder of WebService.FolderService.
func (svc *WebService) MoveFolder(c *gin.Context) {
	user_id := c.GetString("owner_id")

	folderID, ok := requireForm(c, "folder_id")
	if !ok {
		return
	}
	newParentID, ok := requireForm(c, "new_parent_id")
	if !ok {
		return
	}

	if err := svc.client.MoveFolder(user_id, folderID, newParentID); err != nil {
		abort500(c, "failed to move folder", err)
		return
	}

	c.Redirect(302, "/web/folder/"+newParentID)
}

// RenameFolder implements domain.WebService.
// Subtle: this method shadows the method (FolderService).RenameFolder of WebService.FolderService.
func (svc *WebService) RenameFolder(c *gin.Context) {
	user_id := c.GetString("owner_id")

	folderID, ok := requireForm(c, "folder_id")
	if !ok {
		return
	}
	newName, ok := requireForm(c, "new_name")
	if !ok {
		return
	}

	if err := svc.client.RenameFolder(user_id, folderID, newName); err != nil {
		abort500(c, "failed to rename folder", err)
		return
	}

	c.Redirect(302, "/web/folder/"+folderID)
}

// DeleteFolder implements domain.WebService.
func (svc *WebService) DeleteFolder(c *gin.Context) {
	folderID, ok := requireParam(c, "folder_id")
	if !ok {
		return
	}

	if _, err := svc.client.DeleteFolder(folderID); err != nil {
		abort500(c, "failed to delete folder", err)
		return
	}

	if htmxHandled(c) {
		return
	}
	c.JSON(200, response.Success("folder deleted"))
}

// DeleteFolderPermanently implements domain.WebService.
func (svc *WebService) DeleteFolderPermanently(c *gin.Context) {
	user_id := c.GetString("owner_id")

	folderID, ok := requireParam(c, "folder_id")
	if !ok {
		return
	}

	if _, err := svc.client.DeleteFolderPermanently(user_id, folderID); err != nil {
		abort500(c, "failed to delete folder", err)
		return
	}

	if htmxHandled(c) {
		return
	}
	c.JSON(200, response.Success("folder deleted"))
}

// UploadFile implements domain.WebService.
func (svc *WebService) UploadFile(c *gin.Context) {
	user_id := c.GetString("owner_id")

	folderID, ok := requireForm(c, "folder_id")
	if !ok {
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.AbortWithStatusJSON(400, response.Error("invalid form data", err.Error()))
		return
	}

	// Optional ttl (a Go duration like "24h") makes every file in this upload a
	// temp file that expires. Invalid values are ignored (treated as permanent).
	var ttl time.Duration
	if ttlStr := c.PostForm("ttl"); ttlStr != "" {
		ttl, _ = time.ParseDuration(ttlStr)
	}

	for _, file := range form.File["files"] {
		fileName, fileByte, err := fileutil.ProcessFileWithLimit(file, svc.client.MaxFileSize())
		if err != nil {
			abort500(c, "failed to process file", err)
			return
		}

		contentType := file.Header.Get("Content-Type")
		if ttl > 0 {
			_, err = svc.client.UploadFileWithTTL(user_id, folderID, fileName, contentType, fileByte, ttl)
		} else {
			_, err = svc.client.UploadFile(user_id, folderID, fileName, contentType, fileByte)
		}
		if err != nil {
			abort500(c, "failed to create file", err)
			return
		}
	}

	c.Redirect(302, "/web/folder/"+folderID)
}

// DownloadFile implements domain.WebService.
func (svc *WebService) DownloadFile(c *gin.Context) {
	fileID, ok := requireParam(c, "file_id")
	if !ok {
		return
	}

	file, err := svc.client.GetFile(fileID)
	if err != nil {
		abort500(c, "failed to get file", err)
		return
	}

	writeFileHeaders(c, file.Name, file.ContentType, file.Size, "attachment")
	c.Data(200, file.ContentType, file.Data)
}

// RegenerateDerivatives rebuilds the configured image variants for a file. It's
// a no-op if derivatives aren't configured or the file isn't an image.
func (svc *WebService) RegenerateDerivatives(c *gin.Context) {
	fileID, ok := requireParam(c, "file_id")
	if !ok {
		return
	}

	if err := svc.client.GenerateDerivatives(fileID); err != nil {
		abort500(c, "failed to regenerate derivatives", err)
		return
	}

	c.JSON(200, response.Success("derivatives regenerated"))
}

// SetTTL sets (or clears) a file's expiry from the UI. The `ttl` form value is a
// Go duration (e.g. "1h", "24h"); an empty value, "0", or "clear" removes the
// expiry. Answers HTMX with an HX-Refresh so the card re-renders with its new
// expiry badge; redirects non-HTMX callers back to the folder.
func (svc *WebService) SetTTL(c *gin.Context) {
	fileID, ok := requireParam(c, "file_id")
	if !ok {
		return
	}

	ttlStr := c.PostForm("ttl")
	switch ttlStr {
	case "", "0", "clear":
		if err := svc.client.SetFileExpiry(fileID, time.Time{}); err != nil {
			abort500(c, "failed to clear expiry", err)
			return
		}
	default:
		ttl, err := time.ParseDuration(ttlStr)
		if err != nil {
			c.AbortWithStatusJSON(400, response.Error("invalid ttl (use a Go duration like 24h)", err.Error()))
			return
		}
		if err := svc.client.SetFileTTL(fileID, ttl); err != nil {
			abort500(c, "failed to set expiry", err)
			return
		}
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Refresh", "true")
		c.String(200, "")
		return
	}
	c.Redirect(302, c.GetHeader("Referer"))
}

// PurgeExpired permanently deletes every file whose expiry has passed, then
// refreshes the view (HTMX) or redirects back.
func (svc *WebService) PurgeExpired(c *gin.Context) {
	if _, err := svc.client.PurgeExpired(c.Request.Context()); err != nil {
		abort500(c, "failed to purge expired files", err)
		return
	}
	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Refresh", "true")
		c.String(200, "")
		return
	}
	c.Redirect(302, c.GetHeader("Referer"))
}

// MoveFile implements domain.WebService.
func (svc *WebService) MoveFile(c *gin.Context) {
	fileID, ok := requireParam(c, "file_id")
	if !ok {
		return
	}
	newParentID, ok := requireForm(c, "new_parent_id")
	if !ok {
		return
	}

	if err := svc.client.MoveFile(fileID, newParentID); err != nil {
		abort500(c, "failed to move file", err)
		return
	}

	c.Redirect(302, "/web/folder/"+newParentID)
}

// DeleteFile implements domain.WebService.
// Subtle: this method shadows the method (FileService).DeleteFile of WebService.FileService.
func (svc *WebService) DeleteFile(c *gin.Context) {
	fileID, ok := requireParam(c, "file_id")
	if !ok {
		return
	}

	if _, err := svc.client.DeleteFile(fileID); err != nil {
		abort500(c, "failed to delete file", err)
		return
	}

	if htmxHandled(c) {
		return
	}
	c.JSON(200, response.Success("file deleted"))
}

func (svc *WebService) DeleteFilePermanently(c *gin.Context) {
	fileID, ok := requireParam(c, "file_id")
	if !ok {
		return
	}

	if _, err := svc.client.DeleteFilePermanently(fileID); err != nil {
		abort500(c, "failed to delete file", err)
		return
	}

	if htmxHandled(c) {
		return
	}
	c.JSON(200, response.Success("file deleted"))
}
