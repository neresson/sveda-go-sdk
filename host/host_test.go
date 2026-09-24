package host_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neresson/sveda-go-sdk/host"
)

type echoTool struct{}

func (echoTool) Name() string        { return "echo_message" }
func (echoTool) Description() string { return "Echo a message back." }
func (echoTool) Mode() string        { return host.ModeRead }
func (echoTool) Domain() string      { return "demo" }
func (echoTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{"type": "string", "description": "Message to echo"},
		},
		"required": []string{"message"},
	}
}
func (echoTool) Handle(_ context.Context, args map[string]any) (any, error) {
	return map[string]any{
		"success": true,
		"data":    map[string]any{"message": args["message"]},
	}, nil
}

func TestHostStartSessionSendsMCPFields(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any
	sidecar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":      "embed-token",
			"visitor_id": "go-playground",
			"expires_in": 3600,
		})
	}))
	t.Cleanup(sidecar.Close)

	h := host.New(host.Config{
		BaseURL:    sidecar.URL,
		HostAPIKey: "host-secret",
		MCPURL:     "https://app.test/mcp/sveda",
	})
	h.ResolveToolsUsing(func() []host.Tool { return []host.Tool{echoTool{}} })

	session, err := h.StartSession(context.Background(), "go-playground")
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	if session.Token != "embed-token" {
		t.Fatalf("token = %q", session.Token)
	}
	if gotBody["host_mcp_url"] != "https://app.test/mcp/sveda" {
		t.Fatalf("host_mcp_url = %#v", gotBody["host_mcp_url"])
	}
	if gotBody["host_mcp_token"] == "" {
		t.Fatalf("expected minted host_mcp_token")
	}
}

func TestMCPRequiresAuthentication(t *testing.T) {
	t.Parallel()

	h := host.New(host.Config{})
	h.ResolveToolsUsing(func() []host.Tool { return []host.Tool{echoTool{}} })

	req := httptest.NewRequest(http.MethodPost, "/mcp/sveda", bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)))
	rec := httptest.NewRecorder()
	h.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestMCPListsAndCallsTools(t *testing.T) {
	t.Parallel()

	h := host.New(host.Config{
		ServerName:    "Playground Feed",
		Instructions:  "Feed tools for the current user.",
	})
	h.ResolveToolsUsing(func() []host.Tool { return []host.Tool{echoTool{}} })
	token, err := h.TokenStore().Mint()
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	initBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"0.1.0"}}}`
	initReq := httptest.NewRequest(http.MethodPost, "/mcp/sveda", bytes.NewReader([]byte(initBody)))
	initReq.Header.Set("Authorization", "Bearer "+token)
	initRec := httptest.NewRecorder()
	h.Handler().ServeHTTP(initRec, initReq)
	if initRec.Code != http.StatusOK {
		t.Fatalf("initialize status = %d body=%s", initRec.Code, initRec.Body.String())
	}
	var initResp map[string]any
	_ = json.Unmarshal(initRec.Body.Bytes(), &initResp)
	result := initResp["result"].(map[string]any)
	serverInfo := result["serverInfo"].(map[string]any)
	if serverInfo["name"] != "Playground Feed" {
		t.Fatalf("server name = %#v", serverInfo["name"])
	}
	if result["instructions"] != "Feed tools for the current user." {
		t.Fatalf("instructions = %#v", result["instructions"])
	}

	listReq := httptest.NewRequest(http.MethodPost, "/mcp/sveda", bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{"per_page":250}}`)))
	listReq.Header.Set("Authorization", "Bearer "+token)
	listRec := httptest.NewRecorder()
	h.Handler().ServeHTTP(listRec, listReq)
	var listResp map[string]any
	_ = json.Unmarshal(listRec.Body.Bytes(), &listResp)
	tools := listResp["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("tools = %#v", tools)
	}
	tool := tools[0].(map[string]any)
	if tool["name"] != "echo_message" {
		t.Fatalf("tool name = %#v", tool["name"])
	}
	meta := tool["_meta"].(map[string]any)
	if meta["domain"] != "demo" || meta["mode"] != "read" {
		t.Fatalf("meta = %#v", meta)
	}
	if _, ok := meta["confirmation"]; ok {
		t.Fatalf("confirmation = %#v", meta["confirmation"])
	}

	callBody := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo_message","arguments":{"message":"hello"}}}`
	callReq := httptest.NewRequest(http.MethodPost, "/mcp/sveda", bytes.NewReader([]byte(callBody)))
	callReq.Header.Set("Authorization", "Bearer "+token)
	callRec := httptest.NewRecorder()
	h.Handler().ServeHTTP(callRec, callReq)
	var callResp map[string]any
	_ = json.Unmarshal(callRec.Body.Bytes(), &callResp)
	callResult := callResp["result"].(map[string]any)
	if callResult["isError"] != false {
		t.Fatalf("isError = %#v", callResult["isError"])
	}
	content := callResult["content"].([]any)
	text := content[0].(map[string]any)["text"].(string)
	var decoded map[string]any
	if err := json.Unmarshal([]byte(text), &decoded); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	data := decoded["data"].(map[string]any)
	if data["message"] != "hello" {
		t.Fatalf("message = %#v", data["message"])
	}
}

func TestHostStartSessionSendsPolicy(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any
	sidecar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":      "embed-token",
			"visitor_id": "go-playground",
			"expires_in": 3600,
		})
	}))
	t.Cleanup(sidecar.Close)

	h := host.New(host.Config{
		BaseURL:    sidecar.URL,
		HostAPIKey: "host-secret",
		MCPURL:     "https://app.test/mcp/sveda",
	})
	h.ResolveToolsUsing(func() []host.Tool { return []host.Tool{echoTool{}} })
	h.PolicyUsing(func(user any) string { return "reader" })

	session, err := h.StartSession(context.Background(), "go-playground")
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	if session.Token != "embed-token" {
		t.Fatalf("token = %q", session.Token)
	}
	if gotBody["policy"] != "reader" {
		t.Fatalf("policy = %#v", gotBody["policy"])
	}
}

