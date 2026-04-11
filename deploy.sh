#!/bin/bash

set -euo pipefail

echo "Deploying stonks..."

git pull

echo "Building images in parallel..."
docker build -t stonks-api -f Dockerfile.api . &
docker build -t stonks-worker -f Dockerfile.worker . &
docker build -t stonks-discord-bot -f Dockerfile.discord-bot . &
docker build -t stonks-whatsapp-bot -f Dockerfile.whatsapp-bot . &
wait

echo "Stopping old containers..."
docker rm -f stonks-api stonks-worker stonks-discord-bot stonks-whatsapp-bot 2>/dev/null || true
sleep 1
sudo fuser -k 8080/tcp 2>/dev/null || true

echo "Starting containers..."
docker run -d --restart unless-stopped --name stonks-api --add-host=host.docker.internal:host-gateway --env-file .env -p 8080:8080 stonks-api
docker run -d --restart unless-stopped --name stonks-worker --add-host=host.docker.internal:host-gateway --env-file .env stonks-worker
docker run -d --restart unless-stopped --name stonks-discord-bot --add-host=host.docker.internal:host-gateway --env-file .env stonks-discord-bot
docker run -d --restart unless-stopped --name stonks-whatsapp-bot --add-host=host.docker.internal:host-gateway --env-file .env stonks-whatsapp-bot

echo "Cleaning up dangling images..."
docker image prune -f

echo "Done. Running containers:"
docker ps --filter "name=stonks"
