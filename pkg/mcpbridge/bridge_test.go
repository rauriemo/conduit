package mcpbridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rauriemo/conduit/pkg/mcpconfig"
)

func readMCPJSON(t *testing.T, dir string) mcpJSON {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if err != nil {
		t.Fatalf("reading .mcp.json: %v", err)
	}
	var got mcpJSON
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}
	return got
}

func serverMap(t *testing.T, got mcpJSON, name string) map[string]any {
	t.Helper()
	srv, ok := got.MCPServers[name]
	if !ok {
		t.Fatalf("missing %q in output", name)
	}
	m, ok := srv.(map[string]any)
	if !ok {
		t.Fatalf("server %q is not a map", name)
	}
	return m
}

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

	got := readMCPJSON(t, dir)
	m := serverMap(t, got, "test-server")

	if m["command"] != "node" {
		t.Errorf("command = %v, want node", m["command"])
	}
	args, ok := m["args"].([]any)
	if !ok {
		t.Fatal("args is not a list")
	}
	if len(args) != 1 || args[0] != "path/to/server.js" {
		t.Errorf("args = %v, want [path/to/server.js]", args)
	}
	env, ok := m["env"].(map[string]any)
	if !ok {
		t.Fatal("env is not a map")
	}
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

	got := readMCPJSON(t, dir)
	m := serverMap(t, got, "http-server")

	if m["url"] != "http://localhost:8080/mcp" {
		t.Errorf("url = %v, want http://localhost:8080/mcp", m["url"])
	}
	headers, ok := m["headers"].(map[string]any)
	if !ok {
		t.Fatal("headers is not a map")
	}
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

	if err := os.WriteFile(path, []byte(`{"old": true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	servers := map[string]mcpconfig.MCPServerRef{
		"new": {Type: mcpconfig.TransportStdio, Command: "echo"},
	}

	if err := WriteMCPConfig(dir, servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := readMCPJSON(t, dir)
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

func TestWriteMCPConfig_MixedStdioAndHTTP(t *testing.T) {
	dir := t.TempDir()

	servers := map[string]mcpconfig.MCPServerRef{
		"mcp-unity": {
			Type:    mcpconfig.TransportStdio,
			Command: "node",
			Args:    []string{"Server~/build/index.js"},
		},
		"remote-api": {
			Type:    mcpconfig.TransportHTTP,
			URL:     "http://localhost:9090/mcp",
			Headers: map[string]string{"Authorization": "Bearer test"},
		},
	}

	if err := WriteMCPConfig(dir, servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := readMCPJSON(t, dir)
	if len(got.MCPServers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(got.MCPServers))
	}

	sm := serverMap(t, got, "mcp-unity")
	if sm["command"] != "node" {
		t.Errorf("stdio command = %v, want node", sm["command"])
	}

	hm := serverMap(t, got, "remote-api")
	if hm["url"] != "http://localhost:9090/mcp" {
		t.Errorf("http url = %v, want http://localhost:9090/mcp", hm["url"])
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

func TestWriteMCPConfig_EmptyTypeReturnsError(t *testing.T) {
	dir := t.TempDir()

	servers := map[string]mcpconfig.MCPServerRef{
		"no-type": {Command: "node"},
	}

	err := WriteMCPConfig(dir, servers)
	if err == nil {
		t.Fatal("expected validation error for empty type")
	}

	path := filepath.Join(dir, ".mcp.json")
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatal("expected no file when validation fails")
	}
}

func TestWriteMCPConfig_FirstValidSecondInvalid(t *testing.T) {
	dir := t.TempDir()

	servers := map[string]mcpconfig.MCPServerRef{
		"good": {Type: mcpconfig.TransportStdio, Command: "echo"},
		"bad":  {Type: "bogus"},
	}

	err := WriteMCPConfig(dir, servers)
	if err == nil {
		t.Fatal("expected validation error for second server")
	}

	path := filepath.Join(dir, ".mcp.json")
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatal("expected no file when any server fails validation")
	}
}

func TestWriteMCPConfig_MinimalStdio(t *testing.T) {
	dir := t.TempDir()

	servers := map[string]mcpconfig.MCPServerRef{
		"minimal": {Type: mcpconfig.TransportStdio, Command: "echo"},
	}

	if err := WriteMCPConfig(dir, servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}

	raw := string(data)
	if strings.Contains(raw, "args") {
		t.Error("minimal stdio should not contain 'args' key")
	}
	if strings.Contains(raw, "env") {
		t.Error("minimal stdio should not contain 'env' key")
	}
	if !strings.Contains(raw, "echo") {
		t.Error("should contain command value 'echo'")
	}
}

func TestWriteMCPConfig_MinimalHTTP(t *testing.T) {
	dir := t.TempDir()

	servers := map[string]mcpconfig.MCPServerRef{
		"api": {Type: mcpconfig.TransportHTTP, URL: "http://localhost:3000"},
	}

	if err := WriteMCPConfig(dir, servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}

	raw := string(data)
	if strings.Contains(raw, "headers") {
		t.Error("minimal HTTP should not contain 'headers' key")
	}
	if !strings.Contains(raw, "http://localhost:3000") {
		t.Error("should contain url value")
	}
}

func TestWriteMCPConfig_StdioWithURLRejected(t *testing.T) {
	dir := t.TempDir()

	servers := map[string]mcpconfig.MCPServerRef{
		"conflict": {
			Type:    mcpconfig.TransportStdio,
			Command: "node",
			URL:     "http://localhost:8080",
		},
	}

	err := WriteMCPConfig(dir, servers)
	if err == nil {
		t.Fatal("expected validation error for stdio server with URL")
	}
}
