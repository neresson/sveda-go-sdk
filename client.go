package sveda

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

type Config struct {
	BaseURL    string
	HostAPIKey string
	EmbedToken string
	HTTPClient *http.Client
}

type Client struct {
	Embed     *Embed
	Chat      *Chat
	Histories *Histories

	baseURL    string
	hostAPIKey string
	embedToken string
	http       *http.Client
}

func New(cfg Config) *Client {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	client := &Client{
		baseURL:    strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		hostAPIKey: strings.TrimSpace(cfg.HostAPIKey),
		embedToken: strings.TrimSpace(cfg.EmbedToken),
		http:       httpClient,
	}
	client.Embed = &Embed{client: client}
	client.Chat = &Chat{client: client}
	client.Histories = &Histories{client: client}

	return client
}

func (c *Client) url(path string) string {
	return c.baseURL + "/" + strings.TrimLeft(path, "/")
}

func (c *Client) applyAuth(req *http.Request) {
	if c.hostAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.hostAPIKey)
	}
	if c.embedToken != "" {
		req.Header.Set("X-Sveda-Embed-Token", c.embedToken)
	}
}

func (c *Client) requestJSON(ctx context.Context, method, path string, payload any) (map[string]any, error) {
	body, err := c.do(ctx, method, path, payload, "application/json")
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("unable to decode Sveda API response as JSON: %w", err)
	}

	return decoded, nil
}

func (c *Client) requestStream(ctx context.Context, method, path string, payload any) (io.ReadCloser, error) {
	req, err := c.newRequest(ctx, method, path, payload, "application/vnd.sveda.stream+json")
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	if err := checkStatus(resp); err != nil {
		defer resp.Body.Close()
		return nil, err
	}

	return resp.Body, nil
}

func (c *Client) do(ctx context.Context, method, path string, payload any, accept string) ([]byte, error) {
	req, err := c.newRequest(ctx, method, path, payload, accept)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkStatus(resp); err != nil {
		return nil, err
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) newRequest(ctx context.Context, method, path string, payload any, accept string) (*http.Request, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.url(path), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", accept)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.applyAuth(req)

	return req, nil
}

func checkStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return &Error{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("Sveda API authentication failed with status %d", resp.StatusCode),
		}
	}

	message := fmt.Sprintf("Sveda API request failed with status %d", resp.StatusCode)
	var decoded map[string]any
	if len(bytes.TrimSpace(body)) > 0 && json.Unmarshal(body, &decoded) == nil {
		if text, ok := decoded["message"].(string); ok && text != "" {
			message = text
		}
	}

	return &Error{StatusCode: resp.StatusCode, Message: message, Response: decoded}
}

type Error struct {
	StatusCode int
	Message    string
	Response   map[string]any
}

func (e *Error) Error() string {
	if e == nil || e.Message == "" {
		return "sveda: request failed"
	}
	return e.Message
}
