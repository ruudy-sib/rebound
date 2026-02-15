FROM golang:1.23-alpine AS builder
WORKDIR /app

# Copy go.mod and go.sum for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build
COPY . .
RUN go build -o kafka-retry ./cmd/kafka-retry/...

# Runtime image
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/kafka-retry ./
EXPOSE 8080
CMD ["./kafka-retry"]
