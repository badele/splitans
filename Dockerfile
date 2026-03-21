# Configuration variables
ARG GO_VERSION=1.24.3
ARG ALPINE_VERSION=3.22

# Build stage
FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

# Compile application
RUN CGO_ENABLED=0 GOOS=linux go build -o . ./cmd/...

# Runtime stage
ARG ALPINE_VERSION
FROM alpine:${ALPINE_VERSION}

WORKDIR /work

# Copy binary from builder
COPY --from=builder /app/splitans /usr/local/bin/
COPY --from=builder /app/splitansx /usr/local/bin/

# Set entrypoint
ENTRYPOINT ["splitans"]

# Default arguments (can be overridden)
CMD []
