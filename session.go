package sveda

import (
	"context"
	"fmt"
	"strings"
)

type HostSession struct {
	Origin     string         `json:"origin"`
	Token      string         `json:"token"`
	ExpiresIn  int            `json:"expires_in"`
	Appearance map[string]any `json:"appearance"`
}

type HostSessionOptions struct {
	VisitorID    string
	HostMCPURL   string
	HostMCPToken string
}

func StartHostSession(ctx context.Context, cfg Config, opts HostSessionOptions) (HostSession, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	hostKey := strings.TrimSpace(cfg.HostAPIKey)
	if baseURL == "" || hostKey == "" {
		return HostSession{}, fmt.Errorf("SVEDA_CLIENT_BASE_URL and SVEDA_CLIENT_HOST_API_KEY are required")
	}

	cfg.BaseURL = baseURL
	cfg.HostAPIKey = hostKey
	client := New(cfg)

	token, err := client.Embed.CreateToken(ctx, TokenRequest{
		VisitorID:    strings.TrimSpace(opts.VisitorID),
		HostMCPURL:   strings.TrimSpace(opts.HostMCPURL),
		HostMCPToken: strings.TrimSpace(opts.HostMCPToken),
	})
	if err != nil {
		return HostSession{}, err
	}
	if token.Token == "" {
		return HostSession{}, fmt.Errorf("sidecar returned an empty embed token")
	}

	return HostSession{
		Origin:     baseURL,
		Token:      token.Token,
		ExpiresIn:  token.ExpiresIn,
		Appearance: token.Appearance,
	}, nil
}
