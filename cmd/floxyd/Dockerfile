# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build floxyd binary
ARG VERSION=dev
ARG COMMIT=unknown
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-w -s -X 'main.version=${VERSION}' -X 'main.commit=${COMMIT}'" \
    -o floxyd ./cmd/floxyd

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata jq curl && \
    addgroup -S floxyd && \
    adduser -S -G floxyd floxyd

WORKDIR /app

COPY --from=builder /build/floxyd /app/floxyd
RUN chmod +x /app/floxyd

USER floxyd

EXPOSE 8081

ENTRYPOINT ["/app/floxyd"]

