package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/clusterops"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

type durableImageTaskStore struct {
	db             *sql.DB
	node, instance string
	legacy         service.ImageTaskStore
}

func ProvideImageTaskStore(rdb *redis.Client, db *sql.DB, cfg *config.Config, m *clusterops.Manager) service.ImageTaskStore {
	legacy := NewImageTaskStore(rdb)
	if !cfg.Cluster.HAEnabled {
		return legacy
	}
	return &durableImageTaskStore{db: db, node: cfg.Cluster.NodeID, instance: m.InstanceID(), legacy: legacy}
}
func (s *durableImageTaskStore) Save(ctx context.Context, t *service.ImageTaskRecord, _ time.Duration) error {
	raw, err := json.Marshal(t)
	if err != nil {
		return err
	}
	r, err := s.db.ExecContext(ctx, `INSERT INTO cluster_image_tasks(task_id,node_id,instance_id,status,payload,expires_at) VALUES($1,$2,$3,$4,$5::jsonb || jsonb_build_object('_cluster_attempt_id',$7::text),$6) ON CONFLICT(task_id) DO UPDATE SET status=EXCLUDED.status,payload=EXCLUDED.payload || jsonb_build_object('_cluster_attempt_id',cluster_image_tasks.payload->>'_cluster_attempt_id'),expires_at=EXCLUDED.expires_at,updated_at=NOW() WHERE cluster_image_tasks.instance_id=EXCLUDED.instance_id AND cluster_image_tasks.status='processing'`, t.ID, s.node, s.instance, t.Status, raw, time.Unix(t.ExpiresAt, 0), clusterops.AttemptID(ctx))
	if err != nil {
		return err
	}
	count, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return service.ErrImageTaskUnavailable
	}
	return nil
}
func (s *durableImageTaskStore) Get(ctx context.Context, id string) (*service.ImageTaskRecord, error) {
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT payload FROM cluster_image_tasks WHERE task_id=$1 AND expires_at>NOW()`, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return s.legacy.Get(ctx, id)
	}
	if err != nil {
		return nil, err
	}
	var t service.ImageTaskRecord
	if err = json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}
	return &t, nil
}
