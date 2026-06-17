package agentclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

type PlayRequest struct {
	StreamURL string `json:"stream_url"`
}

type StatusResponse struct {
	State string `json:"state"`
	StreamURL string `json:"stream_url,omitempty"`
	Error string `json:"error,omitempty"`
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return err
	}

	defer func() { _ = res.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(res.Body, 128))
	trimmedBody := strings.TrimSpace(string(body))
	if res.StatusCode != http.StatusOK || trimmedBody != "ok" {
		return fmt.Errorf("unexpected health response: %s %q", res.Status, trimmedBody)
	}

	return nil
}
