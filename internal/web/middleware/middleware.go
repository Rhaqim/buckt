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
		// get the user_id from the context
		user_id, ok := c.Get("user_id")
		if !ok {
			c.AbortWithStatusJSON(401, response.WrapError("unauthorised", fmt.Errorf("user_id not found")))
			return
		}

		// convert the user_id to string
		userID, ok := user_id.(string)
		if !ok {
			c.AbortWithStatusJSON(401, response.WrapError("unathorised", fmt.Errorf("failed to convert user_id to string")))
		}

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
