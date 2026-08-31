package domain

import "github.com/gin-gonic/gin"

type APIService interface {
	CreateFolder(c *gin.Context)
	GetFolderContent(c *gin.Context)
	RenameFolder(c *gin.Context)
	MoveFolder(c *gin.Context)
	DeleteFolder(c *gin.Context)
	DeleteFolderPermanently(c *gin.Context)

	UploadFile(c *gin.Context)
	DownloadFile(c *gin.Context)
	ServeFile(c *gin.Context)
	ServeDerivative(c *gin.Context)
	StreamFile(c *gin.Context)
	DeleteFile(c *gin.Context)
	DeleteFilePermanently(c *gin.Context)

	// Presign returns a time-limited direct-download URL for a file.
	Presign(c *gin.Context)

	// SetExpiry sets or clears a file's automatic-deletion time (ttl or absolute at).
	SetExpiry(c *gin.Context)
	// PurgeExpired permanently deletes every file whose expiry has passed.
	PurgeExpired(c *gin.Context)

	// Metrics reports backend operation counts, storage bytes, and cache stats.
	Metrics(c *gin.Context)

	// Backend reports the active storage backend and migration progress.
	Backend(c *gin.Context)

	// TODO: Might not be needed
	GetFilesInFolder(c *gin.Context)
	GetSubFolders(c *gin.Context)
	GetDescendants(c *gin.Context)
}
