# Stage 1: Force your laptop's native architecture for downloading certificates
FROM --platform=$BUILDPLATFORM alpine:3.19 AS certs
RUN apk --no-cache add ca-certificates

# Stage 2: The final clean ARM64 target container (No commands executed!)
FROM alpine:3.19
WORKDIR /app

# 1. Copy the certificates from Stage 1
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# 2. Copy the binary and set permissions at the exact same time!
COPY --chmod=755 stb-bot .

CMD ["./stb-bot"]