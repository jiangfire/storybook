package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRunConfigDefaultsToLocalhostHTTP(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	cfg, err := loadRunConfig("")
	if err != nil {
		t.Fatalf("loadRunConfig returned error: %v", err)
	}

	if cfg.HTTPAddr != "127.0.0.1:8081" {
		t.Fatalf("expected default HTTP addr 127.0.0.1:8081, got %q", cfg.HTTPAddr)
	}
}

func TestLoadRunConfigRejectsNonHTTPTransport(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcp.yaml")
	content := []byte("transport:\n  type: stdio\n")
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := loadRunConfig(configPath)
	if err == nil {
		t.Fatal("expected error for non-http transport")
	}
}
