package sveda_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func loadContract(t *testing.T) map[string]any {
	t.Helper()
	candidates := []string{
		"contracts/sidecar.v1.json",
		filepath.Join("..", "sveda", "packages", "protocol", "contracts", "sidecar.v1.json"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			raw, err := os.ReadFile(candidate)
			if err != nil {
				t.Fatalf("read contract: %v", err)
			}
			var decoded map[string]any
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Fatalf("decode contract: %v", err)
			}
			return decoded
		}
	}
	t.Fatal("sidecar.v1.json contract fixture not found")
	return nil
}

func TestSidecarContractSurface(t *testing.T) {
	contract := loadContract(t)
	if contract["version"] != "1.0" {
		t.Fatalf("unexpected version: %v", contract["version"])
	}
	if contract["prefix"] != "/sveda" {
		t.Fatalf("unexpected prefix: %v", contract["prefix"])
	}
	accept, _ := contract["accept"].(map[string]any)
	if accept["svedaStream"] != "application/vnd.sveda.stream+json" {
		t.Fatalf("unexpected stream accept: %v", accept["svedaStream"])
	}
}
