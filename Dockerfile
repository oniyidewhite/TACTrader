# Build stage
FROM golang:1.23 as build
LABEL maintainer="ea-trader"

WORKDIR /build

# Copy source code
COPY . .

# Fetch dependencies
RUN go mod download

# Build application
RUN CGO_ENABLED=0 go build -o /app ./cmd

# Runner stage
FROM alpine:3.10
# Add certificates for HTTPS (if required)
RUN apk add --no-cache ca-certificates

# Copy binary from build stage
COPY --from=build /app /usr/local/bin/app

# Set binary as entry point
ENTRYPOINT ["/usr/local/bin/app"]
