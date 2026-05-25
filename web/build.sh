#!/bin/sh

echo "Building web image..."
docker build -t zukidocker/chat-web:latest --build-arg VITE_GATEWAY_URL=$GATEWAY_URL --build-arg VITE_ENABLE_PASSWORD_AUTH=${VITE_ENABLE_PASSWORD_AUTH:-} -f ./web/Dockerfile .
echo "Web image built successfully."