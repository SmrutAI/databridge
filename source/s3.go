package source

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/SmrutAI/ingestion-pipeline/internal/core"
)

// S3Source lists objects from an S3 bucket under a prefix and downloads each file.
type S3Source struct {
	workspaceID string
	bucket      string
	prefix      string
	extensions  map[string]bool
	client      *s3.Client
}

// NewS3Source creates an S3 source.
// If extensions is nil, defaultExtensions are used.
// AWS client initialisation is deferred to Open so the constructor performs no I/O.
func NewS3Source(workspaceID, bucket, prefix string, extensions map[string]bool) *S3Source {
	if extensions == nil {
		extensions = defaultExtensions
	}
	return &S3Source{
		workspaceID: workspaceID,
		bucket:      bucket,
		prefix:      prefix,
		extensions:  extensions,
	}
}

// Name returns the source name.
func (s *S3Source) Name() string { return "S3Source" }

// Open initialises the AWS S3 client using the default credential chain (env, ~/.aws, IMDS).
func (s *S3Source) Open(ctx context.Context) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("s3 source: load aws config: %w", err)
	}
	s.client = s3.NewFromConfig(cfg)
	return nil
}

// Records lists all matching objects and downloads each, emitting one Record per file.
// The returned channel is closed when pagination completes or ctx is cancelled.
func (s *S3Source) Records(ctx context.Context) (<-chan *core.Record, error) {
	out := make(chan *core.Record, 100)
	go func() {
		defer close(out)
		paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
			Bucket: aws.String(s.bucket),
			Prefix: aws.String(s.prefix),
		})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return
			}
			for i := range page.Contents {
				key := aws.ToString(page.Contents[i].Key)
				dotIdx := strings.LastIndex(key, ".")
				if dotIdx < 0 {
					continue // no extension — skip
				}
				ext := strings.ToLower(key[dotIdx:])
				if !s.extensions[ext] {
					continue
				}
				content, dlErr := s.downloadObject(ctx, key)
				if dlErr != nil {
					continue // skip objects we can't read
				}
				rel := strings.TrimPrefix(key, s.prefix)
				rel = strings.TrimPrefix(rel, "/")
				r := &core.Record{
					ID:       deterministicID(s.workspaceID, rel),
					SourceID: s.workspaceID,
					Path:     rel,
					Language: langFromExt(ext),
					Content:  string(content),
					Action:   core.ActionUpsert,
					Metadata: map[string]any{"s3_key": key, "bucket": s.bucket},
				}
				select {
				case out <- r:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

func (s *S3Source) downloadObject(ctx context.Context, key string) (_ []byte, err error) {
	result, dlErr := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if dlErr != nil {
		return nil, fmt.Errorf("s3 source: get object %s: %w", key, dlErr)
	}
	defer func() {
		if cerr := result.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("s3 source: close %s: %w", key, cerr)
		}
	}()
	return io.ReadAll(result.Body)
}

// Close is a no-op for S3Source.
func (s *S3Source) Close() error { return nil }
