package domain

import "github.com/gin-gonic/gin"

type WebService interface {
	ViewFolder(c *gin.Context)
	NewFolder(c *gin.Context)
	RenameFolder(c *gin.Context)
	MoveFolder(c *gin.Context)
	DeleteFolder(c *gin.Context)
	DeleteFolderPermanently(c *gin.Context)

	UploadFile(c *gin.Context)
	DownloadFile(c *gin.Context)
	MoveFile(c *gin.Context)
	DeleteFile(c *gin.Context)
	DeleteFilePermanently(c *gin.Context)
}
