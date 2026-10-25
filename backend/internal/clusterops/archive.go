package clusterops

import (
	"context"
	"database/sql"
	"time"
)

type ArchiveHealth struct {
	SampledAt     time.Time  `json:"sampled_at"`
	Enabled       bool       `json:"enabled"`
	Known         bool       `json:"known"`
	LastArchived  *time.Time `json:"last_archived,omitempty"`
	LastFailed    *time.Time `json:"last_failed,omitempty"`
	Pending       *int64     `json:"pending,omitempty"`
	OldestPending *time.Time `json:"oldest_pending,omitempty"`
}

func (m *Manager) sampleArchive(ctx context.Context) *ArchiveHealth {
	v := &ArchiveHealth{SampledAt: time.Now()}
	var mode string
	var archived, failed, oldest sql.NullTime
	if err := m.db.QueryRowContext(ctx, `SELECT current_setting('archive_mode'),last_archived_time,last_failed_time FROM pg_stat_archiver`).Scan(&mode, &archived, &failed); err != nil {
		return v
	}
	v.Known, v.Enabled = true, mode != "off"
	if archived.Valid {
		v.LastArchived = &archived.Time
	}
	if failed.Valid {
		v.LastFailed = &failed.Time
	}
	// Permission denial is unknown, not a healthy empty queue. Quiet databases
	// are not penalized merely because their last archived WAL is old.
	var pending int64
	if err := m.db.QueryRowContext(ctx, `SELECT count(*),min(modification) FROM pg_ls_archive_statusdir() WHERE name LIKE '%.ready'`).Scan(&pending, &oldest); err == nil {
		v.Pending = &pending
		if oldest.Valid {
			v.OldestPending = &oldest.Time
		}
	}
	return v
}

func (m *Manager) collectOperations(ctx context.Context) {
	defer close(m.observationsDone)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		call, cancel := context.WithTimeout(ctx, 3*time.Second)
		m.archive.Store(m.sampleArchive(call))
		cancel()
		if s := m.latest.Load(); s != nil {
			call, cancel = context.WithTimeout(ctx, time.Second)
			_ = m.recordHistory(call, *s)
			cancel()
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
