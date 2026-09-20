# sveda-go-sdk

Go SDK for the [Sveda](https://sveda.dev) sidecar HTTP API.

Docs: [sveda.dev/docs/hosts/go](https://sveda.dev/docs/hosts/go)

Module: `github.com/neresson/sveda-go-sdk`

## Install

```bash
go get github.com/neresson/sveda-go-sdk
```

## Sidecar client

```go
package main

import (
	"context"

	sveda "github.com/neresson/sveda-go-sdk"
)

func main() {
	client := sveda.New(sveda.Config{
		BaseURL:    "https://sveda.example.com",
		HostAPIKey: hostKey,
	})
	token, _ := client.Embed.CreateToken(context.Background(), sveda.TokenRequest{
		VisitorID: "user-1",
	})
	_ = token
}
```

## Host integration (embed session + MCP tools)

The `host` package mirrors the Laravel SDK: mint an embed token with `host_mcp_url` / `host_mcp_token`, and expose `POST /mcp/sveda` for the sidecar to list and call your tools.

```go
import (
	"context"
	"net/http"

	sveda "github.com/neresson/sveda-go-sdk"
	"github.com/neresson/sveda-go-sdk/host"
)

h := host.New(host.Config{
	BaseURL:       os.Getenv("SVEDA_CLIENT_BASE_URL"),
	HostAPIKey:    os.Getenv("SVEDA_CLIENT_HOST_API_KEY"),
	MCPURL:        os.Getenv("SVEDA_CLIENT_MCP_URL"), // public URL the sidecar can reach
	ServerName:    "My App",
	Instructions:  "Tools for the current user.",
})
h.ResolveToolsUsing(func() []host.Tool { return []host.Tool{&SearchPostsTool{}} })

http.Handle("POST /sveda/session", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	session, err := h.StartSession(r.Context(), "user-123")
	// write JSON: origin, token, expires_in, appearance
}))

http.Handle("POST /mcp/sveda", h.Handler())
```

Lower-level session minting (without the MCP server):

```go
session, _ := sveda.StartHostSession(ctx, sveda.Config{
	BaseURL:    baseURL,
	HostAPIKey: hostKey,
}, sveda.HostSessionOptions{
	VisitorID:    "user-1",
	HostMCPURL:   "https://app.example.com/mcp/sveda",
	HostMCPToken: mcpToken,
})
```

By default, `host.Host` mints opaque MCP bearer tokens with an in-memory store (fine for development). Override with `MintTokenUsing` and `AuthenticateUsing` for production auth (for example Sanctum or your session layer).

Implement `host.Tool` with `Name`, `Description`, `InputSchema`, `Mode`, `Domain`, and `Handle`.

## License

GNU Affero General Public License v3.0. See [LICENSE](LICENSE).
