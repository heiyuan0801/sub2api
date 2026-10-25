package clusterops

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

func EvaluateAlerts(nodes []Node, ha bool) []Alert {
	alerts := []Alert{}
	primaries := 0
	versions := map[string]bool{}
	settings := map[string]bool{}
	add := func(n Node, kind, severity, message string) {
		alerts = append(alerts, Alert{Key: n.ID + ":" + kind, NodeID: n.ID, Severity: severity, Message: message})
	}
	for _, n := range nodes {
		if !n.Online {
			add(n, "offline", "critical", "node heartbeat expired")
			continue
		}
		alerts = append(alerts, operationalAlerts(n, ha, time.Now())...)
		if n.Role == "primary" {
			primaries++
		}
		versions[n.Version] = true
		if n.Stats.SettingsVersion != "" {
			settings[n.Stats.SettingsVersion] = true
		}
		if !n.Stats.DatabaseOK {
			add(n, "database", "critical", "database writer unavailable")
		}
		if !n.Stats.RedisOK || !n.Stats.RedisLinkOK {
			add(n, "redis", "critical", "Redis unavailable or replication link down")
		}
		if ha && !n.Stats.LeaseOK {
			add(n, "quorum", "critical", "cluster lease proof unavailable")
		}
		if ha && n.Stats.ReplicationKnown && n.Stats.SyncReplicas == 0 {
			add(n, "single_database_copy", "warning", "no synchronous database replica; reduced redundancy")
		}
		if n.Stats.ReplicationLagBytes > 64*1024*1024 {
			add(n, "replication_lag", "warning", "database replication lag exceeds 64 MiB")
		}
		if n.Stats.DiskTotal > 0 && (n.Stats.DiskFree < 10*1024*1024*1024 || float64(n.Stats.DiskFree)/float64(n.Stats.DiskTotal) < 0.15) {
			add(n, "disk_space", "warning", "application filesystem has less than 10 GiB or 15 percent free")
		}
		if n.Stats.Active >= n.Control.MaxConcurrency {
			add(n, "capacity", "warning", "node request capacity reached")
		}
		if ha && (n.Stats.IngressRevision < n.Control.Revision || (n.Stats.IngressOK != nil && !*n.Stats.IngressOK)) {
			add(n, "ingress_pending", "warning", "ingress has not acknowledged the latest weight and capacity")
		}
		if p := n.Stats.Performance; p != nil && time.Since(p.SampledAt) < 20*time.Second && p.DBWaits > 0 && p.DBWaitMeanMS != nil && *p.DBWaitMeanMS > 100 {
			add(n, "db_pool_wait", "warning", "database connection pool wait exceeds 100 ms")
		}
	}
	if primaries != 1 {
		alerts = append(alerts, Alert{Key: "cluster:primary", Severity: "critical", Message: fmt.Sprintf("expected one application primary; observed %d", primaries)})
	}
	if len(versions) > 1 {
		alerts = append(alerts, Alert{Key: "cluster:versions", Severity: "warning", Message: "application versions differ between nodes"})
	}
	if len(settings) > 1 {
		alerts = append(alerts, Alert{Key: "cluster:settings", Severity: "warning", Message: "runtime settings digests differ between nodes"})
	}
	return alerts
}

