# Start from the official Golang image for building

FROM golang:1.23-alpine AS builder
WORKDIR /app
# Copy go.mod and go.sum first for better caching and dependency install
COPY go.mod go.sum ./
RUN go mod download
# Copy the rest of the source code
COPY . .
RUN go build -o kafka-retry .

# Use a minimal image for running
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/kafka-retry ./
EXPOSE 8080
CMD ["./kafka-retry"]
