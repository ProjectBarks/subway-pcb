# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Cache dependencies
COPY service/go.mod service/go.sum ./
RUN go mod download

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@v0.3.1001

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
COPY service/public/ ./public/
COPY service/vite.config.ts service/tsconfig.json service/biome.json ./
RUN npm run build

# Run stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /subway-server /app/subway-server
COPY --from=frontend /app/static/dist/ /app/static/dist/

RUN mkdir -p /app/data

EXPOSE 8080

CMD ["/app/subway-server", "--port", "8080", "--boards-dir", "static/dist/boards", "--data-dir", "data", "--static-dir", "static"]
