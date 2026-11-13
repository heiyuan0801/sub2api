package repository

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestDocsS3ReusesImageStorageWithoutPublishingObject(t *testing.T) {
	var mu sync.Mutex
	var body []byte
	var contentType string
	var headers http.Header
	const path = "/images/docs/item"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPut:
			contentType, headers = r.Header.Get("Content-Type"), r.Header.Clone()
			var err error
			body, err = io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
			}
		case http.MethodGet:
			_, _ = w.Write(body)
		case http.MethodDelete:
			body = nil
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	ctx := context.Background()
	store, err := NewS3ImageStorage(ctx, &config.ImageStorageConfig{
		Bucket: "images", Endpoint: server.URL, Region: "auto", ForcePathStyle: true,
		AccessKeyID: "test", SecretAccessKey: "test-secret", PublicBaseURL: "https://images.example.test",
	})
	require.NoError(t, err)
	require.NoError(t, store.PutPrivate(ctx, "docs/item", []byte("opaque encrypted payload")))
	reader, err := store.OpenPrivate(ctx, "docs/item")
	require.NoError(t, err)
	payload, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Equal(t, "opaque encrypted payload", string(payload))
	require.NoError(t, store.DeletePrivate(ctx, "docs/item"))
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, "application/octet-stream", contentType)
	require.Empty(t, headers.Get("x-amz-acl"))
	require.Nil(t, body)
}
