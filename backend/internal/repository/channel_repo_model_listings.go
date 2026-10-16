package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// All channel/model-pricing writers take the channel lock before pricing locks.
func lockChannelForModelPricingTx(ctx context.Context, tx *sql.Tx, channelID int64) error {
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM channels WHERE id = $1 FOR UPDATE`, channelID).Scan(&id)
	if err == sql.ErrNoRows {
		return service.ErrChannelNotFound
	}
	if err != nil {
		return fmt.Errorf("lock channel for model pricing: %w", err)
	}
	return nil
}

// recordChannelModelListingsTx uses the saved configuration so partial updates
// and defaulted platform fields follow exactly the same enumeration as the plaza.
// Historical rows survive removals and never receive a replacement date.
func recordChannelModelListingsTx(ctx context.Context, tx *sql.Tx, channelID int64) error {
	var mappingJSON []byte
	err := tx.QueryRowContext(ctx, `SELECT model_mapping FROM channels WHERE id = $1`, channelID).Scan(&mappingJSON)
	if err == sql.ErrNoRows {
		return service.ErrChannelNotFound
	}
	if err != nil {
		return fmt.Errorf("load channel model listing configuration: %w", err)
	}
	channel := service.Channel{ModelMapping: unmarshalModelMapping(mappingJSON)}
	rows, err := tx.QueryContext(ctx,
		`SELECT platform, models FROM channel_model_pricing WHERE channel_id = $1 ORDER BY id`, channelID,
	)
	if err != nil {
		return fmt.Errorf("load model listing pricing: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var pricing service.ChannelModelPricing
		var modelsJSON []byte
		if err := rows.Scan(&pricing.Platform, &modelsJSON); err != nil {
			return fmt.Errorf("scan model listing pricing: %w", err)
		}
		if err := json.Unmarshal(modelsJSON, &pricing.Models); err != nil {
			pricing.Models = []string{}
		}
		channel.ModelPricing = append(channel.ModelPricing, pricing)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate model listing pricing: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close model listing pricing: %w", err)
	}

	seen := make(map[service.ModelListingKey]struct{})
	for _, model := range channel.SupportedModels() {
		seen[service.NormalizeModelListingKey(model.Platform, model.Name)] = struct{}{}
	}
	if len(seen) == 0 {
		return nil
	}
	keys := make([]service.ModelListingKey, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Platform != keys[j].Platform {
			return keys[i].Platform < keys[j].Platform
		}
		return keys[i].Name < keys[j].Name
	})
	platforms := make([]string, len(keys))
	names := make([]string, len(keys))
	for i, key := range keys {
		platforms[i], names[i] = key.Platform, key.Name
	}
	// A consistent key order also prevents opposite cross-channel insertion order
	// from deadlocking when channels introduce overlapping models concurrently.
	_, err = tx.ExecContext(ctx,
		`INSERT INTO channel_model_listings (platform, model_name)
		 SELECT platform, model_name
		 FROM unnest($1::text[], $2::text[]) AS listing(platform, model_name)
		 ORDER BY platform, model_name
		 ON CONFLICT (platform, model_name) DO NOTHING`,
		pq.Array(platforms), pq.Array(names),
	)
	if err != nil {
		return fmt.Errorf("record channel model listings: %w", err)
	}
	return nil
}

func (r *channelRepository) loadModelFirstListedAt(ctx context.Context) (map[service.ModelListingKey]time.Time, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT platform, model_name, first_listed_at FROM channel_model_listings WHERE first_listed_at IS NOT NULL`,
	)
	if err != nil {
		return nil, fmt.Errorf("load model first listing dates: %w", err)
	}
	defer func() { _ = rows.Close() }()
	dates := make(map[service.ModelListingKey]time.Time)
	for rows.Next() {
		var platform, name string
		var firstListedAt time.Time
		if err := rows.Scan(&platform, &name, &firstListedAt); err != nil {
			return nil, fmt.Errorf("scan model first listing date: %w", err)
		}
		if !firstListedAt.IsZero() {
			dates[service.NormalizeModelListingKey(platform, name)] = firstListedAt
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate model first listing dates: %w", err)
	}
	return dates, nil
}
