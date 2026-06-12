FROM alpine:3.19

WORKDIR /app

# Install security certificates and force clear the cache
RUN apk update && apk --no-cache add ca-certificates

# Copy your local pre-compiled binary
COPY stb-bot .

# Ensure permissions are correct
RUN chmod +x stb-bot

CMD ["./stb-bot"]