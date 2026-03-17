# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Download dependencies first for layer caching.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o pulse-agent ./cmd/pulse-agent

# ---- Runtime stage ----
FROM alpine:3

RUN addgroup -S pulse && adduser -S pulse -G pulse

WORKDIR /app
COPY --from=builder /app/pulse-agent ./pulse-agent

USER pulse

EXPOSE 9464

ENTRYPOINT ["./pulse-agent"]
