---
type: Overview
title: databridge
status: stable
tags:
- readme
---

# databridge

`databridge` is Smritea's document- and code-ingestion microservice. It reads
files from a source (a local directory, an S3 bucket, or an Azure Blob
container), parses and chunks each file, embeds the chunks, deduplicates
unchanged content with a Merkle tree, and writes the results to one or more sinks
(Smritea memory, Qdrant, or PostgreSQL).

| Component | Path | Responsibility |
|-----------|------|----------------|
| Sources | `modules/golang/databridge/source/` | Read files from local disk, S3, or Azure Blob |
| Transforms | `modules/golang/databridge/transform/` | Merkle dedup, Go/Python AST parsing, markdown chunking, embedding |
| Sinks | `modules/golang/databridge/sink/` | Write chunks to Smritea, Qdrant, or PostgreSQL |
| HTTP server | `modules/golang/databridge/server/` | Echo app exposing the `/v1/*` ingestion API |
| Entrypoints | `modules/golang/databridge/cmd/` | The `server`, `lambda`, and `codewatch` binaries |
| Internal libs | `modules/golang/databridge/internal/` | Embedder, Merkle tree, parsers, job store, flow engine |
| Azure binding | `modules/golang/databridge/api/` | Azure Functions HTTP-trigger definition |

## Pipeline

Every run assembles the same pipeline: **source → transforms → sinks**. The
transforms run Merkle-tree deduplication (so unchanged files are skipped), Go and
Python AST parsing, markdown chunking, and chunk embedding. Sinks are enabled by
environment variables and at least one must be configured.

## Prerequisites

- Go 1.25+
- `CGO_ENABLED=1` — required for the in-process ONNX embedder
- A PostgreSQL instance if you want async job tracking (server) — optional for the CLI
- At least one sink configured: a Smritea API key and/or a Qdrant host

## Build

All build targets live in the module Makefile (`modules/golang/databridge/Makefile`).
Binaries are written to a `dist` directory under the module:

```bash
make build          # build all three binaries
make build-server   # standalone / Azure Functions server binary
make build-lambda   # AWS Lambda binary (bootstrap)
make build-cli      # local CLI (codewatch)
make test-run       # run unit tests
make lint           # run golangci-lint
```

## Running

### Local CLI (codewatch)

Index a local directory straight into the Smritea sink:

```bash
export SMRITEA_API_KEY=...              # enables the Smritea sink
export CODEWATCH_EMBEDDER_API_KEY=...   # embedding-provider key
./dist/codewatch --input /path/to/repo --workspace my-workspace-id
```

`--input` (directory to index) and `--workspace` (workspace id) are both required.

### HTTP server

```bash
export CODEWATCH_DSN="postgres://user:pass@host:5432/databridge"  # enables job tracking
export SMRITEA_API_KEY=...       # and/or QDRANT_HOST=... to enable sinks
./dist/server                    # listens on $PORT (default 8080)
```

The server exposes:

| Method | Route | Purpose |
|--------|-------|---------|
| GET | /v1/health | Liveness probe |
| POST | /v1/index | Start an async indexing job; returns a job id |
| GET | /v1/jobs/:id | Poll an indexing job's status |
| GET | /v1/flows | List registered flows |
| POST | /v1/flows/:name/run | Run a registered flow synchronously |

`POST /v1/index` accepts a JSON body of `{"source": "local|s3|azure", "workspace_id": "...", "config": { ... }}`.
The `config` keys depend on the source: `input` for local; `bucket` and `prefix`
for S3; `account_url`, `account_name`, `account_key`, `container`, and `prefix`
for Azure.

### Deployment targets

The same server binary runs as an Azure Functions custom handler
(`modules/golang/databridge/host.json` plus `modules/golang/databridge/api/function.json`).
`make build-lambda` produces an AWS Lambda `bootstrap` binary. Docker images:

```bash
make docker-server   # standalone image
make docker-lambda   # AWS Lambda image
make docker-azure    # Azure Functions image
```

## Configuration

Configuration is entirely environment-driven.

### Server / pipeline

| Variable | Default | Purpose |
|----------|---------|---------|
| `CODEWATCH_DSN` | (unset) | PostgreSQL DSN; enables async job tracking. Job tracking is disabled when unset |
| `PORT` / `FUNCTIONS_CUSTOMHANDLER_PORT` | `8080` | HTTP listen port |

### Embedder

Selected by `CODEWATCH_EMBEDDER` (`api` by default, or `hugot` for the in-process
ONNX embedder):

| Variable | Default | Purpose |
|----------|---------|---------|
| `CODEWATCH_EMBEDDER` | `api` | `api` (OpenAI-compatible HTTP) or `hugot` (in-process ONNX) |
| `CODEWATCH_EMBEDDER_API_URL` | Voyage AI endpoint | API-embedder base URL |
| `CODEWATCH_EMBEDDER_API_KEY` | (unset) | API-embedder key |
| `CODEWATCH_EMBEDDER_MODEL` | `voyage-code-3` | API-embedder model |
| `CODEWATCH_EMBEDDER_DIM` | `1024` | Embedding dimension |
| `CODEWATCH_MODEL_PATH` | (unset) | ONNX model path; required when `CODEWATCH_EMBEDDER=hugot` |

### Sinks (auto-detected)

A sink is enabled when its key variable is set. The server flow requires at least one:

| Variable | Enables | Also reads |
|----------|---------|------------|
| `SMRITEA_API_KEY` | Smritea sink | `SMRITEA_APP_ID`, `SMRITEA_BASE_URL` |
| `QDRANT_HOST` | Qdrant sink | `QDRANT_PORT`, `QDRANT_COLLECTION`, `QDRANT_USE_TLS` |
