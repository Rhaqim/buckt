# Buckt Package Documentation

The Buckt package provides a flexible media storage service with optional integration for the Gin Gonic router. It enables you to manage and organize data using a robust and customizable file storage interface. You can configure it to log to files or the terminal, interact with an SQLite database, and access its services directly or via HTTP endpoints.

[![Go Report Card](https://goreportcard.com/badge/github.com/Rhaqim/buckt)](https://goreportcard.com/report/github.com/Rhaqim/buckt)
[![GoDoc](https://godoc.org/github.com/Rhaqim/buckt?status.svg)](https://pkg.go.dev/github.com/Rhaqim/buckt)
[![License](https://img.shields.io/github/license/Rhaqim/buckt)](LICENSE)

## Table of Contents

- [Buckt Package Documentation](#buckt-package-documentation)
  - [Table of Contents](#table-of-contents)
  - [Features](#features)
  - [Getting Started](#getting-started)
  - [Usage](#usage)
    - [Initialization](#initialization)
    - [Configuration](#configuration)
    - [Logging](#logging)
    - [Database](#database)
  - [Services](#services)
    - [Direct Services](#direct-services)
    - [Gin Server](#gin-server)
  - [Examples](#examples)
    - [With Built-in Gin Server](#with-built-in-gin-server)
    - [Using with Other Routers](#using-with-other-routers)
  - [License](#license)
  
## Features

- **Flexible Storage** – Store and manage media files with ease.
- **Gin Gonic Integration** – Use with Gin for HTTP-based file management.
- **Logging Support** – Log to files or the terminal for better debugging.
- **SQLite Support** – Store metadata in an SQLite database.
- **Direct & HTTP Access** – Interact with Buckt programmatically or via API.

## Getting Started

You can install the package using go get:

```bash
go get github.com/Rhaqim/buckt
```

## Usage

### Initialization

To create a new instance of the Buckt package, use the NewBuckt function. It requires a configuration file and optional parameters for logging.

```go
import "github.com/Rhaqim/buckt"

func main() {
    bucktInstance, err := buckt.NewBuckt("config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    defer bucktInstance.Close()

    // Use bucktInstance for your operations...
}
```

### Configuration

The configuration file is a YAML file that defines the settings for the Buckt package. It should contain the following fields:

```yaml
log:
  level: "debug"
  logTerminal: true
  logFile: "log.txt"

database:
  dsn: "db.sqlite"

server:
  host: "localhost"
  port: 8000

mediaDir: "media"

templatesDir: "templates"
```

### Logging

The Buckt package supports logging to files and the terminal. By default the package would log to terminal. You can configure the logging settings in the configuration file. Provide the logTerminal field with a boolean value to enable or disable logging to the terminal. The logFile field should contain the path to the log file.

### Database

The Buckt package interacts with an SQLite database. You can configure the database settings in the configuration file. The DSN field should contain the path to the SQLite database file. By default one is created in the root directory of the project called db.sqlite.

## Services

### Direct Services

The Buckt package exposes the services directly via the Buckt interface. You can use the services to manage and organize data using a robust and customizable interface.

### Gin Server

The Buckt package includes an HTTP server that exposes its services via HTTP endpoints. You can configure the server settings in the configuration file. The host and port fields should contain the address and port for the HTTP server. Alternatively you can use the **GetHandler** method to get the handler and use it with your own router.

## Examples

### With Built-in Gin Server

```go
import (
    "log"

    "github.com/Rhaqim/buckt"
  )   

func main() {
    // Create a new instance of the Buckt package
    bucktInstance, err := buckt.NewBuckt("config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    defer bucktInstance.Close() // Close the Buckt instance when done

    /// Start the router (optional, based on user choice)
    if err := b.StartServer(":8080"); err != nil {
      log.Fatalf("Failed to start Buckt: %v", err)
    }
}
```

### Using with Other Routers

The Buckt package can be integrated with other routers, such as Fiber, Echo, or Chi. You can use the GetHandler method to get the handler and mount it under a specific route.

```go
import (
    "log"

    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/adaptor/v2"
    "github.com/Rhaqim/buckt"
  )

func main() {
    bucktInstance, err := buckt.NewBuckt("config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    defer bucketInstance.Close()

    // Initalise a new fiber mux
    app := fiber.New()

    // Get the handler for the Buckt instance
    handler := bucktInstance.GetHandler()

    // Mount the Buckt router under /api using Fiber's adaptor
    app.Use("/api", adaptor.HTTPHandler(handler))

    // Add additional routes directly in Fiber
    app.Get("/", func(c *fiber.Ctx) error {
      return c.SendString("Welcome to the main application!")
    })

    // Start the Fiber server
    log.Println("Server is running on http://localhost:8080")
    if err := app.Listen(":8080"); err != nil {
      log.Fatalf("Server failed: %v", err)
    }
}
```

More examples can be found in the [examples](example/) directory.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
