package scraper

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

var (
	torClientCache     *http.Client
	torClientCacheMutex sync.RWMutex
	torClientCacheTime time.Time
	torClientCacheTTL  = 5 * time.Minute
)

/*Bu GetTorHTTPClient fonksiyonu, Tor ağı üzerinden HTTP istekleri gönderebilen bir 
http.Client oluşturuyor ve performans için bir önbellekleme mekanizması kullanıyor; 
önce önbellekte geçerli bir client var mı kontrol ediyor, varsa onu döndürüyor, 
yoksa SOCKS5 proxy (varsayılan tor:9050 veya TOR_PROXY ortam değişkeni) üzerinden 
yeni bir client yaratıyor ve önbelleğe kaydediyor. Böylece Tor üzerinden güvenli ve 
anonim HTTP istekleri yapmayı sağlıyor ve tekrar tekrar client yaratma maliyetini önlüyor.
*/
func GetTorHTTPClient() (*http.Client, error) {
	torClientCacheMutex.RLock()
	if torClientCache != nil && time.Since(torClientCacheTime) < torClientCacheTTL {
		client := torClientCache
		torClientCacheMutex.RUnlock()
		return client, nil
	}
	torClientCacheMutex.RUnlock()

	torProxy := os.Getenv("TOR_PROXY")
	if torProxy == "" {
		torProxy = "tor:9050"
	}
	dialer, err := proxy.SOCKS5("tcp", torProxy, nil, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("failed to create Tor dialer: %v", err)
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   5,
		IdleConnTimeout:        90 * time.Second,
		DisableKeepAlives:     false,
		DisableCompression:    false,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 5 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}
	torClientCacheMutex.Lock()
	torClientCache = client
	torClientCacheTime = time.Now()
	torClientCacheMutex.Unlock()

	return client, nil
}

/*Bu TestTorConnection fonksiyonu, Tor ağı üzerinden HTTP isteklerinin çalışıp 
çalışmadığını kontrol ediyor; önce GetTorHTTPClient() ile Tor üzerinden bağlanabilen bir 
HTTP client alıyor, sonra http://check.torproject.org adresine GET isteği gönderiyor ve
eğer bağlantı sağlanamazsa veya istek başarısız olursa uygun hata mesajı döndürüyor. 
Bu şekilde sistemde Tor servisinin çalışıp çalışmadığını ve isteklerin Tor üzerinden 
gidip gitmediğini hızlıca test edebiliyorsun.
*/
func TestTorConnection() error {
	client, err := GetTorHTTPClient()
	if err != nil {
		return fmt.Errorf("failed to create Tor client: %v", err)
	}

	testURL := "http://check.torproject.org"
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create test request: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("Tor connection test failed: %v. Make sure Tor is running", err)
	}
	defer resp.Body.Close()

	return nil
}

func FetchURL(urlString string) (string, error) {
	client, err := GetTorHTTPClient()
	if err != nil {
		return "", err
	}

	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %v", err)
	}

	req, err := http.NewRequest("GET", parsedURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "" {
		isTextContent := strings.HasPrefix(contentType, "text/html") ||
			strings.HasPrefix(contentType, "text/plain") ||
			strings.HasPrefix(contentType, "application/xhtml") ||
			strings.HasPrefix(contentType, "application/xml")
		
		if !isTextContent {
			log.Printf("[SCRAPER] WARNING: Rejecting non-text content type: %s from %s", contentType, urlString)
			return "", fmt.Errorf("unsupported content type: %s (only text/html accepted)", contentType)
		}
	}

	// Read response body
	body := make([]byte, 0)
	buffer := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			body = append(body, buffer[:n]...)
		}
		if err != nil {
			break
		}
	}

		return string(body), nil
}

func FetchURLWithRetry(urlString string, maxRetries int, retryDelay time.Duration) (string, error) {
	var lastErr error
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		content, err := FetchURL(urlString)
		if err == nil {
			if attempt > 1 {
				log.Printf("[TOR] Fetch succeeded on attempt %d/%d for %s", attempt, maxRetries, urlString)
			}
			return content, nil
		}
		
		lastErr = err
		
		errStr := err.Error()
		isRetryable := strings.Contains(errStr, "timeout") ||
			strings.Contains(errStr, "connection") ||
			strings.Contains(errStr, "refused") ||
			strings.Contains(errStr, "network") ||
			strings.Contains(errStr, "temporary")
		
		if !isRetryable && attempt < maxRetries {
			log.Printf("[TOR] Non-retryable error on attempt %d/%d for %s: %v", attempt, maxRetries, urlString, err)
		} else if attempt < maxRetries {
			log.Printf("[TOR] Fetch failed on attempt %d/%d for %s: %v. Retrying in %v...", 
				attempt, maxRetries, urlString, err, retryDelay)
		}
		
		if attempt < maxRetries {
			status, _ := CheckTorReadiness()
			if !status.IsReady {
				log.Printf("[TOR] Tor not ready, waiting before retry...")
				time.Sleep(retryDelay * 2)
			} else {
				time.Sleep(retryDelay)
			}
		}
	}
	
	return "", fmt.Errorf("failed to fetch after %d attempts: %v", maxRetries, lastErr)
}

