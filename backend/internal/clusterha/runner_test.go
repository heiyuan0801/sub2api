package clusterha

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestProofRejectsExpiredForeignAndFuture(t *testing.T) {
	p := filepath.Join(t.TempDir(), "proof.json")
	now := time.Now()
	require.NoError(t, WriteProof(p, Proof{Token: "one", Role: "primary", Until: now.Add(time.Second).UnixNano()}))
	require.NoError(t, CheckProof(p, "one", "primary", now))
	require.Error(t, CheckProof(p, "two", "primary", now))
	require.Error(t, CheckProof(p, "one", "worker", now))
	require.Error(t, CheckProof(p, "one", "primary", now.Add(2*time.Second)))
	require.Error(t, CheckProof(p, "one", "primary", now.Add(-time.Hour)))
	require.NoError(t, os.Remove(p))
	require.Error(t, CheckProof(p, "one", "primary", now))
}

func TestRunnerRequiresQuorumAndProtectedTransport(t *testing.T) {
	c := RunnerConfig{NodeID: "a", Cluster: "test", Endpoints: []string{"http://127.0.0.1:1", "http://127.0.0.1:2", "http://127.0.0.1:3"}, Binary: "/bin/false", WorkingDirectory: "/tmp", ProofFile: "/tmp/proof", LocalLab: true}
	require.NoError(t, c.Validate())
	c.Endpoints[2] = c.Endpoints[1]
	require.Error(t, c.Validate())
	c.Endpoints[2] = "http://10.0.0.1:3"
	require.Error(t, c.Validate())
	c.Endpoints[2] = "http://127.0.0.1:3"
	c.LocalLab = false
	require.Error(t, c.Validate())
}

// Opt in against three real etcd members. No mocked lease semantics.
func TestEtcdSingleOwnerAndLeaseRecovery(t *testing.T) {
	endpoints := os.Getenv("SUB2API_TEST_ETCD_ENDPOINTS")
	if endpoints == "" {
		t.Skip("set SUB2API_TEST_ETCD_ENDPOINTS")
	}
	c, err := clientv3.New(clientv3.Config{Endpoints: strings.Split(endpoints, ","), DialTimeout: 2 * time.Second})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, c.Close()) })
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	a, err := c.Grant(ctx, 10)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = c.Revoke(context.Background(), a.ID) })
	b, err := c.Grant(ctx, 10)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = c.Revoke(context.Background(), b.ID) })
	key := "/sub2api/test/" + uuid.NewString()
	role, err := ResolveRole(ctx, c, key, "a", a.ID)
	require.NoError(t, err)
	require.Equal(t, "primary", role)
	role, err = ResolveRole(ctx, c, key, "b", b.ID)
	require.NoError(t, err)
	require.Equal(t, "worker", role)
	// Reusing an owner's name with a different lease cannot hijack it.
	role, err = ResolveRole(ctx, c, key, "a", b.ID)
	require.NoError(t, err)
	require.Equal(t, "worker", role)
	_, err = c.Revoke(ctx, a.ID)
	require.NoError(t, err)
	role, err = ResolveRole(ctx, c, key, "b", b.ID)
	require.NoError(t, err)
	require.Equal(t, "primary", role)
}
