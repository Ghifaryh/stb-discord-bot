#!/bin/bash

# Exit script immediately if any individual step fails
set -e

echo "⚙️  1. Compiling standalone Go binary for ARM64..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o stb-bot .

echo "🐋 2. Building Docker image locally on laptop..."
docker build --platform linux/arm64 -t stb-bot:latest .

echo "📦 3. Compressing image layers to tar archive..."
docker save stb-bot:latest -o stb-bot.tar

echo "🚀 4. Shipping packed assets to STB storage..."
# Ships only the pre-baked tar payload and the configuration file over Tailscale
scp stb-bot.tar docker-compose.yml root@100.84.225.86:/mnt/ssd/projects/stb-discord-bot/

echo "🔄 5. Ingesting image and hot-swapping container process on STB..."
# Tells the remote host to load the image, drop the old container instance, and deploy the fresh state
ssh root@100.84.225.86 "cd /mnt/ssd/projects/stb-discord-bot && docker load -i stb-bot.tar && rm stb-bot.tar && docker compose up -d --force-recreate"

echo "🧹 6. Cleaning temporary local workspace artifacts..."
rm -f stb-bot stb-bot.tar

echo "🎯 Deployment complete! All commands are live on your server."