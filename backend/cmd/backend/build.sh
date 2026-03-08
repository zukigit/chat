#!/bin/sh

echo "Building backend image..."
docker build -t zukidocker/chat-backend:latest -f ./backend/cmd/backend/Dockerfile .
echo "Backend image built successfully."