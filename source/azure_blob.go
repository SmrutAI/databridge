package source

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	azcontainer "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

	"github.com/SmrutAI/ingestion-pipeline/internal/core"
)

// AzureBlobSource lists blobs from an Azure Blob Storage container under a prefix.
type AzureBlobSource struct {
	workspaceID string
	accountURL  string // https://<account>.blob.core.windows.net
	container   string
	prefix      string
	extensions  map[string]bool
	client      *azblob.Client
}

// NewAzureBlobSource creates an AzureBlob source using a shared key credential.
// accountURL must be "https://<account>.blob.core.windows.net".
// accountName and accountKey are the Azure Storage account credentials.
// If extensions is nil, defaultExtensions are used.
func NewAzureBlobSource(workspaceID, accountURL, accountName, accountKey, container, prefix string, extensions map[string]bool) (*AzureBlobSource, error) {
	if extensions == nil {
		extensions = defaultExtensions
	}
	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, fmt.Errorf("azure blob source: create credential: %w", err)
	}
	client, err := azblob.NewClientWithSharedKeyCredential(accountURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure blob source: create client: %w", err)
	}
	return &AzureBlobSource{
		workspaceID: workspaceID,
		accountURL:  accountURL,
		container:   container,
		prefix:      prefix,
		extensions:  extensions,
		client:      client,
	}, nil
}

// Name returns the source name.
func (s *AzureBlobSource) Name() string { return "AzureBlobSource" }

// Open is a no-op; the client is created in NewAzureBlobSource.
func (s *AzureBlobSource) Open(_ context.Context) error { return nil }

// Records lists all matching blobs and downloads each, emitting one Record per blob.
// The returned channel is closed when pagination completes or ctx is cancelled.
func (s *AzureBlobSource) Records(ctx context.Context) (<-chan *core.Record, error) {
	out := make(chan *core.Record, 100)
	go func() {
		defer close(out)
		containerClient := s.client.ServiceClient().NewContainerClient(s.container)
		pager := containerClient.NewListBlobsFlatPager(&azcontainer.ListBlobsFlatOptions{
			Prefix: &s.prefix,
		})
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return
			}
			for i := range page.Segment.BlobItems {
				item := page.Segment.BlobItems[i]
				if item.Name == nil {
					continue
				}
				name := *item.Name
				dotIdx := strings.LastIndex(name, ".")
				if dotIdx < 0 {
					continue // no extension — skip
				}
				ext := strings.ToLower(name[dotIdx:])
				if !s.extensions[ext] {
					continue
				}
				content, err := s.downloadBlob(ctx, containerClient, name)
				if err != nil {
					continue
				}
				rel := strings.TrimPrefix(name, s.prefix)
				rel = strings.TrimPrefix(rel, "/")
				r := &core.Record{
					ID:       deterministicID(s.workspaceID, rel),
					SourceID: s.workspaceID,
					Path:     rel,
					Language: langFromExt(ext),
					Content:  string(content),
					Action:   core.ActionUpsert,
					Metadata: map[string]any{"blob_name": name, "container": s.container},
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

func (s *AzureBlobSource) downloadBlob(ctx context.Context, containerClient *azcontainer.Client, name string) (_ []byte, err error) {
	blobClient := containerClient.NewBlobClient(name)
	resp, dlErr := blobClient.DownloadStream(ctx, nil)
	if dlErr != nil {
		return nil, fmt.Errorf("azure blob source: download %s: %w", name, dlErr)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("azure blob source: close %s: %w", name, cerr)
		}
	}()
	return io.ReadAll(resp.Body)
}

// Close is a no-op for AzureBlobSource.
func (s *AzureBlobSource) Close() error { return nil }
