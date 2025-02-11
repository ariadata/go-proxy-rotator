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

# Final stage
FROM scratch
WORKDIR /app
COPY --from=builder /build/proxy-server .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 1080
CMD ["./proxy-server"]
