# Multi-stage build for minimal image size
# Stage 1: Build the Go binary
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy dependency files first (better layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /kubesynapse \
    ./cmd/kubesynapse/

# Stage 2: Minimal runtime image
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /kubesynapse /kubesynapse

# Run as non-root user
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/kubesynapse"]
