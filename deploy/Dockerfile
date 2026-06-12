# syntax=docker/dockerfile:1

# ---- Stage 1: build the React SPA ----
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci || npm install
COPY frontend/ ./
RUN npm run build   # outputs to /app/internal/web/dist

# ---- Stage 2: build the Go binary (embeds the SPA) ----
FROM golang:1.24-alpine AS backend
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Bring in the freshly built SPA so go:embed picks it up.
COPY --from=frontend /app/internal/web/dist ./internal/web/dist
ARG VERSION=docker
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X github.com/egorbr4z/panel/internal/api/handler.Version=${VERSION}" \
    -o /vpanel ./cmd/vpanel

# ---- Stage 3: fetch pinned proxy cores ----
FROM alpine:3.20 AS cores
ARG TARGETARCH=amd64
ARG XRAY_VERSION=v1.8.23
ARG SINGBOX_VERSION=1.10.7
RUN apk add --no-cache curl unzip tar
WORKDIR /cores
# Xray-core
RUN case "$TARGETARCH" in \
      amd64) XARCH=64 ;; arm64) XARCH=arm64-v8a ;; *) XARCH=64 ;; esac; \
    curl -fsSL -o xray.zip \
      "https://github.com/XTLS/Xray-core/releases/download/${XRAY_VERSION}/Xray-linux-${XARCH}.zip" && \
    unzip -o xray.zip xray -d . && rm xray.zip && chmod +x xray
# sing-box
RUN curl -fsSL -o singbox.tar.gz \
      "https://github.com/SagerNet/sing-box/releases/download/v${SINGBOX_VERSION}/sing-box-${SINGBOX_VERSION}-linux-${TARGETARCH}.tar.gz" && \
    tar -xzf singbox.tar.gz --strip-components=1 -C . "sing-box-${SINGBOX_VERSION}-linux-${TARGETARCH}/sing-box" && \
    rm singbox.tar.gz && chmod +x sing-box

# ---- Stage 4: final image ----
FROM debian:13-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*
COPY --from=backend /vpanel /usr/local/bin/vpanel
COPY --from=cores /cores/xray /usr/local/bin/xray
COPY --from=cores /cores/sing-box /usr/local/bin/sing-box
ENV VPANEL_DATA_DIR=/var/lib/vpanel \
    VPANEL_XRAY_BIN=/usr/local/bin/xray \
    VPANEL_SINGBOX_BIN=/usr/local/bin/sing-box
VOLUME ["/var/lib/vpanel"]
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/vpanel"]
