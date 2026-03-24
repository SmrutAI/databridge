package core

import "context"

// Source produces Records from an external data origin (filesystem, S3, Azure Blob).
// Implementations must be safe to call from a single goroutine.
type Source interface {
	// Name returns a human-readable identifier used in logs and stats.
	Name() string

	// Open initialises the source (e.g. opens directory, creates S3 client).
	// Must be called once before Records.
	Open(ctx context.Context) error

	// Records returns a channel that receives every Record from the source.
	// The channel is closed when all records have been sent or ctx is cancelled.
	// The caller must drain the channel fully to avoid goroutine leaks.
	Records(ctx context.Context) (<-chan *Record, error)

	// Close releases any resources held by the source.
	Close() error
}

// Transform converts one input Record into zero or more output Records.
// A Transform that returns an empty slice silently drops the record (e.g. MerkleDedup).
// A Transform that returns multiple records performs 1:N fan-out (e.g. ASTParser).
type Transform interface {
	// Name returns a human-readable identifier.
	Name() string

	// Apply processes one Record and returns zero or more Records.
	// Returning (nil, nil) or ([]Record{}, nil) drops the input record.
	Apply(ctx context.Context, in *Record) ([]*Record, error)
}

// Sink consumes Records and writes them to a persistent store.
type Sink interface {
	// Name returns a human-readable identifier.
	Name() string

	// Open initialises the sink (e.g. connects to database).
	Open(ctx context.Context) error

	// Write persists a single Record.
	Write(ctx context.Context, r *Record) error

	// Close flushes any buffered data and releases resources.
	Close() error
}

// Embedder converts text into a dense float32 vector.
type Embedder interface {
	// Embed returns the vector for a single text string.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch returns vectors for multiple text strings in one call.
	// The returned slice has the same length and order as texts.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Dimension returns the length of every embedding vector produced by this embedder.
	Dimension() int

	// Close releases any resources held by the embedder (e.g. ONNX session).
	Close() error
}
