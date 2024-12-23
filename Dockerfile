FROM golang:1.22.5-alpine AS builder
# Install required packages and set up workspace in a single layer
RUN apk add --no-cache ca-certificates && update-ca-certificates

WORKDIR /build

# Copy all necessary files in a single layer
COPY . .

# Build the application
RUN go mod download && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o proxy-server ./cmd/main.go

# Final stage - minimal image
FROM scratch
WORKDIR /app

# Copy only the necessary files from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/proxy-server .
COPY --from=builder /build/proxies.conf ./proxies.conf
COPY --from=builder /build/users.conf ./users.conf

EXPOSE 1080
CMD ["./proxy-server"]