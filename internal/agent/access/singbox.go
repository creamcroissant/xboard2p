package access

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/creamcroissant/xboard/internal/agent/core"
)

type SingboxAccessCollector struct {
	manager *core.Manager
	logger  *slog.Logger
	client  *http.Client
}

func NewSingboxAccessCollector(manager *core.Manager, logger *slog.Logger) *SingboxAccessCollector {
	return &SingboxAccessCollector{
		manager: manager,
		logger:  logger,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *SingboxAccessCollector) Type() string {
	return "sing-box"
}

func (c *SingboxAccessCollector) Collect(ctx context.Context) ([]AccessLogEntry, error) {
	instances := c.manager.ListInstances()
	var entries []AccessLogEntry

	for _, inst := range instances {
		if inst.CoreType != core.CoreTypeSingBox || inst.Status != core.StatusRunning {
			continue
		}

		newEntries, err := c.collectFromInstance(ctx, inst)
		if err != nil {
			// Don't log error for every poll if API is not configured or temporary failure?
			// But logging is good for debugging.
			// Use Debug level to avoid spam if intended.
			c.logger.Debug("failed to collect access log from sing-box instance",
				"instance_id", inst.ID,
				"error", err,
			)
			continue
		}
		entries = append(entries, newEntries...)
	}
	return entries, nil
}

func (c *SingboxAccessCollector) collectFromInstance(ctx context.Context, inst *core.CoreInstance) ([]AccessLogEntry, error) {
	apiAddr, secret, err := c.getClashAPIConfig(inst.ConfigPath)
	if err != nil {
		return nil, nil // API not configured
	}

	url := fmt.Sprintf("http://%s/connections", apiAddr)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Connections []struct {
			ID          string `json:"id"`
			Metadata    struct {
				Network         string `json:"network"`
				Type            string `json:"type"`
				SourceIP        string `json:"sourceIP"`
				DestinationIP   string `json:"destinationIP"`
				SourcePort      int    `json:"sourcePort"`
				DestinationPort int    `json:"destinationPort"`
				Host            string `json:"host"`
			} `json:"metadata"`
			Upload   int64  `json:"upload"`
			Download int64  `json:"download"`
			Start    string `json:"start"`
			Chains   []string `json:"chains"`
		} `json:"connections"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var entries []AccessLogEntry
	for _, conn := range data.Connections {
		t, err := time.Parse(time.RFC3339, conn.Start)
		if err != nil {
			t = time.Now()
		}

		entries = append(entries, AccessLogEntry{
			// UserEmail: unknown for now via this API
			SourceIP:        conn.Metadata.SourceIP,
			TargetDomain:    conn.Metadata.Host,
			TargetIP:        conn.Metadata.DestinationIP,
			TargetPort:      conn.Metadata.DestinationPort,
			Protocol:        conn.Metadata.Network,
			Upload:          conn.Upload,
			Download:        conn.Download,
			ConnectionStart: t,
		})
	}

	return entries, nil
}

func (c *SingboxAccessCollector) getClashAPIConfig(configPath string) (string, string, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return "", "", err
	}

	var config struct {
		Experimental struct {
			ClashAPI struct {
				ExternalController string `json:"external_controller"`
				Secret             string `json:"secret"`
			} `json:"clash_api"`
		} `json:"experimental"`
	}

	if err := json.Unmarshal(content, &config); err != nil {
		return "", "", err
	}

	if config.Experimental.ClashAPI.ExternalController == "" {
		return "", "", fmt.Errorf("clash_api not configured")
	}

	return config.Experimental.ClashAPI.ExternalController, config.Experimental.ClashAPI.Secret, nil
}
