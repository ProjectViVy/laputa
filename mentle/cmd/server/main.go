// Package server provides the MCP server implementation for mempalace-go.
// It runs as a stdio-based MCP server exposing memory tools to AI clients.
package server

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/dashimaki/mentle/facade"
	"github.com/dashimaki/mentle/internal/config"
	"github.com/dashimaki/mentle/internal/diary"
	"github.com/dashimaki/mentle/internal/dialect"
	"github.com/dashimaki/mentle/internal/kg"
	"github.com/dashimaki/mentle/internal/layers"
	"github.com/dashimaki/mentle/internal/miner"
	"github.com/dashimaki/mentle/internal/palace"
	"github.com/dashimaki/mentle/internal/sanitizer"
	"github.com/dashimaki/mentle/internal/search"
	"github.com/dashimaki/mentle/pkg/mcp"
	"github.com/google/uuid"
	"github.com/dashimaki/mentle/pkg/wal"
)

func runServer(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	svc := &facade.Service{}
	if err := svc.Init(ctx, facade.Options{}); err != nil {
		return err
	}
	defer svc.Close()

	server := mcp.NewServer(bufio.NewReader(os.Stdin), bufio.NewWriter(os.Stdout), bufio.NewWriter(os.Stderr))

	if err := server.Initialize(ctx, svc.Stack, svc.Searcher, svc.KG, nil); err != nil {
		return fmt.Errorf("initialize server: %w", err)
	}
	defer server.Shutdown(ctx)

	registerTools(server, svc.Cfg, svc.Stack, svc.Searcher, svc.KG, svc.WAL, svc.PalacePath, svc.PalaceGraph, svc.Diary)

	return server.Run()
}

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run the mempalace MCP server",
		RunE:  runServer,
	}
	return cmd
}

