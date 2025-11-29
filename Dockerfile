# Configuration variables
ARG GO_VERSION=1.25
ARG ALPINE_VERSION=3.22

# Build stage
FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY *.go ./
COPY exporter ./exporter
COPY tokenizer ./tokenizer

# Compile application
RUN CGO_ENABLED=0 GOOS=linux go build -o splitans .

# Runtime stage
ARG ALPINE_VERSION
FROM alpine:${ALPINE_VERSION}

WORKDIR /work

# Copy binary from builder
COPY --from=builder /app/splitans /usr/local/bin/splitans

# Set entrypoint
ENTRYPOINT ["splitans"]

# Default arguments (can be overridden)
CMD []
