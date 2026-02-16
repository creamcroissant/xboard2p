// Package agentclient provides HTTP client for communicating with Agent APIs.
package agentclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents an HTTP client for communicating with an Agent.
type Client struct {
	baseURL   string
	authToken string
	client    *http.Client
}

// NewClient creates a new Agent client.
func NewClient(baseURL, authToken string) *Client {
	return &Client{
		baseURL:   baseURL,
		authToken: authToken,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ConfigFile represents a protocol configuration file from the agent.
type ConfigFile struct {
	Filename    string    `json:"filename"`
	ModTime     time.Time `json:"mod_time"`
	Size        int64     `json:"size"`
	ContentHash string    `json:"content_hash"`
}

// ListConfigs returns all protocol configuration files from the agent.
func (c *Client) ListConfigs(ctx context.Context) ([]ConfigFile, error) {
	var result struct {
		Configs []ConfigFile `json:"configs"`
	}
	if err := c.get(ctx, "/api/v1/protocols", &result); err != nil {
		return nil, err
	}
	return result.Configs, nil
}

// ListInbounds returns only inbound configuration files from the agent.
func (c *Client) ListInbounds(ctx context.Context) ([]ConfigFile, error) {
	var result struct {
		Inbounds []ConfigFile `json:"inbounds"`
	}
	if err := c.get(ctx, "/api/v1/protocols/inbounds", &result); err != nil {
		return nil, err
	}
	return result.Inbounds, nil
}

// GetConfig returns a specific configuration file content from the agent.
func (c *Client) GetConfig(ctx context.Context, filename string) ([]byte, error) {
	url := c.baseURL + "/api/v1/protocols/" + filename
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setAuth(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed: %s", string(body))
	}

	return body, nil
}

// ApplyConfigRequest is the request body for applying a configuration.
type ApplyConfigRequest struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

// ApplyConfig writes a configuration file to the agent and reloads the service.
func (c *Client) ApplyConfig(ctx context.Context, filename, content string) error {
	req := ApplyConfigRequest{
		Filename: filename,
		Content:  content,
	}
	return c.post(ctx, "/api/v1/protocols", req, nil)
}

// ApplyTemplateRequest is the request body for applying a template.
type ApplyTemplateRequest struct {
	Filename string         `json:"filename"`
	Template string         `json:"template"`
	Vars     map[string]any `json:"vars"`
}

// ApplyTemplate creates a config from template on the agent and reloads the service.
func (c *Client) ApplyTemplate(ctx context.Context, filename, template string, vars map[string]any) error {
	req := ApplyTemplateRequest{
		Filename: filename,
		Template: template,
		Vars:     vars,
	}
	return c.post(ctx, "/api/v1/protocols/template", req, nil)
}

// DeleteConfig removes a configuration file from the agent and reloads the service.
func (c *Client) DeleteConfig(ctx context.Context, filename string) error {
	url := c.baseURL + "/api/v1/protocols/" + filename
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	c.setAuth(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %s", string(body))
	}

	return nil
}

// ServiceStatus returns whether the sing-box service is running on the agent.
func (c *Client) ServiceStatus(ctx context.Context) (bool, error) {
	var result struct {
		Running bool `json:"running"`
	}
	if err := c.get(ctx, "/api/v1/service/status", &result); err != nil {
		return false, err
	}
	return result.Running, nil
}

// ReloadService reloads the sing-box service on the agent.
func (c *Client) ReloadService(ctx context.Context) error {
	return c.post(ctx, "/api/v1/service/reload", nil, nil)
}

// Health checks if the agent is reachable.
func (c *Client) Health(ctx context.Context) error {
	url := c.baseURL + "/health"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) setAuth(req *http.Request) {
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}
}

func (c *Client) get(ctx context.Context, path string, result any) error {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	c.setAuth(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed: %s", string(body))
	}

	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("parse response: %w", err)
		}
	}

	return nil
}

func (c *Client) post(ctx context.Context, path string, body, result any) error {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	c.setAuth(req)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request failed: %s", string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("parse response: %w", err)
		}
	}

	return nil
}
