package internal

import (
	"context"
	"fmt"
	"strings"

	"github.com/Danny-Dasilva/CycleTLS/cycletls"
	"github.com/HeapOfChaos/goondvr/server"
)

// CycleTLSReq wraps CycleTLS client for browser-like requests
type CycleTLSReq struct {
	client cycletls.CycleTLS
}

// NewCycleTLSReq creates a new CycleTLS-based HTTP client
func NewCycleTLSReq() *CycleTLSReq {
	client := cycletls.Init()
	return &CycleTLSReq{client: client}
}

// Get sends an HTTP GET request using CycleTLS with proper browser fingerprinting
func (c *CycleTLSReq) Get(ctx context.Context, url string) (string, error) {
	// Build headers
	headers := make(map[string]string)
	
	// Set user agent
	userAgent := server.Config.UserAgent
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	}
	headers["User-Agent"] = strings.TrimSpace(userAgent)
	
	// Add browser-like headers
	headers["Accept"] = "application/json, text/plain, */*"
	headers["Accept-Language"] = "en-US,en;q=0.9"
	headers["Cache-Control"] = "no-cache"
	headers["Pragma"] = "no-cache"
	headers["Sec-Fetch-Dest"] = "empty"
	headers["Sec-Fetch-Mode"] = "cors"
	headers["Sec-Fetch-Site"] = "same-origin"
	headers["Sec-Ch-Ua"] = `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`
	headers["Sec-Ch-Ua-Mobile"] = "?0"
	headers["Sec-Ch-Ua-Platform"] = `"Windows"`
	headers["X-Requested-With"] = "XMLHttpRequest"
	
	// Add cookies
	if server.Config.Cookies != "" {
		headers["Cookie"] = server.Config.Cookies
	}
	
	// Make request with Chrome 120 fingerprint
	response, err := c.client.Do(url, cycletls.Options{
		Body:      "",
		Ja3:       "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0",
		UserAgent: headers["User-Agent"],
		Headers:   headers,
		Timeout:   45,
	}, "GET")
	
	if err != nil {
		return "", fmt.Errorf("cycletls request: %w", err)
	}
	
	if response.Status == 404 {
		return "", ErrNotFound
	}
	
	// Check for Cloudflare protection
	if strings.Contains(response.Body, "<title>Just a moment...</title>") {
		return "", ErrCloudflareBlocked
	}
	
	// Check for Age Verification
	if strings.Contains(response.Body, "Verify your age") {
		return "", ErrAgeVerification
	}
	
	if response.Status == 403 {
		return "", fmt.Errorf("forbidden: %w", ErrPrivateStream)
	}
	
	return response.Body, nil
}

// Post sends an HTTP POST request to Chaturbate API
func (c *CycleTLSReq) Post(ctx context.Context, url, username, csrfToken string) (string, error) {
	// Build headers
	headers := make(map[string]string)
	
	// Set user agent
	userAgent := server.Config.UserAgent
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	}
	headers["User-Agent"] = strings.TrimSpace(userAgent)
	
	// Add POST-specific headers
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	headers["Accept"] = "application/json, text/plain, */*"
	headers["Accept-Language"] = "en-US,en;q=0.9"
	headers["Cache-Control"] = "no-cache"
	headers["Pragma"] = "no-cache"
	headers["Sec-Fetch-Dest"] = "empty"
	headers["Sec-Fetch-Mode"] = "cors"
	headers["Sec-Fetch-Site"] = "same-origin"
	headers["Sec-Ch-Ua"] = `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`
	headers["Sec-Ch-Ua-Mobile"] = "?0"
	headers["Sec-Ch-Ua-Platform"] = `"Windows"`
	headers["X-Requested-With"] = "XMLHttpRequest"
	headers["X-CSRFToken"] = csrfToken
	headers["Referer"] = fmt.Sprintf("https://chaturbate.com/%s", username)
	
	// Add cookies including CSRF token
	cookieStr := fmt.Sprintf("csrftoken=%s", csrfToken)
	if server.Config.Cookies != "" {
		cookieStr = server.Config.Cookies + "; " + cookieStr
	}
	headers["Cookie"] = cookieStr
	
	// Build POST body
	postData := fmt.Sprintf("room_slug=%s&bandwidth=high", username)
	
	// Make POST request
	response, err := c.client.Do(url, cycletls.Options{
		Body:      postData,
		Ja3:       "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0",
		UserAgent: headers["User-Agent"],
		Headers:   headers,
		Timeout:   45,
	}, "POST")
	
	if err != nil {
		return "", fmt.Errorf("cycletls post request: %w", err)
	}
	
	if response.Status == 404 {
		return "", ErrNotFound
	}
	
	// Check for Cloudflare protection
	if strings.Contains(response.Body, "<title>Just a moment...</title>") {
		return "", ErrCloudflareBlocked
	}
	
	if response.Status == 403 {
		return "", fmt.Errorf("forbidden: %w", ErrPrivateStream)
	}
	
	return response.Body, nil
}

// Close closes the CycleTLS client
func (c *CycleTLSReq) Close() {
	c.client.Close()
}
