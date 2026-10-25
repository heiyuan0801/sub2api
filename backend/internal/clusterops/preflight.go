package clusterops

import (
	"context"
	"time"
)

type Check struct {
	Code   string `json:"code"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}
type Preflight struct {
	Checks             []Check   `json:"checks"`
	CheckedAt          time.Time `json:"checked_at"`
	ProductionVerified bool      `json:"production_verified"`
}

func (m *Manager) Preflight(ctx context.Context) (Preflight, error) {
	r := Preflight{Checks: []Check{}, CheckedAt: time.Now()}
	add := func(code, status, detail string) { r.Checks = append(r.Checks, Check{code, status, detail}) }
	if !m.Enabled() {
		add("cluster", "disabled", "cluster mode is disabled")
		return r, nil
	}
	nodes, err := m.Nodes(ctx)
	if err != nil {
		return r, err
	}
	healthy := 0
	versions := map[string]bool{}
	for _, n := range nodes {
		if n.Online && n.Stats.DatabaseOK && n.Stats.RedisOK && n.Stats.LeaseOK && n.Stats.RuntimeConfigOK != nil && *n.Stats.RuntimeConfigOK && !n.Control.Maintenance {
			healthy++
		}
		versions[n.Version] = true
	}
	if healthy >= 2 {
		add("nodes", "pass", "at least two healthy application nodes")
	} else {
		add("nodes", "fail", "fewer than two healthy application nodes")
	}
	if len(versions) == 1 {
		add("versions", "pass", "registered node versions match")
	} else {
		add("versions", "warning", "node versions differ")
	}
	if m.HAEnabled() && m.leaseOK() {
		add("quorum", "pass", "current supervisor has confirmed quorum; this does not prove physical independence")
	} else {
		add("quorum", "fail", "HA lease proof is unavailable")
	}
	var dbMaximum int
	known, configured := true, 0
	for _, node := range nodes {
		if !node.Online {
			continue
		}
		if node.Stats.Performance == nil || node.Stats.Performance.DBLimit <= 0 {
			known = false
			continue
		}
		configured += node.Stats.Performance.DBLimit
	}
	if e := m.db.QueryRowContext(ctx, `SELECT current_setting('max_connections')::int`).Scan(&dbMaximum); e != nil || !known || configured == 0 {
		add("connection_budget", "unknown", "database connection budget cannot be confirmed")
	} else if configured > dbMaximum*80/100 {
		add("connection_budget", "warning", "application pools consume over 80 percent of database connection capacity")
	} else {
		add("connection_budget", "pass", "application pools leave at least 20 percent database connection headroom")
	}
	var archiveMode, archiveCommand string
	if err = m.db.QueryRowContext(ctx, `SELECT current_setting('archive_mode'),current_setting('archive_command')`).Scan(&archiveMode, &archiveCommand); err == nil {
		if archiveMode != "off" && archiveCommand != "" {
			add("wal_archive", "pass", "WAL archiving is configured; verify successful delivery and restore separately")
		} else {
			add("wal_archive", "warning", "continuous WAL archiving is not configured")
		}
	} else {
		add("wal_archive", "unknown", "database role cannot inspect archiving")
	}
	archiveState, archiveReason := archiveStatus(m.sampleArchive(ctx), time.Now())
	add("archive_delivery", archiveState, archiveReason)
	contractStatus := "pass"
	for _, n := range nodes {
		if n.Stats.RuntimeConfigOK == nil || !*n.Stats.RuntimeConfigOK {
			contractStatus = "fail"
		}
	}
	add("runtime_contract", contractStatus, "runtime_contract_check")

	add("shared_storage", "manual", "verify configured image/object storage and batch provider files from both nodes; configuration alone does not prove availability")
	add("fencing", "manual", "verify Linux watchdog and physical/network failure fencing on both hosts")
	add("independent_witness", "manual", "verify the third quorum member is in an independent fault domain")
	add("external_ingress", "manual", "verify the public hostname automatically switches origins and never replays generation POSTs")
	add("restore_drill", "manual", "restore a backup into an isolated database and verify schema, ledger and object references")
	return r, nil
}
