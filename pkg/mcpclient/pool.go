package mcpclient

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rauriemo/conduit/pkg/mcpconfig"
)

type clientSession struct {
	session *mcp.ClientSession
	name    string
}

type Pool struct {
	sessions map[string]*clientSession
	mu       sync.RWMutex
}

func NewPool() *Pool {
	return &Pool{
		sessions: make(map[string]*clientSession),
	}
}

func (p *Pool) Connect(ctx context.Context, name string, ref mcpconfig.MCPServerRef) error {
	if err := ref.Validate(); err != nil {
		return fmt.Errorf("mcpclient: connecting %s: %w", name, err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.sessions[name]; exists {
		return fmt.Errorf("mcpclient: server %q already connected (disconnect first)", name)
	}

	transport, err := buildTransport(ref)
	if err != nil {
		return fmt.Errorf("mcpclient: connecting %s: %w", name, err)
	}

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "conduit",
		Version: "0.1.0",
	}, nil)

	connectCtx := ctx
	if ref.StartupTimeoutMS > 0 {
		var cancel context.CancelFunc
		connectCtx, cancel = context.WithTimeout(ctx, time.Duration(ref.StartupTimeoutMS)*time.Millisecond)
		defer cancel()
	}

	session, err := client.Connect(connectCtx, transport, nil)
	if err != nil {
		return fmt.Errorf("mcpclient: connecting %s: %w", name, err)
	}

	p.sessions[name] = &clientSession{
		session: session,
		name:    name,
	}
	return nil
}

func (p *Pool) ListTools(ctx context.Context, server string) ([]mcp.Tool, error) {
	p.mu.RLock()
	cs, ok := p.sessions[server]
	p.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("mcpclient: server %q not connected", server)
	}

	result, err := cs.session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, fmt.Errorf("mcpclient: listing tools for %s: %w", server, err)
	}

	tools := make([]mcp.Tool, len(result.Tools))
	for i, t := range result.Tools {
		tools[i] = *t
	}
	return tools, nil
}

func (p *Pool) CallTool(ctx context.Context, server, tool string, args map[string]any) (*mcp.CallToolResult, error) {
	p.mu.RLock()
	cs, ok := p.sessions[server]
	p.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("mcpclient: server %q not connected", server)
	}

	result, err := cs.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      tool,
		Arguments: args,
	})
	if err != nil {
		return nil, fmt.Errorf("mcpclient: calling tool %s on %s: %w", tool, server, err)
	}
	return result, nil
}

func (p *Pool) Disconnect(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	cs, ok := p.sessions[name]
	if !ok {
		return fmt.Errorf("mcpclient: server %q not connected", name)
	}

	delete(p.sessions, name)
	if err := cs.session.Close(); err != nil {
		return fmt.Errorf("mcpclient: disconnecting %s: %w", name, err)
	}
	return nil
}

func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for name, cs := range p.sessions {
		if err := cs.session.Close(); err != nil {
			errs = append(errs, fmt.Errorf("mcpclient: closing %s: %w", name, err))
		}
		delete(p.sessions, name)
	}

	if len(errs) > 0 {
		return fmt.Errorf("mcpclient: close errors: %v", errs)
	}
	return nil
}

func buildTransport(ref mcpconfig.MCPServerRef) (mcp.Transport, error) {
	switch ref.Type {
	case mcpconfig.TransportStdio:
		cmd := exec.Command(ref.Command, ref.Args...)
		for k, v := range ref.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
		// Inherit parent env so the subprocess can find binaries.
		if ref.Env != nil {
			cmd.Env = append(os.Environ(), cmd.Env...)
		}
		return &mcp.CommandTransport{Command: cmd}, nil

	case mcpconfig.TransportHTTP:
		transport := &mcp.StreamableClientTransport{
			Endpoint: ref.URL,
		}
		if ref.AuthTokenEnv != "" {
			token := os.Getenv(ref.AuthTokenEnv)
			if token == "" {
				return nil, fmt.Errorf("auth token env %q is empty", ref.AuthTokenEnv)
			}
			transport.HTTPClient = &http.Client{
				Transport: &bearerTransport{
					token:   token,
					headers: ref.Headers,
					base:    http.DefaultTransport,
				},
			}
		} else if len(ref.Headers) > 0 {
			transport.HTTPClient = &http.Client{
				Transport: &bearerTransport{
					headers: ref.Headers,
					base:    http.DefaultTransport,
				},
			}
		}
		return transport, nil

	default:
		return nil, fmt.Errorf("unsupported transport type %q", ref.Type)
	}
}

type bearerTransport struct {
	token   string
	headers map[string]string
	base    http.RoundTripper
}

func (t *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	if t.token != "" {
		r.Header.Set("Authorization", "Bearer "+t.token)
	}
	for k, v := range t.headers {
		r.Header.Set(k, v)
	}
	return t.base.RoundTrip(r)
}
