package acestep

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// Client communicates with the ACE-Step v1.5 REST API.
type Client struct {
	apiURL    string
	apiKey    string
	outputDir string // shared volume mount point
	http      *http.Client
}

// NewClient creates an ACE-Step API client.
func NewClient(apiURL, apiKey, outputDir string) *Client {
	return &Client{
		apiURL:    apiURL,
		apiKey:    apiKey,
		outputDir: outputDir,
		http:      &http.Client{Timeout: 30 * time.Second},
	}
}

// GenerateRequest contains parameters for music generation.
type GenerateRequest struct {
	Caption        string `json:"caption"`
	Lyrics         string `json:"lyrics"`
	Duration       int    `json:"audio_duration"`
	InferenceSteps int    `json:"inference_steps"`
	Seed           int    `json:"seed"`
	BatchSize      int    `json:"batch_size"`
	AudioFormat    string `json:"audio_format"`
}

type releaseResp struct {
	Data struct {
		TaskID string `json:"task_id"`
	} `json:"data"`
	Code  int    `json:"code"`
	Error string `json:"error"`
}

type queryResp struct {
	Data []taskResult `json:"data"`
	Code int          `json:"code"`
}

type taskResult struct {
	TaskID string `json:"task_id"`
	Status int    `json:"status"` // 0=running, 1=success, 2=failed
	Result string `json:"result"` // JSON string with file info
}

type resultItem struct {
	File   string `json:"file"`
	Status int    `json:"status"`
}

// WaitForHealthy blocks until the ACE-Step API responds to health checks.
func (c *Client) WaitForHealthy(ctx context.Context) error {
	log.Println("Waiting for ACE-Step API to be ready...")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := c.http.Get(c.apiURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			log.Println("ACE-Step API is healthy")
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}

		log.Println("ACE-Step not ready, retrying in 5s...")
		time.Sleep(5 * time.Second)
	}
}

// Generate submits a music generation task and returns the task ID.
func (c *Client) Generate(ctx context.Context, req GenerateRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.apiURL+"/release_task", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("submit task: %w", err)
	}
	defer resp.Body.Close()

	var result releaseResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if result.Code != 200 {
		return "", fmt.Errorf("API error (code %d): %s", result.Code, result.Error)
	}

	return result.Data.TaskID, nil
}

// PollUntilDone polls for task completion, returning the audio file path.
func (c *Client) PollUntilDone(ctx context.Context, taskID string, interval time.Duration) (string, error) {
	reqBody, _ := json.Marshal(map[string][]string{
		"task_id_list": {taskID},
	})

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.apiURL+"/query_result", bytes.NewReader(reqBody))
		if err != nil {
			return "", fmt.Errorf("create poll request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		if c.apiKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		}

		resp, err := c.http.Do(httpReq)
		if err != nil {
			log.Printf("Poll error: %v, retrying...", err)
			time.Sleep(interval)
			continue
		}

		var result queryResp
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			log.Printf("Decode error: %v, retrying...", err)
			time.Sleep(interval)
			continue
		}
		resp.Body.Close()

		if len(result.Data) == 0 {
			time.Sleep(interval)
			continue
		}

		task := result.Data[0]
		switch task.Status {
		case 1: // success
			return c.extractAudioPath(task.Result)
		case 2: // failed
			return "", fmt.Errorf("generation failed for task %s", taskID)
		default: // still running
			time.Sleep(interval)
		}
	}
}

// extractAudioPath parses the result JSON and returns the local file path.
func (c *Client) extractAudioPath(resultJSON string) (string, error) {
	var items []resultItem
	if err := json.Unmarshal([]byte(resultJSON), &items); err != nil {
		return "", fmt.Errorf("parse result items: %w", err)
	}

	if len(items) == 0 || items[0].File == "" {
		return "", fmt.Errorf("no audio file in result")
	}

	fileRef := items[0].File

	// Try shared volume first: parse path from URL-style reference
	// ACE-Step returns paths like "/v1/audio?path=outputs/task_xxx/0.mp3"
	if u, err := url.Parse(fileRef); err == nil {
		if relPath := u.Query().Get("path"); relPath != "" {
			localPath := filepath.Join(c.outputDir, relPath)
			if _, err := os.Stat(localPath); err == nil {
				return localPath, nil
			}
		}
	}

	// Fallback: download via HTTP
	return c.downloadAudio(fileRef)
}

// downloadAudio fetches the audio file from the API and saves it locally.
func (c *Client) downloadAudio(fileRef string) (string, error) {
	dlURL := c.apiURL + fileRef
	resp, err := c.http.Get(dlURL)
	if err != nil {
		return "", fmt.Errorf("download audio: %w", err)
	}
	defer resp.Body.Close()

	tmpFile, err := os.CreateTemp("", "drift-*.mp3")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("write audio: %w", err)
	}

	tmpFile.Close()
	return tmpFile.Name(), nil
}
