package repository

import (
	"context"
	"database/sql/driver"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func newMonitorCatalogRepository(t *testing.T) (*channelMonitorV2Repository, sqlmock.Sqlmock, service.ChannelMonitorV2Filter) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	start := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	return &channelMonitorV2Repository{db: db}, mock, service.ChannelMonitorV2Filter{Start: start, End: start.Add(time.Hour), Bucket: time.Hour}
}

func expectMonitorCatalogCoverage(mock sqlmock.Sqlmock, filter service.ChannelMonitorV2Filter) {
	mock.ExpectQuery("SELECT usage_coverage_start, error_coverage_start, data_through, last_successful_at, backfill_cursor").
		WillReturnRows(sqlmock.NewRows([]string{"usage_start", "error_start", "through", "successful", "backfill"}).
			AddRow(filter.Start, filter.Start, filter.End, filter.End, filter.Start))
}

func monitorCatalogFacts(rows ...[]driver.Value) *sqlmock.Rows {
	out := sqlmock.NewRows([]string{
		"bucket", "platform", "group_id", "group_name", "model", "success", "errors", "upstream_affected", "upstream_attempts",
		"input", "output", "cache_creation", "cache_read", "ttft_sum", "ttft_count", "duration_sum", "duration_count",
	})
	for _, row := range rows {
		out.AddRow(row...)
	}
	return out
}

func monitorCatalogHistograms() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"bucket", "platform", "group_id", "model", "user_id", "metric", "upper", "count"})
}

func expectMonitorCatalogMetrics(mock sqlmock.Sqlmock, facts *sqlmock.Rows) {
	mock.ExpectQuery(`SUM\(m.success_requests\)`).WillReturnRows(facts)
	mock.ExpectQuery(`SUM\(h.sample_count\)`).WillReturnRows(monitorCatalogHistograms())
}

