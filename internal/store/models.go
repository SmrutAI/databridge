package store

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	pgvector "github.com/pgvector/pgvector-go"
)

// JSONMap is a map[string]any that serialises to/from Postgres JSONB.
type JSONMap map[string]any

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	b, err := json.Marshal(j)
	if err != nil {
		return nil, fmt.Errorf("jsonmap: marshal: %w", err)
	}
	return string(b), nil
}

func (j *JSONMap) Scan(value any) error {
	if value == nil {
		*j = JSONMap{}
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("jsonmap: unsupported type %T", value)
	}
	return json.Unmarshal(b, j)
}

// MerkleSnapshot tracks the content hash of each file per workspace.
// Maps to the merkle_snapshots table.
type MerkleSnapshot struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WorkspaceID string    `gorm:"column:workspace_id;not null;uniqueIndex:uniq_workspace_filepath"`
	FilePath    string    `gorm:"column:file_path;not null;uniqueIndex:uniq_workspace_filepath"`
	ContentHash string    `gorm:"column:content_hash;not null"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (MerkleSnapshot) TableName() string { return "merkle_snapshots" }

// Chunk holds an indexed code symbol with its embedding.
// Maps to the chunks table.
type Chunk struct {
	ID          string          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WorkspaceID string          `gorm:"column:workspace_id;not null;index:idx_chunks_workspace;uniqueIndex:uniq_chunk"`
	FilePath    string          `gorm:"column:file_path;not null;index:idx_chunks_file,composite:workspace;uniqueIndex:uniq_chunk"`
	Symbol      string          `gorm:"column:symbol;not null;uniqueIndex:uniq_chunk"`
	SymbolType  string          `gorm:"column:symbol_type;not null"`
	Language    string          `gorm:"column:language;not null"`
	Content     string          `gorm:"column:content;not null"`
	ContentHash string          `gorm:"column:content_hash;not null"`
	Embedding   pgvector.Vector `gorm:"column:embedding;type:vector(1024)"`
	Metadata    JSONMap         `gorm:"column:metadata;type:jsonb;not null;default:'{}'"`
	CreatedAt   time.Time       `gorm:"column:created_at"`
	UpdatedAt   time.Time       `gorm:"column:updated_at;autoUpdateTime"`
}

func (Chunk) TableName() string { return "chunks" }

// Job tracks an async indexing request.
// Maps to the index_jobs table.
type Job struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WorkspaceID string    `gorm:"column:workspace_id;not null"`
	Status      string    `gorm:"column:status;not null;default:queued"`
	TotalFiles  int       `gorm:"column:total_files;not null;default:0"`
	Done        int       `gorm:"column:done;not null;default:0"`
	Failed      int       `gorm:"column:failed;not null;default:0"`
	ErrorMsg    string    `gorm:"column:error_msg"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Job) TableName() string { return "index_jobs" }
