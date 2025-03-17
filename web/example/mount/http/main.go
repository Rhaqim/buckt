package main

import (
	"log"
	"net/http"

	"github.com/Rhaqim/buckt"
	_ "github.com/Rhaqim/buckt/web"
)

func main() {
	opts := buckt.BucktConfig{
		Log: buckt.LogConfig{
			LogTerminal: true,
		},
		MediaDir: "media",
	}

	b, err := buckt.New(opts)
	if err != nil {
		log.Fatalf("Failed to initialize Buckt: %v", err)
	}
	defer b.Close() // Ensure resources are cleaned up

	// Initialize router
	err = b.InitRouterService(buckt.WebModeMount)
	if err != nil {
		log.Fatalf("Failed to initialize Buckt router: %v", err)
	}

	handler := b.GetHandler()

	// Create a custom multiplexer
	mux := http.NewServeMux()

	// Mount the Buckt router under /buckt
	mux.Handle("/buckt/", http.StripPrefix("/buckt", handler))

	// Add additional routes
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to the main application!"))
	})

	// Start the server
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Server is running on http://localhost:8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
