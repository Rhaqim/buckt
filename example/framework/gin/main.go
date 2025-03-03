package main

import (
	"log"
	"strings"

	"github.com/Rhaqim/buckt"
	"github.com/gin-gonic/gin"
)

func main() {

	// Initialize Buckt
	opts := buckt.BucktOptions{
		Log: buckt.Log{
			LogTerminal: false,
			LoGfILE:     "buckt.log",
			Debug:       true,
		},
		MediaDir:       "media",
		StandaloneMode: false,
	}

	b, err := buckt.NewBuckt(opts)
	if err != nil {
		log.Fatalf("Failed to initialize Buckt: %v", err)
	}
	defer b.Close() // Ensure resources are cleaned up

	// Get the Buckt handler
	handler := b.GetHandler()

	r := gin.Default()

	// Add additional routes
	r.GET("/", func(c *gin.Context) {
		c.String(200, "Welcome to the main application!")
	})

	r.POST("/post", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "POST request received!",
		})
	})

	// Mount the Buckt router under /buckt
	bucktRouter := r.Group("/buckt")
	bucktRouter.Use(func(c *gin.Context) {

		// Attach user_id to headers before forwarding
		c.Request.Header.Set("buckt-User-ID", "1234")

		c.Next()
	})
	{

		bucktRouter.Any("/*path", func(c *gin.Context) {
			// Trim `/buckt` from the request path
			proxyPath := strings.TrimPrefix(c.Request.URL.Path, "/buckt")

			// Update the request path
			c.Request.URL.Path = proxyPath

			// Forward the request to the handler
			handler.ServeHTTP(c.Writer, c.Request)
		})
	}

	// Start the server
	r.Run(":8080")
}
