package sveda

import (
	"context"
	"encoding/json"
)

type Embed struct {
	client *Client
}

type TokenRequest struct {
	VisitorID    string
	HostMCPURL   string
	HostMCPToken string
}

type TokenResponse struct {
	Token      string
	VisitorID  string
	ExpiresIn  int
	Appearance map[string]any
}

func (e *Embed) CreateToken(ctx context.Context, req TokenRequest) (TokenResponse, error) {
	payload := map[string]any{}
	if req.VisitorID != "" {
		payload["visitor_id"] = req.VisitorID
	}
	if req.HostMCPURL != "" && req.HostMCPToken != "" {
		payload["host_mcp_url"] = req.HostMCPURL
		payload["host_mcp_token"] = req.HostMCPToken
	}

	raw, err := e.client.requestJSON(ctx, "POST", "/sveda/embed/token", payload)
	if err != nil {
		return TokenResponse{}, err
	}

	return tokenFromMap(raw), nil
}

func (e *Embed) Config(ctx context.Context) (map[string]any, error) {
	return e.client.requestJSON(ctx, "GET", "/sveda/embed/config", nil)
}

func tokenFromMap(payload map[string]any) TokenResponse {
	expires := 3600
	if value, ok := payload["expires_in"]; ok {
		expires = max(60, asInt(value))
	}

	appearance, _ := payload["appearance"].(map[string]any)

	return TokenResponse{
		Token:      asString(payload["token"]),
		VisitorID:  asString(payload["visitor_id"]),
		ExpiresIn:  expires,
		Appearance: appearance,
	}
}

func asString(value any) string {
	text, _ := value.(string)
	return text
}

func asInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		n, _ := typed.Int64()
		return int(n)
	default:
		return 0
	}
}
