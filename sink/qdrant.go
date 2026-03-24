package sink

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/qdrant/go-client/qdrant"

	"github.com/SmrutAI/databridge/internal/core"
)

// QdrantSink upserts each Record as a point in a Qdrant collection.
// Config is loaded from environment variables:
//
//	QDRANT_HOST       — default: localhost
//	QDRANT_PORT       — default: 6334
//	QDRANT_COLLECTION — default: codewatch
//	QDRANT_USE_TLS    — default: false
type QdrantSink struct {
	collection   string
	pointsClient qdrant.PointsClient
}

// NewQdrantSink creates a sink connected to Qdrant.
func NewQdrantSink() (*QdrantSink, error) {
	host := os.Getenv("QDRANT_HOST")
	if host == "" {
		host = "localhost"
	}
	portStr := os.Getenv("QDRANT_PORT")
	port := 6334
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("qdrant sink: invalid QDRANT_PORT %q: %w", portStr, err)
		}
		port = p
	}
	collection := os.Getenv("QDRANT_COLLECTION")
	if collection == "" {
		collection = "codewatch"
	}
	useTLS := os.Getenv("QDRANT_USE_TLS") == "true"

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   host,
		Port:   port,
		UseTLS: useTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant sink: create client: %w", err)
	}
	return &QdrantSink{
		collection:   collection,
		pointsClient: client.GetPointsClient(),
	}, nil
}

// Name returns the sink name.
func (s *QdrantSink) Name() string { return "QdrantSink" }

// Open is a no-op for QdrantSink.
func (s *QdrantSink) Open(_ context.Context) error { return nil }

// Write upserts a single Record as a Qdrant point.
// Records with an empty Embedding are skipped — no vector means no point.
// ActionDelete records remove all points matching workspace_id AND file_path.
func (s *QdrantSink) Write(ctx context.Context, r *core.Record) error {
	if r.Action == core.ActionDelete {
		wait := true
		_, err := s.pointsClient.Delete(ctx, &qdrant.DeletePoints{
			CollectionName: s.collection,
			Points: &qdrant.PointsSelector{
				PointsSelectorOneOf: &qdrant.PointsSelector_Filter{
					Filter: &qdrant.Filter{
						Must: []*qdrant.Condition{
							{ConditionOneOf: &qdrant.Condition_Field{Field: &qdrant.FieldCondition{
								Key:   "workspace_id",
								Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: r.SourceID}},
							}}},
							{ConditionOneOf: &qdrant.Condition_Field{Field: &qdrant.FieldCondition{
								Key:   "file_path",
								Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: r.Path}},
							}}},
						},
					},
				},
			},
			Wait: &wait,
		})
		if err != nil {
			return fmt.Errorf("qdrant sink: delete %s: %w", r.Path, err)
		}
		return nil
	}
	if len(r.Embedding) == 0 {
		return nil
	}

	payload := map[string]*qdrant.Value{
		"workspace_id": strVal(r.SourceID),
		"file_path":    strVal(r.Path),
		"symbol":       strVal(r.Symbol),
		"symbol_type":  strVal(r.SymbolType),
		"language":     strVal(r.Language),
		"content_hash": strVal(r.ContentHash),
		"content":      strVal(r.Content),
	}

	wait := true
	_, err := s.pointsClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.collection,
		Points: []*qdrant.PointStruct{
			{
				Id:      qdrant.NewID(r.ID),
				Vectors: qdrant.NewVectorsDense(r.Embedding),
				Payload: payload,
			},
		},
		Wait: &wait,
	})
	if err != nil {
		return fmt.Errorf("qdrant sink: upsert %s#%s: %w", r.Path, r.Symbol, err)
	}
	return nil
}

// Close is a no-op for QdrantSink.
func (s *QdrantSink) Close() error { return nil }

// strVal wraps a string as a *qdrant.Value for use in point payloads.
func strVal(v string) *qdrant.Value {
	return &qdrant.Value{
		Kind: &qdrant.Value_StringValue{StringValue: v},
	}
}
