#Build stage
FROM golang:1.25.3-alpine3.22 AS build
WORKDIR /app

COPY ./backend .

RUN go build -mod=vendor -o chat_backend github.com/zukigit/chat/backend/cmd/backend

#Production stage
FROM alpine:3.22
WORKDIR /app

COPY --from=build /app/chat_backend /app/chat_backend

RUN apk add --no-cache tzdata
RUN chmod +x /app/chat_backend

EXPOSE 1234

ENTRYPOINT ["/app/chat_backend"]
