package middleware

import (
	"fmt"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/pkg/response"
	"github.com/gin-gonic/gin"
)

type bucketMiddleware struct{}

func NewBucketMiddleware() domain.Middleware {
	return &bucketMiddleware{}
}

func (b *bucketMiddleware) APIGuardMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read user_id from header
		userID := c.GetHeader("buckt-User-ID")
		if userID == "" {
			c.AbortWithStatusJSON(401, response.WrapError("unauthorised", fmt.Errorf("user_id not found in headers")))
			return
		}

		// Set in Gin context for further use
		c.Set("owner_id", userID)

		c.Next()
	}
}

// WebGuardMiddleware implements domain.Middleware.
func (b *bucketMiddleware) WebGuardMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Set("owner_id", "1234")

		c.Next()
	}
}
