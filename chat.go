package sveda

import (
	"context"
)

type Chat struct {
	client *Client
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Messages []ChatMessage `json:"messages,omitempty"`
	ChatID   string        `json:"chatId,omitempty"`
}

type MessageResponse struct {
	Explanation string
	TokensUsed  int
	ChatID      string
	Payload     map[string]any
}

func (c *Chat) Create(ctx context.Context, req ChatRequest) (MessageResponse, error) {
	raw, err := c.client.requestJSON(ctx, "POST", "/sveda/message", req)
	if err != nil {
		return MessageResponse{}, err
	}

	return MessageResponse{
		Explanation: asString(raw["explanation"]),
		TokensUsed:  asInt(raw["tokens_used"]),
		ChatID:      asString(raw["chat_id"]),
		Payload:     raw,
	}, nil
}

func (c *Chat) CreateStreamed(ctx context.Context, req ChatRequest) ([]StreamEvent, error) {
	body, err := c.client.requestStream(ctx, "POST", "/sveda/stream", req)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	return ParseStream(body)
}
