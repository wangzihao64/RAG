FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/rag ./cmd/main.go

FROM alpine:3.22

RUN addgroup -S -g 10001 rag && \
    adduser -S -D -H -u 10001 -G rag rag && \
    mkdir -p /app/uploads && \
    chown -R rag:rag /app

WORKDIR /app
COPY --from=builder --chown=rag:rag /out/rag /app/rag
COPY --chown=rag:rag config/config.ini /app/config/config.ini

USER rag
EXPOSE 8080
ENTRYPOINT ["/app/rag"]
