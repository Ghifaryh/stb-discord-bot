#!/bin/bash

# 1. Compile the standalone ARM64 binary locally on your laptop
echo "⚙️ Compiling Go binary for ARM64..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o stb-bot .

# 2. Build the Docker image locally
echo "🐋 Packing Docker image..."
docker build -t stb-bot:latest .

# 3. Export the image to a transportable tar file
echo "📦 Exporting image layers..."
docker save stb-bot:latest -o stb-bot.tar

# 4. Sync BOTH the image payload AND the fresh docker-compose configuration
echo "🚀 Shipping assets to STB..."
scp stb-bot.tar docker-compose.yml root@100.84.225.86:/mnt/ssd/projects/stb-discord-bot/

# 5. Load, purge tar, and force recreate the tracking container
echo "🔄 Reloading container on host..."
ssh root@100.84.225.86 "cd /mnt/ssd/projects/stb-discord-bot && docker load -i stb-bot.tar && rm stb-bot.tar && docker compose up -d --force-recreate"

# 6. Clean up the local tar file on your laptop
rm stb-bot.tar

echo "🎯 Update complete! Check your Discord monitor."