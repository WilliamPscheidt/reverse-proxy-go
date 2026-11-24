# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/proxy ./cmd/proxy

# Runtime stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /bin/proxy /app/proxy

EXPOSE 8080 9000 2112
ENTRYPOINT ["/app/proxy"]
