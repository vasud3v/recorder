package uploader

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	gofileAPIBase = "https://api.gofile.io"
)

// GoFileUploader handles uploading files to GoFile.io
type GoFileUploader struct {
	client *http.Client
}

// NewGoFileUploader creates a new GoFile uploader instance
func NewGoFileUploader() *GoFileUploader {
	return &GoFileUploader{
		client: &http.Client{
			Timeout: 30 * time.Minute, // Long timeout for large video uploads
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  true, // Don't compress video files
			},
		},
	}
}

type getServerResponse struct {
	Status string `json:"status"`
	Data   struct {
		Servers []struct {
			Name string `json:"name"`
			Zone string `json:"zone"`
		} `json:"servers"`
	} `json:"data"`
}

type uploadResponse struct {
	Status string `json:"status"`
	Data   struct {
		DownloadPage string `json:"downloadPage"`
		Code         string `json:"code"`
		ParentFolder string `json:"parentFolder"`
		FileID       string `json:"fileId"`
		FileName     string `json:"fileName"`
		MD5          string `json:"md5"`
	} `json:"data"`
}

// Upload uploads a file to GoFile and returns the download link
func (u *GoFileUploader) Upload(filePath string) (string, error) {
	// Step 1: Get the best server
	server, err := u.getBestServer()
	if err != nil {
		return "", fmt.Errorf("get best server: %w", err)
	}

	// Step 2: Upload the file
	downloadLink, err := u.uploadFile(server, filePath)
	if err != nil {
		return "", fmt.Errorf("upload file: %w", err)
	}

	return downloadLink, nil
}

func (u *GoFileUploader) getBestServer() (string, error) {
	resp, err := u.client.Get(gofileAPIBase + "/servers")
	if err != nil {
		return "", fmt.Errorf("request servers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var serverResp getServerResponse
	if err := json.NewDecoder(resp.Body).Decode(&serverResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if serverResp.Status != "ok" {
		return "", fmt.Errorf("server status not ok: %s", serverResp.Status)
	}

	if len(serverResp.Data.Servers) == 0 {
		return "", fmt.Errorf("no servers available")
	}

	// Return the first server (you could add logic to pick based on zone)
	return serverResp.Data.Servers[0].Name, nil
}

func (u *GoFileUploader) uploadFile(server, filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	// Use pipe to stream the file without loading it all into memory
	pipeReader, pipeWriter := io.Pipe()
	writer := multipart.NewWriter(pipeWriter)

	// Start writing in a goroutine
	errChan := make(chan error, 1)
	go func() {
		defer pipeWriter.Close()

		part, err := writer.CreateFormFile("file", filepath.Base(filePath))
		if err != nil {
			errChan <- fmt.Errorf("create form file: %w", err)
			writer.Close()
			return
		}

		// Use a larger buffer for faster copying (1MB chunks)
		buf := make([]byte, 1024*1024)
		if _, err := io.CopyBuffer(part, file, buf); err != nil {
			errChan <- fmt.Errorf("copy file: %w", err)
			writer.Close()
			return
		}

		// Close writer before signaling success to flush multipart boundary
		if err := writer.Close(); err != nil {
			errChan <- fmt.Errorf("close writer: %w", err)
			return
		}

		errChan <- nil
	}()

	uploadURL := fmt.Sprintf("https://%s.gofile.io/contents/uploadfile", server)
	req, err := http.NewRequest("POST", uploadURL, pipeReader)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := u.client.Do(req)
	if err != nil {
		// Drain error channel to prevent goroutine leak
		select {
		case <-errChan:
		default:
		}
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	// Check for errors from the goroutine
	if err := <-errChan; err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var uploadResp uploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", fmt.Errorf("decode upload response: %w", err)
	}

	if uploadResp.Status != "ok" {
		return "", fmt.Errorf("upload status not ok: %s", uploadResp.Status)
	}

	return uploadResp.Data.DownloadPage, nil
}
