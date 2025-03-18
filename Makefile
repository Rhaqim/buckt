.PHONY: all build run start docker-build docker-run test coverage clean

PORT ?= 8080

PKG_NAME = buckt
PKG = github.com/Rhaqim/${PKG_NAME}
BUILD_DIR = bin
BUILD_NAME = main

all: build run

build:
	go build -o $(BUILD_DIR)/$(BUILD_NAME) $(BUILD_NAME).go

run: build
	./bin/mainb -port=$(PORT)

start:
	go run cmd/main.go -port=$(PORT)

docker-build:
	docker build -t $(PKG_NAME) -f Dockerfile .

docker-run:
	docker run -p $(PORT):$(PORT) $(PKG_NAME)
	# docker run -e PORT=$(PORT) -p $(PORT):$(PORT) $(PKG_NAME)

test:
	go test ./... -v

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

update-sub-deps:
	go get -u ./...

clean: clean-build clean-logs clean-db clean-coverage clean-media

clean-build:
	rm -f bin/*

clean-logs:
	rm -f logs/*

clean-db:
	rm -f db.sqlite

clean-coverage:
	rm -f coverage.*

clean-media:
	rm -rf media/