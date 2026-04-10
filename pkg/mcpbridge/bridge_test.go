package mcpbridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rauriemo/conduit/pkg/mcpconfig"
)

func TestWriteMCPConfig_StdioServer(t *testing.T) {
	dir := t.TempDir()

	servers := map[string]mcpconfig.MCPServerRef{
		"test-server": {
			Type:    mcpconfig.TransportStdio,
			Command: "node",
			Args:    []string{"path/to/server.js"},
			Env:     map[string]string{"KEY": "value"},
		},
	}

	if err := WriteMCPConfig(dir, servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if err != nil {
		t.Fatalf("reading .mcp.json: %v", err)
	}

	var got mcpJSON
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}

	srv, ok := got.MCPServers["test-server"]
	if !ok {
		t.Fatal("missing test-server in output")
	}

	m := srv.(map[string]any)
	if m["command"] != "node" {
		t.Errorf("command = %v, want node", m["command"])
	}
	args := m["args"].([]any)
	if len(args) != 1 || args[0] != "path/to/server.js" {
		t.Errorf("args = %v, want [path/to/server.js]", args)
	}
	env := m["env"].(map[string]any)
	if env["KEY"] != "value" {
		t.Errorf("env.KEY = %v, want value", env["KEY"])
	}
}

func TestWriteMCPConfig_HTTPServer(t *testing.T) {
	dir := t.TempDir()

	servers := map[string]mcpconfig.MCPServerRef{
		"http-server": {
			Type:    mcpconfig.TransportHTTP,
			URL:     "http://localhost:8080/mcp",
			Headers: map[string]string{"X-Custom": "val"},
		},
	}

	if err := WriteMCPConfig(dir, servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if err != nil {
		t.Fatalf("reading .mcp.json: %v", err)
	}

	var got mcpJSON
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}

	srv, ok := got.MCPServers["http-server"]
	if !ok {
		t.Fatal("missing http-server in output")
	}

	m := srv.(map[string]any)
	if m["url"] != "http://localhost:8080/mcp" {
		t.Errorf("url = %v, want http://localhost:8080/mcp", m["url"])
	}
	headers := m["headers"].(map[string]any)
	if headers["X-Custom"] != "val" {
		t.Errorf("headers.X-Custom = %v, want val", headers["X-Custom"])
	}
}

func TestWriteMCPConfig_EmptyServers(t *testing.T) {
	dir := t.TempDir()

	if err := WriteMCPConfig(dir, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := filepath.Join(dir, ".mcp.json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expected no file for empty servers")
	}
}

func TestWriteMCPConfig_EmptyMap(t *testing.T) {
	dir := t.TempDir()

	if err := WriteMCPConfig(dir, map[string]mcpconfig.MCPServerRef{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := filepath.Join(dir, ".mcp.json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expected no file for empty servers map")
	}
}

func TestWriteMCPConfig_CorrectPath(t *testing.T) {
	dir := t.TempDir()

	servers := map[string]mcpconfig.MCPServerRef{
		"s": {Type: mcpconfig.TransportStdio, Command: "echo"},
	}

	if err := WriteMCPConfig(dir, servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(dir, ".mcp.json")
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("file not at expected path %s: %v", expected, err)
	}
}

func TestWriteMCPConfig_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mcp.json")

	if err := os.WriteFile(path, []byte(`{"old": true}`), 0644); err != nil {
		t.Fatal(err)
	}

	servers := map[string]mcpconfig.MCPServerRef{
		"new": {Type: mcpconfig.TransportStdio, Command: "echo"},
	}

	if err := WriteMCPConfig(dir, servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var got mcpJSON
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}
	if _, ok := got.MCPServers["new"]; !ok {
		t.Fatal("overwrite did not produce new content")
	}
}

func TestWriteMCPConfig_NoAuthTokenInOutput(t *testing.T) {
	dir := t.TempDir()

	servers := map[string]mcpconfig.MCPServerRef{
		"secure": {
			Type:         mcpconfig.TransportHTTP,
			URL:          "http://localhost:8080",
			AuthTokenEnv: "SECRET_TOKEN",
		},
	}

	if err := WriteMCPConfig(dir, servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}

	raw := string(data)
	if strings.Contains(raw, "SECRET_TOKEN") || strings.Contains(raw, "auth_token") || strings.Contains(raw, "AuthToken") {
		t.Fatalf("auth token env leaked into .mcp.json: %s", raw)
	}
}

func TestWriteMCPConfig_InvalidRefReturnsError(t *testing.T) {
	dir := t.TempDir()

	servers := map[string]mcpconfig.MCPServerRef{
		"bad": {Type: "invalid"},
	}

	err := WriteMCPConfig(dir, servers)
	if err == nil {
		t.Fatal("expected validation error for invalid ref")
	}
	if !strings.Contains(err.Error(), "mcpbridge") {
		t.Errorf("error %q missing mcpbridge prefix", err.Error())
	}

	path := filepath.Join(dir, ".mcp.json")
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatal("expected no file when validation fails")
	}
}

