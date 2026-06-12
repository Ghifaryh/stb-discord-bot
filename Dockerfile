FROM alpine:3.19

WORKDIR /app

# Install security certificates for Discord HTTPS connections
RUN apk --no-cache add ca-certificates

# Copy the pre-compiled binary that you built on your laptop
COPY stb-bot .

# Grant execution rights to the binary
RUN chmod +x stb-bot

CMD ["./stb-bot"]