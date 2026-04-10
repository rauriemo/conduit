package mcpclient

import (
	"context"
	"strings"
	"testing"

	"github.com/rauriemo/conduit/pkg/mcpconfig"
)

func TestNewPool(t *testing.T) {
	p := NewPool()
	if p == nil {
		t.Fatal("NewPool returned nil")
	}
	if p.sessions == nil {
		t.Fatal("sessions map not initialized")
	}
}

func TestConnect_ValidationError(t *testing.T) {
	p := NewPool()
	err := p.Connect(context.Background(), "bad", &mcpconfig.MCPServerRef{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestConnect_DoubleConnect(t *testing.T) {
	p := NewPool()

	p.sessions["test"] = &clientSession{name: "test"}

	err := p.Connect(context.Background(), "test", &mcpconfig.MCPServerRef{
		Type:    mcpconfig.TransportStdio,
		Command: "echo",
	})
	if err == nil {
		t.Fatal("expected double-connect error")
	}
	if got := err.Error(); !strings.Contains(got, "already connected") {
		t.Fatalf("unexpected error: %v", got)
	}
}

func TestListTools_NotConnected(t *testing.T) {
	p := NewPool()
	_, err := p.ListTools(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing server")
	}
	if got := err.Error(); !strings.Contains(got, "not connected") {
		t.Fatalf("unexpected error: %v", got)
	}
}

func TestCallTool_NotConnected(t *testing.T) {
	p := NewPool()
	_, err := p.CallTool(context.Background(), "missing", "tool", nil)
	if err == nil {
		t.Fatal("expected error for missing server")
	}
	if got := err.Error(); !strings.Contains(got, "not connected") {
		t.Fatalf("unexpected error: %v", got)
	}
}

func TestDisconnect_NotConnected(t *testing.T) {
	p := NewPool()
	err := p.Disconnect("missing")
	if err == nil {
		t.Fatal("expected error for missing server")
	}
	if got := err.Error(); !strings.Contains(got, "not connected") {
		t.Fatalf("unexpected error: %v", got)
	}
}

func TestClose_EmptyPool(t *testing.T) {
	p := NewPool()
	if err := p.Close(); err != nil {
		t.Fatalf("unexpected error closing empty pool: %v", err)
	}
}

func TestBuildTransport_Stdio(t *testing.T) {
	ref := mcpconfig.MCPServerRef{
		Type:    mcpconfig.TransportStdio,
		Command: "echo",
		Args:    []string{"hello"},
		Env:     map[string]string{"FOO": "bar"},
	}
	transport, err := buildTransport(&ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transport == nil {
		t.Fatal("transport is nil")
	}
}

func TestBuildTransport_HTTP(t *testing.T) {
	ref := mcpconfig.MCPServerRef{
		Type: mcpconfig.TransportHTTP,
		URL:  "http://localhost:8080",
	}
	transport, err := buildTransport(&ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transport == nil {
		t.Fatal("transport is nil")
	}
}

func TestBuildTransport_HTTP_EmptyAuthToken(t *testing.T) {
	ref := mcpconfig.MCPServerRef{
		Type:         mcpconfig.TransportHTTP,
		URL:          "http://localhost:8080",
		AuthTokenEnv: "CONDUIT_TEST_NONEXISTENT_TOKEN_VAR",
	}
	_, err := buildTransport(&ref)
	if err == nil {
		t.Fatal("expected error for empty auth token env var")
	}
	if got := err.Error(); !strings.Contains(got, "auth token env") {
		t.Fatalf("unexpected error: %v", got)
	}
}

func TestBuildTransport_HTTP_WithHeaders(t *testing.T) {
	ref := mcpconfig.MCPServerRef{
		Type:    mcpconfig.TransportHTTP,
		URL:     "http://localhost:8080",
		Headers: map[string]string{"X-Custom": "value"},
	}
	transport, err := buildTransport(&ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transport == nil {
		t.Fatal("transport is nil")
	}
}

func TestBuildTransport_UnsupportedType(t *testing.T) {
	ref := mcpconfig.MCPServerRef{
		Type: "grpc",
	}
	_, err := buildTransport(&ref)
	if err == nil {
		t.Fatal("expected error for unsupported transport")
	}
}
