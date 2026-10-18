package admin

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelMonitorAlertPayloadDefaultsToDisabledAndNoEmail(t *testing.T) {
	for _, metric := range []string{"channel_monitor_success_rate", "channel_monitor_cache_rate", "channel_monitor_ttft_ms", "channel_monitor_duration_ms"} {
		t.Run(metric, func(t *testing.T) {
			raw := map[string]json.RawMessage{"name": json.RawMessage(`"Model monitor"`), "metric_type": json.RawMessage(`"` + metric + `"`), "operator": json.RawMessage(`"<"`), "threshold": json.RawMessage(`90`)}
			value, err := validateOpsAlertRulePayload(raw)
			require.NoError(t, err)
			require.False(t, value.Enabled)
			require.False(t, value.NotifyEmail)
			raw["enabled"], raw["notify_email"] = json.RawMessage(`true`), json.RawMessage(`true`)
			value, err = validateOpsAlertRulePayload(raw)
			require.NoError(t, err)
			require.True(t, value.Enabled)
			require.True(t, value.NotifyEmail)
		})
	}
}

func TestChannelMonitorAlertThresholdUnits(t *testing.T) {
	raw := map[string]json.RawMessage{"name": json.RawMessage(`"Model monitor"`), "metric_type": json.RawMessage(`"channel_monitor_success_rate"`), "operator": json.RawMessage(`">"`), "threshold": json.RawMessage(`101`)}
	_, err := validateOpsAlertRulePayload(raw)
	require.Error(t, err)
	raw["metric_type"], raw["threshold"] = json.RawMessage(`"channel_monitor_duration_ms"`), json.RawMessage(`80000`)
	_, err = validateOpsAlertRulePayload(raw)
	require.NoError(t, err)
}
