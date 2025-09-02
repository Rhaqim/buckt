package buckt_web_testing

import (
	"log"
	"net/http"

	"github.com/Rhaqim/buckt"
	"github.com/Rhaqim/buckt/client/web"
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

	config := web.ClientConfig{
		Mode:  web.WebModeMount,
		Debug: true,
	}

	webClient, err := web.NewClient(b, config)
	if err != nil {
		log.Fatalf("Failed to create web client: %v", err)
	}

	// Create a custom multiplexer
	mux := http.NewServeMux()

	// Mount the Buckt router under /buckt
	mux.Handle("/buckt/", http.StripPrefix("/buckt", webClient.Handler()))

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
