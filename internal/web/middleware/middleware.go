package middleware

import (
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/gin-gonic/gin"
)

type bucketMiddleware struct {
}

func NewBucketMiddleware() domain.Middleware {
	return &bucketMiddleware{}
}

func (b *bucketMiddleware) ClientTypeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientType := c.GetHeader("X-Client-Type")
		if clientType == "" {
			clientType = "portal"
		}
		c.Set("clientType", clientType)
		c.Next()
	}
}

func (b *bucketMiddleware) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the user is authenticated
		// If not, return a 401 Unauthorized response

		// temp
		// ownerName := ""

		// owner, err := b.ownerStore.GetBy("name", ownerName)
		// if err != nil {
		// 	c.JSON(401, gin.H{"error": "Unauthorized"})
		// 	c.Abort()
		// 	return
		// }

		c.Set("user_id", "1234")

		c.Next()
	}
}
