package facade

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ProjectViVy/laputa/mentle/internal/palace"
)

// DrawerWriter is the narrow mutation capability granted to import and
// mining adapters. It exposes canonical writes but no vector/raw-DB handle.
type DrawerWriter interface {
	CreateDrawer(context.Context, palace.Drawer, string, string) (MemoryMutationResult, error)
}

// CreateDrawer is the compatibility-shaped adapter for file/conversation
// producers. It converts a drawer into canonical memory input and gives the
// caller no mutable vector-store capability.
func (s *Service) CreateDrawer(ctx context.Context, drawer palace.Drawer, actor, requestID string) (MemoryMutationResult, error) {
	if drawer.Content == "" {
		return MemoryMutationResult{}, fmt.Errorf("memory content is required")
	}
	metadata := make(map[string]any, len(drawer.Metadata)+5)
	for key, value := range drawer.Metadata {
		metadata[key] = value
	}
	if drawer.Wing != "" {
		metadata["wing"] = drawer.Wing
	}
	if drawer.Room != "" {
		metadata["room"] = drawer.Room
	}
	if drawer.SourceFile != "" {
		metadata["source_file"] = drawer.SourceFile
	}
	metadata["chunk_index"] = drawer.ChunkIndex
	if drawer.AddedBy != "" {
		metadata["added_by"] = drawer.AddedBy
	}

	request := CreateMemoryRequest{
		Content: drawer.Content,
		Kind:    "source_artifact",
		Scope:   drawer.Wing,
		Source: MemorySource{
			Type: "import",
			URI:  drawer.SourceFile,
		},
		Metadata:  metadata,
		Actor:     actor,
		RequestID: requestID,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return MemoryMutationResult{}, err
	}
	hash := sha256.Sum256(body)
	key := fmt.Sprintf("drawer:%x", hash[:])
	return s.CreateMemoryWithIndexStatus(ctx, request, key, "sha256:"+hex.EncodeToString(hash[:]))
}
