package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/SmrutAI/databridge/internal/embedder"
	"github.com/SmrutAI/databridge/internal/flow"
	"github.com/SmrutAI/databridge/internal/merkle"
	"github.com/SmrutAI/databridge/sink"
	"github.com/SmrutAI/databridge/source"
	"github.com/SmrutAI/databridge/transform"
)

func main() {
	input := flag.String("input", "", "Path to the directory to index (required)")
	workspace := flag.String("workspace", "", "Workspace ID to associate with indexed files (required)")
	_ = flag.String("source", "local", "Source type: local, s3, azure")
	flag.Parse()

	if *input == "" || *workspace == "" {
		fmt.Fprintln(os.Stderr, "Usage: codewatch --input <dir> --workspace <id>")
		os.Exit(1)
	}

	emb, err := embedder.NewEmbedder()
	if err != nil {
		fmt.Fprintf(os.Stderr, "create embedder: %v\n", err)
		os.Exit(1)
	}

	tree := merkle.NewTree()

	smriteaSink, err := sink.NewSmriteaSink()
	if err != nil {
		fmt.Fprintf(os.Stderr, "create smritea sink: %v\n", err)
		os.Exit(1)
	}

	src := source.NewLocalFileSource(*workspace, *input, nil)
	registry := flow.NewFlowRegistry()

	f := flow.NewFlow("local-index").
		Source(src).
		Transform(transform.NewMerkleDedup(tree)).
		Transform(&transform.GoASTParser{}).
		Transform(transform.NewChunkEmbedder(emb)).
		Sink(smriteaSink)

	if err := registry.Register(f); err != nil {
		fmt.Fprintf(os.Stderr, "register flow: %v\n", err)
		os.Exit(1)
	}

	stats, err := registry.Run(context.Background(), "local-index")
	if err != nil {
		fmt.Fprintf(os.Stderr, "pipeline error: %v\n", err)
		os.Exit(1)
	}

	if _, fErr := fmt.Fprintf(os.Stdout, "Done. in=%d out=%d failed=%d duration=%s\n",
		stats.RecordsIn, stats.RecordsOut, stats.RecordsFailed, stats.Duration); fErr != nil {
		fmt.Fprintf(os.Stderr, "Done. in=%d out=%d failed=%d duration=%s\n",
			stats.RecordsIn, stats.RecordsOut, stats.RecordsFailed, stats.Duration)
	}
}
