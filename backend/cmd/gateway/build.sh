#!/bin/sh

echo "Building gateway binary..."
docker run --rm \
    -v "$(pwd)":/workspace \
    -w /workspace golang:1.25.3-alpine3.22 sh \
    -c "go build -o chat_gateway github.com/zukigit/chat/backend/cmd/gateway"
echo "Gateway binary built successfully."

echo "Building gateway image..."
docker build -t zukidocker/chat-gateway:latest -f ./backend/cmd/gateway/Dockerfile .
echo "Gateway image built successfully."