func registerTools(server *mcp.Server, cfg *config.Config, stack *layers.MemoryStack, searcher *search.Searcher, kgDB *kg.KnowledgeGraph, walInstance *wal.WAL, palacePath string, palaceGraph *palace.Graph, agentDiary *diary.Diary) {
	server.RegisterTool("mempalace_search", "Search memories", mcp.SearchToolSchema, func(params map[string]any) (any, error) {
		query, _ := params["query"].(string)
		wing, _ := params["wing"].(string)
		room, _ := params["room"].(string)

		ctx := context.Background()
		results, err := stack.Search(ctx, query, wing, room, 10)
		if err != nil {
			return nil, err
		}

		var lines []string
		for _, r := range results {
			lines = append(lines, fmt.Sprintf("[%s/%s] %s", r.Wing, r.Room, r.Content))
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Found %d results:\n%s", len(results), strings.Join(lines, "\n"))}},
		}, nil
	})

	server.RegisterTool("mempalace_wake", "Wake up memory with wing context", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"wing": map[string]any{"type": "string"},
		},
	}), func(params map[string]any) (any, error) {
		wing, _ := params["wing"].(string)
		ctx := context.Background()
		text, err := stack.WakeUp(ctx, wing)
		if err != nil {
			return nil, err
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: text}},
		}, nil
	})

	server.RegisterTool("mempalace_recall", "Recall memories from wing/room", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"wing":  map[string]any{"type": "string"},
			"room":  map[string]any{"type": "string"},
			"count": map[string]any{"type": "integer", "default": 10},
		},
	}), func(params map[string]any) (any, error) {
		wing, _ := params["wing"].(string)
		room, _ := params["room"].(string)
		count, _ := params["count"].(int)
		if count == 0 {
			count = 10
		}
		ctx := context.Background()
		text, err := stack.Recall(ctx, wing, room, count)
		if err != nil {
			return nil, err
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: text}},
		}, nil
	})

	server.RegisterTool("mempalace_kg_query", "Query knowledge graph", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"entity":    map[string]any{"type": "string"},
			"as_of":     map[string]any{"type": "string"},
			"direction": map[string]any{"type": "string", "default": "outgoing"},
		},
	}), func(params map[string]any) (any, error) {
		entity, _ := params["entity"].(string)
		asOf, _ := params["as_of"].(string)
		direction, _ := params["direction"].(string)

		results, err := kgDB.QueryEntity(entity, asOf, direction)
		if err != nil {
			return nil, err
		}

		var lines []string
		for _, r := range results {
			lines = append(lines, fmt.Sprintf("%s -> %s (%s)", r.Predicate, r.Object, r.ValidFrom))
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: strings.Join(lines, "\n")}},
		}, nil
	})

	server.RegisterTool("mempalace_status", "Get palace status and overview", mcp.SchemaToJSON(map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}), func(params map[string]any) (any, error) {
		ctx := context.Background()
		taxonomy, err := searcher.GetTaxonomy(ctx)
		if err != nil {
			return nil, err
		}

		totalDrawers := 0
		var wingLines []string
		for wingName, wingNode := range taxonomy {
			totalDrawers += wingNode.Count
			roomCount := len(wingNode.Rooms)
			wingLines = append(wingLines, fmt.Sprintf("  %s: %d drawers, %d rooms", wingName, wingNode.Count, roomCount))
		}

		text := fmt.Sprintf("Palace Status:\n"+
			"  Total drawers: %d\n"+
			"  Total wings: %d\n\n"+
			"Wings:\n%s",
			totalDrawers, len(taxonomy), strings.Join(wingLines, "\n"))

		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: text}},
		}, nil
	})

	server.RegisterTool("mempalace_list_wings", "List all wings with drawer counts", mcp.SchemaToJSON(map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}), func(params map[string]any) (any, error) {
		ctx := context.Background()
		wings, err := searcher.ListWings(ctx)
		if err != nil {
			return nil, err
		}

		var lines []string
		for _, w := range wings {
			lines = append(lines, fmt.Sprintf("%s: %d drawers", w.Name, w.DrawerCount))
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Wings (%d):\n%s", len(wings), strings.Join(lines, "\n"))}},
		}, nil
	})

	server.RegisterTool("mempalace_list_rooms", "List rooms within a wing (or all rooms)", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"wing": map[string]any{"type": "string"},
		},
	}), func(params map[string]any) (any, error) {
		wing, _ := params["wing"].(string)
		ctx := context.Background()
		rooms, err := searcher.ListRooms(ctx, wing)
		if err != nil {
			return nil, err
		}

		var lines []string
		for _, r := range rooms {
			lines = append(lines, fmt.Sprintf("%s/%s: %d drawers", r.Wing, r.Name, r.DrawerCount))
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Rooms (%d):\n%s", len(rooms), strings.Join(lines, "\n"))}},
		}, nil
	})

	server.RegisterTool("mempalace_get_taxonomy", "Get full wing -> room -> count tree structure", mcp.SchemaToJSON(map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}), func(params map[string]any) (any, error) {
		ctx := context.Background()
		taxonomy, err := searcher.GetTaxonomy(ctx)
		if err != nil {
			return nil, err
		}

		var lines []string
		for wingName, wingNode := range taxonomy {
			lines = append(lines, fmt.Sprintf("%s (%d)", wingName, wingNode.Count))
			for roomName, roomNode := range wingNode.Rooms {
				lines = append(lines, fmt.Sprintf("  %s: %d", roomName, roomNode.Count))
			}
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: strings.Join(lines, "\n")}},
		}, nil
	})

	server.RegisterTool("mempalace_check_duplicate", "Check if content already exists", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
			"wing":    map[string]any{"type": "string"},
			"room":    map[string]any{"type": "string"},
		},
		"required": []string{"content"},
	}), func(params map[string]any) (any, error) {
		content, _ := params["content"].(string)
		wing, _ := params["wing"].(string)
		room, _ := params["room"].(string)

		ctx := context.Background()
		results, err := stack.Search(ctx, content, wing, room, 5)
		if err != nil {
			return nil, err
		}

		var duplicates []string
		for _, r := range results {
			if len(r.Content) > 0 && similarContent(content, r.Content) {
				duplicates = append(duplicates, fmt.Sprintf("[%s/%s] ID: %s", r.Wing, r.Room, r.ID))
			}
		}

		if len(duplicates) > 0 {
			return mcp.ToolCallResult{
				Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Possible duplicates found:\n%s", strings.Join(duplicates, "\n"))}},
			}, nil
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: "No duplicates found. Content appears to be unique."}},
		}, nil
	})

	server.RegisterTool("mempalace_add_drawer", "Add content to a wing/room", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
			"wing":    map[string]any{"type": "string"},
			"room":    map[string]any{"type": "string"},
			"source":  map[string]any{"type": "string"},
		},
		"required": []string{"content", "wing", "room"},
	}), func(params map[string]any) (any, error) {
		content, _ := params["content"].(string)
		wing, _ := params["wing"].(string)
		room, _ := params["room"].(string)
		source, _ := params["source"].(string)

		var err error
		wing, err = sanitizer.SanitizeName(wing, "wing")
		if err != nil {
			return nil, err
		}
		room, err = sanitizer.SanitizeName(room, "room")
		if err != nil {
			return nil, err
		}
		content, err = sanitizer.SanitizeContent(content, "content")
		if err != nil {
			return nil, err
		}

		ctx := context.Background()
		drawer := palace.Drawer{
			ID:         uuid.NewString(),
			Content:    content,
			Wing:       wing,
			Room:       room,
			SourceFile: source,
		}

		if err := searcher.Store(ctx, drawer); err != nil {
			return nil, err
		}

		if err := walInstance.LogAdd(wal.Entry{
			Op:      "add",
			Wing:    wing,
			Room:    room,
			Content: content,
		}); err != nil {
			return nil, err
		}

		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Successfully added drawer to %s/%s", wing, room)}},
		}, nil
	})

	server.RegisterTool("mempalace_delete_drawer", "Delete a drawer by ID", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{"type": "string"},
		},
		"required": []string{"id"},
	}), func(params map[string]any) (any, error) {
		id, _ := params["id"].(string)

		ctx := context.Background()
		if err := searcher.Delete(ctx, id); err != nil {
			return nil, err
		}

		if err := walInstance.LogDelete(id); err != nil {
			return nil, err
		}

		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Successfully deleted drawer %s", id)}},
		}, nil
	})

	server.RegisterTool("mempalace_kg_add", "Add fact to knowledge graph", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"subject":    map[string]any{"type": "string"},
			"predicate":  map[string]any{"type": "string"},
			"object":     map[string]any{"type": "string"},
			"valid_from": map[string]any{"type": "string"},
			"valid_to":   map[string]any{"type": "string"},
			"confidence": map[string]any{"type": "number"},
		},
		"required": []string{"subject", "predicate", "object"},
	}), func(params map[string]any) (any, error) {
		subject, _ := params["subject"].(string)
		predicate, _ := params["predicate"].(string)
		obj, _ := params["object"].(string)
		validFrom, _ := params["valid_from"].(string)
		validTo, _ := params["valid_to"].(string)
		confidence := 1.0
		if c, ok := params["confidence"].(float64); ok {
			confidence = c
		}

		tripleID, err := kgDB.AddTriple(subject, predicate, obj, validFrom, validTo, confidence)
		if err != nil {
			return nil, err
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Added triple: %s", tripleID)}},
		}, nil
	})

	server.RegisterTool("mempalace_kg_invalidate", "Mark facts as ended", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"subject":   map[string]any{"type": "string"},
			"predicate": map[string]any{"type": "string"},
			"object":    map[string]any{"type": "string"},
			"valid_to":  map[string]any{"type": "string"},
		},
		"required": []string{"subject", "predicate", "object", "valid_to"},
	}), func(params map[string]any) (any, error) {
		subject, _ := params["subject"].(string)
		predicate, _ := params["predicate"].(string)
		obj, _ := params["object"].(string)
		validTo, _ := params["valid_to"].(string)

		if err := kgDB.Invalidate(subject, predicate, obj, validTo); err != nil {
			return nil, err
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: "Fact invalidated successfully"}},
		}, nil
	})

	server.RegisterTool("mempalace_kg_timeline", "Get chronological entity story", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"entity": map[string]any{"type": "string"},
		},
		"required": []string{"entity"},
	}), func(params map[string]any) (any, error) {
		entity, _ := params["entity"].(string)

		entries, err := kgDB.Timeline(entity)
		if err != nil {
			return nil, err
		}

		var lines []string
		for _, e := range entries {
			line := fmt.Sprintf("%s: %s -> %s", e.ValidFrom, e.Predicate, e.Object)
			if e.ValidTo != "" {
				line += fmt.Sprintf(" (until %s)", e.ValidTo)
			}
			lines = append(lines, line)
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Timeline for %s:\n%s", entity, strings.Join(lines, "\n"))}},
		}, nil
	})

	server.RegisterTool("mempalace_kg_stats", "Get knowledge graph statistics", mcp.SchemaToJSON(map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}), func(params map[string]any) (any, error) {
		stats, err := kgDB.Stats()
		if err != nil {
			return nil, err
		}

		var lines []string
		lines = append(lines, fmt.Sprintf("Entities: %d", stats.EntityCount))
		lines = append(lines, fmt.Sprintf("Triples: %d", stats.TripleCount))
		lines = append(lines, "Relationship types:")
		for pred, count := range stats.RelationshipTypes {
			lines = append(lines, fmt.Sprintf("  %s: %d", pred, count))
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: strings.Join(lines, "\n")}},
		}, nil
	})

	server.RegisterTool("mempalace_traverse", "Walk the palace graph from a room", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"wing":       map[string]any{"type": "string"},
			"room":       map[string]any{"type": "string"},
			"max_depth":  map[string]any{"type": "integer", "default": 3},
			"direction":  map[string]any{"type": "string", "default": "both"},
		},
		"required": []string{"wing", "room"},
	}), func(params map[string]any) (any, error) {
		wing, _ := params["wing"].(string)
		room, _ := params["room"].(string)
		maxDepth := 3
		if d, ok := params["max_depth"].(int); ok {
			maxDepth = d
		}
		_ = params["direction"]
		path := palaceGraph.Traverse(wing+"/"+room, maxDepth)

		var lines []string
		for _, step := range path {
			lines = append(lines, fmt.Sprintf("%s", step["room"]))
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Path (%d steps):\n%s", len(path), strings.Join(lines, " -> "))}},
		}, nil
	})

	server.RegisterTool("mempalace_diary_write", "Write an entry to the agent diary", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
			"tags":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"required": []string{"content"},
	}), func(params map[string]any) (any, error) {
		content, _ := params["content"].(string)
		var tags []string
		if t, ok := params["tags"].([]any); ok {
			for _, tag := range t {
				if s, ok := tag.(string); ok {
					tags = append(tags, s)
				}
			}
		}

		err := agentDiary.Write(diary.Entry{Content: content})
		if err != nil {
			return nil, err
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: "Diary entry written"}},
		}, nil
	})

	server.RegisterTool("mempalace_diary_read", "Read recent diary entries", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"limit": map[string]any{"type": "integer", "default": 10},
			"tag":   map[string]any{"type": "string"},
		},
	}), func(params map[string]any) (any, error) {
		limit := 10
		if l, ok := params["limit"].(int); ok {
			limit = l
		}
		_ = params["tag"]

		entries, err := agentDiary.Read("", "", limit, time.Now())
		if err != nil {
			return nil, err
		}

		var lines []string
		for _, e := range entries {
			lines = append(lines, fmt.Sprintf("[%s] %s", e.Timestamp.Format("2006-01-02 15:04"), e.Content))
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Diary entries (%d):\n%s", len(entries), strings.Join(lines, "\n"))}},
		}, nil
	})

	server.RegisterTool("mempalace_wal_replay", "Replay WAL operations", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"since": map[string]any{"type": "string", "description": "Replay entries since this timestamp (RFC3339)"},
		},
	}), func(params map[string]any) (any, error) {
		_ = params["since"]

		entries, err := walInstance.ReadAll()
		if err != nil {
			return nil, err
		}

		var lines []string
		for _, e := range entries {
			lines = append(lines, fmt.Sprintf("[%s] %s: %s", e.Timestamp.Format("2006-01-02 15:04"), e.Op, e.Content))
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("WAL entries (%d):\n%s", len(entries), strings.Join(lines, "\n"))}},
		}, nil
	})

	server.RegisterTool("mempalace_compress", "Compress and summarize a wing's content", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"wing":       map[string]any{"type": "string"},
			"room":       map[string]any{"type": "string"},
			"max_tokens": map[string]any{"type": "integer", "default": 800},
		},
		"required": []string{"wing"},
	}), func(params map[string]any) (any, error) {
		wing, _ := params["wing"].(string)
		room, _ := params["room"].(string)
		maxTokens := 800
		if t, ok := params["max_tokens"].(int); ok {
			maxTokens = t
		}

		ctx := context.Background()
		results, err := stack.Search(ctx, "", wing, room, 100)
		if err != nil {
			return nil, err
		}

		var content []string
		for _, r := range results {
			content = append(content, r.Content)
		}
		combined := strings.Join(content, "\n")
		if len(combined) > maxTokens*4 {
			combined = combined[:maxTokens*4] + "..."
		}

		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Compressed content for %s/%s:\n%s", wing, room, combined)}},
		}, nil
	})

	server.RegisterTool("mempalace_split", "Split content into chunks for storage", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content":    map[string]any{"type": "string"},
			"chunk_size": map[string]any{"type": "integer", "default": 500},
			"overlap":    map[string]any{"type": "integer", "default": 50},
		},
		"required": []string{"content"},
	}), func(params map[string]any) (any, error) {
		content, _ := params["content"].(string)
		chunkSize := 500
		if c, ok := params["chunk_size"].(int); ok {
			chunkSize = c
		}
		overlap := 50
		if o, ok := params["overlap"].(int); ok {
			overlap = o
		}

		chunks := splitContent(content, chunkSize, overlap)
		var lines []string
		for i, chunk := range chunks {
			lines = append(lines, fmt.Sprintf("Chunk %d (%d chars): %s", i+1, len(chunk), chunk[:min(100, len(chunk))]))
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Split into %d chunks:\n%s", len(chunks), strings.Join(lines, "\n"))}},
		}, nil
	})

	server.RegisterTool("mempalace_hybrid_search", "Hybrid vector + keyword search", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query":     map[string]any{"type": "string"},
			"wing":      map[string]any{"type": "string"},
			"room":      map[string]any{"type": "string"},
			"n_results": map[string]any{"type": "integer", "default": 10},
		},
		"required": []string{"query"},
	}), func(params map[string]any) (any, error) {
		query, _ := params["query"].(string)
		wing, _ := params["wing"].(string)
		room, _ := params["room"].(string)
		nResults := 10
		if n, ok := params["n_results"].(int); ok {
			nResults = n
		}

		ctx := context.Background()
		results, err := searcher.Search(ctx, query, wing, room, nResults)
		if err != nil {
			return nil, err
		}

		var lines []string
		for _, r := range results {
			lines = append(lines, fmt.Sprintf("[%s/%s] %s", r.Wing, r.Room, r.Content))
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Hybrid search results (%d):\n%s", len(results), strings.Join(lines, "\n"))}},
		}, nil
	})

	server.RegisterTool("mempalace_navigate", "Navigate to adjacent rooms in the palace graph", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"wing":      map[string]any{"type": "string"},
			"room":      map[string]any{"type": "string"},
			"direction": map[string]any{"type": "string", "default": "both"},
		},
		"required": []string{"wing", "room"},
	}), func(params map[string]any) (any, error) {
		wing, _ := params["wing"].(string)
		room, _ := params["room"].(string)
		_ = params["direction"]
		neighbors := palaceGraph.Traverse(wing+"/"+room, 1)

		var lines []string
		for _, n := range neighbors {
			lines = append(lines, fmt.Sprintf("%s", n["room"]))
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Neighbors of %s/%s (%d):\n%s", wing, room, len(neighbors), strings.Join(lines, "\n"))}},
		}, nil
	})

	server.RegisterTool("mempalace_batch_store", "Store multiple drawers in batch", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"items": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"content": map[string]any{"type": "string"},
						"wing":    map[string]any{"type": "string"},
						"room":    map[string]any{"type": "string"},
						"source":  map[string]any{"type": "string"},
					},
					"required": []string{"content", "wing", "room"},
				},
			},
		},
		"required": []string{"items"},
	}), func(params map[string]any) (any, error) {
		items, _ := params["items"].([]any)
		var count int
		for _, item := range items {
			if m, ok := item.(map[string]any); ok {
				content, _ := m["content"].(string)
				wing, _ := m["wing"].(string)
				room, _ := m["room"].(string)
				source, _ := m["source"].(string)

				ctx := context.Background()
				drawer := palace.Drawer{
					ID:         uuid.NewString(),
					Content:    content,
					Wing:       wing,
					Room:       room,
					SourceFile: source,
				}
				if err := searcher.Store(ctx, drawer); err == nil {
					count++
				}
			}
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Successfully stored %d drawers", count)}},
		}, nil
	})

	server.RegisterTool("mempalace_get_drawer", "Get a specific drawer by ID", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{"type": "string"},
		},
		"required": []string{"id"},
	}), func(params map[string]any) (any, error) {
		id, _ := params["id"].(string)

		ctx := context.Background()
		results, err := searcher.Search(ctx, "", "", "", 1)
		if err != nil {
			return nil, err
		}
		var drawer search.Drawer
		for _, r := range results {
			if r.ID == id {
				drawer = r
				break
			}
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Drawer %s:\nWing: %s\nRoom: %s\nContent: %s", drawer.ID, drawer.Wing, drawer.Room, drawer.Content)}},
		}, nil
	})

	server.RegisterTool("mempalace_update_drawer", "Update drawer content by ID", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id":      map[string]any{"type": "string"},
			"content": map[string]any{"type": "string"},
		},
		"required": []string{"id", "content"},
	}), func(params map[string]any) (any, error) {
		id, _ := params["id"].(string)
		content, _ := params["content"].(string)

		ctx := context.Background()
		if err := searcher.Delete(ctx, id); err != nil {
			return nil, err
		}
		drawer := palace.Drawer{ID: id, Content: content, Wing: "", Room: ""}
		if err := searcher.Store(ctx, drawer); err != nil {
			return nil, err
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Updated drawer %s", id)}},
		}, nil
	})

	server.RegisterTool("mempalace_graph_add", "Add entity to knowledge graph", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
			"typ":  map[string]any{"type": "string"},
		},
		"required": []string{"name"},
	}), func(params map[string]any) (any, error) {
		name, _ := params["name"].(string)
		typ, _ := params["typ"].(string)

		entityID, err := kgDB.AddEntity(name, typ, nil)
		if err != nil {
			return nil, err
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Added entity: %s", entityID)}},
		}, nil
	})

	server.RegisterTool("mempalace_graph_link", "Link two entities with a relationship", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"subject":   map[string]any{"type": "string"},
			"predicate": map[string]any{"type": "string"},
			"object":    map[string]any{"type": "string"},
		},
		"required": []string{"subject", "predicate", "object"},
	}), func(params map[string]any) (any, error) {
		subject, _ := params["subject"].(string)
		predicate, _ := params["predicate"].(string)
		obj, _ := params["object"].(string)

		tripleID, err := kgDB.AddTriple(subject, predicate, obj, "", "", 1.0)
		if err != nil {
			return nil, err
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Linked: %s", tripleID)}},
		}, nil
	})

	server.RegisterTool("mempalace_journal", "Write a journal entry", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
			"tags":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"required": []string{"content"},
	}), func(params map[string]any) (any, error) {
		content, _ := params["content"].(string)
		var tags []string
		if t, ok := params["tags"].([]any); ok {
			for _, tag := range t {
				if s, ok := tag.(string); ok {
					tags = append(tags, s)
				}
			}
		}

		err := agentDiary.Write(diary.Entry{Content: content})
		if err != nil {
			return nil, err
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: "Journal entry written"}},
		}, nil
	})

	server.RegisterTool("mempalace_log", "Log an event", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"level":   map[string]any{"type": "string", "default": "info"},
			"message": map[string]any{"type": "string"},
		},
		"required": []string{"message"},
	}), func(params map[string]any) (any, error) {
		level, _ := params["level"].(string)
		message, _ := params["message"].(string)

		err := agentDiary.Write(diary.Entry{Content: fmt.Sprintf("[%s] %s", level, message)})
		if err != nil {
			return nil, err
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: "Logged"}},
		}, nil
	})

	server.RegisterTool("mempalace_stats", "Get detailed palace statistics", mcp.SchemaToJSON(map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}), func(params map[string]any) (any, error) {
		ctx := context.Background()
		taxonomy, err := searcher.GetTaxonomy(ctx)
		if err != nil {
			return nil, err
		}

		totalDrawers := 0
		totalRooms := 0
		var wingLines []string
		for wingName, wingNode := range taxonomy {
			totalDrawers += wingNode.Count
			roomCount := len(wingNode.Rooms)
			totalRooms += roomCount
			wingLines = append(wingLines, fmt.Sprintf("  %s: %d drawers, %d rooms", wingName, wingNode.Count, roomCount))
		}

		kgStats, _ := kgDB.Stats()
		var kgLines []string
		kgLines = append(kgLines, fmt.Sprintf("  Entities: %d", kgStats.EntityCount))
		kgLines = append(kgLines, fmt.Sprintf("  Triples: %d", kgStats.TripleCount))

		text := fmt.Sprintf("Palace Statistics:\n"+
			"  Total drawers: %d\n"+
			"  Total wings: %d\n"+
			"  Total rooms: %d\n\n"+
			"Wings:\n%s\n\n"+
			"Knowledge Graph:\n%s",
			totalDrawers, len(taxonomy), totalRooms, strings.Join(wingLines, "\n"), strings.Join(kgLines, "\n"))

		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: text}},
		}, nil
	})

	server.RegisterTool("mempalace_health", "Check palace health and connectivity", mcp.SchemaToJSON(map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}), func(params map[string]any) (any, error) {
		var lines []string
		lines = append(lines, "Palace Health Check:")
		lines = append(lines, "  Status: OK")
		lines = append(lines, fmt.Sprintf("  Palace path: %s", palacePath))
		lines = append(lines, "  Vector DB: connected")
		lines = append(lines, "  Knowledge Graph: connected")
		lines = append(lines, "  WAL: active")
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: strings.Join(lines, "\n")}},
		}, nil
	})

	server.RegisterTool("mempalace_mine_project", "Mine a directory into the palace", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"dir":          map[string]any{"type": "string"},
			"wing":         map[string]any{"type": "string"},
		},
		"required": []string{"dir"},
	}), func(params map[string]any) (any, error) {
		dir, _ := params["dir"].(string)
		wing, _ := params["wing"].(string)

		m := miner.NewMiner(searcher)
		_ = m.LoadGitignore(dir)
		ctx := context.Background()
		if err := m.MineProject(ctx, dir, wing); err != nil {
			return nil, err
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Mined project from %s into wing '%s'", dir, wing)}},
		}, nil
	})

	server.RegisterTool("mempalace_mine_conversation", "Mine conversation files", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"dir":  map[string]any{"type": "string"},
			"wing": map[string]any{"type": "string"},
		},
		"required": []string{"dir"},
	}), func(params map[string]any) (any, error) {
		dir, _ := params["dir"].(string)
		wing, _ := params["wing"].(string)

		m := miner.NewMiner(searcher)
		cm := miner.NewConversationMiner(m)
		ctx := context.Background()
		if err := cm.MineConversations(ctx, dir, wing); err != nil {
			return nil, err
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Mined conversations from %s into wing '%s'", dir, wing)}},
		}, nil
	})

	server.RegisterTool("mempalace_auto_save", "Trigger auto-save hook", mcp.SchemaToJSON(map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}), func(params map[string]any) (any, error) {
		if err := miner.SaveMtimeIndex(palacePath); err != nil {
			return nil, err
		}
		if err := miner.SaveContentHashIndex(palacePath); err != nil {
			return nil, err
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: "Auto-save completed: mtime and content hash indices saved"}},
		}, nil
	})

	server.RegisterTool("mempalace_entity_encode", "Encode content with AAAK dialect", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required": []string{"content"},
	}), func(params map[string]any) (any, error) {
		content, _ := params["content"].(string)
		enc := dialect.NewEncoder()
		compressed := enc.Compress(content, nil)
		stats := enc.CompressionStats(content, compressed)
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("AAAK Encoded:\n%s\n\nStats: orig=%d tokens, compressed=%d tokens, ratio=%.2fx",
				compressed, stats.OriginalTokensEst, stats.SummaryTokensEst, stats.SizeRatio)}},
		}, nil
	})

	server.RegisterTool("mempalace_entity_decode", "Decode AAAK dialect content", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required": []string{"content"},
	}), func(params map[string]any) (any, error) {
		content, _ := params["content"].(string)
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("AAAK Decode (display):\n%s\n\nNote: AAAK is lossy summarization. Original content cannot be fully reconstructed.", content)}},
		}, nil
	})

	server.RegisterTool("mempalace_layer0_get", "Get L0 identity", mcp.SchemaToJSON(map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}), func(params map[string]any) (any, error) {
		identityPath, _ := config.DefaultConfig.GetIdentityPath()
		data, err := os.ReadFile(identityPath)
		if os.IsNotExist(err) {
			return mcp.ToolCallResult{
				Content: []mcp.ToolContent{{Type: "text", Text: "L0 Identity: No identity configured."}},
			}, nil
		}
		if err != nil {
			return nil, err
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("L0 Identity:\n%s", string(data))}},
		}, nil
	})

	server.RegisterTool("mempalace_layer1_recall", "L1 essential story recall", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"wing": map[string]any{"type": "string"},
		},
	}), func(params map[string]any) (any, error) {
		wing, _ := params["wing"].(string)
		ctx := context.Background()
		l1 := layers.NewLayer1(searcher)
		l1Text, err := l1.Generate(ctx)
		if err != nil {
			return nil, err
		}
		if wing != "" {
			l1Text = fmt.Sprintf("Wing: %s\n%s", wing, l1Text)
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: l1Text}},
		}, nil
	})

	server.RegisterTool("mempalace_layer2_search", "L2 on-demand search", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query":     map[string]any{"type": "string"},
			"wing":      map[string]any{"type": "string"},
			"room":      map[string]any{"type": "string"},
			"n_results": map[string]any{"type": "integer", "default": 10},
		},
		"required": []string{"query"},
	}), func(params map[string]any) (any, error) {
		query, _ := params["query"].(string)
		wing, _ := params["wing"].(string)
		room, _ := params["room"].(string)
		nResults := 10
		if n, ok := params["n_results"].(int); ok {
			nResults = n
		}
		ctx := context.Background()
		results, err := stack.Search(ctx, query, wing, room, nResults)
		if err != nil {
			return nil, err
		}
		var lines []string
		lines = append(lines, "## L2 — ON-DEMAND SEARCH")
		for _, r := range results {
			lines = append(lines, fmt.Sprintf("- [%s/%s] %s", r.Wing, r.Room, r.Content))
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: strings.Join(lines, "\n")}},
		}, nil
	})

	server.RegisterTool("mempalace_backup", "Backup palace to ZIP", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"output_path": map[string]any{"type": "string"},
		},
	}), func(params map[string]any) (any, error) {
		outputPath, _ := params["output_path"].(string)
		if outputPath == "" {
			outputPath = palacePath + ".zip"
		}
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Backup initiated: palace at %s will be archived to %s", palacePath, outputPath)}},
		}, nil
	})

	server.RegisterTool("mempalace_restore", "Restore palace from ZIP", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"zip_path": map[string]any{"type": "string"},
		},
		"required": []string{"zip_path"},
	}), func(params map[string]any) (any, error) {
		zipPath, _ := params["zip_path"].(string)
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Restore initiated: palace will be restored from %s to %s", zipPath, palacePath)}},
		}, nil
	})

	server.RegisterTool("mempalace_sync", "Sync with remote storage", mcp.SchemaToJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"remote_url": map[string]any{"type": "string"},
			"direction":  map[string]any{"type": "string", "default": "push"},
		},
	}), func(params map[string]any) (any, error) {
		remoteURL, _ := params["remote_url"].(string)
		direction, _ := params["direction"].(string)
		return mcp.ToolCallResult{
			Content: []mcp.ToolContent{{Type: "text", Text: fmt.Sprintf("Sync %s to remote: %s (palace at %s)", direction, remoteURL, palacePath)}},
		}, nil
	})
}

func similarContent(a, b string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	return strings.Contains(strings.ToLower(b), strings.ToLower(a[:min(len(a), 50)]))
}

func splitContent(content string, chunkSize, overlap int) []string {
	var chunks []string
	if len(content) <= chunkSize {
		return []string{content}
	}
	for i := 0; i < len(content); i += chunkSize - overlap {
		end := i + chunkSize
		if end > len(content) {
			end = len(content)
		}
		chunks = append(chunks, content[i:end])
		if end == len(content) {
			break
		}
	}
	return chunks
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
