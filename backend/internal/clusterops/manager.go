package clusterops

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/clusterha"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Manager struct {
	db         *sql.DB
	rdb        *redis.Client
	cfg        *config.Config
	version    string
	instance   string
	started    time.Time
	control    atomic.Pointer[Control]
	lastOK     atomic.Int64
	requests   atomic.Uint64
	errors     atomic.Uint64
	rejected   atomic.Uint64
	active     atomic.Int64
	background atomic.Int64
	cancel     context.CancelFunc
	done       chan struct{}
	once       sync.Once
	stopOnce   sync.Once
	probe      func() Stats
	notify     func(context.Context, Event) error
	notifyDone chan struct{}
}

func NewManager(db *sql.DB, rdb *redis.Client, cfg *config.Config, version string) *Manager {
	m := &Manager{db: db, rdb: rdb, cfg: cfg, version: version, instance: uuid.NewString(), started: time.Now(), done: make(chan struct{}), notifyDone: make(chan struct{})}
	limit := int64(1000)
	if cfg.Cluster.MaxConcurrency > 0 {
		limit = cfg.Cluster.MaxConcurrency
	}
	m.control.Store(&Control{MaxConcurrency: limit, Weight: 100})
	return m
}
func (m *Manager) Enabled() bool      { return m != nil && m.cfg != nil && m.cfg.Cluster.Enabled }
func (m *Manager) HAEnabled() bool    { return m.Enabled() && m.cfg.Cluster.HAEnabled }
func (m *Manager) InstanceID() string { return m.instance }
func (m *Manager) Control() Control {
	if m == nil {
		return Control{MaxConcurrency: 1000, Weight: 100}
	}
	return *m.control.Load()
}
func (m *Manager) Fresh() bool { return !m.Enabled() || time.Now().Unix()-m.lastOK.Load() < 15 }
func (m *Manager) Begin() bool {
	if !m.Enabled() {
		return true
	}
	c := m.Control()
	if !m.Fresh() || c.Maintenance {
		m.rejected.Add(1)
		return false
	}
	for {
		a := m.active.Load()
		if a+m.background.Load() >= c.MaxConcurrency {
			m.rejected.Add(1)
			return false
		}
		if m.active.CompareAndSwap(a, a+1) {
			m.requests.Add(1)
			return true
		}
	}
}
func (m *Manager) End(status int) {
	if m.Enabled() {
		m.active.Add(-1)
		if status >= 500 {
			m.errors.Add(1)
		}
	}
}
func (m *Manager) TrackTask() func() {
	if !m.Enabled() {
		return func() {}
	}
	m.background.Add(1)
	var once sync.Once
	return func() { once.Do(func() { m.background.Add(-1) }) }
}
func (m *Manager) BackgroundActive() int64 {
	if m == nil {
		return 0
	}
	return m.background.Load()
}

func (m *Manager) Start(probe func() Stats) {
	if !m.Enabled() {
		return
	}
	m.once.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		m.cancel = cancel
		m.probe = probe
		go m.notifyLoop(ctx)
		go func() {
			defer close(m.done)
			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()
			iteration := 0
			for {
				callCtx, stop := context.WithTimeout(ctx, 5*time.Second)
				if err := m.Heartbeat(callCtx); err != nil {
					slog.Warn("cluster heartbeat failed", "node", m.cfg.Cluster.NodeID)
				}
				stop()
				if iteration%5 == 0 && m.cfg.RunsBackgroundJobs() && m.leaseOK() {
					callCtx, stop = context.WithTimeout(ctx, 8*time.Second)
					if err := m.Reconcile(callCtx); err != nil {
						slog.Warn("cluster reconciliation deferred", "node", m.cfg.Cluster.NodeID)
					}
					stop()
				}
				iteration++
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}()
	})
}
func (m *Manager) Stop() {
	if m != nil && m.cancel != nil {
		m.stopOnce.Do(func() {
			m.cancel()
			<-m.done
			<-m.notifyDone
		})
	}
}
func (m *Manager) leaseOK() bool {
	return !m.HAEnabled() || clusterha.CheckProof(m.cfg.Cluster.LeaseFile, m.cfg.Cluster.LeaseToken, m.cfg.Cluster.Role, time.Now()) == nil
}

