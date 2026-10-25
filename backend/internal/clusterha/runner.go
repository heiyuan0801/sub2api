package clusterha

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

type RunnerConfig struct {
	NodeID           string   `json:"node_id"`
	Cluster          string   `json:"cluster"`
	Endpoints        []string `json:"etcd_endpoints"`
	CAFile           string   `json:"ca_file,omitempty"`
	CertFile         string   `json:"cert_file,omitempty"`
	KeyFile          string   `json:"key_file,omitempty"`
	Binary           string   `json:"binary"`
	WorkingDirectory string   `json:"working_directory"`
	ProofFile        string   `json:"proof_file"`
	// LocalLab is an explicit escape hatch for loopback-only fault simulations.
	// Production runs require TLS and systemd's process-group watchdog.
	LocalLab bool `json:"local_lab,omitempty"`
}

func ReadConfig(path string) (RunnerConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return RunnerConfig{}, err
	}
	defer func() { _ = f.Close() }()
	var c RunnerConfig
	d := json.NewDecoder(f)
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return c, err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return c, fmt.Errorf("expected one runner configuration object")
	}
	return c, c.Validate()
}

func (c RunnerConfig) Validate() error {
	valid := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
	if !valid.MatchString(c.NodeID) || !valid.MatchString(c.Cluster) {
		return fmt.Errorf("invalid cluster or node ID")
	}
	if len(c.Endpoints) < 3 {
		return fmt.Errorf("three independent etcd members are required")
	}
	seen := map[string]bool{}
	for _, raw := range c.Endpoints {
		u, err := url.Parse(raw)
		if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || seen[raw] {
			return fmt.Errorf("invalid or duplicate etcd endpoint")
		}
		seen[raw] = true
		ip := net.ParseIP(u.Hostname())
		if c.LocalLab && (ip == nil || !ip.IsLoopback()) {
			return fmt.Errorf("local_lab requires loopback-only etcd endpoints")
		}
		localHTTP := c.LocalLab && u.Scheme == "http" && ip != nil && ip.IsLoopback()
		if u.Scheme != "https" && !localHTTP {
			return fmt.Errorf("etcd requires HTTPS; only local_lab allows loopback HTTP")
		}
	}
	for _, p := range []string{c.Binary, c.WorkingDirectory, c.ProofFile} {
		if !filepath.IsAbs(p) {
			return fmt.Errorf("binary, working_directory and proof_file must be absolute")
		}
	}
	if !c.LocalLab && (c.CAFile == "" || c.CertFile == "" || c.KeyFile == "") {
		return fmt.Errorf("production etcd requires CA and client certificate/key")
	}
	return nil
}

func (c RunnerConfig) Client() (*clientv3.Client, error) {
	cc := clientv3.Config{Endpoints: c.Endpoints, DialTimeout: 2 * time.Second}
	if c.CAFile != "" {
		b, err := os.ReadFile(c.CAFile)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(b) {
			return nil, fmt.Errorf("invalid etcd CA")
		}
		cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
		if err != nil {
			return nil, err
		}
		cc.TLS = &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool, Certificates: []tls.Certificate{cert}}
	}
	return clientv3.New(cc)
}

type child struct {
	cmd         *exec.Cmd
	done        chan struct{}
	role, token string
}

func startChild(c RunnerConfig, role, token string) (*child, error) {
	cmd := exec.Command(c.Binary) //nolint:gosec // Executable comes from the operator-owned runner config, validated as an absolute path; no request input or shell.
	cmd.Dir = c.WorkingDirectory
	for _, v := range os.Environ() {
		key, _, _ := strings.Cut(v, "=")
		switch key {
		case "CLUSTER_ENABLED", "CLUSTER_NODE_ID", "CLUSTER_ROLE", "CLUSTER_HA_ENABLED", "CLUSTER_LEASE_FILE", "CLUSTER_LEASE_TOKEN", "NOTIFY_SOCKET", "WATCHDOG_USEC", "WATCHDOG_PID":
			continue
		}
		cmd.Env = append(cmd.Env, v)
	}
	cmd.Env = append(cmd.Env, "CLUSTER_ENABLED=true", "CLUSTER_NODE_ID="+c.NodeID, "CLUSTER_ROLE="+role, "CLUSTER_HA_ENABLED=true", "CLUSTER_LEASE_FILE="+c.ProofFile, "CLUSTER_LEASE_TOKEN="+token)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	configureChildProcess(cmd)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	p := &child{cmd: cmd, done: make(chan struct{}), role: role, token: token}
	go func() { _ = cmd.Wait(); close(p.done) }()
	slog.Info("HA application started", "node", c.NodeID, "role", role, "pid", cmd.Process.Pid)
	return p, nil
}

