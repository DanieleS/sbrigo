# syntax=docker/dockerfile:1

# --- Frontend -------------------------------------------------------------
FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# --- Backend --------------------------------------------------------------
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/sbrigo ./cmd/sbrigo

# --- Runtime --------------------------------------------------------------
# distroless/static ships CA certificates (needed to reach Logto over HTTPS) and nothing else.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/sbrigo /sbrigo
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/sbrigo"]
