FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY dist/chat_backend /app/chat_backend

EXPOSE 50051

RUN chmod +x /app/chat_backend

ENTRYPOINT ["/app/chat_backend"]
