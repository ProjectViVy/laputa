package mcp

import "encoding/json"

func SchemaToJSON(schema map[string]any) json.RawMessage {
	data, _ := json.Marshal(schema)
	return data
}

var SearchToolSchema = SchemaToJSON(map[string]any{
	"type": "object",
	"properties": map[string]any{
		"query": map[string]any{
			"type":        "string",
			"description": "Search query",
		},
		"wing": map[string]any{
			"type":        "string",
			"description": "Filter by wing",
		},
		"room": map[string]any{
			"type":        "string",
			"description": "Filter by room",
		},
	},
})
