package clusterops

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"time"
)

// Operational samples are disposable Redis data, never a billing ledger.
// One snapshot per five-minute bucket, at most 2017 points per node.
type HistoryPoint struct {
	At          time.Time    `json:"at"`
	Performance *Performance `json:"performance,omitempty"`
	CPU         *float64     `json:"cpu,omitempty"`
	Memory      *float64     `json:"memory,omitempty"`
	Active      int64        `json:"active"`
	Queued      *uint64      `json:"queued,omitempty"`
}
type History struct {
	Node   string         `json:"node"`
	Range  string         `json:"range"`
	Points []HistoryPoint `json:"points"`
}

func historyKey(node string) string {
	digest := sha256.Sum256([]byte(node))
	return "cluster:history:v1:" + hex.EncodeToString(digest[:])
}

const historyWrite = `redis.call('ZREMRANGEBYSCORE',KEYS[1],ARGV[1],ARGV[1])
redis.call('ZADD',KEYS[1],ARGV[1],ARGV[2])
redis.call('ZREMRANGEBYSCORE',KEYS[1],'-inf',ARGV[3])
redis.call('EXPIRE',KEYS[1],691200)
return 1`

func (m *Manager) recordHistory(ctx context.Context, s Stats) error {
	if m.rdb == nil {
		return ErrUnavailable
	}
	if s.Performance == nil || time.Since(s.Performance.SampledAt) > 20*time.Second {
		return nil
	}
	p := HistoryPoint{At: s.Performance.SampledAt, Performance: s.Performance, Active: s.Active}
	if r := s.Resources; r != nil && time.Since(r.SampledAt) < 20*time.Second {
		p.CPU, p.Memory = r.CPUPercent, r.MemoryPercent
	}
	if t := s.IngressTraffic; t != nil && time.Since(t.SampledAt) < 20*time.Second {
		p.Queued = &t.Queued
	}
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	bucket := p.At.Unix() / 300 * 300
	return m.rdb.Eval(ctx, historyWrite, []string{historyKey(m.cfg.Cluster.NodeID)}, bucket, string(b), bucket-7*86400).Err()
}

var ErrInvalidHistory = errors.New("invalid history node or range")

func (m *Manager) History(ctx context.Context, node, window string) (History, error) {
	r := History{Node: node, Range: window, Points: []HistoryPoint{}}
	if !m.Enabled() || m.rdb == nil {
		return r, ErrUnavailable
	}
	duration := 24 * time.Hour
	if window == "7d" {
		duration = 7 * 24 * time.Hour
	} else if window != "24h" {
		return r, ErrInvalidHistory
	}
	nodes, err := m.Nodes(ctx)
	if err != nil {
		return r, err
	}
	found := false
	for _, n := range nodes {
		if n.ID == node {
			found = true
			break
		}
	}
	if !found {
		return r, ErrInvalidHistory
	}
	return m.readHistory(ctx, r, duration)
}
func (m *Manager) readHistory(ctx context.Context, r History, duration time.Duration) (History, error) {
	// Redis command uses a fixed bounded range; no user-supplied key or LIMIT.
	values, err := m.rdb.Do(ctx, "ZRANGEBYSCORE", historyKey(r.Node), strconv.FormatInt(time.Now().Add(-duration).Unix(), 10), time.Now().Unix(), "LIMIT", 0, 2017).StringSlice()
	if err != nil {
		return r, ErrUnavailable
	}
	for _, value := range values {
		var p HistoryPoint
		if json.Unmarshal([]byte(value), &p) != nil {
			return r, ErrUnavailable
		}
		r.Points = append(r.Points, p)
	}
	return r, nil
}
