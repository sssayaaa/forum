# Stage 1: Build the Go application
FROM golang:alpine AS build

# Set CGO_ENABLED to 1 and install dependencies
ENV CGO_ENABLED=1
RUN apk add --no-cache gcc musl-dev

# Set working directory
WORKDIR /app
COPY . .

# Build the Go application
RUN go build -o forum ./cmd/main.go

# Stage 2: Create a minimal runtime container
FROM alpine:latest

# Install the required runtime dependencies (SQLite)
RUN apk add --no-cache sqlite-libs

# Copy the built binary from the build stage
WORKDIR /app
COPY --from=build /app .

# Set the default command to run the binary
CMD ["./forum"]