func (m *Manager) Heartbeat(ctx context.Context) error {
	s := Stats{}
	if m.probe != nil {
		s = m.probe()
	}
	s.Active += m.background.Load()
	s.Requests, s.Errors, s.Rejected = m.requests.Load(), m.errors.Load(), m.rejected.Load()
	s.Goroutines = runtime.NumGoroutine()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	s.HeapBytes = mem.HeapAlloc
	s.LeaseOK = m.leaseOK()
	var readOnly bool
	err := m.db.QueryRowContext(ctx, "SELECT pg_is_in_recovery() OR current_setting('transaction_read_only')::boolean").Scan(&readOnly)
	s.DatabaseOK = err == nil && !readOnly
	if !s.DatabaseOK {
		return ErrUnavailable
	}
	if m.rdb != nil {
		rctx, cancel := context.WithTimeout(ctx, time.Second)
		info, e := m.rdb.Info(rctx, "replication").Result()
		cancel()
		s.RedisOK = e == nil
		if e == nil {
			values := map[string]string{}
			for _, line := range strings.Split(info, "\n") {
				k, v, ok := strings.Cut(strings.TrimSpace(line), ":")
				if ok {
					values[k] = v
				}
			}
			s.RedisRole = values["role"]
			s.RedisReplicas, _ = strconv.Atoi(values["connected_slaves"])
			s.RedisLinkOK = s.RedisRole == "master" || values["master_link_status"] == "up"
		}
	}
	s.DiskTotal, s.DiskFree = diskSpace()
	var statsVisible bool
	replicationErr := m.db.QueryRowContext(ctx, `SELECT count(*), count(*) FILTER (WHERE sync_state IN ('sync','quorum')), COALESCE(MAX(pg_wal_lsn_diff(pg_current_wal_lsn(),replay_lsn)),0)::bigint, pg_has_role(current_user,'pg_read_all_stats','MEMBER') FROM pg_stat_replication`).Scan(&s.ReplicationPeers, &s.SyncReplicas, &s.ReplicationLagBytes, &statsVisible)
	s.ReplicationKnown = replicationErr == nil && statsVisible
	c := m.Control()
	s.ControlRevision = c.Revision
	if m.HAEnabled() {
		var ack struct {
			Revisions map[string]int64 `json:"revisions"`
			UpdatedAt time.Time        `json:"updated_at"`
		}
		if b, e := os.ReadFile(m.cfg.Cluster.LeaseFile + ".proxy-ack"); e == nil && json.Unmarshal(b, &ack) == nil && time.Since(ack.UpdatedAt) < 15*time.Second {
			s.IngressRevision = ack.Revisions[m.cfg.Cluster.NodeID]
		}
	}
	payload, err := json.Marshal(s)
	if err != nil {
		return err
	}
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var oldRole, oldInstance string
	err = tx.QueryRowContext(ctx, `SELECT role,instance_id FROM cluster_nodes WHERE node_id=$1 FOR UPDATE`, m.cfg.Cluster.NodeID).Scan(&oldRole, &oldInstance)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO cluster_nodes(node_id,instance_id,role,version,started_at,snapshot) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(node_id) DO UPDATE SET instance_id=EXCLUDED.instance_id,role=EXCLUDED.role,version=EXCLUDED.version,started_at=EXCLUDED.started_at,snapshot=EXCLUDED.snapshot,heartbeat_at=NOW()`, m.cfg.Cluster.NodeID, m.instance, m.cfg.Cluster.Role, m.version, m.started, payload)
	if err != nil {
		return err
	}
	if oldRole != m.cfg.Cluster.Role || oldInstance != m.instance {
		if _, err = tx.ExecContext(ctx, `INSERT INTO cluster_events(node_id,kind,message) VALUES($1,'node_started',$2)`, m.cfg.Cluster.NodeID, "node started with role "+m.cfg.Cluster.Role); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO cluster_node_controls(node_id,max_concurrency) VALUES($1,$2) ON CONFLICT DO NOTHING`, m.cfg.Cluster.NodeID, c.MaxConcurrency)
	if err != nil {
		return err
	}
	err = tx.QueryRowContext(ctx, `SELECT maintenance,max_concurrency,weight,revision,reason FROM cluster_node_controls WHERE node_id=$1`, m.cfg.Cluster.NodeID).Scan(&c.Maintenance, &c.MaxConcurrency, &c.Weight, &c.Revision, &c.Reason)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	m.control.Store(&c)
	m.lastOK.Store(time.Now().Unix())
	if m.HAEnabled() {
		// Private local IPC survives a role restart, so a drained node cannot
		// immediately reclaim primary before it reads the shared control state.
		v := clusterha.MaintenanceControl{NodeID: m.cfg.Cluster.NodeID, Maintenance: c.Maintenance, Drained: s.Active == 0}
		if err = writePrivateJSON(m.cfg.Cluster.LeaseFile+".control", v); err != nil {
			return err
		}
		nodes, e := m.Nodes(ctx)
		if e != nil {
			return e
		}
		if err = writePrivateJSON(m.cfg.Cluster.LeaseFile+".proxy", map[string]any{"nodes": nodes, "updated_at": time.Now()}); err != nil {
			return err
		}
	}
	return nil
}

