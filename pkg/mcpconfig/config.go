package mcpconfig

import (
	"fmt"
	"net/url"
)

// Transport identifies the MCP transport protocol (stdio or http).
type Transport string

// Supported transport types.
const (
	TransportStdio Transport = "stdio"
	TransportHTTP  Transport = "http"
)

// MCPServerRef describes how to reach a single MCP server.
// It supports both stdio (local subprocess) and HTTP (remote endpoint) transports.
type MCPServerRef struct {
	Type             Transport         `yaml:"type" json:"type"`
	Command          string            `yaml:"command,omitempty" json:"command,omitempty"`
	Args             []string          `yaml:"args,omitempty" json:"args,omitempty"`
	Env              map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	URL              string            `yaml:"url,omitempty" json:"url,omitempty"`
	AuthTokenEnv     string            `yaml:"auth_token_env,omitempty" json:"auth_token_env,omitempty"`
	Headers          map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	StartupTimeoutMS int               `yaml:"startup_timeout_ms,omitempty" json:"startup_timeout_ms,omitempty"`
}

// Validate checks that the ref has a supported transport type, the required
// fields for that transport are present, and conflicting fields are absent.
func (r MCPServerRef) Validate() error {
	switch r.Type {
	case TransportStdio:
		if r.Command == "" {
			return fmt.Errorf("mcpconfig: stdio server requires command")
		}
		if r.URL != "" {
			return fmt.Errorf("mcpconfig: stdio server must not set url")
		}
	case TransportHTTP:
		if r.URL == "" {
			return fmt.Errorf("mcpconfig: http server requires url")
		}
		if r.Command != "" {
			return fmt.Errorf("mcpconfig: http server must not set command")
		}
		if _, err := url.ParseRequestURI(r.URL); err != nil {
			return fmt.Errorf("mcpconfig: invalid url: %w", err)
		}
	case "":
		return fmt.Errorf("mcpconfig: type is required")
	default:
		return fmt.Errorf("mcpconfig: unsupported type %q", r.Type)
	}

	if r.StartupTimeoutMS < 0 {
		return fmt.Errorf("mcpconfig: startup_timeout_ms must not be negative")
	}

	return nil
}
