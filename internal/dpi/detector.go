package dpi

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type DomainStatus int

const (
	StatusUnknown DomainStatus = iota
	StatusAvailable
	StatusBlocked
	StatusChecking
)

type DomainInfo struct {
	Domain       string
	Status       DomainStatus
	LastChecked  time.Time
	ResponseTime time.Duration
	Error        error
}

type AvailabilityDetector struct {
	cache    map[string]*DomainInfo
	cacheTTL time.Duration
	timeout  time.Duration
	mu       sync.RWMutex
	client   *http.Client
}

func NewAvailabilityDetector() *AvailabilityDetector {
	return &AvailabilityDetector{
		cache:    make(map[string]*DomainInfo),
		cacheTTL: 5 * time.Minute, // Cache results for 5 minutes
		timeout:  3 * time.Second, // 3 second timeout for checks
		client: &http.Client{
			Timeout: 3 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects
			},
		},
	}
}

func (d *AvailabilityDetector) CheckDomain(ctx context.Context, domain string) DomainStatus {
	d.mu.RLock()
	if info, exists := d.cache[domain]; exists {
		if time.Since(info.LastChecked) < d.cacheTTL {
			d.mu.RUnlock()
			return info.Status
		}
	}
	d.mu.RUnlock()

	// Need to check
	status := d.performCheck(ctx, domain)

	// Cache result
	d.mu.Lock()
	d.cache[domain] = &DomainInfo{
		Domain:       domain,
		Status:       status,
		LastChecked:  time.Now(),
		ResponseTime: 0, // Will be set by performCheck
	}
	d.mu.Unlock()

	return status
}

func (d *AvailabilityDetector) performCheck(ctx context.Context, domain string) DomainStatus {
	// Try multiple methods to check availability

	// Method 1: Direct TCP connection to port 80
	if d.checkTCPConnection(ctx, domain, 80) {
		return StatusAvailable
	}

	// Method 2: Direct TCP connection to port 443
	if d.checkTCPConnection(ctx, domain, 443) {
		return StatusAvailable
	}

	// Method 3: HTTP request to check for blocking
	if d.checkHTTPRequest(ctx, domain) {
		return StatusAvailable
	}

	return StatusBlocked
}

func (d *AvailabilityDetector) checkTCPConnection(ctx context.Context, domain string, port int) bool {
	address := fmt.Sprintf("%s:%d", domain, port)

	dialer := &net.Dialer{
		Timeout: d.timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return false
	}
	defer conn.Close()

	return true
}

func (d *AvailabilityDetector) checkHTTPRequest(ctx context.Context, domain string) bool {
	urls := []string{
		fmt.Sprintf("http://%s/", domain),
		fmt.Sprintf("https://%s/", domain),
	}

	for _, url := range urls {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		// Set common headers to avoid blocking
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

		resp, err := d.client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()

		// Check if we got a valid response (not a blocking page)
		if resp.StatusCode == 200 || resp.StatusCode == 301 || resp.StatusCode == 302 {
			return true
		}

		// Check for common blocking indicators
		if d.isBlockingResponse(resp) {
			return false
		}
	}

	return false
}

func (d *AvailabilityDetector) isBlockingResponse(resp *http.Response) bool {
	// Check status codes that might indicate blocking
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return true
	}

	// This is a simplified check - in real implementation, you might want
	// to analyze response body for blocking pages
	return false
}

func (d *AvailabilityDetector) IsDomainAvailable(domain string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if info, exists := d.cache[domain]; exists {
		return info.Status == StatusAvailable
	}

	return false // Unknown, assume blocked for safety
}

func (d *AvailabilityDetector) ClearCache() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.cache = make(map[string]*DomainInfo)
}

func (d *AvailabilityDetector) GetCachedInfo(domain string) *DomainInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if info, exists := d.cache[domain]; exists {
		return info
	}

	return nil
}

// Extract domain from hostname (remove port if present)
func (d *AvailabilityDetector) ExtractDomain(host string) string {
	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}
	return host
}
