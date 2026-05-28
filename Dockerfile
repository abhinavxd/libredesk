# Stage 1: Build frontend
FROM node:22-alpine AS frontend-builder
RUN corepack enable && corepack prepare pnpm@9.15.3 --activate
WORKDIR /app/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build:main && pnpm build:widget

# Stage 2: Build Go backend
FROM golang:1.25-alpine AS backend-builder
RUN apk --no-cache add git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
# Install stuffbin
RUN go install github.com/knadh/stuffbin/...@latest
# Build the Go binary
RUN CGO_ENABLED=0 go build -a \
    -ldflags="-s -w" \
    -o libredesk cmd/*.go
# Embed static assets into the binary
RUN stuffbin -a stuff -in libredesk -out libredesk frontend/dist i18n schema.sql static

# Stage 3: Final runtime image
FROM alpine:3.18
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /libredesk
COPY --from=backend-builder /app/libredesk .
COPY config.sample.toml config.toml
EXPOSE 9000
CMD ["./libredesk"]
