package internal

import (
	"compress/gzip"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"math/rand"

	"github.com/HeapOfChaos/goondvr/server"
)

// Req represents an HTTP client with customized settings.
type Req struct {
	client  *http.Client
	isMedia bool   // when true, omits browser-spoofing headers not needed for CDN media requests
	referer string // CDN Referer/Origin override; only used when isMedia is true
}

// NewReq creates a new HTTP client for Chaturbate page requests.
func NewReq() *Req {
	return &Req{
		client: &http.Client{
			Transport: CreateTransport(),
		},
	}
}

// NewMediaReq creates a new HTTP client for CDN media requests (playlists, segments).
// It omits headers like X-Requested-With that are only needed for Chaturbate page fetches.
func NewMediaReq() *Req {
	return &Req{
		client: &http.Client{
			Transport: CreateTransport(),
		},
		isMedia: true,
	}
}

// NewMediaReqWithReferer creates a media HTTP client that sends the given URL as
// Referer and Origin instead of the Chaturbate defaults. Use this for non-Chaturbate CDNs.
func NewMediaReqWithReferer(referer string) *Req {
	return &Req{
		client: &http.Client{
			Transport: CreateTransport(),
		},
		isMedia: true,
		referer: referer,
	}
}

// CreateTransport initializes a custom HTTP transport.
func CreateTransport() *http.Transport {
	// The DefaultTransport allows user changes the proxy settings via environment variables
	// such as HTTP_PROXY, HTTPS_PROXY.
	defaultTransport := http.DefaultTransport.(*http.Transport)

	newTransport := defaultTransport.Clone()
	newTransport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
		// Add more TLS settings to appear more like a real browser
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}
	// Optimize connection settings to avoid being detected as bot
	newTransport.MaxIdleConns = 10
	newTransport.MaxIdleConnsPerHost = 2
	newTransport.IdleConnTimeout = 30 * time.Second
	newTransport.TLSHandshakeTimeout = 15 * time.Second
	newTransport.ExpectContinueTimeout = 1 * time.Second
	newTransport.ResponseHeaderTimeout = 15 * time.Second
	newTransport.DisableKeepAlives = false
	newTransport.DisableCompression = false // Let Go handle compression automatically
	
	// Support proxy from environment variables (HTTP_PROXY, HTTPS_PROXY)
	// This allows using VPN or proxy services
	if server.Config != nil && server.Config.Debug {
		if proxy := newTransport.Proxy; proxy != nil {
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			if proxyURL, _ := proxy(req); proxyURL != nil {
				fmt.Printf("[DEBUG] Using proxy: %s\n", proxyURL.String())
			}
		}
	}
	
	return newTransport
}

// Get sends an HTTP GET request and returns the response as a string.
func (h *Req) Get(ctx context.Context, url string) (string, error) {
	resp, err := h.GetBytes(ctx, url)
	if err != nil {
		return "", fmt.Errorf("get bytes: %w", err)
	}
	return string(resp), nil
}

// PostJSON sends an HTTP POST request with JSON body and returns the response as a string.
func (h *Req) PostJSON(ctx context.Context, url string, jsonBody string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}
	
	h.SetRequestHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	
	return string(body), nil
}

