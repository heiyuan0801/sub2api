package clusterops

import (
	"context"
	"log/slog"
	"time"
)

// SetNotifier must be called before Start. Delivery reuses the configured ops
// email policy; events stay durable until acknowledged by the notifier.
func (m *Manager) SetNotifier(f func(context.Context, Event) error) {
	if m != nil {
		m.notify = f
	}
}
func (m *Manager) notifyLoop(ctx context.Context) {
	defer close(m.notifyDone)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if !m.cfg.RunsBackgroundJobs() || !m.leaseOK() || m.notify == nil {
			continue
		}
		call, cancel := context.WithTimeout(ctx, 15*time.Second)
		if err := m.deliverNotifications(call); err != nil {
			slog.Warn("cluster alert delivery deferred")
		}
		cancel()
	}
}
func (m *Manager) deliverNotifications(ctx context.Context) error {
	rows, err := m.db.QueryContext(ctx, `SELECT id,node_id,kind,severity,message,created_at FROM cluster_events WHERE kind IN ('alert_raised','alert_resolved') AND NOT notification_done AND created_at>NOW()-INTERVAL '1 day' ORDER BY id LIMIT 10`)
	if err != nil {
		return err
	}
	events := []Event{}
	for rows.Next() {
		var e Event
		if err = rows.Scan(&e.ID, &e.NodeID, &e.Kind, &e.Severity, &e.Message, &e.CreatedAt); err != nil {
			_ = rows.Close()
			return err
		}
		events = append(events, e)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return err
	}
	for _, e := range events {
		if !m.leaseOK() {
			return ErrUnavailable
		}
		if err = m.notify(ctx, e); err != nil {
			return err
		}
		if _, err = m.db.ExecContext(ctx, `UPDATE cluster_events SET notification_done=true WHERE id=$1`, e.ID); err != nil {
			return err
		}
	}
	return nil
}
