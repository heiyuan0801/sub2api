package clusterprobe

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestStorageCrossReadAndScopedCleanup(t *testing.T) {
	var mu sync.Mutex
	objects := map[string][]byte{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(403)
			return
		}
		switch r.Method {
		case "PUT":
			if _, ok := objects[r.URL.Path]; ok && r.Header.Get("If-None-Match") == "*" {
				w.WriteHeader(412)
				return
			}
			objects[r.URL.Path], _ = io.ReadAll(r.Body)
			w.WriteHeader(200)
		case "GET":
			b, ok := objects[r.URL.Path]
			if !ok {
				w.WriteHeader(404)
				return
			}
			_, _ = w.Write(b)
		case "DELETE":
			delete(objects, r.URL.Path)
			w.WriteHeader(204)
		default:
			w.WriteHeader(405)
		}
	}))
	defer server.Close()
	_, err := New(server.URL, "test", "test", "a", "b", false)
	require.Error(t, err)
	client, err := New(server.URL, "test", "test", "a", "b", true)
	require.NoError(t, err)
	ctx := context.Background()
	challenge := uuid.NewString()
	a, err := client.Write(ctx, challenge, "a")
	require.NoError(t, err)
	b, err := client.Write(ctx, challenge, "b")
	require.NoError(t, err)
	_, err = client.Read(ctx, "a", b)
	require.NoError(t, err)
	_, err = client.Read(ctx, "b", a)
	require.NoError(t, err)
	_, err = client.Write(ctx, challenge, "a")
	require.Error(t, err, "never overwrite an existing canary")
	corrupt := a
	corrupt.Digest = "0000000000000000000000000000000000000000000000000000000000000000"
	_, err = client.Read(ctx, "b", corrupt)
	require.Error(t, err)
	_, err = client.Write(ctx, challenge, "../../users")
	require.Error(t, err)
	require.NoError(t, client.Cleanup(ctx, a))
	require.NoError(t, client.Cleanup(ctx, b))
	require.Empty(t, objects)
}
