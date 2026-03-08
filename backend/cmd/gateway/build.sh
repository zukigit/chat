#!/bin/sh

echo "Building gateway image..."
docker build -t zukidocker/chat-gateway:latest -f ./backend/cmd/gateway/Dockerfile .
echo "Gateway image built successfully."