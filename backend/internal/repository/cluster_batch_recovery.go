package repository

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

func (r *batchImageRepository) ListRecoverableBatchImageJobs(ctx context.Context, after int64, limit int) ([]*service.BatchImageJob, error) {
	if limit < 1 || limit > 100 {
		limit = 100
	}
	rows, err := r.sql.QueryContext(ctx, batchImageJobSelectSQL+` WHERE id>$1 AND status IN ('submitted','running','indexing','settling') AND COALESCE(provider_job_name,'')<>'' ORDER BY id LIMIT $2`, after, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanBatchImageJobs(rows)
}

var batchImageRepairQueueScript = redis.NewScript(`
if redis.call('ZSCORE',KEYS[2],ARGV[1]) or redis.call('ZSCORE',KEYS[3],ARGV[1]) or redis.call('LPOS',KEYS[1],ARGV[1]) or redis.call('EXISTS',KEYS[5])==1 then return 0 end
redis.call('SET',KEYS[4],ARGV[1],'PX',ARGV[2])
redis.call('LPUSH',KEYS[1],ARGV[1])
return 1
`)

func (q *batchImageQueue) EnsureQueued(ctx context.Context, id string) (bool, error) {
	if !service.IsValidBatchImageID(id) {
		return false, service.ErrInvalidBatchImageQueuePayload
	}
	n, err := batchImageRepairQueueScript.Run(ctx, q.rdb, []string{q.readyKey, q.delayedKey, q.activeKey, q.inflightPrefix + id, q.lockPrefix + id}, id, q.inflightTTL.Milliseconds()).Int()
	return n == 1, err
}
