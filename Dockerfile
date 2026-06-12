FROM alpine:3.19

WORKDIR /app

# Install certificates and the native Docker CLI tool
RUN apk --no-cache add ca-certificates docker-cli

# Copy your local pre-compiled binary
COPY stb-bot .
RUN chmod +x stb-bot

CMD ["./stb-bot"]