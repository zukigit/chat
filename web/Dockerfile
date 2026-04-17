# ── Build stage ───────────────────────────────────────────────────────────────
FROM node:alpine AS builder

WORKDIR /app

COPY ./web .

ARG VITE_GATEWAY_URL
RUN npm ci
RUN npm run build

# ── Serve stage ───────────────────────────────────────────────────────────────
FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY ./web/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
