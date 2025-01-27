# Stage 1: Build the Go application
FROM golang:1.22.5-alpine AS builder
# Install required system packages
RUN apk update && \
    apk upgrade && \
    apk add --no-cache ca-certificates && \
    update-ca-certificates

WORKDIR /build

# Copy go mod and source files
COPY go.mod go.sum ./
COPY *.go ./

# Download dependencies
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o proxy-server .

# Stage 2: Build the nc binary
FROM alpine:latest AS nc-builder
RUN apk add --no-cache netcat-openbsd

# Stage 3: Final scratch image
FROM scratch
WORKDIR /app

# Copy the Go binary and CA certificates
COPY --from=builder /build/proxy-server .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the nc binary from the nc-builder stage
COPY --from=nc-builder /usr/bin/nc /bin/nc

EXPOSE 1080
CMD ["./proxy-server"]
