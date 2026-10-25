// Package clusterops owns the infrastructure control and observation plane.
// It never replays provider requests or modifies user balances.
package clusterops

import (
	"context"
	"errors"
	"time"
)

var (
	ErrConflict    = errors.New("cluster control changed; refresh before retrying")
	ErrUnavailable = errors.New("cluster operations are unavailable")
	ErrLastNode    = errors.New("cannot drain the last healthy application node")
)

type attemptContextKey struct{}

func WithAttempt(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, attemptContextKey{}, id)
}
func AttemptID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := ctx.Value(attemptContextKey{}).(string)
	return id
}

type Control struct {
	Maintenance    bool   `json:"maintenance"`
	MaxConcurrency int64  `json:"max_concurrency"`
	Weight         int    `json:"weight"`
	Revision       int64  `json:"revision"`
	Reason         string `json:"reason"`
}

type Stats struct {
	Resources           *Resources `json:"resources,omitempty"`
	DatabaseLatencyMS   *float64   `json:"database_latency_ms,omitempty"`
	RedisLatencyMS      *float64   `json:"redis_latency_ms,omitempty"`
	IngressOK           *bool      `json:"ingress_ok,omitempty"`
	DiskTotal           uint64     `json:"disk_total_bytes"`
	DiskFree            uint64     `json:"disk_free_bytes"`
	Active              int64      `json:"active_requests"`
	Requests            uint64     `json:"requests"`
	Errors              uint64     `json:"errors"`
	Rejected            uint64     `json:"rejected"`
	Goroutines          int        `json:"goroutines"`
	HeapBytes           uint64     `json:"heap_bytes"`
	SettingsVersion     string     `json:"settings_version"`
	DatabaseOK          bool       `json:"database_ok"`
	RedisOK             bool       `json:"redis_ok"`
	LeaseOK             bool       `json:"lease_ok"`
	ControlRevision     int64      `json:"control_revision"`
	IngressRevision     int64      `json:"ingress_revision"`
	ReplicationPeers    int        `json:"replication_peers"`
	SyncReplicas        int        `json:"sync_replicas"`
	ReplicationLagBytes int64      `json:"replication_lag_bytes"`
	ReplicationKnown    bool       `json:"replication_known"`
	RedisRole           string     `json:"redis_role"`
	RedisReplicas       int        `json:"redis_replicas"`
	RedisLinkOK         bool       `json:"redis_link_ok"`
}
type Node struct {
	ID          string    `json:"id"`
	InstanceID  string    `json:"instance_id"`
	Role        string    `json:"role"`
	Version     string    `json:"version"`
	HeartbeatAt time.Time `json:"heartbeat_at"`
	StartedAt   time.Time `json:"started_at"`
	Online      bool      `json:"online"`
	Stats       Stats     `json:"stats"`
	Control     Control   `json:"control"`
}
type Event struct {
	ID        int64     `json:"id"`
	NodeID    string    `json:"node_id"`
	Kind      string    `json:"kind"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}
type Alert struct {
	Key      string    `json:"key"`
	NodeID   string    `json:"node_id"`
	Severity string    `json:"severity"`
	Message  string    `json:"message"`
	RaisedAt time.Time `json:"raised_at"`
}
type Request struct {
	ID            string     `json:"id"`
	RequestID     string     `json:"request_id"`
	NodeID        string     `json:"node_id"`
	UserID        int64      `json:"user_id"`
	APIKeyID      int64      `json:"api_key_id"`
	Route         string     `json:"route"`
	State         string     `json:"state"`
	HTTPStatus    int        `json:"http_status"`
	StartedAt     time.Time  `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
	ReviewNote    string     `json:"review_note"`
	ReviewedAt    *time.Time `json:"reviewed_at"`
	EvidenceCount int        `json:"evidence_count"`
	SettledCount  int        `json:"settled_count"`
	UsageCount    int        `json:"usage_count"`
	ExpectedCost  string     `json:"expected_cost"`
	RecordedCost  string     `json:"recorded_cost"`
	BillingState  string     `json:"billing_state"`
}
type Snapshot struct {
	Enabled     bool      `json:"enabled"`
	HAEnabled   bool      `json:"ha_enabled"`
	Nodes       []Node    `json:"nodes"`
	Alerts      []Alert   `json:"alerts"`
	Events      []Event   `json:"events"`
	ReviewCount int64     `json:"review_count"`
	Now         time.Time `json:"now"`
}
