FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY dist/gateway /app/gateway

EXPOSE 8080

ENTRYPOINT ["/app/gateway"]
