package clusterops

import (
	"encoding/json"
	"time"
)

type IngressTraffic struct {
	SampledAt      time.Time `json:"sampled_at"`
	Queued         uint64    `json:"queued"`
	Active         uint64    `json:"active"`
	BytesIn        uint64    `json:"bytes_in"`
	BytesOut       uint64    `json:"bytes_out"`
	CrossNodeBytes *uint64   `json:"cross_node_bytes"`
}
type IngressProbe struct {
	ConsecutiveFailures  int        `json:"consecutive_failures"`
	ID                   string     `json:"id"`
	Status               string     `json:"status"`
	Reason               string     `json:"reason,omitempty"`
	LatencyMS            *float64   `json:"latency_ms"`
	CertificateExpiresAt *time.Time `json:"certificate_expires_at"`
}
type Verification struct {
	Storage          *StorageVerification `json:"storage,omitempty"`
	CheckedAt        time.Time            `json:"checked_at"`
	Scope            string               `json:"scope"`
	Endpoints        []IngressProbe       `json:"endpoints"`
	ArtifactsMatch   *bool                `json:"artifacts_match"`
	RestoreCheckedAt *time.Time           `json:"restore_checked_at"`
	RestoreScope     string               `json:"restore_scope"`
}

type StorageVerification struct {
	CheckedAt time.Time `json:"checked_at"`
	Status    string    `json:"status"`
	Scope     string    `json:"scope"`
}

func (m *Manager) readVerification() *Verification {
	if !m.HAEnabled() {
		return nil
	}
	b, err := readPrivateFile(m.cfg.Cluster.LeaseFile + ".verification")
	if err != nil || len(b) > 32768 {
		return nil
	}
	var v Verification
	if json.Unmarshal(b, &v) != nil || len(v.Endpoints) < 2 || len(v.Endpoints) > 8 || v.CheckedAt.IsZero() || time.Until(v.CheckedAt) > 5*time.Second || (v.Scope != "local_lab" && v.Scope != "external_probe") {
		return nil
	}
	if v.RestoreCheckedAt != nil && time.Until(*v.RestoreCheckedAt) > 5*time.Second {
		return nil
	}
	if s := v.Storage; s != nil {
		if (s.Status != "pass" && s.Status != "fail") || (s.Scope != "local_lab" && s.Scope != "cross_node_probe") || s.CheckedAt.IsZero() || time.Until(s.CheckedAt) > 5*time.Second {
			return nil
		}
	}
	seen := map[string]bool{}
	for _, e := range v.Endpoints {
		if seen[e.ID] || e.ID == "" || e.ConsecutiveFailures < 0 || e.ConsecutiveFailures > 1000 {
			return nil
		}
		seen[e.ID] = true
		if len(e.ID) > 64 || (e.Status != "pass" && e.Status != "fail") || (e.LatencyMS != nil && finiteMetric(*e.LatencyMS) == nil) {
			return nil
		}
	}
	return &v
}
