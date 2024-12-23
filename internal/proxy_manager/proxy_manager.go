package proxy_manager

import (
	"bufio"
	"fmt"
	"go-proxy-rotator/internal/proxy_dialer"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type Manager struct {
	proxies    []*proxy_dialer.ProxyConfig
	currentIdx int
	mu         sync.Mutex
	enableEdge bool
	lastUsed   time.Time
}

func NewManager(enableEdge bool) *Manager {
	return &Manager{
		proxies:    make([]*proxy_dialer.ProxyConfig, 0),
		enableEdge: enableEdge,
		lastUsed:   time.Now(),
	}
}

func (pm *Manager) LoadProxies(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		if pm.enableEdge {
			return nil
		}
		return err
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

		proxyURL, err := url.Parse(line)
		if err != nil {
			return fmt.Errorf("invalid proxy URL: %s", err)
		}

		pm.proxies = append(pm.proxies, &proxy_dialer.ProxyConfig{ProxyURL: proxyURL})
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if len(pm.proxies) == 0 && !pm.enableEdge {
		return fmt.Errorf("no proxies loaded from configuration and edge mode is disabled")
	}

	return nil
}

func (pm *Manager) GetNextProxy() (*proxy_dialer.ProxyConfig, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.proxies) == 0 {
		return nil, fmt.Errorf("no proxies available")
	}

	proxy := pm.proxies[pm.currentIdx]
	pm.currentIdx = (pm.currentIdx + 1) % len(pm.proxies)
	pm.lastUsed = time.Now()

	return proxy, nil
}

func (pm *Manager) ShouldUseDirect() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.enableEdge {
		return false
	}

	if len(pm.proxies) == 0 {
		return true
	}

	return time.Since(pm.lastUsed) > 5*time.Second
}

func (pm *Manager) HasProxies() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return len(pm.proxies) > 0
}

func (pm *Manager) IsEdgeEnabled() bool {
	return pm.enableEdge
}

func (pm *Manager) ProxyCount() int {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return len(pm.proxies)
}
