package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) do(method, path string, body, out any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	if out != nil {
		return json.Unmarshal(respBody, out)
	}
	return nil
}

type EnrollRequest struct {
	Token        string `json:"token"`
	Hostname     string `json:"hostname"`
	OSType       string `json:"os_type"`
	Arch         string `json:"arch"`
	AgentVersion string `json:"agent_version"`
}

type EnrollResponse struct {
	Device struct {
		ID string `json:"id"`
	} `json:"device"`
	DeviceToken string `json:"device_token"`
}

func (c *Client) Enroll(req EnrollRequest) (*EnrollResponse, error) {
	var resp EnrollResponse
	if err := c.do("POST", "/api/v1/auth/enroll", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Heartbeat() error {
	return c.do("POST", "/api/v1/agent/heartbeat", map[string]string{}, nil)
}

func (c *Client) GetDEK() (string, error) {
	var resp struct {
		DEK string `json:"dek"`
	}
	if err := c.do("GET", "/api/v1/agent/dek", nil, &resp); err != nil {
		return "", err
	}
	return resp.DEK, nil
}

func (c *Client) CheckChunks(hashes []string) ([]string, error) {
	var resp struct {
		Missing []string `json:"missing"`
	}
	if err := c.do("POST", "/api/v1/agent/chunks/check",
		map[string][]string{"hashes": hashes}, &resp); err != nil {
		return nil, err
	}
	return resp.Missing, nil
}

func (c *Client) ConfirmChunk(hash string, plaintextSize, storedSize int64, objectKey string) error {
	return c.do("POST", "/api/v1/agent/chunks/confirm", map[string]any{
		"hash":           hash,
		"plaintext_size": plaintextSize,
		"stored_size":    storedSize,
		"object_key":     objectKey,
	}, nil)
}

type FileManifestEntry struct {
	Path        string   `json:"path"`
	ParentPath  string   `json:"parent_path"`
	Name        string   `json:"name"`
	Size        int64    `json:"size"`
	Mtime       *string  `json:"mtime,omitempty"`
	Mode        int      `json:"mode"`
	IsDir       bool     `json:"is_dir"`
	ChunkHashes []string `json:"chunk_hashes"`
}

func (c *Client) CreateSnapshot(jobID string, entries []FileManifestEntry) (string, error) {
	var resp struct {
		ID string `json:"id"`
	}
	if err := c.do("POST", "/api/v1/agent/snapshots", map[string]any{
		"job_id":  jobID,
		"entries": entries,
	}, &resp); err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (c *Client) CreateJob() (string, error) {
	var resp struct {
		JobID string `json:"job_id"`
	}
	if err := c.do("POST", "/api/v1/agent/jobs", map[string]string{}, &resp); err != nil {
		return "", err
	}
	return resp.JobID, nil
}

func (c *Client) GetPresignURL(hash string) (presignURL, objectKey string, err error) {
	var resp struct {
		PresignURL string `json:"presign_url"`
		ObjectKey  string `json:"object_key"`
	}
	if err := c.do("POST", "/api/v1/agent/chunks/presign",
		map[string]string{"hash": hash}, &resp); err != nil {
		return "", "", err
	}
	return resp.PresignURL, resp.ObjectKey, nil
}

func (c *Client) UpdateJob(jobID, status string, errMsg *string, bytesScanned, bytesChanged, chunksTotal, chunksUploaded, chunksDeduped int64) error {
	return c.do("PUT", "/api/v1/agent/jobs/"+jobID, map[string]any{
		"status":          status,
		"error_msg":       errMsg,
		"bytes_scanned":   bytesScanned,
		"bytes_changed":   bytesChanged,
		"chunks_total":    chunksTotal,
		"chunks_uploaded": chunksUploaded,
		"chunks_deduped":  chunksDeduped,
	}, nil)
}
