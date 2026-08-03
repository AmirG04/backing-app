# --- Etapa 1: build ---
FROM golang:1.25-bookworm AS builder

# TigerBeetle usa CGO (codigo nativo), necesitamos gcc y libc de verdad (no musl)
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o /banking-api ./cmd/api
RUN CGO_ENABLED=1 GOOS=linux go build -o /seed ./cmd/seed

# --- Etapa 2: imagen final ---
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /banking-api .
COPY --from=builder /seed .

EXPOSE 8080
CMD ["./banking-api"]
