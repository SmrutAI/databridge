package embedder

import (
	"fmt"
	"os"
	"strconv"

	"github.com/SmrutAI/ingestion-pipeline/internal/core"
)

// NewEmbedder creates an Embedder based on the CODEWATCH_EMBEDDER environment variable.
//
// Supported values:
//   - "hugot" — in-process ONNX via hugot; requires CODEWATCH_MODEL_PATH
//   - "api"   — OpenAI-compatible HTTP API (default)
//
// For "api", the following env vars are also read:
//   - CODEWATCH_EMBEDDER_API_URL  (default: "https://api.voyageai.com/v1")
//   - CODEWATCH_EMBEDDER_API_KEY  (optional)
//   - CODEWATCH_EMBEDDER_MODEL    (default: "voyage-code-3")
//   - CODEWATCH_EMBEDDER_DIM      (default: 1024)
func NewEmbedder() (core.Embedder, error) {
	provider := os.Getenv("CODEWATCH_EMBEDDER")
	if provider == "" {
		provider = "api"
	}

	switch provider {
	case "hugot":
		modelPath := os.Getenv("CODEWATCH_MODEL_PATH")
		if modelPath == "" {
			return nil, fmt.Errorf("embedder factory: CODEWATCH_MODEL_PATH is required for hugot embedder")
		}
		return NewHugotEmbedder(modelPath)

	case "api":
		apiURL := os.Getenv("CODEWATCH_EMBEDDER_API_URL")
		if apiURL == "" {
			apiURL = "https://api.voyageai.com/v1"
		}
		apiKey := os.Getenv("CODEWATCH_EMBEDDER_API_KEY")
		model := os.Getenv("CODEWATCH_EMBEDDER_MODEL")
		if model == "" {
			model = "voyage-code-3"
		}
		dim := 1024
		if d := os.Getenv("CODEWATCH_EMBEDDER_DIM"); d != "" {
			parsed, err := strconv.Atoi(d)
			if err != nil {
				return nil, fmt.Errorf("embedder factory: invalid CODEWATCH_EMBEDDER_DIM %q: %w", d, err)
			}
			dim = parsed
		}
		return NewAPIEmbedder(apiURL, apiKey, model, dim), nil

	default:
		return nil, fmt.Errorf("embedder factory: unknown CODEWATCH_EMBEDDER value %q (want: hugot, api)", provider)
	}
}
