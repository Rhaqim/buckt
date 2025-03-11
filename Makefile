.PHONY: all build run start docker-build docker-run test coverage clean

PORT ?= 8080

all: build run

build:
	go build -o bin/main cmd/main.go

run: build
	./bin/mainb -port=$(PORT)

start:
	go run cmd/main.go -port=$(PORT)

docker-build:
	docker build -t buckt -f Dockerfile .

docker-run:
	docker run -p $(PORT):$(PORT) buckt
	# docker run -e PORT=$(PORT) -p $(PORT):$(PORT) buckt

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