func (p *child) stop() {
	if p == nil {
		return
	}
	// Proof has already been removed. Fail-closed demotion must not spend the
	// entire lease on draining. The node also has its independent proof watchdog.
	signalChildProcess(p.cmd, false)
	select {
	case <-p.done:
		return
	case <-time.After(2 * time.Second):
	}
	signalChildProcess(p.cmd, true)
	<-p.done
}

// ResolveRole uses a linearizable transaction. Matching value alone is not
// enough: the key must belong to the current lease (including after reconnect).
func ResolveRole(ctx context.Context, cli *clientv3.Client, key, owner string, lease clientv3.LeaseID) (string, error) {
	r, err := cli.Txn(ctx).If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).Then(clientv3.OpPut(key, owner, clientv3.WithLease(lease))).Else(clientv3.OpGet(key)).Commit()
	if err != nil {
		return "", err
	}
	if r.Succeeded {
		return "primary", nil
	}
	if len(r.Responses) != 1 || r.Responses[0].GetResponseRange() == nil {
		return "", fmt.Errorf("invalid lease response")
	}
	kvs := r.Responses[0].GetResponseRange().Kvs
	if len(kvs) == 1 && string(kvs[0].Value) == owner && kvs[0].Lease == int64(lease) {
		return "primary", nil
	}
	return "worker", nil
}

func Run(ctx context.Context, c RunnerConfig) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if !c.LocalLab && (os.Getenv("NOTIFY_SOCKET") == "" || os.Getenv("WATCHDOG_USEC") != "10000000") {
		return fmt.Errorf("production runner requires systemd Type=notify, WatchdogSec=10s, KillMode=control-group")
	}
	cli, err := c.Client()
	if err != nil {
		return err
	}
	defer func() { _ = cli.Close() }()
	owner := c.NodeID + ":" + uuid.NewString()
	key := "/sub2api/ha/" + c.Cluster + "/primary"
	var session *concurrency.Session
	var proc *child
	stop := func() { _ = os.Remove(c.ProofFile); proc.stop(); proc = nil }
	closeSession := func() {
		if session != nil {
			session.Orphan()
			revokeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_, _ = cli.Revoke(revokeCtx, session.Lease())
			cancel()
			session = nil
		}
	}
	defer func() { stop(); closeSession() }()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	_ = notifySystemd("READY=1")
	for {
		if ctx.Err() != nil {
			return nil
		}
		started := time.Now()
		if session != nil {
			select {
			case <-session.Done():
				stop()
				closeSession()
			default:
			}
		}
		if session == nil {
			// Session uses a bounded client for grant; its keepalive must outlive
			// individual requests. Stop the child before orphaning/revoking it.
			grantCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			grant, grantErr := cli.Grant(grantCtx, 30)
			cancel()
			if grantErr == nil {
				session, err = concurrency.NewSession(cli, concurrency.WithLease(grant.ID), concurrency.WithTTL(30), concurrency.WithContext(ctx))
			} else {
				err = grantErr
			}
		}
		role := ""
		if session != nil {
			probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			role, err = ResolveRole(probeCtx, cli, key, owner, session.Lease())
			cancel()
		}
		if err != nil || role == "" || time.Since(started) > 3*time.Second {
			stop()
			closeSession()
			slog.Warn("HA quorum unavailable; application fenced", "node", c.NodeID)
		} else {
			if proc != nil {
				select {
				case <-proc.done:
					proc = nil
				default:
				}
			}
			if proc != nil && proc.role != role {
				stop()
			}
			token := ""
			if proc != nil {
				token = proc.token
			} else {
				token = uuid.NewString()
			}
			if err = WriteProof(c.ProofFile, Proof{Token: token, Role: role, Until: started.Add(5 * time.Second).UnixNano()}); err != nil {
				stop()
				return fmt.Errorf("write HA proof: %w", err)
			}
			if proc == nil {
				proc, err = startChild(c, role, token)
				if err != nil {
					stop()
					slog.Error("HA application launch failed", "node", c.NodeID)
				}
			}
		}
		_ = notifySystemd("WATCHDOG=1")
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