// GetBytes sends an HTTP GET request and returns the response as a byte slice.
func (h *Req) GetBytes(ctx context.Context, url string) ([]byte, error) {
	req, cancel, err := h.CreateRequest(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	defer cancel()

	if server.Config.Debug && h.isMedia && strings.Contains(url, ".m3u8") {
		fmt.Printf("[DEBUG] HLS request to: %s\n", url)
		fmt.Printf("[DEBUG]   Cookies: %s\n", req.Header.Get("Cookie"))
		fmt.Printf("[DEBUG]   User-Agent: %s\n", req.Header.Get("User-Agent"))
		fmt.Printf("[DEBUG]   Referer: %s\n", req.Header.Get("Referer"))
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client do: %w", err)
	}
	defer resp.Body.Close()

	if server.Config.Debug && resp.StatusCode >= 400 {
		fmt.Printf("[DEBUG] HTTP %d: %s\n", resp.StatusCode, req.URL)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	// Handle gzip compression
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	b, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	// Check for Cloudflare protection
	if strings.Contains(string(b), "<title>Just a moment...</title>") {
		if server.Config.Debug {
			fmt.Printf("[DEBUG] CF response for %s (status %d)\n", req.URL, resp.StatusCode)
			tmpFile, ferr := os.CreateTemp("", "chaturbate-debug-cf-*.html")
			if ferr == nil {
				if _, werr := tmpFile.Write(b); werr == nil {
					fmt.Printf("[DEBUG]   Full body written to: %s\n", tmpFile.Name())
				}
				tmpFile.Close()
			}
		}
		return nil, ErrCloudflareBlocked
	}
	// Check for Age Verification
	if strings.Contains(string(b), "Verify your age") {
		return nil, ErrAgeVerification
	}

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("forbidden: %w", ErrPrivateStream)
	}

	return b, err
}

// CreateRequest constructs an HTTP GET request with necessary headers.
func (h *Req) CreateRequest(ctx context.Context, url string) (*http.Request, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second) // increased timeout to 45 seconds

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	h.SetRequestHeaders(req)
	
	// Add small random delay to avoid being detected as bot
	if !h.isMedia {
		delay := time.Duration(500+(time.Now().UnixNano()%1000)) * time.Millisecond
		select {
		case <-ctx.Done():
			cancel()
			return nil, nil, ctx.Err()
		case <-time.After(delay):
		}
	}
	
	return req, cancel, nil
}

// DoRequest executes an already-constructed *http.Request and returns the
// response body as a string. This allows callers to set extra headers on the
// request before executing it (e.g. site-specific Referer or X-Requested-With).
func (h *Req) DoRequest(req *http.Request) (string, error) {
	resp, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("client do: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	// Check for Cloudflare protection
	if strings.Contains(string(b), "<title>Just a moment...</title>") {
		return "", ErrCloudflareBlocked
	}
	// Check for Age Verification
	if strings.Contains(string(b), "Verify your age") {
		return "", ErrAgeVerification
	}

	if resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("forbidden: %w", ErrPrivateStream)
	}

	return string(b), nil
}

// getRandomUserAgent returns a random user agent to avoid detection
func getRandomUserAgent() string {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:132.0) Gecko/20100101 Firefox/132.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.1 Safari/605.1.15",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	}
	return userAgents[rand.Intn(len(userAgents))]
}

// SetRequestHeaders applies necessary headers to the request.
func (h *Req) SetRequestHeaders(req *http.Request) {
	if h.isMedia {
		ref := h.referer
		if ref == "" {
			ref = "https://chaturbate.com/"
		}
		req.Header.Set("Referer", ref)
		req.Header.Set("Origin", strings.TrimRight(ref, "/"))
	} else {
		// X-Requested-With helps bypass Cloudflare on chaturbate.com page fetches.
		// Do NOT send it to CDN media hosts (mmcdn.com) as it may cause rejection.
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
	}
	
	// Set comprehensive browser-like headers to avoid detection
	userAgent := server.Config.UserAgent
	if userAgent == "" {
		// Use a random user agent to avoid detection patterns
		userAgent = getRandomUserAgent()
	}
	// Clean the user agent string to remove any invalid characters
	userAgent = strings.TrimSpace(strings.ReplaceAll(userAgent, "\n", ""))
	userAgent = strings.ReplaceAll(userAgent, "\r", "")
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	
	// Add more browser-like headers
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	
	if server.Config.Cookies != "" {
		cookies := ParseCookies(server.Config.Cookies)
		for name, value := range cookies {
			req.AddCookie(&http.Cookie{Name: name, Value: value})
		}
	}
}

// ParseCookies converts a cookie string into a map.
func ParseCookies(cookieStr string) map[string]string {
	cookies := make(map[string]string)
	pairs := strings.Split(cookieStr, ";")

	// Iterate over each cookie pair and extract key-value pairs
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) == 2 {
			// Trim spaces around key and value
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Store cookie name and value in the map
			cookies[key] = value
		}
	}
	return cookies
}
