package sveda

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateTokenWithHostCredentials(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotAuth string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":      "sveda_embed_test",
			"visitor_id": "visitor-1",
			"expires_in": 3600,
			"appearance": map[string]any{"accent": "#c45c26"},
		})
	}))
	t.Cleanup(server.Close)

	client := New(Config{BaseURL: server.URL, HostAPIKey: "host-secret"})
	tok, err := client.Embed.CreateToken(context.Background(), TokenRequest{
		VisitorID:    "visitor-1",
		HostMCPURL:   "https://app.test/mcp/sveda",
		HostMCPToken: "mcp-token",
	})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	if tok.Token != "sveda_embed_test" {
		t.Fatalf("token = %q", tok.Token)
	}
	if tok.VisitorID != "visitor-1" {
		t.Fatalf("visitor = %q", tok.VisitorID)
	}
	if tok.ExpiresIn != 3600 {
		t.Fatalf("expires = %d", tok.ExpiresIn)
	}
	if tok.Appearance["accent"] != "#c45c26" {
		t.Fatalf("appearance = %#v", tok.Appearance)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q", gotMethod)
	}
	if gotPath != "/sveda/embed/token" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "Bearer host-secret" {
		t.Fatalf("authorization = %q", gotAuth)
	}
	if gotBody["visitor_id"] != "visitor-1" {
		t.Fatalf("payload visitor_id = %#v", gotBody["visitor_id"])
	}
	if gotBody["host_mcp_url"] != "https://app.test/mcp/sveda" {
		t.Fatalf("payload host_mcp_url = %#v", gotBody["host_mcp_url"])
	}
	if gotBody["host_mcp_token"] != "mcp-token" {
		t.Fatalf("payload host_mcp_token = %#v", gotBody["host_mcp_token"])
	}
}

func TestCreateStreamedChatEvents(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotToken, gotAccept string
	var gotBody ChatRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotToken = r.Header.Get("X-Sveda-Embed-Token")
		gotAccept = r.Header.Get("Accept")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"type\":\"message.start\"}\n\ndata: {\"type\":\"text.delta\",\"delta\":\"Hi\"}\n\ndata: [DONE]\n\n")
	}))
	t.Cleanup(server.Close)

	client := New(Config{BaseURL: server.URL, EmbedToken: "embed-token"})
	events, err := client.Chat.CreateStreamed(context.Background(), ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "Hello"}},
		ChatID:   "chat-1",
	})
	if err != nil {
		t.Fatalf("CreateStreamed: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("events = %#v", events)
	}
	if events[0].Type != "message.start" {
		t.Fatalf("event 0 = %#v", events[0])
	}
	if events[1].Type != "text.delta" {
		t.Fatalf("event 1 = %#v", events[1])
	}
	if events[1].Payload["delta"] != "Hi" {
		t.Fatalf("delta = %#v", events[1].Payload["delta"])
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q", gotMethod)
	}
	if gotPath != "/sveda/stream" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotToken != "embed-token" {
		t.Fatalf("embed token = %q", gotToken)
	}
	if gotAccept != "application/vnd.sveda.stream+json" {
		t.Fatalf("accept = %q", gotAccept)
	}
	if gotBody.ChatID != "chat-1" {
		t.Fatalf("chatId = %q", gotBody.ChatID)
	}
}

func TestCreateMessageAndHistories(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/sveda/message":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"explanation": "Hello",
				"tokens_used": 12,
				"chat_id":     "chat-1",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/sveda/chat-histories":
			_ = json.NewEncoder(w).Encode(map[string]any{"histories": []any{}})
		case r.Method == http.MethodGet && r.URL.Path == "/sveda/chat-histories/chat-1":
			_ = json.NewEncoder(w).Encode(map[string]any{"history": map[string]any{"chatId": "chat-1"}})
		case r.Method == http.MethodPatch && r.URL.Path == "/sveda/chat-histories/chat-1":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		case r.Method == http.MethodDelete && r.URL.Path == "/sveda/chat-histories/chat-1":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client := New(Config{BaseURL: server.URL, EmbedToken: "embed-token"})
	ctx := context.Background()

	message, err := client.Chat.Create(ctx, ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "Hello"}},
		ChatID:   "chat-1",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if message.Explanation != "Hello" || message.TokensUsed != 12 || message.ChatID != "chat-1" {
		t.Fatalf("message = %#v", message)
	}

	histories, err := client.Histories.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, ok := histories["histories"]; !ok {
		t.Fatalf("histories = %#v", histories)
	}

	detail, err := client.Histories.Get(ctx, "chat-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if _, ok := detail["history"]; !ok {
		t.Fatalf("detail = %#v", detail)
	}

	renamed, err := client.Histories.Rename(ctx, "chat-1", "New title")
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if renamed["success"] != true {
		t.Fatalf("rename = %#v", renamed)
	}

	deleted, err := client.Histories.Delete(ctx, "chat-1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deleted["success"] != true {
		t.Fatalf("delete = %#v", deleted)
	}
}

func TestStartHostSession(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer host-secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":      "sveda_embed_host",
			"visitor_id": "go-playground",
			"expires_in": 1800,
		})
	}))
	t.Cleanup(server.Close)

	session, err := StartHostSession(context.Background(), Config{
		BaseURL:    server.URL + "/",
		HostAPIKey: "host-secret",
	}, "go-playground")
	if err != nil {
		t.Fatalf("StartHostSession: %v", err)
	}

	if session.Origin != server.URL {
		t.Fatalf("origin = %q", session.Origin)
	}
	if session.Token != "sveda_embed_host" {
		t.Fatalf("token = %q", session.Token)
	}
	if session.ExpiresIn != 1800 {
		t.Fatalf("expires = %d", session.ExpiresIn)
	}
}

func TestStartHostSessionRequiresConfig(t *testing.T) {
	t.Parallel()

	_, err := StartHostSession(context.Background(), Config{}, "go-playground")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "SVEDA_CLIENT_BASE_URL") || !strings.Contains(err.Error(), "SVEDA_CLIENT_HOST_API_KEY") {
		t.Fatalf("error = %v", err)
	}
}

func TestAuthenticationError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	client := New(Config{BaseURL: server.URL, HostAPIKey: "bad"})
	_, err := client.Embed.CreateToken(context.Background(), TokenRequest{VisitorID: "go-playground"})
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("err type %T", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d", apiErr.StatusCode)
	}
}

func TestTokenOmitsPartialMCP(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":      "tok",
			"visitor_id": "go-playground",
		})
	}))
	t.Cleanup(server.Close)

	client := New(Config{BaseURL: server.URL, HostAPIKey: "host-secret"})
	_, err := client.Embed.CreateToken(context.Background(), TokenRequest{
		VisitorID:  "go-playground",
		HostMCPURL: "https://app.test/mcp/sveda",
	})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	if _, ok := gotBody["host_mcp_url"]; ok {
		t.Fatalf("expected MCP fields omitted, got %#v", gotBody)
	}
}
