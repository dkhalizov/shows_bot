FROM golang:1.24-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o tv-shows-bot cmd/main.go

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /app/tv-shows-bot /app/tv-shows-bot

ENTRYPOINT ["/app/tv-shows-bot"]

EXPOSE 8080

USER nonroot:nonroot

LABEL org.opencontainers.image.source="https://github.com/deniskhalizov/shows_bot"
LABEL org.opencontainers.image.description="Telegram bot for TV shows notifications"
LABEL org.opencontainers.image.licenses="MIT"