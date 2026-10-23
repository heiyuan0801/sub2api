package repository

import (
	"context"
	"errors"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/setting"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// SetContentModerationConfig holds live group rows until the setting is written.
// Deletion takes FOR UPDATE on the group before cleaning this setting, so both
// operations follow the same lock order: groups, then settings.
func (r *settingRepository) SetContentModerationConfig(ctx context.Context, value string, groupIDs []int64) error {
	tx, err := r.client.Tx(ctx)
	if err != nil && !errors.Is(err, ent.ErrTxStarted) {
		return err
	}
	client := r.client
	if err == nil {
		defer func() { _ = tx.Rollback() }()
		client = tx.Client()
	}
	if err := lockLiveGroups(ctx, client, groupIDs); err != nil {
		return err
	}
	now := time.Now()
	if err := client.Setting.Create().SetKey(service.SettingKeyContentModerationConfig).
		SetValue(value).SetUpdatedAt(now).OnConflictColumns(setting.FieldKey).
		UpdateNewValues().Exec(ctx); err != nil {
		return err
	}
	if tx != nil {
		return r.notifyChange(ctx, tx.Commit())
	}
	return r.notifyChange(ctx, nil)
}

type settingRepository struct {
	client       *ent.Client
	clusterRedis *redis.Client
}

func ProvideSettingRepository(client *ent.Client, rdb *redis.Client, cfg *config.Config) service.SettingRepository {
	r := &settingRepository{client: client}
	if cfg != nil && cfg.Cluster.Enabled {
		r.clusterRedis = rdb
	}
	return r
}

// Events carry no setting names or values. Periodic snapshots cover lost events.
func (r *settingRepository) notifyChange(ctx context.Context, writeErr error) error {
	if writeErr == nil && r.clusterRedis != nil {
		pubCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		if err := r.clusterRedis.Publish(pubCtx, "cluster:settings:changed", "changed").Err(); err != nil {
			slog.Warn("cluster settings notification unavailable; polling will reconcile")
		}
	}
	return writeErr
}

func (r *settingRepository) SubscribeSettingChanges(ctx context.Context) (<-chan struct{}, func(), error) {
	if r.clusterRedis == nil {
		return nil, func() {}, nil
	}
	pubsub := r.clusterRedis.Subscribe(ctx, "cluster:settings:changed")
	readyCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	_, err := pubsub.Receive(readyCtx)
	cancel()
	if err != nil {
		_ = pubsub.Close()
		return nil, func() {}, err
	}
	ch := make(chan struct{}, 1)
	go func() {
		defer close(ch)
		messages := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-messages:
				if !ok {
					return
				}
				select {
				case ch <- struct{}{}:
				default:
				}
			}
		}
	}()
	return ch, func() { _ = pubsub.Close() }, nil
}

func NewSettingRepository(client *ent.Client) service.SettingRepository {
	return &settingRepository{client: client}
}

func (r *settingRepository) Get(ctx context.Context, key string) (*service.Setting, error) {
	m, err := r.client.Setting.Query().Where(setting.KeyEQ(key)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, service.ErrSettingNotFound
		}
		return nil, err
	}
	return &service.Setting{
		ID:        m.ID,
		Key:       m.Key,
		Value:     m.Value,
		UpdatedAt: m.UpdatedAt,
	}, nil
}

func (r *settingRepository) GetValue(ctx context.Context, key string) (string, error) {
	setting, err := r.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

func (r *settingRepository) Set(ctx context.Context, key, value string) error {
	now := time.Now()
	err := r.client.Setting.
		Create().
		SetKey(key).
		SetValue(value).
		SetUpdatedAt(now).
		OnConflictColumns(setting.FieldKey).
		UpdateNewValues().
		Exec(ctx)
	return r.notifyChange(ctx, err)
}

func (r *settingRepository) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return map[string]string{}, nil
	}
	settings, err := r.client.Setting.Query().Where(setting.KeyIn(keys...)).All(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	return result, nil
}

func (r *settingRepository) SetMultiple(ctx context.Context, settings map[string]string) error {
	if len(settings) == 0 {
		return nil
	}

	now := time.Now()
	builders := make([]*ent.SettingCreate, 0, len(settings))
	for key, value := range settings {
		builders = append(builders, r.client.Setting.Create().SetKey(key).SetValue(value).SetUpdatedAt(now))
	}
	err := r.client.Setting.
		CreateBulk(builders...).
		OnConflictColumns(setting.FieldKey).
		UpdateNewValues().
		Exec(ctx)
	return r.notifyChange(ctx, err)
}

func (r *settingRepository) GetAll(ctx context.Context) (map[string]string, error) {
	settings, err := r.client.Setting.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	return result, nil
}

func (r *settingRepository) Delete(ctx context.Context, key string) error {
	_, err := r.client.Setting.Delete().Where(setting.KeyEQ(key)).Exec(ctx)
	return r.notifyChange(ctx, err)
}
