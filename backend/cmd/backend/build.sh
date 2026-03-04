#!/bin/sh

echo "Building backend binary..."
docker run --rm \
    -v "$(pwd)":/workspace \
    -w /workspace golang:1.25.3-alpine3.22 sh \
    -c "go build -o chat_backend github.com/zukigit/chat/backend/cmd/backend"
echo "Backend binary built successfully."

echo "Building backend image..."
docker build -t zukidocker/chat-backend:latest -f ./backend/cmd/backend/Dockerfile .
echo "Backend image built successfully."