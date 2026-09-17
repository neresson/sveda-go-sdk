package sveda

import (
	"strings"
	"testing"
)

func TestParseStreamEventsAndSkipDone(t *testing.T) {
	t.Parallel()

	content := strings.Join([]string{
		": connected",
		"",
		`data: {"type":"message.start"}`,
		"",
		`data: {"type":"text.delta","delta":"Hello"}`,
		"",
		`data: {"type":"message.end","finishReason":"stop"}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")

	events, err := ParseStream(strings.NewReader(content))
	if err != nil {
		t.Fatalf("ParseStream: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("events = %#v", events)
	}
	if events[0].Type != "message.start" {
		t.Fatalf("event 0 = %#v", events[0])
	}
	if events[1].Type != "text.delta" {
		t.Fatalf("event 1 = %#v", events[1])
	}
	if events[1].Payload["delta"] != "Hello" {
		t.Fatalf("delta = %#v", events[1].Payload["delta"])
	}
	if events[2].Type != "message.end" {
		t.Fatalf("event 2 = %#v", events[2])
	}
}

func TestParseStreamIgnoresInvalidLines(t *testing.T) {
	t.Parallel()

	events, err := ParseStream(strings.NewReader("event: ping\ndata: not-json\ndata: {\"type\":\"unknown.event\"}\n"))
	if err != nil {
		t.Fatalf("ParseStream: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("events = %#v", events)
	}
}

func TestSSEDoneLineMatchesContract(t *testing.T) {
	t.Parallel()

	if SSEDoneLine != "data: [DONE]" {
		t.Fatalf("SSEDoneLine = %q", SSEDoneLine)
	}
}
