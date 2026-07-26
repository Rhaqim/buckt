.PHONY: all build test test-all coverage clean sync-client

PORT ?= 8080

# Version to point client/web at when syncing (e.g. make sync-client VERSION=v1.6.0).
VERSION ?= latest

PKG_NAME = buckt
PKG = github.com/Rhaqim/${PKG_NAME}
BUILD_DIR = bin

all: build run

build:
	go build -o $(BUILD_DIR)/$(PKG_NAME) cmd/$(PKG_NAME).go

run: build
	./bin/$(PKG_NAME)

test:
	go test ./... -v

# Test every module (root + cloud/* + client/web) through the workspace, so the
# sub-modules exercise the local buckt, not their pinned release.
test-all:
	go test ./...
	cd client/web && go test ./...
	cd cloud/aws && go test ./...
	cd cloud/azure && go test ./...
	cd cloud/gcp && go test ./...

# During development, go.work keeps client/web (and the example) on the local
# buckt automatically. For a RELEASE, after tagging a new buckt version, point
# client/web at it and tidy so it builds standalone:
#   make sync-client VERSION=v1.6.0
# (The cloud/* modules do not depend on buckt — the dependency is one-way — so
# they need no sync.)
sync-client:
	cd client/web && go get github.com/Rhaqim/buckt@$(VERSION) && go mod tidy

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