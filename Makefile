.PHONY: all build run start test coverage clean

all: run

build:
	go build -o bin/main cmd/main.go

run: build
	./bin/main

start:
	go run cmd/main.go

test:
	go test ./... -v

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

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