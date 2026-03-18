# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Cache dependencies
COPY service/backend/go.mod service/backend/go.sum ./
RUN go mod download

# Copy backend source
COPY service/backend/ .

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /subway-server ./cmd/subway-server/

# Frontend build stage
FROM node:20-alpine AS frontend

WORKDIR /app

COPY service/frontend/package.json service/frontend/package-lock.json ./
RUN npm ci
COPY service/frontend/ .
RUN npm run build -- --outDir /frontend-dist

# Run stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /subway-server /app/subway-server
COPY service/backend/led_map.json /app/led_map.json
COPY service/backend/leds.json /app/leds.json
COPY service/backend/templates/ /app/templates/
COPY --from=frontend /frontend-dist/ /app/static/dist/

RUN mkdir -p /app/data

EXPOSE 8080

CMD ["/app/subway-server", "--port", "8080", "--led-map", "led_map.json", "--data-dir", "data", "--template-dir", "templates", "--static-dir", "static"]
