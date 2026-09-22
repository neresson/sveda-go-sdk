package host

const hostManifestSchema = "sveda.host/v1"

func (h *Host) registeredHooks() map[string]bool {
	return map[string]bool{
		"resolve_tools": h.resolveTools != nil || h.resolveToolsFor != nil,
		"policy":        h.policyUsing != nil,
		"authorize":     h.authorize != nil,
		"visitor_id":    false,
		"mint_token":    h.mintTokenCustom,
	}
}

func (h *Host) Describe(user any) map[string]any {
	authenticated := user != nil
	var policy any
	if authenticated {
		value := h.PolicyFor(user)
		if value != "" {
			policy = value
		}
	}

	return map[string]any{
		"schema": hostManifestSchema,
		"sdk": map[string]any{
			"language": "go",
			"version":  "unknown",
		},
		"subject": map[string]any{
			"authenticated": authenticated,
			"policy":        policy,
		},
		"hooks": h.registeredHooks(),
		"tools": h.listTools(user),
	}
}
