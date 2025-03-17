package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/Rhaqim/buckt"
	_ "github.com/Rhaqim/buckt/web"
)

func main() {
	b, err := buckt.Default(buckt.FlatNameSpaces(true), buckt.WithCache(NewCache()))
	if err != nil {
		log.Fatalf("Failed to initialize Buckt: %v", err)
	}
	defer b.Close() // Ensure resources are cleaned up

	// initialize router
	err = b.InitRouterService(buckt.WebModeAll)
	if err != nil {
		log.Fatalf("Failed to initialize Buckt router: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Allow overriding via command-line flag
	flagPort := flag.String("port", port, "Port to run the server on")
	flag.Parse()

	// Start the router (optional, based on user choice)
	if err := b.StartServer(":" + *flagPort); err != nil {
		log.Fatalf("Failed to start Buckt: %v", err)
	}
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

func (c *Cache) GetBucktValue(key string) (any, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	fmt.Println("Cache get", key)

	if val, ok := c.store[key]; ok {

		fmt.Println("Cache hit", key)

		return val, nil
	}

	return nil, nil
}

func (c *Cache) SetBucktValue(key string, value any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	fmt.Println("Cache set", key)

	c.store[key] = value
	return nil
}

func (c *Cache) DeleteBucktValue(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.store, key)
	return nil
}
