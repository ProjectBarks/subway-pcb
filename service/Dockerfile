# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Cache dependencies
COPY service/go.mod service/go.sum ./
RUN go mod download

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

# Copy backend source
COPY service/cmd/ ./cmd/
COPY service/internal/ ./internal/
COPY service/ui/ ./ui/
COPY service/gen/ ./gen/

# Generate templ files and build
RUN templ generate && go generate ./ui/... && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /subway-server ./cmd/subway-server/

# Frontend build stage
FROM node:20-alpine AS frontend

WORKDIR /app

COPY service/package.json service/package-lock.json ./
RUN npm ci
COPY service/ui/ ./ui/
COPY service/vite.config.ts service/tsconfig.json service/biome.json ./
RUN npm run build -- --outDir /frontend-dist

# Run stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /subway-server /app/subway-server
COPY service/led_map.json /app/led_map.json
COPY service/leds.json /app/leds.json
COPY --from=frontend /frontend-dist/ /app/static/dist/

RUN mkdir -p /app/data

EXPOSE 8080

CMD ["/app/subway-server", "--port", "8080", "--led-map", "led_map.json", "--data-dir", "data", "--static-dir", "static"]
