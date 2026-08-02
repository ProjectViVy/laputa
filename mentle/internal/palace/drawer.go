package palace

import "time"

type Drawer struct {
	ID         string            `json:"id"`
	Content    string            `json:"content"`
	Wing       string            `json:"wing"`
	Room       string            `json:"room"`
	SourceFile string            `json:"source_file"`
	ChunkIndex int               `json:"chunk_index"`
	AddedBy    string            `json:"added_by"`
	FiledAt    time.Time         `json:"filed_at"`
	Metadata   map[string]string `json:"metadata"`
}