func TestMCPFiltersToolsByAuthenticatedUser(t *testing.T) {
	t.Parallel()

	h := host.New(host.Config{})
	h.ResolveToolsForUsing(func(user any) []host.Tool {
		userMap, _ := user.(map[string]any)
		if userMap != nil && userMap["id"] == "user-1" {
			return []host.Tool{echoTool{}}
		}
		return nil
	})

	allowed, err := h.TokenStore().MintFor("user-1")
	if err != nil {
		t.Fatalf("MintFor: %v", err)
	}
	denied, err := h.TokenStore().MintFor("other")
	if err != nil {
		t.Fatalf("MintFor: %v", err)
	}

	callBody := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo_message","arguments":{"message":"hello"}}}`

	allowedReq := httptest.NewRequest(http.MethodPost, "/mcp/sveda", bytes.NewReader([]byte(callBody)))
	allowedReq.Header.Set("Authorization", "Bearer "+allowed)
	allowedRec := httptest.NewRecorder()
	h.Handler().ServeHTTP(allowedRec, allowedReq)
	var allowedResp map[string]any
	_ = json.Unmarshal(allowedRec.Body.Bytes(), &allowedResp)
	if allowedResp["result"].(map[string]any)["isError"] != false {
		t.Fatalf("allowed isError = %#v", allowedResp["result"])
	}

	deniedReq := httptest.NewRequest(http.MethodPost, "/mcp/sveda", bytes.NewReader([]byte(callBody)))
	deniedReq.Header.Set("Authorization", "Bearer "+denied)
	deniedRec := httptest.NewRecorder()
	h.Handler().ServeHTTP(deniedRec, deniedReq)
	var deniedResp map[string]any
	_ = json.Unmarshal(deniedRec.Body.Bytes(), &deniedResp)
	result := deniedResp["result"].(map[string]any)
	if result["isError"] != true {
		t.Fatalf("denied isError = %#v", result["isError"])
	}
	text := result["content"].([]any)[0].(map[string]any)["text"].(string)
	if text != "unknown tool: echo_message" {
		t.Fatalf("denied text = %#v", text)
	}
}

func TestZeroArgResolveToolsStillWorks(t *testing.T) {
	t.Parallel()

	h := host.New(host.Config{})
	h.ResolveToolsUsing(func() []host.Tool { return []host.Tool{echoTool{}} })
	token, err := h.TokenStore().MintFor("user-1")
	if err != nil {
		t.Fatalf("MintFor: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodPost, "/mcp/sveda", bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)))
	listReq.Header.Set("Authorization", "Bearer "+token)
	listRec := httptest.NewRecorder()
	h.Handler().ServeHTTP(listRec, listReq)
	var listResp map[string]any
	_ = json.Unmarshal(listRec.Body.Bytes(), &listResp)
	tools := listResp["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("tools = %#v", tools)
	}
}

type deleteTool struct{ echoTool }

func (deleteTool) Name() string         { return "delete_post" }
func (deleteTool) Mode() string         { return host.ModeDelete }
func (deleteTool) Confirmation() string { return "required" }

func TestConfirmationMetaIsPublishedWhenRequired(t *testing.T) {
	t.Parallel()

	h := host.New(host.Config{})
	h.ResolveToolsUsing(func() []host.Tool { return []host.Tool{echoTool{}, deleteTool{}} })
	token, err := h.TokenStore().Mint()
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp/sveda", bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.Handler().ServeHTTP(rec, req)
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	tools := resp["result"].(map[string]any)["tools"].([]any)
	byName := map[string]map[string]any{}
	for _, item := range tools {
		tool := item.(map[string]any)
		byName[tool["name"].(string)] = tool["_meta"].(map[string]any)
	}
	if _, ok := byName["echo_message"]["confirmation"]; ok {
		t.Fatalf("echo meta = %#v", byName["echo_message"])
	}
	if byName["delete_post"]["confirmation"] != "required" || byName["delete_post"]["mode"] != "delete" {
		t.Fatalf("delete meta = %#v", byName["delete_post"])
	}
}

func TestDescribeMatchesMcpToolsList(t *testing.T) {
	t.Parallel()

	h := host.New(host.Config{})
	h.ResolveToolsUsing(func() []host.Tool { return []host.Tool{echoTool{}} })
	user := map[string]any{"id": "user-1"}

	manifest := h.Describe(user)
	if manifest["schema"] != "sveda.host/v1" {
		t.Fatalf("schema = %#v", manifest["schema"])
	}

	token, err := h.TokenStore().MintFor("user-1")
	if err != nil {
		t.Fatalf("MintFor: %v", err)
	}
	listReq := httptest.NewRequest(http.MethodPost, "/mcp/sveda", bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"per_page":250}}`)))
	listReq.Header.Set("Authorization", "Bearer "+token)
	listRec := httptest.NewRecorder()
	h.Handler().ServeHTTP(listRec, listReq)
	var listResp map[string]any
	_ = json.Unmarshal(listRec.Body.Bytes(), &listResp)
	listed := listResp["result"].(map[string]any)["tools"].([]any)
	byName := map[string]map[string]any{}
	for _, item := range listed {
		tool := item.(map[string]any)
		byName[tool["name"].(string)] = tool
	}

	for _, tool := range manifest["tools"].([]map[string]any) {
		name := tool["name"].(string)
		if byName[name]["description"] != tool["description"] {
			t.Fatalf("description mismatch for %s", name)
		}
	}
}
