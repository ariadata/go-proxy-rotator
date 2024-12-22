# Go Proxy Rotator

A high-performance SOCKS5 proxy server written in Go that rotates through multiple upstream proxies. Perfect for distributed scraping, API access, and general proxy needs.

## Features

- SOCKS5 proxy server with username/password authentication
- Support for multiple upstream proxy protocols:
  - HTTP proxies
  - HTTPS proxies (encrypted)
  - SOCKS5 proxies
  - SOCKS5H proxies (proxy performs DNS resolution)
- Round-robin proxy rotation
- Edge mode for fallback to direct connections
- Multi-user support via configuration file
- Docker and docker-compose support
- Configurable port
- Zero runtime dependencies
- Comments support in configuration files
- Automatic proxy failover
- IPv6 support

## Quick Start with Docker Compose (Recommended)

1. Clone the repository:
```bash
git clone https://github.com/ariadata/go-proxy-rotator.git
cd go-proxy-rotator
```

2. Set up configuration files:
```bash
# Copy environment example
cp .env.example .env

# Create users file
echo "user1:password1" > users.conf
echo "user2:password2" >> users.conf

# Create proxies file (add your proxies)
touch proxies.conf
```

3. Create `docker-compose.yml`:
```yaml
version: '3.8'

services:
  proxy-rotator:
    image: 'ghcr.io/ariadata/go-proxy-rotator:latest'
    ports:
      - "${DC_SOCKS_PROXY_PORT}:1080"
    volumes:
      - ./proxies.conf:/app/proxies.conf:ro
      - ./users.conf:/app/users.conf:ro
    env_file:
      - .env
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "nc", "-z", "localhost", "1080"]
      interval: 30s
      timeout: 10s
      retries: 3
```

4. Start the service:
```bash
docker-compose up -d
```

5. Test your connection:
```bash
curl --proxy socks5h://user1:password1@localhost:60255 https://api.ipify.org?format=json
```

## Installation with Go

1. Clone and enter the repository:
```bash
git clone https://github.com/ariadata/go-proxy-rotator.git
cd go-proxy-rotator
```

2. Install dependencies:
```bash
go mod download
```

3. Set up configuration files:
```bash
cp .env.example .env
# Edit users.conf and proxies.conf
```

4. Build and run:
```bash
go build -o proxy-server
./proxy-server
```

## Configuration

### Environment Variables (.env)

```env
# Project name for docker-compose
COMPOSE_PROJECT_NAME=go-proxy-rotator

# Port for the SOCKS5 server
DC_SOCKS_PROXY_PORT=60255

# Enable direct connections when proxies fail
ENABLE_EDGE_MODE=true
```

### User Configuration (users.conf)

Format:
```
username1:password1
username2:password2
# Comments are supported
```

### Proxy Configuration (proxies.conf)

The proxy configuration file supports various proxy formats:

```
# HTTP proxies
http://proxy1.example.com:8080
http://user:password@proxy2.example.com:8080

# HTTPS proxies (encrypted connection to proxy)
https://secure-proxy.example.com:8443
https://user:password@secure-proxy2.example.com:8443

# SOCKS5 proxies (standard)
socks5://socks-proxy.example.com:1080
socks5://user:password@socks-proxy2.example.com:1080

# SOCKS5H proxies (proxy performs DNS resolution)
socks5h://socks-proxy3.example.com:1080
socks5h://user:password@socks-proxy4.example.com:1080

# IPv6 support
http://[2001:db8::1]:8080
socks5://user:password@[2001:db8::2]:1080

# Real-world format examples
http://proxy-user:Abcd1234@103.1.2.3:8080
https://proxy-user:Abcd1234@103.1.2.4:8443
socks5://socks-user:Abcd1234@103.1.2.5:1080
```

## Edge Mode

When edge mode is enabled (`ENABLE_EDGE_MODE=true`), the server will:

1. First attempt a direct connection
2. If direct connection fails, rotate through available proxies
3. If all proxies fail, return an error

This is useful for:
- Accessing both internal and external resources
- Reducing latency for local/fast connections
- Automatic failover to direct connection

## Usage Examples

### With cURL
```bash
# Basic usage
curl --proxy socks5h://user:pass@localhost:60255 https://api.ipify.org?format=json

# With specific DNS resolution
curl --proxy socks5h://user:pass@localhost:60255 https://example.com

# With insecure mode (skip SSL verification)
curl --proxy socks5h://user:pass@localhost:60255 -k https://example.com
```

### With Python Requests
```python
import requests

proxies = {
    'http': 'socks5h://user:pass@localhost:60255',
    'https': 'socks5h://user:pass@localhost:60255'
}

response = requests.get('https://api.ipify.org?format=json', proxies=proxies)
print(response.json())
```

### With Node.js
```javascript
const SocksProxyAgent = require('socks-proxy-agent');

const proxyOptions = {
    hostname: 'localhost',
    port: 60255,
    userId: 'user',
    password: 'pass',
    protocol: 'socks5:'
};

const agent = new SocksProxyAgent(proxyOptions);

fetch('https://api.ipify.org?format=json', { agent })
    .then(res => res.json())
    .then(data => console.log(data));
```

## Building for Production

For production builds, use:

```bash
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o proxy-server .
```

## Security Notes

- Always use strong passwords in `users.conf`
- Consider using HTTPS/SOCKS5 proxies for sensitive traffic
- The server logs minimal information for privacy

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License

## Acknowledgments

Built using:
- [go-socks5](https://github.com/armon/go-socks5) - SOCKS5 server implementation
- Go's standard library for proxy and networking features