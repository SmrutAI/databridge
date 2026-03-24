package embedder

import (
	"context"
	"fmt"

	"github.com/knights-analytics/hugot"
	"github.com/knights-analytics/hugot/backends"
	"github.com/knights-analytics/hugot/pipelines"
)

// HugotEmbedder runs all-MiniLM-L6-v2 in-process via hugot (ONNX runtime).
// It produces 384-dimensional embeddings.
type HugotEmbedder struct {
	session  *hugot.Session
	pipeline *pipelines.FeatureExtractionPipeline
	dim      int
}

// NewHugotEmbedder creates an embedder using the ONNX model at the given path.
// modelPath should point to the directory containing model.onnx and tokenizer files.
// Typical value: a local cache of https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2
func NewHugotEmbedder(modelPath string) (*HugotEmbedder, error) {
	session, err := hugot.NewORTSession()
	if err != nil {
		return nil, fmt.Errorf("hugot: create session: %w", err)
	}

	pipe, err := hugot.NewPipeline(session, backends.PipelineConfig[*pipelines.FeatureExtractionPipeline]{
		ModelPath:    modelPath,
		Name:         "all-MiniLM-L6-v2",
		OnnxFilename: "model.onnx",
	})
	if err != nil {
		_ = session.Destroy()
		return nil, fmt.Errorf("hugot: create pipeline: %w", err)
	}

	return &HugotEmbedder{
		session:  session,
		pipeline: pipe,
		dim:      384,
	}, nil
}

// Embed returns the embedding for a single text.
func (e *HugotEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	vecs, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

// EmbedBatch returns embeddings for multiple texts in a single ONNX inference call.
func (e *HugotEmbedder) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	result, err := e.pipeline.RunPipeline(texts)
	if err != nil {
		return nil, fmt.Errorf("hugot: run pipeline: %w", err)
	}
	return result.Embeddings, nil
}

// Dimension returns 384 (all-MiniLM-L6-v2 output size).
func (e *HugotEmbedder) Dimension() int { return e.dim }

// Close destroys the underlying hugot session and releases ONNX resources.
func (e *HugotEmbedder) Close() error {
	return e.session.Destroy()
}
