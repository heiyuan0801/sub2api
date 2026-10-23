package server

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func nodeTestRouter(n *NodeRuntime) *gin.Engine {
	r := gin.New()
	r.Use(n.Middleware())
	r.GET("/readyz", n.Ready)
	r.GET("/health", func(c *gin.Context) { c.Status(200) })
	return r
}

func TestNodeReadinessDependenciesAndDrain(t *testing.T) {
	var failed atomic.Bool
	n := newNodeRuntime(config.ClusterConfig{Enabled: true, NodeID: "node-a", Role: "primary"},
		func(context.Context) error { return nil },
		func(context.Context) error {
			if failed.Load() {
				return errors.New("private-credential")
			}
			return nil
		})
	r := nodeTestRouter(n)
	check := func(path string, status int) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		require.Equal(t, status, w.Code)
		require.NotContains(t, w.Body.String(), "private-credential")
	}
	check("/readyz?node=node-a&role=primary", 200)
	check("/readyz?node=wrong", 503)
	check("/readyz?role=worker", 503)
	failed.Store(true)
	check("/readyz", 503)
	check("/health", 200)
	failed.Store(false)
	check("/readyz", 200)
	n.BeginDrain()
	check("/readyz", 503)
	check("/health", 200)
	// A single-node instance with an accidentally retained node_id must not
	// pass the cluster ingress check and clear peers' slots on a later restart.
	n.cfg.Enabled = false
	n.draining.Store(false)
	check("/readyz?node=node-a", 503)
}

func TestNodeWorkerRejectsManagementAndLocalTasks(t *testing.T) {
	n := newNodeRuntime(config.ClusterConfig{Enabled: true, Role: "worker"})
	r := nodeTestRouter(n)
	r.NoRoute(func(c *gin.Context) { c.Status(204) })
	for _, path := range []string{"/api/v1/admin/settings", "/v1/images/batches", "/v1/videos", "/home", "/v1/images/generations/async"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("POST", path, nil))
		require.Equal(t, 503, w.Code, path)
	}
	for _, path := range []string{"/v1/responses", "/v1/messages", "/v1/images/generations", "/v1beta/models/gemini:generateContent"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("POST", path, nil))
		require.Equal(t, 204, w.Code, path)
	}
}

func TestNodeDrainWaitsForStream(t *testing.T) {
	n := newNodeRuntime(config.ClusterConfig{Enabled: true, Role: "primary"})
	r := nodeTestRouter(n)
	started, finish := make(chan struct{}), make(chan struct{})
	r.GET("/stream", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		_, _ = c.Writer.WriteString("data: first\n\n")
		c.Writer.Flush()
		close(started)
		<-finish
		_, _ = c.Writer.WriteString("data: done\n\n")
	})
	ts := httptest.NewServer(r)
	defer ts.Close()
	resp, err := ts.Client().Get(ts.URL + "/stream")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- n.Drain(ctx, ts.Config, 100*time.Millisecond) }()
	require.Eventually(t, n.draining.Load, time.Second, time.Millisecond)
	select {
	case <-done:
		t.Fatal("drain returned before stream finished")
	case <-time.After(150 * time.Millisecond):
	}
	close(finish)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "data: done")
	require.NoError(t, <-done)
}

func TestNodeDrainClosesHijackedConnectionsAtDeadline(t *testing.T) {
	n := newNodeRuntime(config.ClusterConfig{Enabled: true, Role: "primary"})
	r := nodeTestRouter(n)
	closed := make(chan struct{})
	r.GET("/ws", func(c *gin.Context) {
		conn, rw, err := c.Writer.Hijack()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
		_ = rw.Flush()
		_, _ = conn.Read(make([]byte, 1))
		close(closed)
	})
	ts := httptest.NewServer(r)
	defer ts.Close()
	conn, err := net.Dial("tcp", strings.TrimPrefix(ts.URL, "http://"))
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()
	_, err = io.WriteString(conn, "GET /ws HTTP/1.1\r\nHost: localhost\r\n\r\n")
	require.NoError(t, err)
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	require.NoError(t, err)
	require.Equal(t, 101, resp.StatusCode)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, n.Drain(ctx, ts.Config, 0), context.DeadlineExceeded)
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("hijacked connection leaked")
	}
}
