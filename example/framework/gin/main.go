package main

import (
	"sync"

	_ "github.com/lib/pq"

	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/Rhaqim/buckt"
	"github.com/gin-gonic/gin"
)

func main() {

	var err error
	var db *sql.DB

	// Postgres database
	conn_string := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		"localhost", 5432, "postgres", "password", "postgres")

	db, err = sql.Open("postgres", conn_string)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	// File cache
	cache := NewCache()

	// Initialize Buckt
	opts := buckt.BucktConfig{
		DB: buckt.DBConfig{
			Driver:   buckt.Postgres,
			Database: db,
		}, // Pass the database connection
		Cache: cache,
		Log: buckt.LogConfig{
			LogTerminal: false,
			LoGfILE:     "logs",
			Debug:       true,
		},
		MediaDir:       "media",
		StandaloneMode: false,
	}

	b, err := buckt.New(opts)
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

type Cache struct {
	// Cache
	mu    sync.RWMutex
	store map[string]any
}

func NewCache() *Cache {
	return &Cache{
		store: make(map[string]any),
	}
}

func (c *Cache) Get(key string) (any, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	fmt.Println("Cache get", key)

	if val, ok := c.store[key]; ok {

		fmt.Println("Cache hit", key)

		return val, nil
	}

	return nil, nil
}

func (c *Cache) Set(key string, value any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	fmt.Println("Cache set", key)

	c.store[key] = value
	return nil
}

func (c *Cache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.store, key)
	return nil
}
