FROM golang:1.25.6-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/go-proxy ./cmd

FROM alpine:3.20

RUN apk add --no-cache ca-certificates && \
    addgroup -S app && adduser -S -G app app

WORKDIR /app
COPY --from=builder /out/go-proxy /app/go-proxy

USER app
EXPOSE 8000

ENTRYPOINT ["/app/go-proxy"]
