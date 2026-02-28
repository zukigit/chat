FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY dist/chat_gateway /app/chat_gateway

EXPOSE 8080

RUN chmod +x /app/chat_gateway

ENTRYPOINT ["/app/chat_gateway"]
