package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

func TestParseUpstreamBalance(t *testing.T) {
	for _, tt := range []struct {
		body, kind string
		amount     float64
	}{
		{`{"mode":"unrestricted","isValid":true,"unit":"USD","balance":123.95}`, "wallet", 123.95},
		{`{"mode":"unrestricted","isValid":true,"unit":"USD","balance":0}`, "wallet", 0},
		{`{"mode":"unrestricted","isValid":true,"unit":"USD","balance":-1}`, "wallet", -1},
		{`{"mode":"quota_limited","isValid":true,"quota":{"remaining":9.95,"unit":"USD"}}`, "quota", 9.95},
		{`{"mode":"unrestricted","isValid":true,"unit":"USD","remaining":20,"subscription":{}}`, "subscription", 20},
	} {
		got, err := parseUpstreamBalance([]byte(tt.body))
		require.NoError(t, err)
		require.Equal(t, tt.kind, got.Kind)
		require.Equal(t, tt.amount, *got.Amount)
	}
	for _, body := range []string{`{}`, `{"balance":12}`, `{"mode":"unrestricted","isValid":false,"unit":"USD","balance":0}`, `{"mode":"unrestricted","isValid":true,"unit":"CNY","balance":12}`, `{"mode":"quota_limited","isValid":true,"rate_limits":[]}`, `{"mode":"unrestricted","isValid":true,"unit":"USD","remaining":50}`, `{"mode":"unrestricted","isValid":true,"unit":"USD","balance":"12"}`} {
		_, err := parseUpstreamBalance([]byte(body))
		require.Error(t, err, body)
	}
}

func TestProbeAccountBalancePersistsSanitizedSnapshot(t *testing.T) {
	account := &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"base_url": "https://upstream.example/v1", "api_key": "test-key"}}
	repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{1: account}}
	upstream := &httpUpstreamRecorder{resp: &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"mode":"unrestricted","isValid":true,"unit":"USD","balance":123.95,"secret":"not-persisted"}`))}}
	svc := newUpstreamBillingProbeTestService(repo, upstream, &upstreamBillingProbeSettingRepo{})
	snapshot, err := svc.ProbeAccountBalance(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, "https://upstream.example/v1/usage?days=1", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer test-key", upstream.lastReq.Header.Get("Authorization"))
	require.True(t, HTTPUpstreamRedirectsDisabled(upstream.lastReq.Context()))
	require.Equal(t, "ok", snapshot.Balance.Status)
	require.Equal(t, 123.95, *snapshot.Balance.Amount)
	require.NotNil(t, decodeUpstreamBillingProbeSnapshot(account.Extra))
	require.Nil(t, account.RateMultiplier)
	// A failed query must not claim zero funds or retain an apparently fresh balance.
	upstream.resp = &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader(`{"error":"secret"}`))}
	snapshot, err = svc.ProbeAccountBalance(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, "failed", snapshot.Balance.Status)
	require.Nil(t, snapshot.Balance.Amount)
}

func TestProbeAccountBalanceSkipsOfficialAndPreservesRate(t *testing.T) {
	account := &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"base_url": "https://api.openai.com", "api_key": "test-key"}, Extra: map[string]any{UpstreamBillingProbeExtraKey: map[string]any{"status": "unsupported"}}}
	repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{1: account}}
	upstream := &httpUpstreamRecorder{}
	svc := newUpstreamBillingProbeTestService(repo, upstream, &upstreamBillingProbeSettingRepo{})
	snapshot, err := svc.ProbeAccountBalance(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, "unsupported", snapshot.Balance.Status)
	require.Nil(t, upstream.lastReq)
	// Later rate refreshes keep the separately queried balance.
	freshAccount, err := repo.GetByID(context.Background(), 1)
	require.NoError(t, err)
	rateSnapshot := &UpstreamBillingProbeSnapshot{Status: UpstreamBillingProbeStatusFailed}
	require.NoError(t, svc.updateSnapshot(context.Background(), freshAccount, rateSnapshot, nil))
	require.NotNil(t, rateSnapshot.Balance)
}

func TestWindowStatsRetainsCallMetrics(t *testing.T) {
	latency := 1234.0
	stats := windowStatsFromAccountStats(&usagestats.AccountStats{Requests: 2, CacheReadTokens: 900, CacheInputTokens: 1000, AverageFirstTokenMs: &latency})
	require.EqualValues(t, 900, stats.CacheReadTokens)
	require.EqualValues(t, 1000, stats.CacheInputTokens)
	require.Equal(t, 1234.0, *stats.AverageFirstTokenMs)
	require.Nil(t, windowStatsFromAccountStats(nil).AverageFirstTokenMs)
}
