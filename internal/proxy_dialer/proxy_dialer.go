package proxy_dialer

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
)

type ProxyConfig struct {
	ProxyURL *url.URL
}

type ProxyManager interface {
	GetNextProxy() (*ProxyConfig, error)
	ShouldUseDirect() bool
	HasProxies() bool
	IsEdgeEnabled() bool
}

type ProxyDialer struct {
	manager ProxyManager
}

func NewProxyDialer(manager ProxyManager) *ProxyDialer {
	return &ProxyDialer{
		manager: manager,
	}
}

func (d *ProxyDialer) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	var lastError error

	if d.manager.ShouldUseDirect() {
		conn, err := net.Dial(network, addr)
		if err == nil {
			return conn, nil
		}
		lastError = err
	}

	if d.manager.HasProxies() {
		proxy, err := d.manager.GetNextProxy()
		if err != nil {
			if lastError != nil {
				return nil, fmt.Errorf("direct connection failed: %v, proxy error: %v", lastError, err)
			}
			return nil, err
		}

		conn, err := d.dialWithProxy(proxy, network, addr)
		if err == nil {
			return conn, nil
		}
		lastError = err
	} else if !d.manager.IsEdgeEnabled() {
		return nil, fmt.Errorf("no proxies available and edge mode is disabled")
	}

	if lastError != nil {
		return nil, fmt.Errorf("all connection attempts failed, last error: %v", lastError)
	}
	return nil, fmt.Errorf("no connection methods available")
}

func (d *ProxyDialer) dialWithProxy(proxy *ProxyConfig, network, addr string) (net.Conn, error) {
	switch proxy.ProxyURL.Scheme {
	case "socks5", "socks5h":
		return d.dialSocks5(proxy, addr)
	case "http", "https":
		return d.dialHTTP(proxy, network, addr)
	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", proxy.ProxyURL.Scheme)
	}
}

func (d *ProxyDialer) dialSocks5(proxy *ProxyConfig, addr string) (net.Conn, error) {
	conn, err := net.Dial("tcp", proxy.ProxyURL.Host)
	if err != nil {
		return nil, err
	}

	if proxy.ProxyURL.User != nil {
		err = performSocks5Handshake(conn, proxy.ProxyURL)
		if err != nil {
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
	conn, err := net.Dial("tcp", proxy.ProxyURL.Host)
	if err != nil {
		return nil, err
	}

	if proxy.ProxyURL.Scheme == "https" {
		tlsConn := tls.Client(conn, &tls.Config{
			InsecureSkipVerify: true,
		})
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

	if proxy.ProxyURL.User != nil {
		basicAuth := base64.StdEncoding.EncodeToString([]byte(proxy.ProxyURL.User.String()))
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

func performSocks5Handshake(conn net.Conn, proxyURL *url.URL) error {
	_, err := conn.Write([]byte{0x05, 0x01, 0x02})
	if err != nil {
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
	auth = append(auth, []byte(username)...)
	auth = append(auth, byte(len(password)))
	auth = append(auth, []byte(password)...)

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

	if ip == nil {
		req = append(req, 0x03, byte(len(host)))
		req = append(req, []byte(host)...)
	} else if ip4 := ip.To4(); ip4 != nil {
		req = append(req, 0x01)
		req = append(req, ip4...)
	} else {
		req = append(req, 0x04)
		req = append(req, ip.To16()...)
	}

	portNum := uint16(0)
	_, _ = fmt.Sscanf(port, "%d", &portNum)
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
		_, err = io.ReadFull(conn, make([]byte, 4+2))
	case 0x03:
		size := make([]byte, 1)
		_, err = io.ReadFull(conn, size)
		if err == nil {
			_, err = io.ReadFull(conn, make([]byte, int(size[0])+2))
		}
	case 0x04:
		_, err = io.ReadFull(conn, make([]byte, 16+2))
	}

	return err
}
