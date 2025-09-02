package middleware

import (
	"log"

	"github.com/Rhaqim/buckt/web/domain"
	"github.com/gin-gonic/gin"
)

type bucketMiddleware struct {
	logger  *log.Logger
	mounted bool
}

func NewBucketMiddleware(bucktLog *log.Logger, mounted bool) domain.Middleware {
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
				b.logger.Printf("unauthorised: user_id not found in headers")
				c.AbortWithStatusJSON(401, gin.H{"error": "unauthorised", "message": "user_id not found in headers"})
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
