// mcp exposes the site lookup as a minimal JSON-RPC 2.0 stdio MCP server.
package main

import (
	"bufio"
	"context"
	"design-public/internal/county"
	"encoding/json"
	"os"
)

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func main() {
	s := bufio.NewScanner(os.Stdin)
	out := json.NewEncoder(os.Stdout)
	a := county.KingCounty{}
	for s.Scan() {
		var r request
		if json.Unmarshal(s.Bytes(), &r) != nil {
			continue
		}
		result := any(nil)
		switch r.Method {
		case "initialize":
			result = map[string]any{"protocolVersion": "2024-11-05", "serverInfo": map[string]string{"name": "design-public", "version": "0.1.0"}, "capabilities": map[string]any{"tools": map[string]any{}}}
		case "tools/list":
			result = map[string]any{"tools": []any{map[string]any{"name": "site_lookup", "description": "Look up a King County public-data site study.", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"address": map[string]string{"type": "string"}}, "required": []string{"address"}}}}}
		case "tools/call":
			var p struct {
				Name      string `json:"name"`
				Arguments struct {
					Address string `json:"address"`
				} `json:"arguments"`
			}
			_ = json.Unmarshal(r.Params, &p)
			site, err := a.Lookup(context.Background(), p.Arguments.Address)
			if err != nil {
				result = map[string]any{"content": []any{map[string]string{"type": "text", "text": err.Error()}}, "isError": true}
			} else {
				b, _ := json.Marshal(site)
				result = map[string]any{"content": []any{map[string]string{"type": "text", "text": string(b)}}}
			}
		default:
			result = map[string]any{"error": "method not found"}
		}
		_ = out.Encode(map[string]any{"jsonrpc": "2.0", "id": r.ID, "result": result})
	}
}
