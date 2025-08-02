package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"github.com/armon/go-socks5"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

type ProxyConfig struct {
	proxyURL *url.URL
	isDirect bool
}

type ProxyManager struct {
	proxies    []*ProxyConfig
	currentIdx int
	mu         sync.Mutex
	enableEdge bool
}

func NewProxyManager(enableEdge bool) *ProxyManager {
	pm := &ProxyManager{
		proxies:    make([]*ProxyConfig, 0),
		enableEdge: enableEdge,
	}

	if enableEdge {
		pm.proxies = append(pm.proxies, &ProxyConfig{isDirect: true})
	}

	return pm
}

func (pm *ProxyManager) LoadProxies(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		if pm.enableEdge {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		proxyURL, err := url.Parse(line)
		if err != nil {
			return fmt.Errorf("invalid proxy URL: %s", err)
		}

		pm.proxies = append(pm.proxies, &ProxyConfig{
			proxyURL: proxyURL,
			isDirect: false,
		})
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if len(pm.proxies) == 0 && !pm.enableEdge {
		return fmt.Errorf("no proxies loaded and edge mode disabled")
	}

	return nil
}

func (pm *ProxyManager) GetNextProxy() (*ProxyConfig, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.proxies) == 0 {
		return nil, fmt.Errorf("no proxies available")
	}

	proxy := pm.proxies[pm.currentIdx]
	pm.currentIdx = (pm.currentIdx + 1) % len(pm.proxies)
	return proxy, nil
}

type ProxyDialer struct {
	manager *ProxyManager
}

func (d *ProxyDialer) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	proxy, err := d.manager.GetNextProxy()
	if err != nil {
		return nil, err
	}

	if proxy.isDirect {
		return net.Dial(network, addr)
	}

	return d.dialWithProxy(proxy, network, addr)
}

func (d *ProxyDialer) dialWithProxy(proxy *ProxyConfig, network, addr string) (net.Conn, error) {
	switch proxy.proxyURL.Scheme {
	case "socks5", "socks5h":
		return d.dialSocks5(proxy, addr)
	case "http", "https":
		return d.dialHTTP(proxy, network, addr)
	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", proxy.proxyURL.Scheme)
	}
}

func (d *ProxyDialer) dialSocks5(proxy *ProxyConfig, addr string) (net.Conn, error) {
	conn, err := net.Dial("tcp", proxy.proxyURL.Host)
	if err != nil {
		return nil, err
	}

	if proxy.proxyURL.User != nil {
		if err := performSocks5Handshake(conn, proxy.proxyURL); err != nil {
			_ = conn.Close()
			return nil, err
		}
	}

	if err := sendSocks5Connect(conn, addr); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return conn, nil
}

func (d *ProxyDialer) dialHTTP(proxy *ProxyConfig, network, addr string) (net.Conn, error) {
	conn, err := net.Dial("tcp", proxy.proxyURL.Host)
	if err != nil {
		return nil, err
	}

	if proxy.proxyURL.Scheme == "https" {
		tlsConn := tls.Client(conn, &tls.Config{InsecureSkipVerify: true})
		if err := tlsConn.Handshake(); err != nil {
			_ = conn.Close()
			return nil, err
		}
		conn = tlsConn
	}

	req := &http.Request{
		Method: "CONNECT",
		URL:    &url.URL{Opaque: addr},
		Host:   addr,
		Header: make(http.Header),
	}

	if proxy.proxyURL.User != nil {
		basicAuth := base64.StdEncoding.EncodeToString([]byte(proxy.proxyURL.User.String()))
		req.Header.Set("Proxy-Authorization", "Basic "+basicAuth)
	}

	if err := req.Write(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if resp.StatusCode != 200 {
		_ = conn.Close()
		return nil, fmt.Errorf("proxy error: %s", resp.Status)
	}

	return conn, nil
}

func loadUserCredentials(filename string) (socks5.StaticCredentials, error) {
	credentials := make(socks5.StaticCredentials)

	file, err := os.Open(filename)
	if err != nil {
		return credentials, nil // Return empty credentials if file missing
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid credential format: %s", line)
		}
		credentials[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return credentials, nil
}

func main() {
	credentials, err := loadUserCredentials("users.conf")
	if err != nil {
		log.Fatal(err)
	}

	enableEdge := os.Getenv("ENABLE_EDGE_MODE") == "true"

	proxyManager := NewProxyManager(enableEdge)
	if err := proxyManager.LoadProxies("proxies.conf"); err != nil {
		log.Fatal(err)
	}

	conf := &socks5.Config{
		Dial: (&ProxyDialer{manager: proxyManager}).Dial,
	}

	if len(credentials) > 0 {
		conf.Credentials = credentials
		conf.AuthMethods = []socks5.Authenticator{
			socks5.UserPassAuthenticator{Credentials: credentials},
		}
	}

	server, err := socks5.New(conf)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("SOCKS5 server running on :1080 (Auth: %t, Edge Mode: %t, Proxies: %d)",
		len(credentials) > 0, enableEdge, len(proxyManager.proxies))

	if err := server.ListenAndServe("tcp", ":1080"); err != nil {
		log.Fatal(err)
	}
}

func performSocks5Handshake(conn net.Conn, proxyURL *url.URL) error {
	if _, err := conn.Write([]byte{0x05, 0x01, 0x02}); err != nil {
		return err
	}

	resp := make([]byte, 2)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return err
	}

	if resp[0] != 0x05 || resp[1] != 0x02 {
		return fmt.Errorf("unsupported auth method")
	}

	username := proxyURL.User.Username()
	password, _ := proxyURL.User.Password()

	auth := []byte{0x01}
	auth = append(auth, byte(len(username)))
	auth = append(auth, username...)
	auth = append(auth, byte(len(password)))
	auth = append(auth, password...)

	if _, err := conn.Write(auth); err != nil {
		return err
	}

	authResp := make([]byte, 2)
	if _, err := io.ReadFull(conn, authResp); err != nil {
		return err
	}

	if authResp[1] != 0x00 {
		return fmt.Errorf("authentication failed")
	}

	return nil
}

func sendSocks5Connect(conn net.Conn, addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}

	req := []byte{0x05, 0x01, 0x00}
	ip := net.ParseIP(host)

	switch {
	case ip == nil:
		req = append(req, 0x03, byte(len(host)))
		req = append(req, host...)
	case ip.To4() != nil:
		req = append(req, 0x01)
		req = append(req, ip.To4()...)
	default:
		req = append(req, 0x04)
		req = append(req, ip.To16()...)
	}

	portNum := uint16(0)
	fmt.Sscanf(port, "%d", &portNum)
	req = append(req, byte(portNum>>8), byte(portNum&0xff))

	if _, err := conn.Write(req); err != nil {
		return err
	}

	resp := make([]byte, 4)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return err
	}

	if resp[1] != 0x00 {
		return fmt.Errorf("connect failed: %d", resp[1])
	}

	switch resp[3] {
	case 0x01:
		_, err = io.ReadFull(conn, make([]byte, 6))
	case 0x03:
		size := make([]byte, 1)
		if _, err = io.ReadFull(conn, size); err == nil {
			_, err = io.ReadFull(conn, make([]byte, int(size[0])+2))
		}
	case 0x04:
		_, err = io.ReadFull(conn, make([]byte, 18))
	}

	return err
}
