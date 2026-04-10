# Conduit -- Claude Code Context

## What Is Conduit

Conduit is a reusable Go module providing MCP (Model Context Protocol) substrate: configuration models, a client connection pool, and a `.mcp.json` bridge writer. It exists as a separate repo because MCP plumbing must be portable across orchestrators -- Anthem today, Anthropic managed agents tomorrow, other runtimes later.

**Module path:** `github.com/rauriemo/conduit`

**Primary dependency:** `github.com/modelcontextprotocol/go-sdk` (official Go SDK for MCP servers and clients)

## Design Principles

- **Thin v1:** Three small packages, no servers. Every line earns its place.
- **Portable:** No Anthem imports. No orchestration logic. No HTTP/API tool brokering. No feature context. No artifact management. Those are Anthem's concerns.
- **Idiomatic Go:** `pkg/` layout for importable packages. Table-driven tests. Interface-based mocks. `log/slog` for logging. Error wrapping with `fmt.Errorf("context: %w", err)`.

## Module Layout

```
conduit/
  go.mod
  go.sum
  pkg/
    mcpconfig/
      config.go            # MCPServerRef, Transport, Validate
      config_test.go
    mcpclient/
      pool.go              # Connection pool wrapping go-sdk
      pool_test.go
    mcpbridge/
      bridge.go            # .mcp.json writer for Claude Code compatibility
      bridge_test.go
  CLAUDE.md
  README.md
```

Import paths:
- `github.com/rauriemo/conduit/pkg/mcpconfig`
- `github.com/rauriemo/conduit/pkg/mcpclient`
- `github.com/rauriemo/conduit/pkg/mcpbridge`

## Package Specifications

### pkg/mcpconfig -- Configuration Model

The canonical MCP server reference used by all consumers (Anthem, future adapters).

```go
package mcpconfig

type Transport string

const (
    TransportStdio Transport = "stdio"
    TransportHTTP  Transport = "http"
)

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

func (r *MCPServerRef) Validate() error
```

**Naming:** `TransportHTTP` not `TransportURL` -- names the protocol per MCP spec.

**Validate() rules:**
- `Type` is required, must be `stdio` or `http`
- If `stdio`: `Command` is required, `URL` must be empty
- If `http`: `URL` is required, `Command` must be empty. URL must parse as valid URL.
- `StartupTimeoutMS` must not be negative (0 means no extra deadline beyond parent context)
- Return wrapped errors with field context (e.g., `"mcpconfig: stdio server requires command"`)

**Tests:** Table-driven covering all validation branches, both transports, edge cases (empty type, conflicting fields).

### pkg/mcpclient -- Connection Pool

Wraps the official go-sdk to manage named MCP server connections.

```go
package mcpclient

import (
    "context"
    "sync"

    "github.com/modelcontextprotocol/go-sdk/mcp"
    "github.com/rauriemo/conduit/pkg/mcpconfig"
)

type Pool struct {
    sessions map[string]*clientSession
    mu       sync.RWMutex
}

func NewPool() *Pool

func (p *Pool) Connect(ctx context.Context, name string, ref *mcpconfig.MCPServerRef) error
func (p *Pool) ListTools(ctx context.Context, server string) ([]mcp.Tool, error)
func (p *Pool) CallTool(ctx context.Context, server, tool string, args map[string]any) (*mcp.CallToolResult, error)
func (p *Pool) Disconnect(name string) error
func (p *Pool) Close() error
```

**Transport selection via `ref.Type`:**
- `stdio` -> spawn subprocess via go-sdk stdio transport. Honor `StartupTimeoutMS` via context timeout.
- `http` -> Streamable HTTP via go-sdk. Bearer token from `os.Getenv(ref.AuthTokenEnv)`.

