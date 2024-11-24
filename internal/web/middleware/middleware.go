package middleware

import "github.com/gin-gonic/gin"

func ClientTypeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientType := c.GetHeader("X-Client-Type")
		if clientType == "" {
			clientType = "portal"
		}
		c.Set("clientType", clientType)
		c.Next()
	}
}
