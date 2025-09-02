package middleware

import (
	"fmt"

	"github.com/Rhaqim/buckt/pkg/logger"
	webDomain "github.com/Rhaqim/buckt/web/domain"
	"github.com/gin-gonic/gin"
)

type bucketMiddleware struct {
	logger  *logger.BucktLogger
	mounted bool
}

func NewBucketMiddleware(bucktLog *logger.BucktLogger, mounted bool) webDomain.Middleware {
	return &bucketMiddleware{
		logger:  bucktLog,
		mounted: mounted,
	}
}

func (b *bucketMiddleware) APIGuardMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !b.mounted {
			// Read user_id from header
			userID := c.GetHeader("buckt-User-ID")
			if userID == "" {
				c.AbortWithStatusJSON(401, b.logger.WrapError("unauthorised", fmt.Errorf("user_id not found in headers")))
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