**Implementation notes:**
- `clientSession` holds the go-sdk client instance, transport, and connection metadata
- `Connect` validates the ref before connecting (call `ref.Validate()`)
- `Connect` on an already-connected name returns an error (disconnect first)
- `CallTool` / `ListTools` on a non-existent name returns a clear error
- `Close` iterates all sessions, calls `Disconnect` on each, collects errors
- Thread-safe: `sync.RWMutex` on the sessions map. Read lock for `ListTools`/`CallTool`, write lock for `Connect`/`Disconnect`/`Close`.
- For HTTP transport: if `AuthTokenEnv` is set, resolve via `os.Getenv`. If the env var is empty at runtime, return an error (not a silent empty header).

**Tests:**
- Unit tests with mock/stub sessions for connect/disconnect/list/call lifecycle
- Validate that double-connect returns error
- Validate that call on missing server returns error
- Validate that Close cleans up all sessions
- Integration tests (build-tagged `//go:build integration`) against a real stdio server if available

### pkg/mcpbridge -- .mcp.json Bridge Writer

Writes `.mcp.json` files for Claude Code auto-discovery. **Anthem v1** uses this path for guest agents (merged global + per-guest `mcp_servers`); Claude Code performs tool execution. Use the Pool when an orchestrator holds the MCP session in Go.

```go
package mcpbridge

import "github.com/rauriemo/conduit/pkg/mcpconfig"

func WriteMCPConfig(wsPath string, servers map[string]mcpconfig.MCPServerRef) error
```

**Claude Code .mcp.json format:**

```json
{
  "mcpServers": {
    "server-name": {
      "command": "node",
      "args": ["path/to/server.js"],
      "env": {"KEY": "value"}
    }
  }
}
```

**Implementation notes:**
- stdio servers: map directly to `command`/`args`/`env`
- HTTP servers: map `URL` to `"url"` key, include headers if present
- Write atomically: write to temp file in same directory, then rename
- Overwrite existing `.mcp.json` (caller manages the full server set)
- Empty servers map: no file written, return nil

**Tests:**
- Verify JSON output format for stdio servers
- Verify JSON output format for HTTP servers
- Verify empty servers produces no file
- Verify file is written to correct path
- Verify atomic write (temp + rename)

## What Conduit Does NOT Own

- Orchestration, dispatch, or execution loop logic (Anthem)
- HTTP/API tool brokering (Anthem)
- Shared feature context files or schema (project repos like RebelTower)
- Artifact registration or notifications (Anthem)
- Guest agent definitions or policy enforcement (Anthem)
- Allowlist logic (Anthem)
- Template substitution for HTTP tools (Anthem)

## Coding Standards

- **Language**: Go (latest stable)
- No unnecessary comments. Don't narrate what code does. Only comment non-obvious intent.
- Table-driven tests for all unit tests.
- Wrap errors with context: `fmt.Errorf("mcpclient: connecting %s: %w", name, err)`
- Use `log/slog` for logging where needed.
- No global mutable state -- dependency injection via constructors.
- `go vet` and `golangci-lint` must pass.
- Test with `-race` flag.

## Security Constraints

- MCP servers using Streamable HTTP bind to `localhost` by default. No `0.0.0.0`.
- Auth tokens are NEVER written to `.mcp.json` or any file. Only env var names are stored. Resolved at call time via `os.Getenv`.
- If `AuthTokenEnv` is set but the env var is empty at runtime, `Connect` fails with a clear error.
- `.mcp.json` bridge writes are localhost-only by design.

## Relationship to Other Repos

- **Anthem** (`github.com/rauriemo/anthem`): primary consumer. Imports Conduit types and mirrors `WriteMCPConfig` behavior in `internal/harness`; merges guest `mcp_servers` into workspace `.mcp.json` for Claude Code. Brokered in-process `Pool.CallTool` is deferred.
- **RebelTower**: game project; guest agents (e.g. Eiji) declare `mcp_servers` pointing at **Unity’s official MCP relay** (`com.unity.ai.assistant` 2.x, relay + `--mcp`) or other stdio servers. Not a direct Conduit Go dependency.
- **Unity MCP**: first-party Editor integration; relay binary speaks MCP over stdio to Claude Code / Cursor. Third-party `npx mcp-unity` is an alternative community stack, not Conduit-specific.
