-- Enable pgvector extension (must be done once per database by a superuser)
-- CREATE EXTENSION IF NOT EXISTS vector;

-- Merkle snapshots: track content hash per file per workspace
CREATE TABLE IF NOT EXISTS merkle_snapshots (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    TEXT NOT NULL,
    file_path       TEXT NOT NULL,
    content_hash    TEXT NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id, file_path)
);

-- Code chunks with embeddings
CREATE TABLE IF NOT EXISTS chunks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    TEXT NOT NULL,
    file_path       TEXT NOT NULL,
    symbol          TEXT NOT NULL,
    symbol_type     TEXT NOT NULL,
    language        TEXT NOT NULL,
    content         TEXT NOT NULL,
    content_hash    TEXT NOT NULL,
    embedding       vector(1024),
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id, file_path, symbol)
);

CREATE INDEX IF NOT EXISTS idx_chunks_workspace ON chunks(workspace_id);
CREATE INDEX IF NOT EXISTS idx_chunks_file ON chunks(workspace_id, file_path);
CREATE INDEX IF NOT EXISTS idx_merkle_workspace ON merkle_snapshots(workspace_id);

-- Job tracking for async indexing requests
CREATE TABLE IF NOT EXISTS index_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'queued',
    total_files     INT NOT NULL DEFAULT 0,
    done            INT NOT NULL DEFAULT 0,
    failed          INT NOT NULL DEFAULT 0,
    error_msg       TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
