package host

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

const mcpProtocolVersion = "2025-11-25"

type jsonRPCRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params"`
}

type jsonRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (h *Host) serveMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	if err := h.authenticate(ctx, bearerToken(r)); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if h.authorize != nil {
		if err := h.authorize(ctx); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}
	if h.afterAuth != nil {
		if err := h.afterAuth(ctx); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var req jsonRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.JSONRPC != "2.0" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	switch req.Method {
	case "notifications/initialized":
		w.WriteHeader(http.StatusAccepted)
		return
	case "initialize":
		h.writeJSONRPC(w, req.ID, h.initializeResult())
	case "tools/list":
		user := h.userFromBearer(bearerToken(r))
		h.writeJSONRPC(w, req.ID, map[string]any{"tools": h.listTools(user)})
	case "tools/call":
		user := h.userFromBearer(bearerToken(r))
		result, callErr := h.callTool(ctx, req.Params, user)
		if callErr != nil {
			h.writeJSONRPC(w, req.ID, map[string]any{
				"content": []map[string]any{{"type": "text", "text": callErr.Error()}},
				"isError": true,
			})
			return
		}
		h.writeJSONRPC(w, req.ID, result)
	default:
		w.WriteHeader(http.StatusAccepted)
	}
}

func (h *Host) initializeResult() map[string]any {
	result := map[string]any{
		"protocolVersion": mcpProtocolVersion,
		"capabilities": map[string]any{
			"tools": map[string]any{"listChanged": false},
		},
		"serverInfo": map[string]any{
			"name":    h.cfg.ServerName,
			"version": h.cfg.ServerVersion,
		},
	}
	if instructions := strings.TrimSpace(h.cfg.Instructions); instructions != "" {
		result["instructions"] = instructions
	}
	return result
}

func (h *Host) listTools(user any) []map[string]any {
	tools := h.ToolsFor(user)
	out := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		schema := tool.InputSchema()
		if schema == nil {
			schema = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		meta := map[string]any{
			"domain": tool.Domain(),
			"mode":   tool.Mode(),
		}
		if confirming, ok := tool.(ConfirmingTool); ok && confirming.Confirmation() == "required" {
			meta["confirmation"] = "required"
		}
		entry := map[string]any{
			"name":        tool.Name(),
			"title":       tool.Name(),
			"description": tool.Description(),
			"inputSchema": schema,
			"_meta":       meta,
		}
		out = append(out, entry)
	}
	return out
}

func (h *Host) callTool(ctx context.Context, params map[string]any, user any) (map[string]any, error) {
	name, _ := params["name"].(string)
	args, _ := params["arguments"].(map[string]any)
	if args == nil {
		args = map[string]any{}
	}

	for _, tool := range h.ToolsFor(user) {
		if tool.Name() != name {
			continue
		}
		result, err := tool.Handle(ctx, args)
		if err != nil {
			return nil, err
		}
		text, err := encodeToolResult(result)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"content": []map[string]any{{"type": "text", "text": text}},
			"isError": false,
		}, nil
	}
	return nil, errors.New("unknown tool: " + name)
}

func encodeToolResult(result any) (string, error) {
	switch v := result.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}

func bearerToken(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return ""
	}
	prefix := "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func (h *Host) writeJSONRPC(w http.ResponseWriter, id any, result any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}
