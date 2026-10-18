package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestOpsMonitorCheckpointLocksRuleAndRejectsStaleRevision(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM ops_alert_rules WHERE id = $1 FOR UPDATE`)).WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectQuery(`SELECT id, dimensions, status FROM ops_alert_events`).WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"id", "dimensions", "status"}).AddRow(42, []byte(`{"channel_monitor_state":{"revision":3}}`), service.OpsAlertStatusFiring))
	mock.ExpectRollback()
	_, err := repo.SaveChannelMonitorAlertEvent(context.Background(), &service.OpsAlertEvent{ID: 42, RuleID: 1}, 42, 2)
	require.ErrorIs(t, err, service.ErrOpsAlertCheckpointConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsMonitorCheckpointInitialInsertUsesExistingEventsTable(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}
	now := time.Now().UTC()
	event := &service.OpsAlertEvent{RuleID: 1, Severity: "P2", Status: "pending", Title: "Model monitor", FiredAt: now, Dimensions: map[string]any{"model": "gpt-image-demo", "channel_monitor_state": map[string]any{"revision": 1}}}
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id FROM ops_alert_rules .* FOR UPDATE`).WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectQuery(`SELECT id, dimensions, status FROM ops_alert_events`).WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"id", "dimensions", "status"}))
	mock.ExpectQuery(`INSERT INTO ops_alert_events`).WithArgs(event.RuleID, event.Severity, event.Status, event.Title, "", nil, nil, sqlmock.AnyArg(), now, nil, false).WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(50, now))
	mock.ExpectCommit()
	saved, err := repo.SaveChannelMonitorAlertEvent(context.Background(), event, 0, 0)
	require.NoError(t, err)
	require.Equal(t, int64(50), saved.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsMonitorCheckpointUpdatesIncidentAtomically(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}
	now := time.Now().UTC()
	event := &service.OpsAlertEvent{ID: 50, RuleID: 1, Severity: "P2", Status: service.OpsAlertStatusResolved, Title: "Model monitor", FiredAt: now.Add(-time.Hour), ResolvedAt: &now, EmailSent: true, Dimensions: map[string]any{"channel_monitor_state": map[string]any{"revision": 4}}}
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id FROM ops_alert_rules .* FOR UPDATE`).WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectQuery(`SELECT id, dimensions, status FROM ops_alert_events`).WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"id", "dimensions", "status"}).AddRow(50, []byte(`{"channel_monitor_state":{"revision":3}}`), service.OpsAlertStatusFiring))
	mock.ExpectExec(`UPDATE ops_alert_events SET status=\$2`).WithArgs(event.ID, event.Status, event.Title, "", nil, nil, sqlmock.AnyArg(), event.FiredAt, now, true).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	_, err := repo.SaveChannelMonitorAlertEvent(context.Background(), event, 50, 3)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsMonitorPendingCheckpointsAreHiddenFromIncidentLists(t *testing.T) {
	for _, filter := range []*service.OpsAlertEventFilter{nil, {}, {Status: service.OpsAlertStatusFiring}} {
		query, _ := buildOpsAlertEventsWhere(filter)
		require.Contains(t, query, "status <> 'pending'")
	}
}

func TestOpsMonitorCheckpointCannotReopenManuallyResolvedIncident(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id FROM ops_alert_rules .* FOR UPDATE`).WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectQuery(`SELECT id, dimensions, status FROM ops_alert_events`).WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"id", "dimensions", "status"}).AddRow(42, []byte(`{"channel_monitor_state":{"revision":3}}`), service.OpsAlertStatusManualResolved))
	mock.ExpectRollback()
	_, err := repo.SaveChannelMonitorAlertEvent(context.Background(), &service.OpsAlertEvent{ID: 42, RuleID: 1, Status: service.OpsAlertStatusFiring}, 42, 3)
	require.ErrorIs(t, err, service.ErrOpsAlertCheckpointConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}
