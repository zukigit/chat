FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY dist/backend /app/backend

EXPOSE 50051

RUN chmod +x /app/backend

ENTRYPOINT ["/app/backend"]
