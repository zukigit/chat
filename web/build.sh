#!/bin/sh

echo "Building web image..."
docker build -t zukidocker/chat-web:latest -f ./web/Dockerfile .
echo "Web image built successfully."