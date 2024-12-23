package main

import (
	"github.com/armon/go-socks5"
	"go-proxy-rotator/internal/config"
	"go-proxy-rotator/internal/proxy_dialer"
	"go-proxy-rotator/internal/proxy_manager"
	"log"
)

func main() {
	// Load configuration
	cfg := config.NewConfig()

	// Load user credentials
	credentials, err := config.LoadUserCredentials(cfg.UsersFile)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize proxy manager
	proxyManager := proxy_manager.NewManager(cfg.EnableEdgeMode)
	if err := proxyManager.LoadProxies(cfg.ProxiesFile); err != nil {
		log.Fatal(err)
	}

	// Initialize proxy dialer
	dialer := proxy_dialer.NewProxyDialer(proxyManager)

	// Create SOCKS5 server configuration with authentication
	serverConfig := &socks5.Config{
		Dial:        dialer.Dial,
		Credentials: credentials,
		AuthMethods: []socks5.Authenticator{socks5.UserPassAuthenticator{
			Credentials: credentials,
		}},
	}

	server, err := socks5.New(serverConfig)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("SOCKS5 server running on %s (Edge Mode: %v, Users: %d)\n",
		cfg.ListenAddr,
		cfg.EnableEdgeMode,
		len(credentials))

	// Start server
	if err := server.ListenAndServe("tcp", cfg.ListenAddr); err != nil {
		log.Fatal(err)
	}
}
