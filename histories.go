package sveda

import (
	"context"
	"net/url"
)

type Histories struct {
	client *Client
}

func (h *Histories) List(ctx context.Context) (map[string]any, error) {
	return h.client.requestJSON(ctx, "GET", "/sveda/chat-histories", nil)
}

func (h *Histories) Get(ctx context.Context, chatID string) (map[string]any, error) {
	return h.client.requestJSON(ctx, "GET", "/sveda/chat-histories/"+url.PathEscape(chatID), nil)
}

func (h *Histories) Rename(ctx context.Context, chatID, title string) (map[string]any, error) {
	return h.client.requestJSON(ctx, "PATCH", "/sveda/chat-histories/"+url.PathEscape(chatID), map[string]any{
		"title": title,
	})
}

func (h *Histories) Delete(ctx context.Context, chatID string) (map[string]any, error) {
	return h.client.requestJSON(ctx, "DELETE", "/sveda/chat-histories/"+url.PathEscape(chatID), nil)
}
