package mcpbridge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rauriemo/conduit/pkg/mcpconfig"
)

type mcpJSON struct {
	MCPServers map[string]any `json:"mcpServers"`
}

func WriteMCPConfig(wsPath string, servers map[string]mcpconfig.MCPServerRef) error {
	if len(servers) == 0 {
		return nil
	}

	out := mcpJSON{MCPServers: make(map[string]any, len(servers))}

	for name, ref := range servers {
		entry := make(map[string]any)

		switch ref.Type {
		case mcpconfig.TransportStdio:
			entry["command"] = ref.Command
			if len(ref.Args) > 0 {
				entry["args"] = ref.Args
			}
			if len(ref.Env) > 0 {
				entry["env"] = ref.Env
			}
		case mcpconfig.TransportHTTP:
			entry["url"] = ref.URL
			if len(ref.Headers) > 0 {
				entry["headers"] = ref.Headers
			}
		}

		out.MCPServers[name] = entry
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("mcpbridge: marshaling config: %w", err)
	}
	data = append(data, '\n')

	dest := filepath.Join(wsPath, ".mcp.json")
	dir := filepath.Dir(dest)

	tmp, err := os.CreateTemp(dir, ".mcp.json.tmp.*")
	if err != nil {
		return fmt.Errorf("mcpbridge: creating temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("mcpbridge: writing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("mcpbridge: closing temp file: %w", err)
	}

	if err := os.Rename(tmpName, dest); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("mcpbridge: renaming to %s: %w", dest, err)
	}

	return nil
}
