package agentclient

import (
	"bytes"
	"context"
	"encoding/json"
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
	State     string `json:"state"`
	StreamURL string `json:"stream_url,omitempty"`
	Error     string `json:"error,omitempty"`
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
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

func (c *Client) Status(ctx context.Context) (StatusResponse, error) {
	var out StatusResponse
	err := c.getJson(ctx, "/status", &out)

	return out, err
}

func (c *Client) Play(ctx context.Context, streamURL string) (StatusResponse, error) {
	var out StatusResponse
	err := c.postJson(ctx, "/play", PlayRequest{StreamURL: streamURL}, &out)
	return out, err
}

func (c *Client) Stop(ctx context.Context) (StatusResponse, error) {
	var out StatusResponse
	err := c.postJson(ctx, "/stop", nil, &out)
	return out, err
}

func (c *Client) doJson(req *http.Request, out any) error {
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode < 200 || res.StatusCode > 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("%s: %s", res.Status, strings.TrimSpace(string(body)))
	}

	return json.NewDecoder(res.Body).Decode(out)
}

func (c *Client) getJson(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}

	return c.doJson(req, out)
}

func (c *Client) postJson(ctx context.Context, path string, body any, out any) error {
	var r io.Reader
	if body != nil {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return err
		}
		r = &buf
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, r)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	return c.doJson(req, out)
}