func TestChannelMonitorV2CatalogNoTrafficAppearsThroughPublicQueries(t *testing.T) {
	for _, view := range []string{"dimensions", "models", "matrix"} {
		t.Run(view, func(t *testing.T) {
			repo, mock, filter := newMonitorCatalogRepository(t)
			filter.Catalog = []service.PlazaGroup{
				{ID: 10, Name: "standard", Platform: service.PlatformOpenAI, Models: []service.PlazaModel{
					{Name: "gpt-6.1-sol", Platform: service.PlatformOpenAI},
					{Name: "gpt-image-2.5-flare", Platform: service.PlatformOpenAI},
				}},
				{ID: 20, Name: "premium", Platform: service.PlatformOpenAI, Models: []service.PlazaModel{
					{Name: "gpt-6.1-sol", Platform: service.PlatformOpenAI},
				}},
				{ID: 30, Name: "claude", Platform: service.PlatformAnthropic, Models: []service.PlazaModel{
					{Name: "claude-sonnet-4-6", Platform: service.PlatformAnthropic},
				}},
			}
			cfg := service.ChannelMonitorV2Config{Platforms: []service.ChannelMonitorV2PlatformConfig{
				{Platform: service.PlatformOpenAI, Enabled: true},
				{Platform: service.PlatformAnthropic, Enabled: true},
			}}
			expectMonitorCatalogCoverage(mock, filter)
			if view == "dimensions" {
				mock.ExpectQuery("SELECT m.platform").WillReturnRows(sqlmock.NewRows([]string{"platform", "group_name", "group_platform", "group_id", "model", "count"}))
				got, err := repo.GetDimensions(context.Background(), filter, cfg)
				require.NoError(t, err)
				require.Len(t, got.Groups, 3)
				require.Len(t, got.Models, 3)
				for _, model := range got.Models {
					require.Zero(t, model.RequestCount)
					require.NotEqual(t, service.ChannelMonitorV2OtherModel, model.Value)
					if model.Value == "gpt-6.1-sol" {
						require.Equal(t, []int64{10, 20}, model.GroupIDs)
					}
				}
			} else {
				expectMonitorCatalogMetrics(mock, monitorCatalogFacts())
				if view == "models" {
					got, err := repo.GetModels(context.Background(), filter, cfg, false)
					require.NoError(t, err)
					require.Len(t, got.Items, 3)
					for _, row := range got.Items {
						require.False(t, row.Metrics.HasData)
						require.Zero(t, row.Metrics.RequestCount)
						require.Equal(t, "unknown", row.Health.Overall)
						require.Nil(t, row.Metrics.TTFT.AvgMs)
						require.Equal(t, row.Model == "gpt-image-2.5-flare", row.IsImageModel)
					}
				} else {
					got, err := repo.GetMatrix(context.Background(), filter, cfg, service.ChannelMonitorV2GroupByPlatformGroupModel, false)
					require.NoError(t, err)
					require.Len(t, got.Items, 4)
					for _, row := range got.Items {
						require.NotNil(t, row.GroupID)
						require.NotEmpty(t, row.GroupName)
						require.NotEmpty(t, row.Model)
						require.False(t, row.Metrics.HasData)
						require.Equal(t, "unknown", row.Health.Overall)
						require.Empty(t, row.Buckets)
					}
				}
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestChannelMonitorV2CatalogStrictWhitelistIncludesOnlySelectedImages(t *testing.T) {
	for _, matrix := range []bool{false, true} {
		t.Run(fmt.Sprintf("matrix=%t", matrix), func(t *testing.T) {
			repo, mock, filter := newMonitorCatalogRepository(t)
			filter.Catalog = []service.PlazaGroup{{ID: 10, Name: "openai", Platform: service.PlatformOpenAI, Models: []service.PlazaModel{
				{Name: "gpt-6.1-sol", Platform: service.PlatformOpenAI},
				{Name: "gpt-image-2.5-flare", Platform: service.PlatformOpenAI},
				{Name: "gpt-image-2.5-sunburst", Platform: service.PlatformOpenAI},
			}}}
			cfg := service.ChannelMonitorV2Config{Platforms: []service.ChannelMonitorV2PlatformConfig{{
				Platform: service.PlatformOpenAI, Enabled: true, Models: []string{"gpt-6.1-sol", "gpt-image-2.5-flare", "gpt-unlisted-yet"},
			}}}
			expectMonitorCatalogCoverage(mock, filter)
			expectMonitorCatalogMetrics(mock, monitorCatalogFacts(
				[]driver.Value{filter.Start, "openai", 10, "openai", "gpt-6.1-sol", 8, 2, 0, 0, 100, 40, 0, 100, 8000, 8, 16000, 8},
				[]driver.Value{filter.Start, "openai", 10, "openai", "gpt-image-2.5-flare", 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				[]driver.Value{filter.Start, "openai", 10, "openai", "gpt-image-2.5-sunburst", 100, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				[]driver.Value{filter.Start, "openai", 10, "openai", "outside-whitelist", 100, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			))
			metrics := map[string]service.ChannelMonitorV2Metric{}
			if matrix {
				got, err := repo.GetMatrix(context.Background(), filter, cfg, service.ChannelMonitorV2GroupByPlatformGroupModel, true)
				require.NoError(t, err)
				for _, row := range got.Items {
					metrics[row.Model] = row.Metrics
				}
			} else {
				got, err := repo.GetModels(context.Background(), filter, cfg, true)
				require.NoError(t, err)
				for _, row := range got.Items {
					metrics[row.Model] = row.Metrics
				}
			}
			require.Len(t, metrics, 3)
			require.NotContains(t, metrics, service.ChannelMonitorV2OtherModel)
			require.NotContains(t, metrics, "outside-whitelist")
			require.NotContains(t, metrics, "gpt-image-2.5-sunburst")
			require.Equal(t, int64(10), metrics["gpt-6.1-sol"].RequestCount)
			require.InDelta(t, 0.8, metrics["gpt-6.1-sol"].SuccessRate, 1e-10)
			require.InDelta(t, 0.5, metrics["gpt-6.1-sol"].CacheRate, 1e-10)
			require.Equal(t, int64(1), metrics["gpt-image-2.5-flare"].ErrorRequests)
			require.Zero(t, metrics["gpt-image-2.5-flare"].SuccessRate)
			require.False(t, metrics["gpt-unlisted-yet"].HasData)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestChannelMonitorV2CatalogCompositeUsesConcreteModelPlatforms(t *testing.T) {
	repo, mock, filter := newMonitorCatalogRepository(t)
	filter.Catalog = []service.PlazaGroup{{ID: 10, Name: "mixed", Platform: service.PlatformComposite, Models: []service.PlazaModel{
		{Name: "shared", Platform: service.PlatformOpenAI},
		{Name: "shared", Platform: service.PlatformAnthropic},
		{Name: "grok-imagine-image", Platform: service.PlatformGrok},
	}}}
	cfg := service.ChannelMonitorV2Config{Platforms: []service.ChannelMonitorV2PlatformConfig{
		{Platform: service.PlatformOpenAI, Enabled: true},
		{Platform: service.PlatformAnthropic, Enabled: true},
		{Platform: service.PlatformGrok, Enabled: true},
	}}
	expectMonitorCatalogCoverage(mock, filter)
	expectMonitorCatalogMetrics(mock, monitorCatalogFacts())
	got, err := repo.GetMatrix(context.Background(), filter, cfg, service.ChannelMonitorV2GroupByPlatformGroupModel, false)
	require.NoError(t, err)
	require.Len(t, got.Items, 3)
	seen := map[string]bool{}
	for _, row := range got.Items {
		require.Equal(t, int64(10), *row.GroupID)
		require.Equal(t, "mixed", row.GroupName)
		seen[row.Platform+"/"+row.Model] = true
	}
	require.Equal(t, map[string]bool{"openai/shared": true, "anthropic/shared": true, "grok/grok-imagine-image": true}, seen)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelMonitorV2CatalogDimensionsLinkCompositeGroupsToEveryPlatform(t *testing.T) {
	repo, mock, filter := newMonitorCatalogRepository(t)
	filter.Platforms = []string{service.PlatformOpenAI}
	filter.Models = []string{"shared"}
	filter.Catalog = []service.PlazaGroup{{ID: 10, Name: "mixed", Platform: service.PlatformComposite, Models: []service.PlazaModel{
		{Name: "shared", Platform: service.PlatformOpenAI},
		{Name: "shared", Platform: service.PlatformAnthropic},
		{Name: "grok-imagine-image", Platform: service.PlatformGrok},
	}}}
	cfg := service.ChannelMonitorV2Config{Platforms: []service.ChannelMonitorV2PlatformConfig{
		{Platform: service.PlatformOpenAI, Enabled: true},
		{Platform: service.PlatformAnthropic, Enabled: true},
		{Platform: service.PlatformGrok, Enabled: true},
	}}
	expectMonitorCatalogCoverage(mock, filter)
	mock.ExpectQuery("SELECT m.platform").WillReturnRows(sqlmock.NewRows([]string{"platform", "group_name", "group_platform", "group_id", "model", "count"}))
	got, err := repo.GetDimensions(context.Background(), filter, cfg)
	require.NoError(t, err)
	require.Len(t, got.Groups, 1)
	require.Equal(t, []string{"anthropic", "grok", "openai"}, got.Groups[0].Platforms)
	require.Len(t, got.Models, 3, "dimensions retain the authorized picker menu when an individual platform/model is selected")
	for _, model := range got.Models {
		require.Equal(t, []int64{10}, model.GroupIDs)
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelMonitorV2CatalogDimensionsApplyImageWhitelistToTrafficAndSeeds(t *testing.T) {
	repo, mock, filter := newMonitorCatalogRepository(t)
	filter.Catalog = []service.PlazaGroup{{ID: 10, Name: "openai", Platform: service.PlatformOpenAI, Models: []service.PlazaModel{
		{Name: "gpt-image-2.5-flare", Platform: service.PlatformOpenAI},
		{Name: "gpt-image-2.5-sunburst", Platform: service.PlatformOpenAI},
	}}}
	cfg := service.ChannelMonitorV2Config{Platforms: []service.ChannelMonitorV2PlatformConfig{{
		Platform: service.PlatformOpenAI, Enabled: true, Models: []string{"gpt-image-2.5-flare", "gpt-unlisted-yet"},
	}}}
	expectMonitorCatalogCoverage(mock, filter)
	mock.ExpectQuery("SELECT m.platform").WillReturnRows(sqlmock.NewRows([]string{"platform", "group_name", "group_platform", "group_id", "model", "count"}).
		AddRow("openai", "openai", "openai", 10, "gpt-image-2.5-flare", 2).
		AddRow("openai", "openai", "openai", 10, "gpt-image-2.5-sunburst", 100))
	got, err := repo.GetDimensions(context.Background(), filter, cfg)
	require.NoError(t, err)
	require.Len(t, got.Models, 2)
	counts := map[string]int64{}
	for _, model := range got.Models {
		counts[model.Value] = model.RequestCount
	}
	require.Equal(t, map[string]int64{"gpt-image-2.5-flare": 2, "gpt-unlisted-yet": 0}, counts)
	require.Equal(t, int64(2), got.Platforms[0].RequestCount)
	require.Equal(t, int64(2), got.Groups[0].RequestCount)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelMonitorV2CatalogWithoutWhitelistRetainsObservedRealModels(t *testing.T) {
	for _, matrix := range []bool{false, true} {
		t.Run(fmt.Sprintf("matrix=%t", matrix), func(t *testing.T) {
			repo, mock, filter := newMonitorCatalogRepository(t)
			filter.Catalog = []service.PlazaGroup{{ID: 10, Name: "openai", Platform: service.PlatformOpenAI, Models: []service.PlazaModel{
				{Name: "gpt-6.1-sol", Platform: service.PlatformOpenAI},
			}}}
			cfg := service.ChannelMonitorV2Config{Platforms: []service.ChannelMonitorV2PlatformConfig{{Platform: service.PlatformOpenAI, Enabled: true}}}
			expectMonitorCatalogCoverage(mock, filter)
			expectMonitorCatalogMetrics(mock, monitorCatalogFacts([]driver.Value{
				filter.Start, "openai", 10, "openai", "observed-image-alias", 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2000, 1,
			}))
			names := map[string]bool{}
			if matrix {
				got, err := repo.GetMatrix(context.Background(), filter, cfg, service.ChannelMonitorV2GroupByPlatformGroupModel, true)
				require.NoError(t, err)
				for _, row := range got.Items {
					names[row.Model] = row.Metrics.HasData
				}
			} else {
				got, err := repo.GetModels(context.Background(), filter, cfg, true)
				require.NoError(t, err)
				for _, row := range got.Items {
					names[row.Model] = row.Metrics.HasData
				}
			}
			require.Equal(t, map[string]bool{"gpt-6.1-sol": false, "observed-image-alias": true}, names)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestChannelMonitorV2CatalogScopeUsesViewerConfigAndSelectionIntersection(t *testing.T) {
	for _, view := range []string{"dimensions", "models", "matrix"} {
		t.Run(view, func(t *testing.T) {
			repo, mock, filter := newMonitorCatalogRepository(t)
			filter.RestrictGroups = true
			filter.AllowedGroupIDs = []int64{10, 20}
			filter.GroupIDs = []int64{10}
			filter.Platforms = []string{service.PlatformOpenAI}
			filter.Catalog = []service.PlazaGroup{
				{ID: 10, Name: "allowed", Platform: service.PlatformOpenAI, Models: []service.PlazaModel{{Name: "allowed", Platform: service.PlatformOpenAI}}},
				{ID: 20, Name: "not-monitored", Platform: service.PlatformOpenAI, Models: []service.PlazaModel{{Name: "not-monitored", Platform: service.PlatformOpenAI}}},
				{ID: 30, Name: "not-authorized", Platform: service.PlatformOpenAI, Models: []service.PlazaModel{{Name: "not-authorized", Platform: service.PlatformOpenAI}}},
			}
			cfg := service.ChannelMonitorV2Config{GroupIDs: []int64{10, 30}, Platforms: []service.ChannelMonitorV2PlatformConfig{{Platform: service.PlatformOpenAI, Enabled: true}}}
			expectMonitorCatalogCoverage(mock, filter)
			if view == "dimensions" {
				mock.ExpectQuery("SELECT m.platform").WithArgs(filter.Start, filter.End, pq.Array([]string{"openai"}), pq.Array([]int64{10}), 3600).
					WillReturnRows(sqlmock.NewRows([]string{"platform", "group_name", "group_platform", "group_id", "model", "count"}))
				got, err := repo.GetDimensions(context.Background(), filter, cfg)
				require.NoError(t, err)
				require.Len(t, got.Groups, 1)
				require.Equal(t, int64(10), got.Groups[0].ID)
				require.Len(t, got.Models, 1)
				require.Equal(t, []int64{10}, got.Models[0].GroupIDs)
			} else {
				mock.ExpectQuery(`SUM\(m.success_requests\)`).WithArgs(filter.Start, filter.End, pq.Array([]string{"openai"}), pq.Array([]int64{10}), 3600).
					WillReturnRows(monitorCatalogFacts())
				mock.ExpectQuery(`SUM\(h.sample_count\)`).WithArgs(filter.Start, filter.End, pq.Array([]string{"openai"}), pq.Array([]int64{10}), 3600, int64(0)).
					WillReturnRows(monitorCatalogHistograms())
				if view == "models" {
					got, err := repo.GetModels(context.Background(), filter, cfg, false)
					require.NoError(t, err)
					require.Len(t, got.Items, 1)
					require.Equal(t, "allowed", got.Items[0].Model)
				} else {
					got, err := repo.GetMatrix(context.Background(), filter, cfg, service.ChannelMonitorV2GroupByPlatformGroupModel, false)
					require.NoError(t, err)
					require.Len(t, got.Items, 1)
					require.Equal(t, int64(10), *got.Items[0].GroupID)
				}
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestChannelMonitorV2CatalogSeedsDoNotDiluteAggregatedMetrics(t *testing.T) {
	for _, groupBy := range []service.ChannelMonitorV2GroupBy{service.ChannelMonitorV2GroupByPlatform, service.ChannelMonitorV2GroupByPlatformModel} {
		t.Run(string(groupBy), func(t *testing.T) {
			repo, mock, filter := newMonitorCatalogRepository(t)
			filter.Catalog = []service.PlazaGroup{
				{ID: 10, Name: "first", Platform: service.PlatformOpenAI, Models: []service.PlazaModel{{Name: "gpt-6.1-sol", Platform: service.PlatformOpenAI}, {Name: "unused", Platform: service.PlatformOpenAI}}},
				{ID: 20, Name: "second", Platform: service.PlatformOpenAI, Models: []service.PlazaModel{{Name: "gpt-6.1-sol", Platform: service.PlatformOpenAI}}},
			}
			cfg := service.ChannelMonitorV2Config{Platforms: []service.ChannelMonitorV2PlatformConfig{{Platform: service.PlatformOpenAI, Enabled: true}}}
			expectMonitorCatalogCoverage(mock, filter)
			expectMonitorCatalogMetrics(mock, monitorCatalogFacts(
				[]driver.Value{filter.Start, "openai", 10, "first", "gpt-6.1-sol", 8, 2, 0, 0, 100, 0, 0, 100, 8000, 8, 16000, 8},
				[]driver.Value{filter.Start, "openai", 20, "second", "gpt-6.1-sol", 1, 9, 0, 0, 900, 0, 0, 100, 4000, 1, 8000, 1},
			))
			got, err := repo.GetMatrix(context.Background(), filter, cfg, groupBy, true)
			require.NoError(t, err)
			var actual *service.ChannelMonitorV2MatrixRow
			for i := range got.Items {
				row := &got.Items[i]
				if row.Model == "gpt-6.1-sol" || groupBy == service.ChannelMonitorV2GroupByPlatform {
					actual = row
				}
			}
			require.NotNil(t, actual)
			require.Equal(t, int64(20), actual.Metrics.RequestCount)
			require.InDelta(t, 0.45, actual.Metrics.SuccessRate, 1e-10)
			require.InDelta(t, 1.0/6, actual.Metrics.CacheRate, 1e-10)
			require.InDelta(t, 12000.0/9, *actual.Metrics.TTFT.AvgMs, 1e-10)
			require.Len(t, actual.Buckets, 1)
			require.Equal(t, actual.Metrics.SuccessRate, actual.Buckets[0].Metrics.SuccessRate)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
