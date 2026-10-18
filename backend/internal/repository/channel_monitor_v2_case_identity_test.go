package repository

import (
	"context"
	"database/sql/driver"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestChannelMonitorV2CaseIdentityMergesCatalogFactsAndBuckets(t *testing.T) {
	for _, whitelist := range []bool{false, true} {
		for _, view := range []string{"models", "matrix", "dimensions"} {
			t.Run(fmt.Sprintf("%s/whitelist=%t", view, whitelist), func(t *testing.T) {
				repo, mock, filter := newMonitorCatalogRepository(t)
				filter.Models = []string{"sHaReD"}
				filter.Catalog = []service.PlazaGroup{
					{ID: 10, Name: "openai", Platform: service.PlatformOpenAI, Models: []service.PlazaModel{
						{Name: "ShArEd", Platform: service.PlatformOpenAI},
						{Name: "shared", Platform: service.PlatformOpenAI},
					}},
					{ID: 30, Name: "claude", Platform: service.PlatformAnthropic, Models: []service.PlazaModel{
						{Name: "sHARED", Platform: service.PlatformAnthropic},
					}},
				}
				cfg := service.ChannelMonitorV2Config{
					Platforms: []service.ChannelMonitorV2PlatformConfig{
						{Platform: service.PlatformOpenAI, Enabled: true},
						{Platform: service.PlatformAnthropic, Enabled: true},
					},
					IgnoredErrorCategories: []string{"authentication"},
				}
				wantName := map[string]string{"openai": "ShArEd", "anthropic": "sHARED"}
				if whitelist {
					cfg.Platforms[0].Models = []string{"SHARED"}
					cfg.Platforms[1].Models = []string{"Shared"}
					wantName = map[string]string{"openai": "SHARED", "anthropic": "Shared"}
				}
				expectMonitorCatalogCoverage(mock, filter)
				if view == "dimensions" {
					mock.ExpectQuery("SELECT m.platform").WillReturnRows(sqlmock.NewRows([]string{"platform", "group_name", "group_platform", "group_id", "model", "count"}).
						AddRow("openai", "openai", "openai", 10, "shared", 10).
						AddRow("openai", "openai", "openai", 10, "SHARED", 2).
						AddRow("anthropic", "claude", "anthropic", 30, "shared", 5))
					got, err := repo.GetDimensions(context.Background(), filter, cfg)
					require.NoError(t, err)
					require.Len(t, got.Models, 2, "catalog spelling and observed spelling are one model per platform")
					for _, row := range got.Models {
						require.Equal(t, wantName[row.Platform], row.Value)
						if row.Platform == "openai" {
							require.Equal(t, int64(12), row.RequestCount)
							require.Equal(t, []int64{10}, row.GroupIDs)
						} else {
							require.Equal(t, int64(5), row.RequestCount)
							require.Equal(t, []int64{30}, row.GroupIDs)
						}
					}
					require.NoError(t, mock.ExpectationsWereMet())
					return
				}
				mock.ExpectQuery(`SUM\(m.success_requests\)`).WillReturnRows(monitorCatalogFacts(
					[]driver.Value{filter.Start, "openai", 10, "openai", "shared", 8, 2, 0, 0, 100, 40, 0, 100, 16000, 8, 24000, 8},
					[]driver.Value{filter.Start, "openai", 10, "openai", "SHARED", 1, 1, 0, 0, 100, 40, 0, 100, 4000, 1, 6000, 1},
					[]driver.Value{filter.Start, "anthropic", 30, "claude", "shared", 5, 0, 0, 0, 50, 10, 0, 0, 5000, 5, 10000, 5},
				))
				mock.ExpectQuery(`SUM\(h.sample_count\)`).WillReturnRows(monitorCatalogHistograms().
					AddRow(filter.Start, "openai", 10, "SHared", 0, "ttft", 2000, 8).
					AddRow(filter.Start, "openai", 10, "sHared", 0, "ttft", 4000, 1))
				if view == "models" {
					mock.ExpectQuery("SELECT e.platform, e.model").WillReturnRows(sqlmock.NewRows([]string{"platform", "model", "count"}).
						AddRow("openai", "shaRED", 1).AddRow("openai", "SHARED", 1))
					got, err := repo.GetModels(context.Background(), filter, cfg, true)
					require.NoError(t, err)
					require.Len(t, got.Items, 2)
					for _, row := range got.Items {
						require.Equal(t, wantName[row.Platform], row.Model)
						if row.Platform == "openai" {
							assertMonitorCaseMetric(t, row.Metrics)
						} else {
							require.Equal(t, int64(5), row.Metrics.RequestCount)
						}
					}
				} else {
					mock.ExpectQuery("e.platform, e.group_id, e.model").WillReturnRows(sqlmock.NewRows([]string{"bucket", "platform", "group_id", "model", "count"}).
						AddRow(filter.Start, "openai", 10, "shaRED", 1).AddRow(filter.Start, "openai", 10, "SHARED", 1))
					got, err := repo.GetMatrix(context.Background(), filter, cfg, service.ChannelMonitorV2GroupByPlatformGroupModel, true)
					require.NoError(t, err)
					require.Len(t, got.Items, 2)
					for _, row := range got.Items {
						require.Equal(t, wantName[row.Platform], row.Model)
						if row.Platform == "openai" {
							assertMonitorCaseMetric(t, row.Metrics)
							require.Len(t, row.Buckets, 1)
							assertMonitorCaseMetric(t, row.Buckets[0].Metrics)
						} else {
							require.Equal(t, int64(5), row.Metrics.RequestCount)
						}
					}
				}
				require.NoError(t, mock.ExpectationsWereMet())
			})
		}
	}
}

func assertMonitorCaseMetric(t *testing.T, metrics service.ChannelMonitorV2Metric) {
	t.Helper()
	require.Equal(t, int64(12), metrics.RequestCount)
	require.Equal(t, int64(9), metrics.SuccessRequests)
	require.InDelta(t, 0.75, metrics.SuccessRate, 1e-10)
	require.InDelta(t, 1.0/12, metrics.ErrorRate, 1e-10, "ignored error variants must join the same model")
	require.Equal(t, int64(9), metrics.TTFT.SampleCount)
	require.NotNil(t, metrics.TTFT.P50Ms)
	require.Equal(t, int64(2000), *metrics.TTFT.P50Ms)
}

func TestChannelMonitorV2CaseIdentityNamesKeepConfiguredSpelling(t *testing.T) {
	cfg := service.ChannelMonitorV2Config{Platforms: []service.ChannelMonitorV2PlatformConfig{
		{Platform: "openai", Enabled: true, Models: []string{"GPT-6.1-SOL"}},
		{Platform: "anthropic", Enabled: true},
	}}
	filter := service.ChannelMonitorV2Filter{Models: []string{"gpt-6.1-sol"}, Catalog: []service.PlazaGroup{
		{ID: 10, Platform: "openai", Models: []service.PlazaModel{{Name: "gPt-6.1-sol", Platform: "openai"}}},
		{ID: 20, Platform: "anthropic", Models: []service.PlazaModel{{Name: "GPT-6.1-sol", Platform: "anthropic"}}},
	}}
	require.True(t, channelMonitorV2ModelSelected(filter, cfg, "openai", "gPt-6.1-SoL"))
	require.Equal(t, []string{"GPT-6.1-SOL"}, configuredChannelMonitorV2Models(cfg, "openai", filter))
	require.Equal(t, "GPT-6.1-SOL", channelMonitorV2ModelDisplayName(filter, cfg, "openai", "gpt-6.1-sol", ""))
	require.Equal(t, "GPT-6.1-sol", channelMonitorV2ModelDisplayName(filter, cfg, "anthropic", "gpt-6.1-sol", ""))
	require.Equal(t, "GPT-6.1-SOL", channelMonitorV2DisplayModel(cfg, "openai", "gpt-6.1-sol"))

	// A traffic-only name stays an original spelling and is stable across SQL row order.
	for _, names := range [][]string{{"raw-MODEL", "RAW-model"}, {"RAW-model", "raw-MODEL"}} {
		name := ""
		for _, raw := range names {
			name = channelMonitorV2ModelDisplayName(service.ChannelMonitorV2Filter{}, cfg, "openai", raw, name)
		}
		require.Equal(t, "RAW-model", name)
	}
}
