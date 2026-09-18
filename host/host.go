package host

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	sveda "github.com/neresson/sveda-go-sdk"
)

type Config struct {
	BaseURL       string
	HostAPIKey    string
	MCPURL        string
	MCPPath       string
	ServerName    string
	ServerVersion string
	Instructions  string
	TokenTTL      time.Duration
}

// Host coordinates embed session minting and the local MCP endpoint.
type Host struct {
	cfg Config

	resolveTools func() []Tool
	mintToken    func(context.Context) (string, error)
	authenticate func(context.Context, string) error
	authorize    func(context.Context) error
	afterAuth    func(context.Context) error

	tokenStore *MemoryTokenStore
	tools      []Tool
}

func New(cfg Config) *Host {
	if cfg.MCPPath == "" {
		cfg.MCPPath = "/mcp/sveda"
	}
	if cfg.ServerName == "" {
		cfg.ServerName = "Host Application"
	}
	if cfg.ServerVersion == "" {
		cfg.ServerVersion = "0.1.0"
	}
	if cfg.TokenTTL <= 0 {
		cfg.TokenTTL = time.Hour
	}
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	cfg.HostAPIKey = strings.TrimSpace(cfg.HostAPIKey)
	cfg.MCPURL = strings.TrimRight(strings.TrimSpace(cfg.MCPURL), "/")
	cfg.MCPPath = normalizePath(cfg.MCPPath)

	store := NewMemoryTokenStore(cfg.TokenTTL)
	h := &Host{
		cfg:        cfg,
		tokenStore: store,
		mintToken: func(context.Context) (string, error) {
			return store.Mint()
		},
		authenticate: func(_ context.Context, bearer string) error {
			if store.Validate(bearer) {
				return nil
			}
			return errUnauthorized
		},
	}
	return h
}

func (h *Host) ResolveToolsUsing(fn func() []Tool) {
	h.resolveTools = fn
}

func (h *Host) MintTokenUsing(fn func(context.Context) (string, error)) {
	if fn != nil {
		h.mintToken = fn
	}
}

func (h *Host) AuthenticateUsing(fn func(context.Context, string) error) {
	if fn != nil {
		h.authenticate = fn
	}
}

func (h *Host) AuthorizeUsing(fn func(context.Context) error) {
	h.authorize = fn
}

func (h *Host) AfterAuthenticateUsing(fn func(context.Context) error) {
	h.afterAuth = fn
}

func (h *Host) TokenStore() *MemoryTokenStore {
	return h.tokenStore
}

func (h *Host) MCPURL() string {
	if h.cfg.MCPURL != "" {
		return h.cfg.MCPURL
	}
	return h.cfg.MCPPath
}

func (h *Host) Configured() bool {
	return h.cfg.BaseURL != "" && h.cfg.HostAPIKey != ""
}

func (h *Host) Tools() []Tool {
	if h.resolveTools != nil {
		return h.resolveTools()
	}
	return h.tools
}

func (h *Host) RegisterTool(tool Tool) {
	h.tools = append(h.tools, tool)
}

func (h *Host) StartSession(ctx context.Context, visitorID string) (sveda.HostSession, error) {
	if !h.Configured() {
		return sveda.HostSession{}, fmt.Errorf("SVEDA_CLIENT_BASE_URL and SVEDA_CLIENT_HOST_API_KEY are required")
	}

	mcpURL := h.MCPURL()
	mcpToken, err := h.mintToken(ctx)
	if err != nil {
		return sveda.HostSession{}, err
	}

	opts := sveda.HostSessionOptions{
		VisitorID: strings.TrimSpace(visitorID),
	}
	if mcpURL != "" && mcpToken != "" {
		opts.HostMCPURL = mcpURL
		opts.HostMCPToken = mcpToken
	}

	return sveda.StartHostSession(ctx, sveda.Config{
		BaseURL:    h.cfg.BaseURL,
		HostAPIKey: h.cfg.HostAPIKey,
	}, opts)
}

func (h *Host) Handler() http.Handler {
	return http.HandlerFunc(h.serveMCP)
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/mcp/sveda"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}
