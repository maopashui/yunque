# YunQue — multi-stage image: frontend SPA + Go API (single process).
# Build:  docker build -t yunque .
#         docker build --build-arg YUNQUE_BASE_PATH=/yunque -t yunque .
# Run:    docker run --rm -p 8080:8080 -v yunque-data:/data yunque
# Base path must match at build (frontend assets) and runtime (API mount).

# ---------- frontend ----------
FROM node:20-alpine AS frontend
WORKDIR /src/frontend
ARG YUNQUE_BASE_PATH=/
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN YUNQUE_BASE_PATH="$YUNQUE_BASE_PATH" npm run build

# ---------- backend ----------
FROM golang:1.26-alpine AS backend
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/yunque ./cmd/server

# ---------- runtime ----------
FROM alpine:3.21
ARG YUNQUE_BASE_PATH=/
RUN apk add --no-cache ca-certificates tzdata \
  && adduser -D -H -u 10001 yunque
WORKDIR /app

COPY --from=backend /out/yunque /app/yunque
COPY --from=backend /src/migrations /app/migrations
COPY --from=frontend /src/frontend/dist /app/static

RUN mkdir -p /data/assets && chown -R yunque:yunque /data /app

USER yunque
ENV YUNQUE_ADDR=:8080 \
    YUNQUE_BASE_PATH=${YUNQUE_BASE_PATH} \
    YUNQUE_DB_PATH=/data/yunque.db \
    YUNQUE_ASSETS_DIR=/data/assets \
    YUNQUE_MIGRATIONS_DIR=/app/migrations \
    YUNQUE_STATIC_DIR=/app/static \
    YUNQUE_ALLOWED_ORIGIN=

VOLUME ["/data"]
EXPOSE 8080
ENTRYPOINT ["/app/yunque"]
