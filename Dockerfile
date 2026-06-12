# Start with an absolutely empty 0-byte layer
FROM scratch

WORKDIR /app

# Copy the pre-compiled binary from your SSD project directory
COPY stb-bot .

# Run it directly
CMD ["./stb-bot"]