package sveda

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

const SSEDoneLine = "data: [DONE]"

var streamEventTypes = map[string]struct{}{
	"message.start":   {},
	"text.delta":      {},
	"reasoning.delta": {},
	"tool.call":       {},
	"tool.result":     {},
	"tool.progress":   {},
	"context.usage":   {},
	"chat.title":      {},
	"max_steps":       {},
	"message.end":     {},
	"error":           {},
}

type StreamEvent struct {
	Type    string
	Payload map[string]any
}

func ParseStream(r io.Reader) ([]StreamEvent, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var events []StreamEvent
	for scanner.Scan() {
		event, ok := parseSSELine(scanner.Text())
		if ok {
			events = append(events, event)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func parseSSELine(line string) (StreamEvent, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "data:") {
		return StreamEvent{}, false
	}

	payload := strings.TrimSpace(trimmed[5:])
	if payload == "" || payload == "[DONE]" {
		return StreamEvent{}, false
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		return StreamEvent{}, false
	}

	eventType, _ := decoded["type"].(string)
	if _, ok := streamEventTypes[eventType]; !ok {
		return StreamEvent{}, false
	}

	return StreamEvent{Type: eventType, Payload: decoded}, true
}
