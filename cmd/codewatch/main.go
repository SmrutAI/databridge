package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/SmrutAI/ingestion-pipeline/internal/flow"
	"github.com/SmrutAI/ingestion-pipeline/source"
)

func main() {
	input := flag.String("input", "", "Path to the directory to index (required)")
	workspace := flag.String("workspace", "", "Workspace ID to associate with indexed files (required)")
	flag.Parse()

	if *input == "" || *workspace == "" {
		fmt.Fprintln(os.Stderr, "Usage: codewatch --input <dir> --workspace <id>")
		os.Exit(1)
	}

	src := source.NewLocalFileSource(*workspace, *input, nil)
	registry := flow.NewFlowRegistry()

	f := flow.NewFlow("local-index").
		Source(src)

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
