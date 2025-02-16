package domain

import "github.com/gin-gonic/gin"

type Middleware interface {
	AuthMiddleware() gin.HandlerFunc
	ClientTypeMiddleware() gin.HandlerFunc
}

type APIService interface {
	UploadFile(c *gin.Context)
	DownloadFile(c *gin.Context)
	ServeFile(c *gin.Context)
	DeleteFile(c *gin.Context)
	CreateFolder(c *gin.Context)
	RenameFolder(c *gin.Context)
	MoveFolder(c *gin.Context)
	DeleteFolder(c *gin.Context)
	GetFolderContent(c *gin.Context)
	GetFilesInFolder(c *gin.Context)
	GetSubFolders(c *gin.Context)
	GetDescendants(c *gin.Context)
}
