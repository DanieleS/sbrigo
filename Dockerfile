# syntax=docker/dockerfile:1

# --- Frontend -------------------------------------------------------------
# Build stages run on the builder's own architecture and cross-compile for the target, so
# multi-arch images (amd64 + arm64) do not pay the cost of emulating npm and go.
FROM --platform=$BUILDPLATFORM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# --- Backend --------------------------------------------------------------
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/sbrigo ./cmd/sbrigo

# --- Runtime --------------------------------------------------------------
# distroless/static ships CA certificates (needed to reach Logto over HTTPS) and nothing else.
FROM gcr.io/distroless/static-debian12:nonroot
LABEL org.opencontainers.image.source="https://github.com/DanieleS/sbrigo"
COPY --from=build /out/sbrigo /sbrigo
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/sbrigo"]
