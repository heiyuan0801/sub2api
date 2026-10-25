package clusterops

import "time"

// Stable reason codes are translated in the console. Optional checks remain
// unknown until observed; an idle archive is distinct from a stuck queue.
func archiveStatus(a *ArchiveHealth, now time.Time) (string, string) {
	if a == nil || !a.Known || now.Sub(a.SampledAt) > 90*time.Second {
		return "unknown", "archive_unknown"
	}
	if !a.Enabled {
		return "warning", "archive_disabled"
	}
	if a.LastFailed != nil && (a.LastArchived == nil || a.LastFailed.After(*a.LastArchived)) {
		return "fail", "archive_failed"
	}
	if a.Pending == nil {
		return "unknown", "archive_queue_unknown"
	}
	if a.OldestPending != nil && now.Sub(*a.OldestPending) > 5*time.Minute {
		return "fail", "archive_backlog"
	}
	if a.LastArchived == nil {
		return "unknown", "archive_unobserved"
	}
	return "pass", "archive_healthy"
}
func operationalAlerts(n Node, ha bool, now time.Time) []Alert {
	out := []Alert{}
	add := func(code, severity string) {
		out = append(out, Alert{Key: n.ID + ":" + code, NodeID: n.ID, Severity: severity, Message: code})
	}
	if n.Stats.RuntimeConfigOK != nil && !*n.Stats.RuntimeConfigOK {
		add("runtime_contract", "critical")
	}
	if !ha {
		return out
	}
	grace := n.StartedAt.IsZero() || now.Sub(n.StartedAt) < 5*time.Minute
	if !grace {
		if status, code := archiveStatus(n.Stats.Archive, now); status != "pass" {
			severity := "warning"
			if status == "fail" {
				severity = "critical"
			}
			add(code, severity)
		}
		if n.Stats.RuntimeConfigOK == nil {
			add("runtime_contract_unknown", "warning")
		}
	}
	v := n.Stats.Verification
	if v == nil || now.Sub(v.CheckedAt) > 5*time.Minute || now.Sub(v.CheckedAt) < -5*time.Second {
		if !grace {
			add("verification_stale", "warning")
		}
		return out
	}
	for _, p := range v.Endpoints {
		if p.Status == "fail" && p.ConsecutiveFailures >= 3 {
			add("ingress_probe_failed", "critical")
			break
		}
	}
	for _, p := range v.Endpoints {
		if p.CertificateExpiresAt != nil && p.CertificateExpiresAt.Sub(now) < 14*24*time.Hour {
			severity := "warning"
			if p.CertificateExpiresAt.Before(now) {
				severity = "critical"
			}
			add("certificate_expiry", severity)
			break
		}
	}
	if v.ArtifactsMatch != nil && !*v.ArtifactsMatch {
		add("artifact_mismatch", "warning")
	}
	if v.RestoreCheckedAt == nil || now.Sub(*v.RestoreCheckedAt) > 30*24*time.Hour {
		if !grace {
			add("restore_overdue", "warning")
		}
	}
	if v.Storage == nil || now.Sub(v.Storage.CheckedAt) > 24*time.Hour {
		if !grace {
			add("storage_unverified", "warning")
		}
	} else if v.Storage.Status != "pass" {
		add("storage_failed", "critical")
	}
	return out
}
