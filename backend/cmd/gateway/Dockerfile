#Build stage
FROM golang:1.25.3-alpine3.22 AS build
WORKDIR /app

COPY ./backend .

RUN go build -mod=vendor -o chat_gateway github.com/zukigit/chat/backend/cmd/gateway

#Production stage
FROM alpine:3.22
WORKDIR /app

COPY --from=build /app/chat_gateway /app/chat_gateway

RUN apk add --no-cache tzdata
RUN chmod +x /app/chat_gateway

EXPOSE 8080

ENTRYPOINT ["/app/chat_gateway"]
