# Multi-stage build for optimized image size

# Stage 1: Build Go binary
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o rph ./cmd/rph

# Stage 2: Runtime with LaTeX
FROM ubuntu:22.04

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install LaTeX and dependencies
RUN apt-get update && apt-get install -y \
    texlive-latex-base \
    texlive-latex-extra \
    texlive-fonts-recommended \
    latexmk \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create app directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/rph /app/rph

# Copy config file
COPY config/config.yaml /app/config/config.yaml

# Create necessary directories
RUN mkdir -p /app/lib /app/tex_files /app/reports /app/.metadata

# Make binary executable
RUN chmod +x /app/rph

# Set environment variables
ENV PATH="/app:${PATH}"

# Default command
ENTRYPOINT ["/app/rph"]
CMD ["--help"]
