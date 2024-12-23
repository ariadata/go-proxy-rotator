package config

import (
	"bufio"
	"fmt"
	"github.com/armon/go-socks5"
	"os"
	"strings"
)

type Config struct {
	EnableEdgeMode bool
	ProxiesFile    string
	UsersFile      string
	ListenAddr     string
}

func NewConfig() *Config {
	return &Config{
		EnableEdgeMode: os.Getenv("ENABLE_EDGE_MODE") == "true",
		ProxiesFile:    "proxies.conf",
		UsersFile:      "users.conf",
		ListenAddr:     ":1080",
	}
}

func LoadUserCredentials(filename string) (socks5.StaticCredentials, error) {
	credentials := make(socks5.StaticCredentials)

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid credentials format in users.conf: %s", line)
		}
		credentials[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(credentials) == 0 {
		return nil, fmt.Errorf("no valid credentials found in users.conf")
	}

	return credentials, nil
}
