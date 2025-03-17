package domain

import "github.com/gin-gonic/gin"

type Middleware interface {
	APIGuardMiddleware() gin.HandlerFunc
	WebGuardMiddleware() gin.HandlerFunc
}
