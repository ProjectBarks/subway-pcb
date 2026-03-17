# Build stage
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Cache dependencies
COPY server/go.mod server/go.sum ./
RUN go mod download

# Copy server source
COPY server/ .

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /subway-server ./cmd/subway-server/

# Run stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /subway-server /app/subway-server
COPY server/led_map.json /app/led_map.json
COPY server/static/ /app/static/
COPY server/templates/ /app/templates/

RUN mkdir -p /app/data
VOLUME ["/app/data"]

EXPOSE 8080

CMD ["/app/subway-server", "--port", "8080", "--led-map", "led_map.json", "--data-dir", "data", "--template-dir", "templates"]
