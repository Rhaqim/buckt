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
    - [Configuration Options](#configuration-options)
    - [Database](#database)
    - [Caching](#caching)
    - [Logging](#logging)
  - [Initialization](#initialization)
  - [Services](#services)
    - [Direct Services](#direct-services)
    - [Web Server](#web-server)
    - [Cloud Backend](#cloud-backend)
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

### Configuration Options

The Config struct holds the configuration options for the Buckt package. It includes settings for the database, cache, logging, media directory, and flat namespaces.

```go
  type Config struct {
    DB    DBConfig
    Cache CacheConfig
    Log   LogConfig

    MediaDir       string
    FlatNameSpaces bool
}
```

### Database

By default Buckt uses an SQLite database to store metadata on the media files, optionally you can provide a custom database connection using the DB field in the Config struct. Gorm is used as the ORM to interact with the database. On instantiation it migrates the database schema for the `files` and `folders` tables.

The `files` table stores metadata information about the media files, such as the file name, size, and MIME type.

The `folders` table stores logical mappings of folders to put files in.

The DBConfig struct allows for BYODB (Bring Your Own Database) configurations. You can specify the database driver and provide a custom database connection. If not provided, a default SQLite connection will be used.

>Note: Parent application handles closing the database connection.

```go
  type DBConfig struct {
    Driver   DBDrivers
    Database *sql.DB
  }
```

### Caching

The Buckt package supports caching using the CacheManager interface. You can provide a custom cache manager by implementing the interface and passing it to the Cache field in the BucktOptions struct. The application also uses LRUCache for file caching.

The CacheManager interface allows the application to manage cached values.

```go
  type CacheManager interface {
    // Set a value in the cache for a given key.
    SetBucktValue(key string, value any) error

    // Get a value from the cache using a key.
    GetBucktValue(key string) (any, error)

    // Delete a key-value pair from the cache.
    DeleteBucktValue(key string) error
  }
```

### Logging

The Buckt package supports logging to files and the terminal. By default the package would log to terminal. You can configure the logging settings in the BucktOptions. Provide the logTerminal field with a boolean value to enable or disable logging to the terminal. The logFile field should contain the path to the log file.

The Log struct allows the parent application to configure logging options.

```go
  type Log struct {
    LogTerminal bool
    LogFile     string
    Debug       bool
  }
```

The rest of the configuration options are as follows:

- **MediaDir** – The path to the media directory on the file system.
- **FlatNameSpaces** – `true` for flat namespaces, `false` for hierarchical namespaces.

## Initialization

To create a new instance of the Buckt package, use the New or Default function. It requires the BucktConfig struct as an argument. The New function returns a new instance of the Buckt package, while the Default function initializes the package with default settings.

```go
import "github.com/Rhaqim/buckt"

func main() {
    // Create a new instance of the Buckt package
    client, err := buckt.Default()
    if err != nil {
        log.Fatal(err)
    }

    defer client.Close()

    // Use client for your operations...
}
```

## Services

### Direct Services

The Buckt package exposes the services directly via the Buckt interface. You can use the services to manage and organize data using a robust and customizable interface.

A detailed example for direct usage can be found in the [Direct Example](example/direct/main.go) directory.

### Web Server

The Buckt package includes an HTTP server that exposes its services via HTTP endpoints. You can configure the server settings in the configuration file. The host and port fields should contain the address and port for the HTTP server. Alternatively you can use the **GetHandler** method to get the handler and use it with your own router.

The Buckt package can be integrated with other routers, such as Fiber, Echo, Chi or Go's HTTP package . You can use the GetHandler method to get the handler and mount it under a specific route.

You can find a Postman collection with the API endpoints at [<img src="https://run.pstmn.io/button.svg" alt="Run In Postman" style="width: 128px; height: 32px;">](https://app.getpostman.com/run-collection/17061476-00806d0d-9584-4889-ade7-f8407932dba2?action=collection%2Ffork&source=rip_markdown&collection-url=entityId%3D17061476-00806d0d-9584-4889-ade7-f8407932dba2%26entityType%3Dcollection%26workspaceId%3D28697276-d953-482a-bd39-c4695366a55a)

More examples for router usage can be found in the [AWS Example](example/client/web/main.go) directory.

### Cloud Backend

The Buckt package can be integrated with cloud services like `Amazon S3`, `Google Cloud Storage`, or `Azure Blob Storage` as the backend storage solution. If you want to use a cloud service as the backend, you need to import the submodule for the specific cloud provider. If no backend is specified, Buckt will use the local file system as the default storage.

A detailed example for cloud usage can be found in the [AWS Example](example/cloud/aws/main.go) directory.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