func (m *Manager) Snapshot(ctx context.Context) (Snapshot, error) {
	s := Snapshot{Enabled: m.Enabled(), HAEnabled: m.HAEnabled(), Nodes: []Node{}, Alerts: []Alert{}, Events: []Event{}, Now: time.Now()}
	if !s.Enabled {
		return s, nil
	}
	var err error
	s.Nodes, err = m.Nodes(ctx)
	if err != nil {
		return s, err
	}
	rows, err := m.db.QueryContext(ctx, `SELECT alert_key,node_id,severity,message,raised_at FROM cluster_alerts WHERE active ORDER BY raised_at DESC LIMIT 100`)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var a Alert
		if err = rows.Scan(&a.Key, &a.NodeID, &a.Severity, &a.Message, &a.RaisedAt); err != nil {
			_ = rows.Close()
			return s, err
		}
		s.Alerts = append(s.Alerts, a)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return s, err
	}
	rows, err = m.db.QueryContext(ctx, `SELECT id,node_id,kind,severity,message,created_at FROM cluster_events ORDER BY created_at DESC,id DESC LIMIT 60`)
	if err != nil {
		return s, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var e Event
		if err = rows.Scan(&e.ID, &e.NodeID, &e.Kind, &e.Severity, &e.Message, &e.CreatedAt); err != nil {
			return s, err
		}
		s.Events = append(s.Events, e)
	}
	if err = rows.Err(); err != nil {
		return s, err
	}
	err = m.db.QueryRowContext(ctx, `SELECT LEAST(count(*),100) FROM cluster_request_journal WHERE state='needs_review' AND reviewed_at IS NULL`).Scan(&s.ReviewCount)
	if err != nil {
		return s, err
	}
	return s, nil
}

