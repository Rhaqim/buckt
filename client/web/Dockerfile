# Use a minimal base image
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Copy source code
COPY . .

# Install necessary packages for SQLite and CGO
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Enable CGO
ENV CGO_ENABLED=1

# Build the Go application
RUN go build -o myapp cmd/main.go

# Use a smaller runtime image
FROM alpine:latest

# Set working directory
WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/myapp .

# Expose port (default 8080, can be changed via env variable)
ENV PORT=8080

# Run the application with the specified port
CMD ["sh", "-c", "./myapp -port=$PORT"]