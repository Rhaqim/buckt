package domain

import (
	"github.com/gin-gonic/gin"
)

type StorageHTTPService interface {
	Upload(*gin.Context)
	Download(*gin.Context)
	ServeFile(*gin.Context)
	Delete(*gin.Context)
	NewUser(c *gin.Context)
	NewBucket(c *gin.Context)
}
