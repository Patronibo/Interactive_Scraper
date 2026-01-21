package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type TorStatus struct {
	IsConnected bool   `json:"is_connected"`
	ExitIP      string `json:"exit_ip"`
	Message     string `json:"message"`
}

var (
	torStatusCache      *TorStatus
	torStatusCacheMutex sync.RWMutex
	torStatusCacheTime  time.Time
	torStatusCacheTTLDisconnected = 1 * time.Second
	torStatusCacheTTLConnected    = 15 * time.Second
)

func CheckTorStatus() (*TorStatus, error) {
	torStatusCacheMutex.RLock()
	if torStatusCache != nil {
		ttl := torStatusCacheTTLDisconnected
		if torStatusCache.IsConnected {
			ttl = torStatusCacheTTLConnected
		}
		if time.Since(torStatusCacheTime) < ttl {
			cached := *torStatusCache
			torStatusCacheMutex.RUnlock()
			return &cached, nil
		}
	}
	torStatusCacheMutex.RUnlock()

	torProxy := os.Getenv("TOR_PROXY")
	if torProxy == "" {
		torProxy = "tor:9050"
	}
	host, port := torProxy, "9050"
	if h, p, err := net.SplitHostPort(torProxy); err == nil {
		host, port = h, p
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 1*time.Second)
	if err != nil {
		status := &TorStatus{
			IsConnected: false,
			ExitIP:      "",
			Message:     fmt.Sprintf("Tor SOCKS5 port unreachable at %s:%s: %v", host, port, err),
		}
		torStatusCacheMutex.Lock()
		torStatusCache = status
		torStatusCacheTime = time.Now()
		torStatusCacheMutex.Unlock()
		return status, nil
	}
	_ = conn.Close()

	client, err := GetTorHTTPClient()
	if err != nil {
		return &TorStatus{
			IsConnected: false,
			ExitIP:      "",
			Message:     fmt.Sprintf("Tor proxy connection failed (%s). Make sure Tor is running.", torProxy),
		}, nil
	}

	var lastError error
	var mu sync.Mutex
	ipCh := make(chan string, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	validateIP := func(ip string) bool {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			return false
		}
		ip = strings.ReplaceAll(ip, "\n", "")
		ip = strings.ReplaceAll(ip, "\r", "")
		ip = strings.ReplaceAll(ip, "\"", "")
		ip = strings.ReplaceAll(ip, "'", "")
		ip = strings.TrimSpace(ip)
		
		if strings.Contains(ip, ".") {
			parts := strings.Split(ip, ".")
			if len(parts) == 4 {
				for _, part := range parts {
					if len(part) == 0 || len(part) > 3 {
						return false
					}
					for _, c := range part {
						if c < '0' || c > '9' {
							return false
						}
					}
				}
				return true
			}
		}
		if strings.Contains(ip, ":") {
			return len(ip) > 2
		}
		return false
	}

	tryPlainIP := func(url string) {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Accept", "*/*")
		resp, err := client.Do(req.WithContext(ctx))
		if err != nil {
			mu.Lock()
			if lastError == nil {
				lastError = err
			}
			mu.Unlock()
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 50))
		if err != nil {
			return
		}
		ip := strings.TrimSpace(string(body))
		if validateIP(ip) {
			select {
			case ipCh <- ip:
			default:
			}
		}
	}

	tryJSONIP := func(url string, jsonField string) {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Accept", "application/json")
		resp, err := client.Do(req.WithContext(ctx))
		if err != nil {
			mu.Lock()
			if lastError == nil {
				lastError = err
			}
			mu.Unlock()
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 500))
		if err != nil {
			return
		}
		
		var ipResp map[string]interface{}
		if json.Unmarshal(body, &ipResp) == nil {
			fields := []string{jsonField, "ip", "IP", "origin", "Origin", "query", "Query"}
			for _, field := range fields {
				if val, ok := ipResp[field]; ok {
					if ipStr, ok := val.(string); ok {
						ip := strings.TrimSpace(ipStr)
						if validateIP(ip) {
							select {
							case ipCh <- ip:
							default:
							}
							return
						}
					}
				}
			}
		}
	}

	go tryPlainIP("http://icanhazip.com")
	go tryPlainIP("http://ifconfig.me/ip")
	go tryPlainIP("http://ipinfo.io/ip")
	go tryPlainIP("http://api.ipify.org")
	go tryPlainIP("http://checkip.amazonaws.com")
	go tryPlainIP("http://ipecho.net/plain")
	go tryPlainIP("http://ident.me")
	go tryJSONIP("http://api.ipify.org?format=json", "ip")
	go tryJSONIP("https://api.ipify.org?format=json", "ip")
	go tryJSONIP("https://check.torproject.org/api/ip", "IP")

	var exitIP string
	select {
	case exitIP = <-ipCh:
		exitIP = strings.TrimSpace(exitIP)
		exitIP = strings.ReplaceAll(exitIP, "\n", "")
		exitIP = strings.ReplaceAll(exitIP, "\r", "")
		exitIP = strings.ReplaceAll(exitIP, "\"", "")
		exitIP = strings.ReplaceAll(exitIP, "'", "")
		exitIP = strings.TrimSpace(exitIP)
		if exitIP == "" {
			fmt.Printf("[TOR] Warning: Received empty IP from service\n")
		} else {
			fmt.Printf("[TOR] Successfully retrieved exit IP: %s\n", exitIP)
		}
	case <-ctx.Done():
		exitIP = ""
		fmt.Printf("[TOR] Timeout: No IP received from any service after 10 seconds. Last error: %v\n", lastError)
	}
	
	isConnected := exitIP != ""
	
	message := "Tor connection active"
	if !isConnected {
		torProxy := os.Getenv("TOR_PROXY")
		if torProxy == "" {
			torProxy = "tor:9050"
		}
		if lastError != nil {
			message = fmt.Sprintf("Tor connected but IP check failed: %v. Tor proxy: %s", lastError, torProxy)
		} else {
			message = fmt.Sprintf("Tor proxy active but unable to retrieve exit IP. Tor proxy: %s", torProxy)
		}
	}

	status := &TorStatus{
		IsConnected: isConnected,
		ExitIP:      exitIP,
		Message:     message,
	}

	torStatusCacheMutex.Lock()
	torStatusCache = status
	torStatusCacheTime = time.Now()
	torStatusCacheMutex.Unlock()

	return status, nil
}

