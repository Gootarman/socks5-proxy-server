FROM alpine:3.20

RUN apk add --no-cache ca-certificates && \
    addgroup -S app && adduser -S -G app app

WORKDIR /app
COPY dist/proxy /app/proxy

USER app
EXPOSE 8000

ENTRYPOINT ["/app/proxy"]