func writePrivateJSON(path string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err = os.WriteFile(path+".tmp", b, 0600); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}

func (m *Manager) Nodes(ctx context.Context) ([]Node, error) {
	nodes := []Node{}
	rows, err := m.db.QueryContext(ctx, `SELECT n.node_id,n.instance_id,n.role,n.version,n.heartbeat_at,n.started_at,n.snapshot,c.maintenance,c.max_concurrency,c.weight,c.revision,c.reason FROM cluster_nodes n JOIN cluster_node_controls c USING(node_id) ORDER BY n.node_id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var n Node
		var raw []byte
		if err = rows.Scan(&n.ID, &n.InstanceID, &n.Role, &n.Version, &n.HeartbeatAt, &n.StartedAt, &raw, &n.Control.Maintenance, &n.Control.MaxConcurrency, &n.Control.Weight, &n.Control.Revision, &n.Control.Reason); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &n.Stats); err != nil {
			return nil, err
		}
		n.Online = time.Since(n.HeartbeatAt) < 20*time.Second
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

func (m *Manager) UpdateControl(ctx context.Context, id string, next Control, actor int64) error {
	if !m.Enabled() {
		return ErrUnavailable
	}
	if next.MaxConcurrency < 1 || next.MaxConcurrency > 20000 || next.Weight < 1 || next.Weight > 256 || len(strings.TrimSpace(next.Reason)) < 3 || len(next.Reason) > 500 {
		return fmt.Errorf("capacity must be 1-20000, weight 1-256 and a reason of 3-500 bytes is required")
	}
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(7438147)`); err != nil {
		return err
	}
	if next.Maintenance {
		if !m.HAEnabled() {
			var role string
			if err = tx.QueryRowContext(ctx, `SELECT role FROM cluster_nodes WHERE node_id=$1`, id).Scan(&role); err != nil {
				return err
			}
			if role == "primary" {
				return fmt.Errorf("%w: draining a primary requires the HA supervisor", ErrConflict)
			}
		}
		var available int
		err = tx.QueryRowContext(ctx, `SELECT count(*) FROM cluster_nodes n JOIN cluster_node_controls c USING(node_id) WHERE n.node_id<>$1 AND n.heartbeat_at>NOW()-INTERVAL '20 seconds' AND NOT c.maintenance AND (n.snapshot->>'database_ok')::boolean AND (n.snapshot->>'redis_ok')::boolean AND (n.snapshot->>'lease_ok')::boolean`, id).Scan(&available)
		if err != nil {
			return err
		}
		if available == 0 {
			return ErrLastNode
		}
	}
	r, err := tx.ExecContext(ctx, `UPDATE cluster_node_controls SET maintenance=$2,max_concurrency=$3,weight=$4,reason=$5,revision=revision+1,updated_at=NOW(),updated_by=$6 WHERE node_id=$1 AND revision=$7`, id, next.Maintenance, next.MaxConcurrency, next.Weight, next.Reason, actor, next.Revision)
	if err != nil {
		return err
	}
	count, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrConflict
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO cluster_events(node_id,kind,message,actor_id) VALUES($1,'control_changed',$2,$3)`, id, fmt.Sprintf("maintenance=%t capacity=%d weight=%d: %s", next.Maintenance, next.MaxConcurrency, next.Weight, next.Reason), actor)
	if err != nil {
		return err
	}
	return tx.Commit()
}
