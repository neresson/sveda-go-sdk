package host

import "context"

const (
	ModeRead   = "read"
	ModeWrite  = "write"
	ModeDelete = "delete"
)

// Tool is a host MCP tool the sidecar can list and call.
type Tool interface {
	Name() string
	Description() string
	InputSchema() map[string]any
	Mode() string
	Domain() string
	Handle(ctx context.Context, arguments map[string]any) (any, error)
}

// ConfirmingTool optionally requires a chat user to approve the call before it runs.
type ConfirmingTool interface {
	Tool
	Confirmation() string
}
