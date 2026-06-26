# syntax=docker/dockerfile:1

# --- Build stage ---
FROM golang:1.25-alpine AS build

WORKDIR /src

# Cache dependencies first.
COPY go.mod go.sum ./
RUN go mod download

# Build the statically linked Linux binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/botontheclocktower ./cmd/bot

# --- Runtime stage ---
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/botontheclocktower /app/botontheclocktower

# BOTAPIKEY must be supplied at runtime, e.g. `docker run -e BOTAPIKEY=...`.
USER nonroot:nonroot
ENTRYPOINT ["/app/botontheclocktower"]
