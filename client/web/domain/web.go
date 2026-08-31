package domain

import "github.com/gin-gonic/gin"

type WebService interface {
	ViewFolder(c *gin.Context)
	ViewTrash(c *gin.Context)
	RestoreFile(c *gin.Context)
	RestoreFolder(c *gin.Context)
	NewFolder(c *gin.Context)
	RenameFolder(c *gin.Context)
	MoveFolder(c *gin.Context)
	DeleteFolder(c *gin.Context)
	DeleteFolderPermanently(c *gin.Context)

	UploadFile(c *gin.Context)
	DownloadFile(c *gin.Context)
	RegenerateDerivatives(c *gin.Context)
	MoveFile(c *gin.Context)
	DeleteFile(c *gin.Context)
	DeleteFilePermanently(c *gin.Context)

	// SetTTL sets or clears a file's expiry from the UI.
	SetTTL(c *gin.Context)
	// PurgeExpired permanently deletes every file whose expiry has passed.
	PurgeExpired(c *gin.Context)
}
