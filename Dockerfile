# --- STAGE 1: Build the Binary ---
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy dependency manifests first for caching optimizations
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Cross-compile for ARM64 architecture (the target STB environment)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o stb-bot .

# --- STAGE 2: Microscopic Runtime ---
FROM alpine:3.19

WORKDIR /app

# Install security certificates so the bot can connect over HTTPS safely to Discord
RUN apk --no-cache add ca-certificates

# Copy the lightweight binary from the builder stage
COPY --from=builder /app/stb-bot .

# Run the binary
CMD ["./stb-bot"]