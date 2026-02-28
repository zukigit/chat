FROM debian:13

RUN apt-get update && apt-get install -y ca-certificates tzdata

WORKDIR /app

COPY dist/chat_gateway /app/chat_gateway

EXPOSE 8080

RUN chmod +x /app/chat_gateway

ENTRYPOINT ["/app/chat_gateway"]
