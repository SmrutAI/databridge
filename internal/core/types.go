package core

import "time"

// ActionUpsert indicates the record should be inserted or updated in sinks.
const ActionUpsert = "upsert"

// ActionDelete indicates the record (and all its chunks) should be removed from sinks.
const ActionDelete = "delete"

// Record is the canonical data unit flowing through the pipeline.
// A single source file produces one Record (with Content = full file text),
// which transforms then expand into multiple chunk Records (one per symbol/section).
type Record struct {
	// ID is a unique identifier for this record within a pipeline run.
	// Set by the source or transform that created it.
	ID string

	// SourceID is the workspace identifier — groups all records from one indexing job.
	SourceID string

	// Path is the relative file path within the workspace (e.g. "pkg/auth/login.go").
	Path string

	// Symbol is the named code unit within the file (e.g. "Login", "UserService").
	// Empty for whole-file records produced by sources.
	Symbol string

	// SymbolType describes the kind of symbol: "func", "method", "type", "class", "section".
	// Empty for whole-file records.
	SymbolType string

	// Language is the programming language: "go", "python", "markdown", etc.
	Language string

	// Content is the raw text: full file content for source records,
	// extracted symbol body for chunk records.
	Content string

	// ContentHash is the SHA-256 hex digest of Content.
	// Used by MerkleDedup to detect unchanged content.
	ContentHash string

	// Embedding is the vector representation of Content, populated by ChunkEmbedder.
	// Length matches the embedder's Dimension() — typically 384 for all-MiniLM-L6-v2.
	Embedding []float32

	// Metadata holds arbitrary key-value pairs for downstream sinks.
	Metadata map[string]any

	// Action is either ActionUpsert (default) or ActionDelete.
	Action string
}

// RecordBatch groups all Records derived from a single source file.
// It is the pipeline unit that flows through transform stages, enabling
// WorkerModeTransaction for all non-source nodes while still supporting
// 1:N fan-out per file (e.g. GoASTParser returning one record per symbol).
type RecordBatch []*Record

// FlowStats summarises the result of a single pipeline run.
type FlowStats struct {
	// FlowName is the name passed to NewFlow.
	FlowName string

	// RecordsIn is the total number of records emitted by the source.
	RecordsIn int64

	// RecordsOut is the number of records successfully written by all sinks.
	RecordsOut int64

	// RecordsSkipped is the number of records dropped by MerkleDedup (unchanged).
	RecordsSkipped int64

	// RecordsDeleted is the number of delete-action records processed.
	RecordsDeleted int64

	// RecordsFailed is the total number of records that encountered errors during
	// the pipeline run, as reported by the conveyor error stats.
	RecordsFailed int64

	// ErrorsByStage is a per-stage error breakdown keyed by "StageName:*RootErrorType".
	// Populated from the conveyor error stats snapshot after the pipeline completes.
	ErrorsByStage map[string]int64

	// Duration is the wall-clock time for the full pipeline run.
	Duration time.Duration

	// Error holds the first non-nil error returned by the pipeline, if any.
	Error string
}
