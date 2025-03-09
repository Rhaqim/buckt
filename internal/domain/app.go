package domain

import "github.com/gin-gonic/gin"

type Middleware interface {
	APIGuardMiddleware() gin.HandlerFunc
	WebGuardMiddleware() gin.HandlerFunc
}

type APIService interface {
	UploadFile(c *gin.Context)
	DownloadFile(c *gin.Context)
	ServeFile(c *gin.Context)
	DeleteFile(c *gin.Context)

	CreateFolder(c *gin.Context)
	GetFolderContent(c *gin.Context)
	RenameFolder(c *gin.Context)
	MoveFolder(c *gin.Context)
	DeleteFolder(c *gin.Context)
	DeleteFilePermanently(c *gin.Context)

	// TODO: Might not be needed
	GetFilesInFolder(c *gin.Context)
	GetSubFolders(c *gin.Context)
	GetDescendants(c *gin.Context)
}

type WebService interface {
	ViewFolder(c *gin.Context)
	NewFolder(c *gin.Context)
	RenameFolder(c *gin.Context)
	MoveFolder(c *gin.Context)
	DeleteFolder(c *gin.Context)

	UploadFile(c *gin.Context)
	DownloadFile(c *gin.Context)
	MoveFile(c *gin.Context)
	DeleteFile(c *gin.Context)
	DeleteFilePermanently(c *gin.Context)
}
