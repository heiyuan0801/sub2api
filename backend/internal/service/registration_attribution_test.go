package service

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRegistrationAttributionKeepsMarketingFieldsSeparate(t *testing.T) {
	got := ParseRegistrationAttribution([]byte(`{
		"selected_source":" OTHER ","source_note":" 社区\r\n推荐 ",
		"utm_source":" GitHub ","utm_medium":"newsletter","utm_campaign":"launch",
		"signup_source":"google","user_id":7
	}`))
	require.Equal(t, RegistrationAttribution{
		SelectedSource: "other", SourceNote: "社区推荐", UTMSource: "github",
		UTMMedium: "newsletter", UTMCampaign: "launch",
	}, got)
	normalized := ParseRegistrationAttribution([]byte(`{"selected_source":"google","source_note":"ignored","utm_source":"unknown"}`))
	require.Equal(t, RegistrationAttribution{SelectedSource: "google"}, normalized)
	require.Empty(t, ParseRegistrationAttribution([]byte(`{"selected_source":["github"]}`)))
	require.Empty(t, ParseRegistrationAttribution([]byte(strings.Repeat("a", MaxRegistrationAttributionBytes+1))))
	ctx := WithRegistrationAttribution(context.Background(), got)
	fromContext, ok := RegistrationAttributionFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, got, fromContext)
}
