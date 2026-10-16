//go:build unit

package repository

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

var channelModelPricingTimePricingColumns = []string{
	"id", "channel_id", "platform", "models", "billing_mode", "input_price", "output_price",
	"cache_write_price", "cache_write_1h_price", "cache_read_price", "fast_multiplier", "flex_multiplier", "reasoning_effort_multipliers", "image_input_price", "image_output_price",
	"per_request_price", "time_pricing", "created_at", "updated_at",
}

const channelModelPricingTimePricingJSON = `{"timezone":"Asia/Shanghai","periods":[{"start_time":"09:00","end_time":"12:00","multiplier":2}]}`

func newChannelModelPricingTimePricingRepo(t *testing.T) (*channelRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return &channelRepository{db: db}, mock
}

func modelPricingTimePricingRow(timePricing any) *sqlmock.Rows {
	return sqlmock.NewRows(channelModelPricingTimePricingColumns).AddRow(
		int64(11), int64(7), "openai", `["gpt-5"]`, service.BillingModeToken,
		nil, nil, nil, nil, nil, nil, nil, "{}", nil, nil, nil, timePricing,
		time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC), time.Date(2026, 8, 17, 1, 0, 0, 0, time.UTC),
	)
}

func expectEmptyModelPricingIntervals(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(`SELECT id, pricing_id, min_tokens, max_tokens, tier_label`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
}

func expectChannelModelPricingLock(mock sqlmock.Sqlmock, channelID int64) {
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM channels WHERE id = $1 FOR UPDATE")).
		WithArgs(channelID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(channelID))
}

func expectChannelModelPricingWriteTx(mock sqlmock.Sqlmock, channelID int64) {
	mock.ExpectBegin()
	expectChannelModelPricingLock(mock, channelID)
}

func expectChannelModelPricingUpdateTx(mock sqlmock.Sqlmock, pricingID, channelID int64) {
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT channel_id FROM channel_model_pricing WHERE id = $1")).
		WithArgs(pricingID).
		WillReturnRows(sqlmock.NewRows([]string{"channel_id"}).AddRow(channelID))
	expectChannelModelPricingLock(mock, channelID)
}

func expectChannelModelPricingListing(mock sqlmock.Sqlmock, channelID int64, platform, modelsJSON, modelName string) *sqlmock.ExpectedExec {
	mock.ExpectQuery(regexp.QuoteMeta("SELECT model_mapping FROM channels WHERE id = $1")).
		WithArgs(channelID).
		WillReturnRows(sqlmock.NewRows([]string{"model_mapping"}).AddRow(`{}`))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT platform, models FROM channel_model_pricing WHERE channel_id = $1 ORDER BY id")).
		WithArgs(channelID).
		WillReturnRows(sqlmock.NewRows([]string{"platform", "models"}).AddRow(platform, modelsJSON)).
		RowsWillBeClosed()
	return mock.ExpectExec(`(?s)INSERT INTO channel_model_listings \(platform, model_name\).*ON CONFLICT \(platform, model_name\) DO NOTHING$`).
		WithArgs(pq.Array([]string{platform}), pq.Array([]string{modelName}))
}

func TestChannelModelPricingTimePricingListRoundTrip(t *testing.T) {
	repo, mock := newChannelModelPricingTimePricingRepo(t)
	mock.ExpectQuery(`(?s)SELECT .*per_request_price, time_pricing, created_at, updated_at.*FROM channel_model_pricing.*channel_id = \$1`).
		WithArgs(int64(7)).
		WillReturnRows(modelPricingTimePricingRow(channelModelPricingTimePricingJSON))
	expectEmptyModelPricingIntervals(mock)

	pricing, err := repo.ListModelPricing(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, pricing, 1)
	require.NotNil(t, pricing[0].TimePricing)
	require.Equal(t, "Asia/Shanghai", pricing[0].TimePricing.Timezone)
	require.Len(t, pricing[0].TimePricing.Periods, 1)
	require.Equal(t, 2.0, pricing[0].TimePricing.Periods[0].Multiplier)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelModelPricingTimePricingListNullAndMalformed(t *testing.T) {
	t.Run("SQL NULL maps to nil", func(t *testing.T) {
		repo, mock := newChannelModelPricingTimePricingRepo(t)
		mock.ExpectQuery(`(?s)SELECT .*per_request_price, time_pricing, created_at, updated_at.*FROM channel_model_pricing.*channel_id = \$1`).
			WithArgs(int64(7)).
			WillReturnRows(modelPricingTimePricingRow(nil))
		expectEmptyModelPricingIntervals(mock)

		pricing, err := repo.ListModelPricing(context.Background(), 7)
		require.NoError(t, err)
		require.Len(t, pricing, 1)
		require.Nil(t, pricing[0].TimePricing)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("malformed JSON returns repository error", func(t *testing.T) {
		repo, mock := newChannelModelPricingTimePricingRepo(t)
		mock.ExpectQuery(`(?s)SELECT .*per_request_price, time_pricing, created_at, updated_at.*FROM channel_model_pricing.*channel_id = \$1`).
			WithArgs(int64(7)).
			WillReturnRows(modelPricingTimePricingRow(`{"timezone":`))

		_, err := repo.ListModelPricing(context.Background(), 7)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), "unmarshal time pricing"), "unexpected error: %v", err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestChannelModelPricingTimePricingCreateAndUpdateRoundTrip(t *testing.T) {
	pricing := &service.ChannelModelPricing{
		ID:        11,
		ChannelID: 7,
		Platform:  "openai",
		Models:    []string{"gpt-5"},
		TimePricing: &service.ChannelTimePricing{
			Timezone: "Asia/Shanghai",
			Periods: []service.ChannelTimePricingPeriod{{
				StartTime: "09:00", EndTime: "12:00", Multiplier: 2,
			}},
		},
	}

	t.Run("create writes JSON", func(t *testing.T) {
		repo, mock := newChannelModelPricingTimePricingRepo(t)
		expectChannelModelPricingWriteTx(mock, 7)
		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO channel_model_pricing (channel_id, platform, models, billing_mode, input_price, output_price, cache_write_price, cache_write_1h_price, cache_read_price, fast_multiplier, flex_multiplier, reasoning_effort_multipliers, image_input_price, image_output_price, per_request_price, time_pricing)")).
			WithArgs(
				int64(7), "openai", []byte(`["gpt-5"]`), service.BillingModeToken,
				nil, nil, nil, nil, nil, nil, nil, "{}", nil, nil, nil, channelModelPricingTimePricingJSON,
			).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(11), time.Time{}, time.Time{}))
		expectChannelModelPricingListing(mock, 7, "openai", `["gpt-5"]`, "gpt-5").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		require.NoError(t, repo.CreateModelPricing(context.Background(), pricing))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("update writes JSON and entry ID", func(t *testing.T) {
		repo, mock := newChannelModelPricingTimePricingRepo(t)
		expectChannelModelPricingUpdateTx(mock, 11, 7)
		mock.ExpectExec(`(?s)UPDATE channel_model_pricing.*per_request_price = \$13, time_pricing = \$14, platform = \$15.*WHERE id = \$16`).
			WithArgs(
				[]byte(`["gpt-5"]`), service.BillingModeToken,
				nil, nil, nil, nil, nil, nil, nil, "{}", nil, nil, nil, channelModelPricingTimePricingJSON, "openai", int64(11),
			).
			WillReturnResult(sqlmock.NewResult(0, 1))
		expectChannelModelPricingListing(mock, 7, "openai", `["gpt-5"]`, "gpt-5").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		// Ownership comes from the stored pricing entry, not a caller's ChannelID.
		updated := *pricing
		updated.ChannelID = 999
		require.NoError(t, repo.UpdateModelPricing(context.Background(), &updated))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestChannelModelPricingTimePricingCreateAndUpdateWriteNullWhenDisabled(t *testing.T) {
	tests := []struct {
		name        string
		timePricing *service.ChannelTimePricing
	}{
		{name: "nil", timePricing: nil},
		{name: "empty periods", timePricing: &service.ChannelTimePricing{Timezone: "Asia/Shanghai"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newPricing := func() *service.ChannelModelPricing {
				return &service.ChannelModelPricing{
					ID:          11,
					ChannelID:   7,
					Platform:    "openai",
					Models:      []string{"gpt-5"},
					TimePricing: tt.timePricing,
				}
			}

			t.Run("create writes SQL NULL", func(t *testing.T) {
				repo, mock := newChannelModelPricingTimePricingRepo(t)
				expectChannelModelPricingWriteTx(mock, 7)
				mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO channel_model_pricing (channel_id, platform, models, billing_mode, input_price, output_price, cache_write_price, cache_write_1h_price, cache_read_price, fast_multiplier, flex_multiplier, reasoning_effort_multipliers, image_input_price, image_output_price, per_request_price, time_pricing)")).
					WithArgs(
						int64(7), "openai", []byte(`["gpt-5"]`), service.BillingModeToken,
						nil, nil, nil, nil, nil, nil, nil, "{}", nil, nil, nil, nil,
					).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(11), time.Time{}, time.Time{}))
				expectChannelModelPricingListing(mock, 7, "openai", `["gpt-5"]`, "gpt-5").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()

				require.NoError(t, repo.CreateModelPricing(context.Background(), newPricing()))
				require.NoError(t, mock.ExpectationsWereMet())
			})

			t.Run("update writes SQL NULL", func(t *testing.T) {
				repo, mock := newChannelModelPricingTimePricingRepo(t)
				expectChannelModelPricingUpdateTx(mock, 11, 7)
				mock.ExpectExec(`(?s)UPDATE channel_model_pricing.*per_request_price = \$13, time_pricing = \$14, platform = \$15.*WHERE id = \$16`).
					WithArgs(
						[]byte(`["gpt-5"]`), service.BillingModeToken,
						nil, nil, nil, nil, nil, nil, nil, "{}", nil, nil, nil, nil, "openai", int64(11),
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
				expectChannelModelPricingListing(mock, 7, "openai", `["gpt-5"]`, "gpt-5").
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()

				require.NoError(t, repo.UpdateModelPricing(context.Background(), newPricing()))
				require.NoError(t, mock.ExpectationsWereMet())
			})
		})
	}
}

func TestChannelModelPricingListingFailureRollsBackPricing(t *testing.T) {
	repo, mock := newChannelModelPricingTimePricingRepo(t)
	pricing := &service.ChannelModelPricing{ChannelID: 7, Platform: "openai", Models: []string{"gpt-5"}}
	expectChannelModelPricingWriteTx(mock, 7)
	mock.ExpectQuery(`INSERT INTO channel_model_pricing .*reasoning_effort_multipliers`).
		WithArgs(int64(7), "openai", []byte(`["gpt-5"]`), service.BillingModeToken,
			nil, nil, nil, nil, nil, nil, nil, "{}", nil, nil, nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(11), time.Time{}, time.Time{}))
	expectChannelModelPricingListing(mock, 7, "openai", `["gpt-5"]`, "gpt-5").
		WillReturnError(errors.New("listing unavailable"))
	mock.ExpectRollback()

	err := repo.CreateModelPricing(context.Background(), pricing)
	require.ErrorContains(t, err, "record channel model listings: listing unavailable")
	require.NoError(t, mock.ExpectationsWereMet())
}
