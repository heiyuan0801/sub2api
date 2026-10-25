package clusterlb

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func exampleConfig() Config {
	return Config{Listen: "127.0.0.1:8088", Nodes: []Node{
		{ID: "primary-a", Address: "10.70.0.1:8080", Role: "primary", Weight: 100},
		{ID: "worker-b", Address: "10.70.0.2:8080", Role: "worker", Weight: 100},
	}}
}

func TestClusterInventoryValidation(t *testing.T) {
	require.NoError(t, exampleConfig().Validate())
	for _, mutate := range []func(*Config){
		func(c *Config) { c.Listen = "0.0.0.0:8088" },
		func(c *Config) { c.Nodes[1].Address = "8.8.8.8:8080" },
		func(c *Config) { c.Nodes[1].Address = c.Nodes[0].Address },
		func(c *Config) { c.Nodes[1].ID = c.Nodes[0].ID },
		func(c *Config) { c.Nodes[1].ID = "bad\nbackend injected" },
		func(c *Config) { c.Nodes[1].Weight = 0 },
		func(c *Config) { c.Nodes[1].Role = "primary" },
		func(c *Config) { c.Nodes[0].Disabled = true },
	} {
		cfg := exampleConfig()
		mutate(&cfg)
		require.Error(t, cfg.Validate())
	}
	_, err := Read(strings.NewReader(`{"listen":"127.0.0.1:8088","typo":true}`))
	require.Error(t, err)
}

func TestClusterRouteClassification(t *testing.T) {
	for _, path := range []string{"/v1/responses", "/responses/compact", "/backend-api/codex/responses", "/v1/chat/completions", "/v1/messages", "/v1/images/edits", "/v1beta/models/gemini:streamGenerateContent", "/antigravity/v1/messages"} {
		require.True(t, WorkerPath(path), path)
	}
	for _, path := range []string{"/api/v1/admin/settings", "/api/v1/payment/notify", "/v1/images/batches/123/content", "/v1/images/tasks/123", "/v1/images/edits/async", "/v1/videos", "/v1/live", "/backend-api/codex/realtime/calls", "/responses-evil", "/unknown", "/uploads/pic.png"} {
		require.False(t, WorkerPath(path), path)
	}
}

func TestClusterHAElectsManagementByReadiness(t *testing.T) {
	c := exampleConfig()
	c.HA = true
	for i := range c.Nodes {
		c.Nodes[i].Role = "candidate"
	}
	r, err := c.Render()
	require.NoError(t, err)
	primary := strings.Split(strings.Split(r, "\nbackend primary\n")[1], "\nbackend gateway\n")[0]
	require.Contains(t, primary, "&role=primary")
	require.Contains(t, primary, "server primary-a")
	require.Contains(t, primary, "server worker-b")
	require.Contains(t, r, "nbsrv(primary) gt 0")
	require.Contains(t, r, "nbsrv(gateway) gt 0")
	require.Contains(t, r, "nbsrv(responses) gt 0")
	c.Nodes[0].Role = "primary"
	require.Error(t, c.Validate())
}

