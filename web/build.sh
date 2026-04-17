#!/bin/sh

echo "Building web image..."
docker build -t zukidocker/chat-web:latest --build-arg VITE_GATEWAY_URL=$GATEWAY_URL -f ./web/Dockerfile .
echo "Web image built successfully."