# === Dev stage (hot reload with air) ===
FROM golang:1.23-alpine AS dev
RUN go install github.com/air-verse/air@latest
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true
COPY . .
CMD ["air", "-c", ".air.toml"]

# === Builder ===
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/wessley ./cmd/wessley

# === Runtime ===
FROM alpine:3.19 AS wessley
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/wessley /usr/local/bin/wessley
EXPOSE 8080
ENTRYPOINT ["wessley"]
