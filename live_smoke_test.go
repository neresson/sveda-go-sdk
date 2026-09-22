//go:build live

package sveda_test

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	sveda "github.com/neresson/sveda-go-sdk"
)

func TestLiveSmokeFlow(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("SVEDA_BASE_URL")), "/")
	hostKey := strings.TrimSpace(os.Getenv("SVEDA_HOST_KEY"))
	if baseURL == "" || hostKey == "" {
		t.Skip("SVEDA_BASE_URL and SVEDA_HOST_KEY are required for live smoke tests")
	}

	ctx := context.Background()
	httpClient := &http.Client{Timeout: 60 * http.Second}
	for _, path := range []string{"/sveda/health", "/sveda/ready"} {
		resp, err := httpClient.Get(baseURL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s: status %d", path, resp.StatusCode)
		}
	}

	host := sveda.New(sveda.Config{BaseURL: baseURL, HostAPIKey: hostKey})
	token, err := host.Embed.CreateToken(ctx, sveda.TokenRequest{VisitorID: "sdk-compat-go"})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if !strings.HasPrefix(token.Token, "sveda_embed_") {
		t.Fatalf("unexpected token prefix: %s", token.Token)
	}

	embed := sveda.New(sveda.Config{BaseURL: baseURL, EmbedToken: token.Token})
	events, err := embed.Chat.CreateStreamed(ctx, sveda.ChatRequest{
		ChatID: "sdk-compat-go",
		Messages: []sveda.ChatMessage{
			{Role: "user", Content: "compat stream"},
		},
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if len(events) == 0 {
		t.Fatalf("empty stream")
	}

	message, err := embed.Chat.Create(ctx, sveda.ChatRequest{
		ChatID: "sdk-compat-go-json",
		Messages: []sveda.ChatMessage{
			{Role: "user", Content: "compat smoke"},
		},
	})
	if err != nil {
		t.Fatalf("message: %v", err)
	}
	if strings.TrimSpace(message.Explanation) == "" {
		t.Fatalf("empty explanation")
	}

	histories, err := embed.Histories.List(ctx)
	if err != nil {
		t.Fatalf("histories: %v", err)
	}
	if histories["histories"] == nil {
		t.Fatalf("histories payload missing list key")
	}
}
