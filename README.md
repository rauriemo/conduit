# Conduit

[![Go](https://img.shields.io/badge/Go-1.26.1+-00ADD8?logo=go)](https://go.dev/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Conduit** is a reusable Go module providing [MCP](https://modelcontextprotocol.io/) (Model Context Protocol) substrate: a typed configuration model, a named connection pool, and a `.mcp.json` bridge writer. It exists so that MCP plumbing is portable across orchestrators -- [Anthem](https://github.com/rauriemo/anthem) today, Anthropic managed agents tomorrow, other runtimes later.

Built on the [official Go SDK](https://github.com/modelcontextprotocol/go-sdk) for MCP.

## Quick Start

```bash
go get github.com/rauriemo/conduit
```

```go
import (
    "github.com/rauriemo/conduit/pkg/mcpconfig"
    "github.com/rauriemo/conduit/pkg/mcpclient"
    "github.com/rauriemo/conduit/pkg/mcpbridge"
)
```

## Packages

### `pkg/mcpconfig` -- Configuration Model

Canonical MCP server reference with explicit transport discriminator.

```go
ref := mcpconfig.MCPServerRef{
    Type:    mcpconfig.TransportStdio,
    Command: "node",
    Args:    []string{"path/to/server.js"},
}

if err := ref.Validate(); err != nil {
    log.Fatal(err)
}
```

Supports two transports per the MCP spec:
- **stdio** -- spawn a subprocess (`Command`, `Args`, `Env`)
- **http** -- Streamable HTTP (`URL`, `AuthTokenEnv`, `Headers`)

### `pkg/mcpclient` -- Connection Pool

Named connection pool wrapping the official go-sdk. Manages concurrent MCP server sessions.

```go
pool := mcpclient.NewPool()
defer pool.Close()

err := pool.Connect(ctx, "mcp-unity", ref)

tools, err := pool.ListTools(ctx, "mcp-unity")

result, err := pool.CallTool(ctx, "mcp-unity", "update_gameobject", map[string]any{
    "name": "Player",
    "tag":  "Player",
})
```

Thread-safe. Transport selected automatically from `MCPServerRef.Type`. Bearer auth resolved from environment variables at connect time.

### `pkg/mcpbridge` -- .mcp.json Bridge Writer

Writes `.mcp.json` for Claude Code auto-discovery. Secondary execution path -- direct Pool calls are preferred.

```go
err := mcpbridge.WriteMCPConfig("/path/to/workspace", servers)
```

## Architecture

```
┌─────────────────────────────────────────────┐
│              Your Orchestrator              │
│         (Anthem, managed agents, …)         │
└──────┬──────────────┬──────────────┬────────┘
       │              │              │
       v              v              v
  mcpconfig      mcpclient      mcpbridge
  (types +       (pool +        (.mcp.json
   validate)      go-sdk)        writer)
       │              │
       │              v
       │     ┌────────────────┐
       └────>│  go-sdk (MCP)  │
             └────────┬───────┘
                      │
              ┌───────┴───────┐
              v               v
         stdio server    HTTP server
         (mcp-unity)     (future)
```

Conduit owns the boxes in the middle row. Everything above (orchestration, policy, dispatch) and below (actual MCP servers) belongs to other repos.

## What Conduit Does NOT Own

- Orchestration, dispatch, or execution loops
- HTTP/API tool brokering (simple REST calls)
- Shared feature context or artifact management
- Guest agent definitions or policy enforcement
- Allowlist logic

These are [Anthem](https://github.com/rauriemo/anthem)'s concerns. Conduit is the substrate, not the platform.

## Module Layout

```
conduit/
  go.mod
  pkg/
    mcpconfig/
      config.go          # MCPServerRef, Transport, Validate
      config_test.go
    mcpclient/
      pool.go            # Connection pool wrapping go-sdk
      pool_test.go
    mcpbridge/
      bridge.go          # .mcp.json writer
      bridge_test.go
  CLAUDE.md
  README.md
```

## Development

```bash
go build ./...           # Build
go test ./... -race      # Test with race detector
go vet ./...             # Vet
```

## Related Projects

- [Anthem](https://github.com/rauriemo/anthem) -- agent orchestrator, primary Conduit consumer
- [Prism](https://github.com/rauriemo/prism) -- visual workstation for agent interaction
- [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk) -- official Go SDK for MCP

## License

[MIT](LICENSE)
