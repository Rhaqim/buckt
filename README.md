# Buckt Package Documentation

The Buckt package provides a flexible storage service with optional integration for the Gin Gonic router. It enables you to manage and organize data using a robust and customizable file storage interface. You can configure it to log to files or the terminal, interact with an SQLite database, and access its services directly or via HTTP endpoints.

## Table of Contents

- [Buckt Package Documentation](#buckt-package-documentation)
  - [Table of Contents](#table-of-contents)
  - [Installation](#installation)
  - [Usage](#usage)
    - [Initialization](#initialization)
    - [Configuration](#configuration)
    - [Logging](#logging)
    - [Database](#database)
  - [Services](#services)
    - [File Storage](#file-storage)
    - [HTTP Server](#http-server)
  - [Examples](#examples)
    - [Basic Usage](#basic-usage)
    - [Using with Gin Router](#using-with-gin-router)
  - [License](#license)

## Installation

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
    bucktInstance, err := buckt.NewBuckt("config.yaml", true, "/path/to/logs")
    if err != nil {
        log.Fatal(err)
    }

    // Use bucktInstance for your operations...
}
```

### Configuration

The configuration file is a YAML file that defines the settings for the Buckt package. It should contain the following fields:

```yaml
log:
  logToFileAndTerminal: true
  level: "debug"
  saveDir: 

database:
  dsn: "db.sqlite"

server:
  host: "localhost"
  port: 8080

media:
  dir: "media"

endpoint:
  url: "http://localhost:8080"
```

### Logging

The Buckt package supports logging to files and the terminal. You can configure the logging settings in the configuration file. The log level can be set to "debug", "info", "warn", "error", or "fatal".

### Database

The Buckt package can interact with an SQLite database. You can configure the database settings in the configuration file. The DSN field should contain the path to the SQLite database file.

## Services

### File Storage

The Buckt package provides a file storage service that allows you to manage and organize data using a robust and customizable interface. You can configure it to log to files or the terminal, interact with an SQLite database, and access its services directly or via HTTP endpoints.

### HTTP Server

The Buckt package includes an HTTP server that exposes its services via HTTP endpoints. You can configure the server settings in the configuration file. The host and port fields should contain the address and port for the HTTP server.

## Examples

### Basic Usage

```go
import "github.com/Rhaqim/buckt"    

func main() {
    bucktInstance, err := buckt.NewBuckt("config.yaml", true, "/path/to/logs")
    if err != nil {
        log.Fatal(err)
    }

    // Use bucktInstance for your operations...
}
```

### Using with Gin Router

The Buckt package can be integrated with the Gin Gonic router to expose its services via HTTP endpoints. You can configure the server settings in the configuration file. The host and port fields should contain the address and port for the HTTP server.

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/Rhaqim/buckt"
)

func main() {
    bucktInstance, err := buckt.NewBuckt("config.yaml", true, "/path/to/logs")
    if err != nil {
        log.Fatal(err)
    }

    router := gin.Default()
    bucktInstance.UseWithGin(router)

    router.Run()
}
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
