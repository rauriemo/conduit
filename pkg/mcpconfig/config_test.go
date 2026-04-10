package mcpconfig

import (
	"strings"
	"testing"
)

func TestMCPServerRef_Validate(t *testing.T) {
	tests := []struct {
		name    string
		ref     MCPServerRef
		wantErr string
	}{
		{
			name:    "empty type",
			ref:     MCPServerRef{},
			wantErr: "type is required",
		},
		{
			name:    "unsupported type",
			ref:     MCPServerRef{Type: "grpc"},
			wantErr: "unsupported type",
		},
		{
			name: "valid stdio",
			ref: MCPServerRef{
				Type:    TransportStdio,
				Command: "node",
				Args:    []string{"server.js"},
			},
		},
		{
			name:    "stdio missing command",
			ref:     MCPServerRef{Type: TransportStdio},
			wantErr: "stdio server requires command",
		},
		{
			name: "stdio with url",
			ref: MCPServerRef{
				Type:    TransportStdio,
				Command: "node",
				URL:     "http://localhost:8080",
			},
			wantErr: "stdio server must not set url",
		},
		{
			name: "valid http",
			ref: MCPServerRef{
				Type: TransportHTTP,
				URL:  "http://localhost:8080/mcp",
			},
		},
		{
			name:    "http missing url",
			ref:     MCPServerRef{Type: TransportHTTP},
			wantErr: "http server requires url",
		},
		{
			name: "http with command",
			ref: MCPServerRef{
				Type:    TransportHTTP,
				URL:     "http://localhost:8080",
				Command: "node",
			},
			wantErr: "http server must not set command",
		},
		{
			name: "http invalid url",
			ref: MCPServerRef{
				Type: TransportHTTP,
				URL:  "://bad",
			},
			wantErr: "invalid url",
		},
		{
			name: "negative startup timeout",
			ref: MCPServerRef{
				Type:             TransportStdio,
				Command:          "node",
				StartupTimeoutMS: -1,
			},
			wantErr: "startup_timeout_ms must not be negative",
		},
		{
			name: "zero startup timeout is valid",
			ref: MCPServerRef{
				Type:    TransportStdio,
				Command: "node",
			},
		},
		{
			name: "positive startup timeout is valid",
			ref: MCPServerRef{
				Type:             TransportStdio,
				Command:          "node",
				StartupTimeoutMS: 5000,
			},
		},
		{
			name: "http with auth token env",
			ref: MCPServerRef{
				Type:         TransportHTTP,
				URL:          "http://localhost:8080",
				AuthTokenEnv: "MY_TOKEN",
			},
		},
		{
			name: "stdio with env vars",
			ref: MCPServerRef{
				Type:    TransportStdio,
				Command: "node",
				Args:    []string{"server.js"},
				Env:     map[string]string{"KEY": "value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ref.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if got := err.Error(); !strings.Contains(got, tt.wantErr) {
				t.Fatalf("error %q does not contain %q", got, tt.wantErr)
			}
		})
	}
}
