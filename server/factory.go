package server

import (
	"fmt"
	"os"

	"github.com/SmrutAI/databridge/internal/core"
	"github.com/SmrutAI/databridge/internal/embedder"
	"github.com/SmrutAI/databridge/internal/flow"
	"github.com/SmrutAI/databridge/internal/merkle"
	"github.com/SmrutAI/databridge/sink"
	"github.com/SmrutAI/databridge/source"
	"github.com/SmrutAI/databridge/transform"
)

// BuildFlow constructs a ready-to-run *flow.Flow from the given parameters.
//
// sourceType selects which data source to use:
//   - "local" or "" — LocalFileSource; config["input"] must be non-empty
//   - "s3"          — S3Source; config["bucket"] and config["prefix"] are used
//   - "azure"       — AzureBlobSource; config["account_url"], config["account_name"],
//     config["account_key"], config["container"], and config["prefix"] are used
//
// Sinks are auto-detected from environment variables:
//
//	SMRITEA_API_KEY — enables SmriteaSink
//	QDRANT_HOST     — enables QdrantSink
//
// At least one sink must be reachable or an error is returned.
func BuildFlow(workspaceID, sourceType string, config map[string]string) (*flow.Flow, error) {
	// --- Source ---
	var src core.Source

	switch sourceType {
	case "", "local":
		input := config["input"]
		if input == "" {
			return nil, fmt.Errorf("build flow: source type %q requires config[\"input\"]", sourceType)
		}
		src = source.NewLocalFileSource(workspaceID, input, nil)

	case "s3":
		src = source.NewS3Source(workspaceID, config["bucket"], config["prefix"], nil)

	case "azure":
		azSrc, err := source.NewAzureBlobSource(
			workspaceID,
			config["account_url"],
			config["account_name"],
			config["account_key"],
			config["container"],
			config["prefix"],
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("build flow: azure source: %w", err)
		}
		src = azSrc

	default:
		return nil, fmt.Errorf("build flow: unknown source type %q (want: local, s3, azure)", sourceType)
	}

	// --- Transforms ---
	tree := merkle.NewTree()
	dedup := transform.NewMerkleDedup(tree)

	emb, err := embedder.NewEmbedder()
	if err != nil {
		return nil, fmt.Errorf("build flow: embedder: %w", err)
	}
	chunkEmb := transform.NewChunkEmbedder(emb)

	// --- Sinks (auto-detected from environment) ---
	var sinkList []core.Sink

	if os.Getenv("SMRITEA_API_KEY") != "" {
		smrSink, sinkErr := sink.NewSmriteaSink()
		if sinkErr != nil {
			return nil, fmt.Errorf("build flow: smritea sink: %w", sinkErr)
		}
		sinkList = append(sinkList, smrSink)
	}

	if os.Getenv("QDRANT_HOST") != "" {
		qdSink, sinkErr := sink.NewQdrantSink()
		if sinkErr != nil {
			return nil, fmt.Errorf("build flow: qdrant sink: %w", sinkErr)
		}
		sinkList = append(sinkList, qdSink)
	}

	if len(sinkList) == 0 {
		return nil, fmt.Errorf("build flow: no sinks configured (set SMRITEA_API_KEY or QDRANT_HOST)")
	}

	// --- Assemble the flow ---
	f := flow.NewFlow(workspaceID).
		Source(src).
		Transform(dedup).
		Transform(&transform.GoASTParser{}).
		Transform(&transform.PythonASTParser{}).
		Transform(&transform.MarkdownChunker{}).
		Transform(chunkEmb)

	for i := range sinkList {
		f = f.Sink(sinkList[i])
	}

	return f, nil
}