func (m *Manager) Reconcile(ctx context.Context) error {
	if !m.Enabled() || !m.leaseOK() {
		return ErrUnavailable
	}
	nodes, err := m.Nodes(ctx)
	if err != nil {
		return err
	}
	alerts := EvaluateAlerts(nodes, m.HAEnabled())
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var locked bool
	if err = tx.QueryRowContext(ctx, `SELECT pg_try_advisory_xact_lock(7438148)`).Scan(&locked); err != nil {
		return err
	}
	if !locked {
		return nil
	}
	_, err = tx.ExecContext(ctx, `UPDATE cluster_request_journal j SET state='needs_review' WHERE state='running' AND started_at<NOW()-INTERVAL '1 minute' AND NOT EXISTS(SELECT 1 FROM cluster_nodes n WHERE n.node_id=j.node_id AND n.instance_id=j.instance_id AND n.heartbeat_at>NOW()-INTERVAL '20 seconds')`)
	if err != nil {
		return err
	}
	// Completed images survive a Redis failover. An interrupted synchronous
	// generation is terminal/uncertain, never silently sent to a provider again.
	_, err = tx.ExecContext(ctx, `WITH interrupted AS (UPDATE cluster_image_tasks t SET status='interrupted',payload=payload || jsonb_build_object('status','failed','http_status',503,'completed_at',EXTRACT(EPOCH FROM NOW())::bigint,'error',jsonb_build_object('type','execution_interrupted','message','Worker stopped before the result was persisted; check billing before resubmitting')),updated_at=NOW() WHERE status='processing' AND updated_at<NOW()-INTERVAL '1 minute' AND NOT EXISTS(SELECT 1 FROM cluster_nodes n WHERE n.node_id=t.node_id AND n.instance_id=t.instance_id AND n.heartbeat_at>NOW()-INTERVAL '20 seconds') RETURNING payload->>'_cluster_attempt_id' AS attempt) UPDATE cluster_request_journal SET state='needs_review' WHERE attempt_id::text IN (SELECT attempt FROM interrupted)`)
	if err != nil {
		return err
	}
	// A 202 response ends before asynchronous billing; reconcile its terminal
	// task through the durable attempt association without scanning text history.
	_, err = tx.ExecContext(ctx, `UPDATE cluster_request_journal j SET state='needs_review' FROM cluster_image_tasks t WHERE t.status<>'processing' AND t.payload->>'_cluster_attempt_id'=j.attempt_id::text AND j.state='finished' AND (`+billingMismatchSQL+`)`)
	if err != nil {
		return err
	}
	active := map[string]bool{}
	rows, err := tx.QueryContext(ctx, `SELECT alert_key FROM cluster_alerts WHERE active`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var k string
		if err = rows.Scan(&k); err != nil {
			_ = rows.Close()
			return err
		}
		active[k] = true
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return err
	}
	changed := []Alert{}
	for _, a := range alerts {
		if !active[a.Key] {
			_, err = tx.ExecContext(ctx, `INSERT INTO cluster_events(node_id,kind,severity,message) VALUES($1,'alert_raised',$2,$3)`, a.NodeID, a.Severity, a.Message)
			if err != nil {
				return err
			}
			changed = append(changed, a)
		}
		delete(active, a.Key)
		_, err = tx.ExecContext(ctx, `INSERT INTO cluster_alerts(alert_key,node_id,severity,message) VALUES($1,$2,$3,$4) ON CONFLICT(alert_key) DO UPDATE SET active=true,severity=EXCLUDED.severity,message=EXCLUDED.message,raised_at=CASE WHEN cluster_alerts.active THEN cluster_alerts.raised_at ELSE NOW() END,updated_at=NOW()`, a.Key, a.NodeID, a.Severity, a.Message)
		if err != nil {
			return err
		}
	}
	for key := range active {
		_, err = tx.ExecContext(ctx, `UPDATE cluster_alerts SET active=false,updated_at=NOW() WHERE alert_key=$1`, key)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO cluster_events(node_id,kind,severity,message) SELECT node_id,'alert_resolved',severity,alert_key FROM cluster_alerts WHERE alert_key=$1`, key)
		if err != nil {
			return err
		}
	}
	// Bound routine history while preserving every unresolved billing case.
	for _, q := range []string{
		`DELETE FROM cluster_request_journal j WHERE attempt_id IN (SELECT j.attempt_id FROM cluster_request_journal j WHERE j.started_at<NOW()-INTERVAL '30 days' AND (j.reviewed_at IS NOT NULL OR (j.state='finished' AND NOT EXISTS(SELECT 1 FROM cluster_billing_evidence b LEFT JOIN usage_logs u ON u.request_id=b.request_id AND u.api_key_id=b.api_key_id WHERE (b.attempt_id=j.attempt_id OR (b.request_id=j.request_id AND b.api_key_id=j.api_key_id)) AND (u.id IS NULL OR ABS(u.actual_cost-b.expected_cost)>0.00000001 OR NOT EXISTS(SELECT 1 FROM usage_billing_dedup d WHERE d.request_id=b.request_id AND d.api_key_id=b.api_key_id) AND NOT EXISTS(SELECT 1 FROM usage_billing_dedup_archive d WHERE d.request_id=b.request_id AND d.api_key_id=b.api_key_id))))) LIMIT 1000)`,
		`DELETE FROM cluster_billing_evidence b WHERE (request_id,api_key_id) IN (SELECT b.request_id,b.api_key_id FROM cluster_billing_evidence b WHERE b.created_at<NOW()-INTERVAL '90 days' AND NOT EXISTS(SELECT 1 FROM cluster_request_journal j WHERE j.attempt_id=b.attempt_id OR (j.request_id=b.request_id AND j.api_key_id=b.api_key_id)) AND EXISTS(SELECT 1 FROM usage_logs u WHERE u.request_id=b.request_id AND u.api_key_id=b.api_key_id AND ABS(u.actual_cost-b.expected_cost)<=0.00000001) LIMIT 1000)`,
		`DELETE FROM cluster_events WHERE id IN(SELECT id FROM cluster_events WHERE created_at<NOW()-INTERVAL '90 days' LIMIT 1000)`,
		`DELETE FROM cluster_alerts WHERE NOT active AND updated_at<NOW()-INTERVAL '90 days'`,
		`DELETE FROM cluster_image_tasks WHERE task_id IN(SELECT task_id FROM cluster_image_tasks WHERE expires_at<NOW() LIMIT 1000)`,
	} {
		if _, err = tx.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	for _, a := range changed {
		slog.Warn("cluster alert", "code", a.Key, "severity", a.Severity, "message", a.Message)
	}
	return nil
}
