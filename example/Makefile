.PHONY: build $(TARGETS)

# Port defaults
PORT ?= 8080

# Find all subprojects with a main.go
DIRECTORIES := $(shell find direct cloud client/web -type f -name main.go -exec dirname {} \;)
TARGETS := $(notdir $(DIRECTORIES))

# Run target: go run <dir>
$(TARGETS):
	go run ./$@ $(if $(filter $@,ui api mount),-port=$(PORT))

# Build everything into ./bin
build:
	@for dir in $(DIRECTORIES); do \
		name=$$(basename $$dir); \
		echo "Building $$name..."; \
		go build -o bin/$$name ./$$dir; \
	done