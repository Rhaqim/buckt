package app

import (
	"github.com/Rhaqim/buckt/internal/domain"
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
		"Path":    folderContent.Path,
		"Folders": folderContent.Folders,
		"Files":   folderContent.Files,
	})
}
