package middleware

import (
	"fmt"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/gin-gonic/gin"
)

type bucketMiddleware struct {
	*logger.BucktLogger
	Standalone bool
}

func NewBucketMiddleware(bucktLog *logger.BucktLogger, standalone bool) domain.Middleware {
	return &bucketMiddleware{
		BucktLogger: bucktLog,
		Standalone:  standalone,
	}
}

func (b *bucketMiddleware) APIGuardMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !b.Standalone {
			// Read user_id from header
			userID := c.GetHeader("buckt-User-ID")
			if userID == "" {
				c.AbortWithStatusJSON(401, b.WrapError("unauthorised", fmt.Errorf("user_id not found in headers")))
				return
			}

			// Set in Gin context for further use
			c.Set("owner_id", userID)

			c.Next()
		}

		c.Set("owner_id", "default")

		c.Next()
	}
}

// WebGuardMiddleware implements domain.Middleware.
func (b *bucketMiddleware) WebGuardMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Set("owner_id", "default")

		c.Next()
	}
}