// Opt-in executable test: run with HAPROXY_BIN=/path/to/haproxy (3.2+).
// Real TCP requests pass through the generated proxy to two controlled nodes.
// No upstream model or payment provider is called.
func TestClusterHAProxyEndToEnd(t *testing.T) {
	binary := os.Getenv("HAPROXY_BIN")
	if binary == "" {
		t.Skip("set HAPROXY_BIN to exercise real HAProxy")
	}
	var workerDown atomic.Bool
	var primaryDown atomic.Bool
	var inferenceDown atomic.Bool
	var failedRequests atomic.Int32
	hold, started := make(chan struct{}), make(chan string, 2)
	var releaseOnce sync.Once
	releaseHold := func() { releaseOnce.Do(func() { close(hold) }) }
	makeNode := func(id, role string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/readyz" {
				if inferenceDown.Load() && r.URL.Query().Get("role") == "" {
					w.WriteHeader(503)
					return
				}
				if (role == "worker" && workerDown.Load()) || (role == "primary" && primaryDown.Load()) || r.URL.Query().Get("node") != id || (r.URL.Query().Get("role") != "" && r.URL.Query().Get("role") != role) {
					w.WriteHeader(503)
					return
				}
				w.WriteHeader(200)
				_, _ = io.WriteString(w, `{"status":"ready"}`)
				return
			}
			w.Header().Set("X-Test-Node", id)
			if r.URL.Query().Get("hold") == "1" {
				started <- id
				<-hold
			}
			if r.URL.Query().Get("fail") == "1" {
				failedRequests.Add(1)
				w.WriteHeader(503)
				return
			}
			if r.URL.Query().Get("sse") == "1" {
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = fmt.Fprint(w, "data: first\n\n")
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
				<-hold
				_, _ = fmt.Fprint(w, "data: done\n\n")
				return
			}
			if r.Header.Get("Upgrade") == "websocket" {
				hijacker, ok := w.(http.Hijacker)
				if !ok {
					return
				}
				conn, rw, err := hijacker.Hijack()
				if err != nil {
					return
				}
				defer func() { _ = conn.Close() }()
				_, _ = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: websocket\r\nSec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=\r\n\r\n")
				_ = rw.Flush()
				line, _ := rw.ReadString('\n')
				_, _ = rw.WriteString(line)
				_ = rw.Flush()
				return
			}
			_, _ = io.Copy(io.Discard, r.Body)
			_, _ = fmt.Fprint(w, id)
		}))
	}
	a, b := makeNode("primary-a", "primary"), makeNode("worker-b", "worker")
	defer a.Close()
	defer b.Close()
	defer releaseHold()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	require.NoError(t, listener.Close())
	cfg := exampleConfig()
	cfg.Listen = addr
	cfg.Nodes[0].Address = strings.TrimPrefix(a.URL, "http://")
	cfg.Nodes[1].Address = strings.TrimPrefix(b.URL, "http://")
	rendered, err := cfg.Render()
	require.NoError(t, err)
	statsPath := fmt.Sprintf("/tmp/sub2api-lb-%d-%d.sock", os.Getpid(), time.Now().UnixNano())
	t.Cleanup(func() { _ = os.Remove(statsPath) })
	rendered = strings.Replace(rendered, "global\n", "global\n    stats socket "+statsPath+" mode 600\n", 1)
	file := filepath.Join(t.TempDir(), "haproxy.cfg")
	require.NoError(t, os.WriteFile(file, []byte(rendered), 0600))
	out, err := exec.Command(binary, "-c", "-f", file).CombinedOutput() //nolint:gosec // HAPROXY_BIN is a trusted test-runner setting; no shell or application request input.
	require.NoError(t, err, string(out))
	logFile, err := os.CreateTemp(t.TempDir(), "proxy-log")
	require.NoError(t, err)
	defer func() { _ = logFile.Close() }()
	cmd := exec.Command(binary, "-db", "-f", file) //nolint:gosec // HAPROXY_BIN is a trusted test-runner setting; no shell or application request input.
	cmd.Stdout, cmd.Stderr = logFile, logFile
	require.NoError(t, cmd.Start())
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	client := &http.Client{Timeout: 15 * time.Second}
	base := "http://" + addr
	request := func(path, key string) (int, string, error) {
		req, _ := http.NewRequest("POST", base+path, bytes.NewBufferString(`{"model":"test"}`))
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := client.Do(req)
		if err != nil {
			return 0, "", err
		}
		defer func() { _ = resp.Body.Close() }()
		_, _ = io.Copy(io.Discard, resp.Body)
		return resp.StatusCode, resp.Header.Get("X-Test-Node"), nil
	}
	require.Eventually(t, func() bool { status, _, err := request("/ingress-ready", "test"); return err == nil && status == 200 }, 10*time.Second, 50*time.Millisecond)
	// Management alone must not mark an origin ready for paid inference.
	inferenceDown.Store(true)
	require.Eventually(t, func() bool { status, _, err := request("/ingress-ready", "test"); return err == nil && status == 503 }, 10*time.Second, 50*time.Millisecond)
	status, _, err := request("/home", "test")
	require.NoError(t, err)
	require.Equal(t, 200, status)
	inferenceDown.Store(false)
	require.Eventually(t, func() bool { status, _, err := request("/ingress-ready", "test"); return err == nil && status == 200 }, 10*time.Second, 50*time.Millisecond)
	// Each backend has independent rise checks. One healthy server is enough
	// for ingress; distribution assertions require both servers in every pool.
	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("unix", statsPath, time.Second)
		if err != nil {
			return false
		}
		defer func() { _ = conn.Close() }()
		_ = conn.SetDeadline(time.Now().Add(time.Second))
		_, _ = io.WriteString(conn, "show stat\n")
		reader := csv.NewReader(conn)
		reader.FieldsPerRecord = -1
		rows, err := reader.ReadAll()
		if err != nil {
			return false
		}
		up := 0
		for _, row := range rows {
			if len(row) > 17 && (row[0] == "gateway" || row[0] == "responses") && (row[1] == "primary-a" || row[1] == "worker-b") && row[17] == "UP" {
				up++
			}
		}
		return up == 4
	}, 10*time.Second, 50*time.Millisecond)
	for _, path := range []string{"/home", "/api/v1/admin/settings", "/api/v1/payment/notify", "/v1/images/batches", "/v1/images/generations/async", "/v1/videos"} {
		status, node, err := request(path, "test")
		require.NoError(t, err)
		require.Equal(t, 200, status)
		require.Equal(t, "primary-a", node, path)
	}
	seen := map[string]bool{}
	for i := 0; i < 40; i++ {
		key := fmt.Sprintf("key-%d", i)
		_, expected, err := request("/v1/responses", key)
		require.NoError(t, err)
		require.NotEmpty(t, expected)
		seen[expected] = true
		for _, headers := range []http.Header{
			{"X-Api-Key": {key}},
			{"X-Goog-Api-Key": {key}},
			{"Authorization": {"bearer  " + key + " "}},
		} {
			req, _ := http.NewRequest("POST", base+"/responses", nil)
			req.Header = headers
			resp, err := client.Do(req)
			require.NoError(t, err)
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			require.Equal(t, expected, resp.Header.Get("X-Test-Node"), headers)
		}
		for _, path := range []string{"/responses", "/backend-api/codex/responses"} {
			_, node, err := request(path, key)
			require.NoError(t, err)
			require.Equal(t, expected, node)
		}
	}
	require.Len(t, seen, 2, "API keys should distribute across both nodes")
	// One busy API key can use both nodes without moving an individual session.
	sessionNodes := map[string]bool{}
	for i := 0; i < 40; i++ {
		var previous string
		for j := 0; j < 2; j++ {
			req, _ := http.NewRequest("POST", base+"/v1/responses", nil)
			req.Header.Set("Authorization", "Bearer one-large-customer")
			req.Header.Set("X-Sub2API-Session-ID", fmt.Sprintf("conversation-%d", i))
			resp, err := client.Do(req)
			require.NoError(t, err)
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			node := resp.Header.Get("X-Test-Node")
			require.NotEmpty(t, node)
			if j > 0 {
				require.Equal(t, previous, node)
			}
			previous = node
			sessionNodes[node] = true
		}
	}
	require.Len(t, sessionNodes, 2)
	// With one active request, leastconn chooses the other node.
	done := make(chan error, 2)
	go func() { _, _, err := request("/v1/chat/completions?hold=1", "same-key"); done <- err }()
	first := <-started
	go func() { _, _, err := request("/v1/chat/completions?hold=1", "same-key"); done <- err }()
	second := <-started
	require.NotEqual(t, first, second)
	// The first SSE event must arrive before the handler is allowed to finish.
	resp, err := client.Get(base + "/v1/chat/completions?sse=1")
	require.NoError(t, err)
	line, err := bufio.NewReader(resp.Body).ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "data: first\n", line)
	releaseHold()
	_ = resp.Body.Close()
	require.NoError(t, <-done)
	require.NoError(t, <-done)
	status, _, err = request("/v1/chat/completions?fail=1", "test")
	require.NoError(t, err)
	require.Equal(t, 503, status)
	require.EqualValues(t, 1, failedRequests.Load(), "must not replay billed requests")
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	_, err = io.WriteString(conn, "GET /v1/responses HTTP/1.1\r\nHost: localhost\r\nAuthorization: Bearer ws-test\r\nConnection: Upgrade\r\nUpgrade: websocket\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n")
	require.NoError(t, err)
	rw := bufio.NewReader(conn)
	upgrade, err := http.ReadResponse(rw, nil)
	require.NoError(t, err)
	require.Equal(t, 101, upgrade.StatusCode)
	_, err = io.WriteString(conn, "ping\n")
	require.NoError(t, err)
	line, err = rw.ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "ping\n", line)
	workerDown.Store(true)
	require.Eventually(t, func() bool {
		for i := 0; i < 10; i++ {
			status, node, err := request("/v1/responses", fmt.Sprintf("key-%d", i))
			if err != nil || status != 200 || node != "primary-a" {
				return false
			}
		}
		return true
	}, 10*time.Second, 100*time.Millisecond)
	workerDown.Store(false)
	require.Eventually(t, func() bool {
		for i := 0; i < 20; i++ {
			_, node, _ := request("/v1/responses", fmt.Sprintf("key-%d", i))
			if node == "worker-b" {
				return true
			}
		}
		return false
	}, 10*time.Second, 100*time.Millisecond)
	primaryDown.Store(true)
	require.Eventually(t, func() bool {
		status, _, _ := request("/api/v1/payment/notify", "test")
		return status == 503
	}, 10*time.Second, 100*time.Millisecond)
	require.Eventually(t, func() bool {
		status, node, err := request("/v1/chat/completions", "test")
		return err == nil && status == 200 && node == "worker-b"
	}, 10*time.Second, 100*time.Millisecond)
}
