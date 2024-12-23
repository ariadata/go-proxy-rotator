# Go Proxy Rotator

A high-performance SOCKS5 proxy server written in Go that rotates through multiple upstream proxies. Perfect for distributed scraping, API access, and general proxy needs.

## Features

- SOCKS5 proxy server with username/password authentication
- Multiple proxy protocol support (HTTP, HTTPS, SOCKS5, SOCKS5H)
- Round-robin proxy rotation
- Edge mode for fallback to direct connections
- Multi-user support
- Docker support
- Zero runtime dependencies
- IPv6 support

## Quick Start

1. Clone the repository:
```bash
git clone https://github.com/yourusername/go-proxy-rotator.git
cd go-proxy-rotator
```

2. Set up configuration files:
```bash
cp .env.example .env
cp users.conf.example users.conf
cp proxies.conf.example proxies.conf
```

3. Edit the configuration files:
  - `users.conf`: Add your username:password pairs
  - `proxies.conf`: Add your proxy servers
  - `.env`: Adjust settings if needed

4. Run with Docker:
```bash
docker compose up -d
```

## Configuration

### Environment Variables (.env)
```env
COMPOSE_PROJECT_NAME=go-proxy-rotator
DC_SOCKS_PROXY_PORT=60255
ENABLE_EDGE_MODE=true
```

### User Configuration (users.conf)
```
username1:password1
username2:password2
```

### Proxy Configuration (proxies.conf)
```
# HTTP/HTTPS proxies
http://proxy1.example.com:8080
https://user:pass@proxy2.example.com:8443

# SOCKS5 proxies
socks5://proxy3.example.com:1080
socks5h://user:pass@proxy4.example.com:1080
```

## Testing

Test your connection:
```bash
curl --proxy socks5h://username1:password1@localhost:60255 https://api.ipify.org?format=json
```

## Building from Source

```bash
make build
```

## Docker Commands

Build image:
```bash
docker build -t go-proxy-rotator .
```

Run container:
```bash
docker run -d \
  -p 60255:1080 \
  -v $(pwd)/proxies.conf:/app/proxies.conf:ro \
  -v $(pwd)/users.conf:/app/users.conf:ro \
  -e ENABLE_EDGE_MODE=true \
  go-proxy-rotator
```

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.