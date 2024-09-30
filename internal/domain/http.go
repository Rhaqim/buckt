package domain

import (
	"github.com/gin-gonic/gin"
)

type StorageHTTPService interface {
	Download(*gin.Context)
	Upload(*gin.Context)
	Delete(*gin.Context)
}
