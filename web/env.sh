#!/bin/sh

# Generate runtime environment config from Docker environment variables
# This script runs at container startup before nginx starts

cat <<EOF > /usr/share/nginx/html/env-config.js
window.ENV = {
  API_URL: "${API_URL:-http://localhost:8080}"
};
EOF

echo "Runtime config generated with API_URL=${API_URL:-http://localhost:8080}"
