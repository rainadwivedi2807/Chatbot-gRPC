# Build stage
FROM golang:1.25.3-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project
COPY . .

# Build the server binary
RUN go build -o chatbot-server server/main.go

# Final stage
FROM alpine:latest

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/chatbot-server .

# Expose the gRPC port
EXPOSE 9000

# Run the server
CMD ["./chatbot-server"]
