package buckt_web_testing

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/Rhaqim/buckt"
	"github.com/Rhaqim/buckt/client/web"
)

func main() {
	client, err := buckt.Default(buckt.FlatNameSpaces(true))
	if err != nil {
		log.Fatalf("Failed to initialize Buckt: %v", err)
	}
	defer client.Close() // Ensure resources are cleaned up

	config := web.Config{
		Mode:  web.WebModeMount,
		Debug: true,
	}

	webClient, err := web.NewClient(client, config)
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Allow overriding via command-line flag
	flagPort := flag.String("port", port, "Port to run the server on")
	flag.Parse()

	// Start the server
	server := &http.Server{
		Addr:    ":" + *flagPort,
		Handler: mux,
	}

	log.Println("Server is running on http://localhost:8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
