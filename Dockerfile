# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/spotifish ./cmd/server

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/spotifish .
COPY --from=builder /app/migrations ./migrations

# Create album art directory
RUN mkdir -p /var/lib/spotifish/art

EXPOSE 8080

ENV PORT=8080
ENV ALBUM_ART_PATH=/var/lib/spotifish/art

CMD ["./spotifish"]
