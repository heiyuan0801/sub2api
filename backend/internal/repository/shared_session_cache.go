package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

func sharedSessionKey(key string) string {
	digest := sha256.Sum256([]byte(key))
	return "cluster:session:" + hex.EncodeToString(digest[:])
}

func (c *gatewayCache) GetSessionValues(ctx context.Context, keys []string) ([]string, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	if len(keys) > 256 {
		return nil, fmt.Errorf("too many session keys")
	}
	redisKeys := make([]string, len(keys))
	for i, key := range keys {
		redisKeys[i] = sharedSessionKey(key)
	}
	values, err := c.rdb.MGet(ctx, redisKeys...).Result()
	if err != nil {
		return nil, err
	}
	result := make([]string, len(keys))
	for i, value := range values {
		result[i], _ = value.(string)
	}
	return result, nil
}

func (c *gatewayCache) SetSessionValue(ctx context.Context, key, value string, ttl time.Duration) error {
	if ttl <= 0 || len(value) > 64*1024 {
		return fmt.Errorf("invalid session metadata size or TTL")
	}
	return c.rdb.Set(ctx, sharedSessionKey(key), value, ttl).Err()
}

func (c *gatewayCache) DeleteSessionValue(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, sharedSessionKey(key)).Err()
}

// Save the existing account binding and node origin in one Redis round trip.
func (c *gatewayCache) SetResponseRoute(ctx context.Context, groupID int64, responseID, accountKey string, accountID int64, originKey, instanceID string, ttl time.Duration) error {
	pipe := c.rdb.TxPipeline()
	pipe.Set(ctx, buildSessionKey(groupID, accountKey), accountID, ttl)
	pipe.Set(ctx, sharedSessionKey(originKey), instanceID, ttl)
	_, err := pipe.Exec(ctx)
	return err
}
