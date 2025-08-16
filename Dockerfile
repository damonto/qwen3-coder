FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY *.go ./

RUN go mod download

ARG BUILD_VERSION

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s -X main.Version=$BUILD_VERSION" -o qwen3-coder .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/qwen3-coder .

RUN mkdir -p /data

EXPOSE 9527

COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
