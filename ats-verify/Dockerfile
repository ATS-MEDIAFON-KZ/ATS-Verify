# ===== Stage 1: Build React Frontend =====
FROM node:22-alpine AS frontend-builder

WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci --silent
COPY web/ .
RUN npm run build

# ===== Stage 2: Build Go Backend =====
FROM golang:1.24-alpine AS backend-builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# ===== Stage 3: Production Runtime =====
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -S ats && adduser -S ats -G ats

WORKDIR /app

# Copy Go binary
COPY --from=backend-builder /server .

# Copy frontend build (served by Go or reverse proxy)
COPY --from=frontend-builder /app/web/dist ./web/dist

# Copy database schema for reference
COPY database/ ./database/

# Switch to non-root
USER ats

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["./server"]
