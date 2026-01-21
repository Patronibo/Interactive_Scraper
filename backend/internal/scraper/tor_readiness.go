package scraper

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// TorReadinessStatus represents the readiness state of Tor
type TorReadinessStatus struct {
	IsReady      bool
	BootstrapPct int    // Bootstrap percentage (0-100)
	Message      string
}

// CheckTorReadiness checks if Tor is fully ready (SOCKS5 port + bootstrap complete)
// Returns true only when Tor is ready to handle requests
func CheckTorReadiness() (*TorReadinessStatus, error) {
	torProxy := os.Getenv("TOR_PROXY")
	if torProxy == "" {
		torProxy = "tor:9050"
	}

	// Split host:port
	host, port, err := net.SplitHostPort(torProxy)
	if err != nil {
		// If no port, assume default
		host = torProxy
		port = "9050"
	}

	// Step 1: Check if SOCKS5 port is reachable
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 3*time.Second)
	if err != nil {
		return &TorReadinessStatus{
			IsReady:      false,
			BootstrapPct: 0,
			Message:      fmt.Sprintf("SOCKS5 port unreachable at %s:%s: %v", host, port, err),
		}, nil
	}
	conn.Close()

	// Step 2: Verify Tor can actually route traffic (bootstrap check)
	// We do this by attempting a simple connection through Tor
	client, err := GetTorHTTPClient()
	if err != nil {
		return &TorReadinessStatus{
			IsReady:      false,
			BootstrapPct: 0,
			Message:      fmt.Sprintf("Failed to create Tor client: %v", err),
		}, nil
	}

	// Try a lightweight test request to verify Tor is routing
	// Use a simple endpoint that responds quickly
	testURL := "http://icanhazip.com"
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return &TorReadinessStatus{
			IsReady:      false,
			BootstrapPct: 0,
			Message:      fmt.Sprintf("Failed to create test request: %v", err),
		}, nil
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		// Check if error suggests bootstrap in progress
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "connection refused") ||
			strings.Contains(errStr, "no route to host") ||
			strings.Contains(errStr, "timeout") {
			return &TorReadinessStatus{
				IsReady:      false,
				BootstrapPct: 50, // Estimated - bootstrap in progress
				Message:      fmt.Sprintf("Tor bootstrap in progress: %v", err),
			}, nil
		}
		return &TorReadinessStatus{
			IsReady:      false,
			BootstrapPct: 0,
			Message:      fmt.Sprintf("Tor routing test failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	// If we got a response, Tor is ready
	if resp.StatusCode == 200 {
		return &TorReadinessStatus{
			IsReady:      true,
			BootstrapPct: 100,
			Message:      "Tor is ready and routing traffic",
		}, nil
	}

	return &TorReadinessStatus{
		IsReady:      false,
		BootstrapPct: 75,
		Message:      fmt.Sprintf("Tor responded but with status %d", resp.StatusCode),
	}, nil
}

// WaitForTorReady waits for Tor to become ready with exponential backoff
// Returns error only if max retries exceeded
func WaitForTorReady(maxRetries int, initialDelay time.Duration) error {
	log.Printf("[TOR] Waiting for Tor to become ready...")
	
	delay := initialDelay
	for attempt := 1; attempt <= maxRetries; attempt++ {
		status, err := CheckTorReadiness()
		if err != nil {
			log.Printf("[TOR] Error checking readiness (attempt %d/%d): %v", attempt, maxRetries, err)
		} else if status.IsReady {
			log.Printf("[TOR] ✓ Tor is ready! Bootstrap: %d%%, Message: %s", status.BootstrapPct, status.Message)
			return nil
		} else {
			log.Printf("[TOR] Tor not ready yet (attempt %d/%d): Bootstrap: %d%%, Message: %s", 
				attempt, maxRetries, status.BootstrapPct, status.Message)
		}

		if attempt < maxRetries {
			log.Printf("[TOR] Retrying in %v...", delay)
			time.Sleep(delay)
			// Exponential backoff: 3s, 5s, 8s, 12s, etc.
			delay = time.Duration(float64(delay) * 1.5)
			if delay > 15*time.Second {
				delay = 15 * time.Second // Cap at 15 seconds
			}
		}
	}

	return fmt.Errorf("Tor did not become ready after %d attempts", maxRetries)
}

