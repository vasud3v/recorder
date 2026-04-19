package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/HeapOfChaos/goondvr/server"
)

type FlareSolverrRequest struct {
	Cmd        string `json:"cmd"`
	URL        string `json:"url"`
	MaxTimeout int    `json:"maxTimeout"`
	Session    string `json:"session,omitempty"`
	Proxy      struct {
		URL string `json:"url,omitempty"`
	} `json:"proxy,omitempty"`
}

type FlareSolverrResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Solution struct {
		URL      string `json:"url"`
		Status   int    `json:"status"`
		Response string `json:"response"` // HTML content
		Cookies []struct {
			Name     string  `json:"name"`
			Value    string  `json:"value"`
			Domain   string  `json:"domain"`
			Path     string  `json:"path"`
			Expires  float64 `json:"expires"`
			Size     int     `json:"size"`
			HttpOnly bool    `json:"httpOnly"`
			Secure   bool    `json:"secure"`
			SameSite string  `json:"sameSite"`
		} `json:"cookies"`
		UserAgent string `json:"userAgent"`
	} `json:"solution"`
}

// GetFreshCookies uses FlareSolverr to bypass Cloudflare and get fresh cookies
func GetFreshCookies(ctx context.Context, url string) (string, string, error) {
	flaresolverrURL := os.Getenv("FLARESOLVERR_URL")
	if flaresolverrURL == "" {
		flaresolverrURL = "http://localhost:8191/v1"
	}

	// Create a unique session for this request to avoid conflicts
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())
	
	// First, create a session
	createSessionReq := FlareSolverrRequest{
		Cmd: "sessions.create",
		Session: sessionID,
	}
	
	jsonData, err := json.Marshal(createSessionReq)
	if err != nil {
		return "", "", fmt.Errorf("marshal session request: %w", err)
	}
	
	if server.Config.Debug {
		fmt.Printf("[DEBUG] FlareSolverr: creating session %s\n", sessionID)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", flaresolverrURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("create session request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("create session: %w", err)
	}
	resp.Body.Close()
	
	// Now make the actual request with the session

	reqBody := FlareSolverrRequest{
		Cmd:        "request.get",
		URL:        url,
		MaxTimeout: 180000, // 180 seconds for Cloudflare challenges
		Session:    sessionID,
	}

	jsonData, err = json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("marshal request: %w", err)
	}

	if server.Config.Debug {
		fmt.Printf("[DEBUG] FlareSolverr: requesting %s\n", url)
	}

	req, err = http.NewRequestWithContext(ctx, "POST", flaresolverrURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client = &http.Client{Timeout: 360 * time.Second} // 6 minutes for Cloudflare challenges + queue wait
	resp, err = client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("read body: %w", err)
	}

	var fsResp FlareSolverrResponse
	if err := json.Unmarshal(body, &fsResp); err != nil {
		return "", "", fmt.Errorf("unmarshal response: %w", err)
	}

	// Clean up the session
	defer func() {
		destroyReq := FlareSolverrRequest{
			Cmd: "sessions.destroy",
			Session: sessionID,
		}
		destroyData, _ := json.Marshal(destroyReq)
		destroyHttpReq, _ := http.NewRequest("POST", flaresolverrURL, bytes.NewBuffer(destroyData))
		destroyHttpReq.Header.Set("Content-Type", "application/json")
		client.Do(destroyHttpReq)
	}()

	if fsResp.Status != "ok" {
		return "", "", fmt.Errorf("flaresolverr error: %s", fsResp.Message)
	}

	// Extract cookies
	var cookieParts []string
	for _, cookie := range fsResp.Solution.Cookies {
		if cookie.Name == "cf_clearance" || cookie.Name == "csrftoken" {
			cookieParts = append(cookieParts, fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
		}
	}

	if len(cookieParts) == 0 {
		return "", "", fmt.Errorf("no cookies found in response")
	}

	cookieStr := strings.Join(cookieParts, "; ")
	userAgent := fsResp.Solution.UserAgent

	if server.Config.Debug {
		fmt.Printf("[DEBUG] FlareSolverr: got %d cookies, user-agent: %s\n", len(cookieParts), userAgent)
	}

	return cookieStr, userAgent, nil
}
