package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type channelMonitorV2GroupAuthorizerStub struct {
	groups []service.Group
	err    error
	calls  []int64
}

func (s *channelMonitorV2GroupAuthorizerStub) GetAvailableGroups(_ context.Context, userID int64) ([]service.Group, error) {
	s.calls = append(s.calls, userID)
	return s.groups, s.err
}

func TestChannelMonitorV2QueryListSupportsRepeatedAndCommaValues(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)
	c.Request = httptest.NewRequest("GET", "/?platform=openai,grok&platform=anthropic", nil)
	require.Equal(t, []string{"openai", "grok", "anthropic"}, queryList(c, "platform"))
}

func TestChannelMonitorV2IncidentWindow(t *testing.T) {
	h := NewChannelMonitorV2Handler(service.NewChannelMonitorV2Service(nil), nil)
	end := time.Now().UTC().Truncate(time.Hour)
	start := end.Add(-time.Hour)
	for _, tc := range []struct {
		name, start, end string
		valid            bool
	}{
		{"exact bucket", start.Format(time.RFC3339), end.Format(time.RFC3339), true},
		{"missing end", start.Format(time.RFC3339), "", false},
		{"invalid", "yesterday", end.Format(time.RFC3339), false},
		{"reversed", end.Format(time.RFC3339), start.Format(time.RFC3339), false},
		{"partial minute", start.Add(time.Second).Format(time.RFC3339), end.Format(time.RFC3339), false},
		{"partial rollup", start.Add(time.Minute).Format(time.RFC3339), end.Format(time.RFC3339), false},
		{"too old", start.Add(-31 * 24 * time.Hour).Format(time.RFC3339), end.Format(time.RFC3339), false},
		{"too wide", end.Add(-25 * time.Hour).Format(time.RFC3339), end.Format(time.RFC3339), false},
		{"future", end.Format(time.RFC3339), end.Add(48 * time.Hour).Format(time.RFC3339), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, "/?range=24h&platform=openai&model=gpt-image-1&group_id=7&start="+tc.start+"&end="+tc.end, nil)
			filter, ok := h.parseFilter(c)
			require.Equal(t, tc.valid, ok)
			if ok {
				require.Equal(t, start, filter.Start)
				require.Equal(t, end, filter.End)
				require.Equal(t, []string{"openai"}, filter.Platforms)
				require.Equal(t, []int64{7}, filter.GroupIDs)
				require.Equal(t, []string{"gpt-image-1"}, filter.Models)
			} else {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			}
		})
	}
}

func TestChannelMonitorV2IncidentWindowsMatchRetainedRollupBoundaries(t *testing.T) {
	svc := service.NewChannelMonitorV2Service(nil)
	h := NewChannelMonitorV2Handler(svc, nil)
	for _, rangeValue := range []string{"90m", "24h", "7d", "30d"} {
		t.Run(rangeValue, func(t *testing.T) {
			base, err := svc.ParseFilter(rangeValue, nil, nil, nil)
			require.NoError(t, err)
			for _, tc := range []struct {
				name  string
				start time.Time
				end   time.Time
				valid bool
			}{
				{name: "current complete bucket edges", start: base.End.Add(-base.Bucket), end: base.End, valid: true},
				{name: "minute aligned but partial coarse bucket", start: base.End.Add(-base.Bucket).Add(time.Minute), end: base.End, valid: false},
				{name: "oldest supported bucket", start: base.End.Add(-30 * 24 * time.Hour), end: base.End.Add(-30 * 24 * time.Hour).Add(base.Bucket), valid: true},
				{name: "future only bucket", start: base.End, end: base.End.Add(base.Bucket), valid: false},
			} {
				t.Run(tc.name, func(t *testing.T) {
					recorder := httptest.NewRecorder()
					c, _ := gin.CreateTestContext(recorder)
					c.Request = httptest.NewRequest(http.MethodGet, "/?range="+rangeValue+"&start="+tc.start.Format(time.RFC3339)+"&end="+tc.end.Format(time.RFC3339), nil)
					filter, ok := h.parseFilter(c)
					require.Equal(t, tc.valid, ok)
					if ok {
						require.Equal(t, tc.start, filter.Start)
						require.Equal(t, tc.end, filter.End)
						require.Equal(t, base.Bucket, filter.Bucket)
					} else {
						require.Equal(t, http.StatusBadRequest, recorder.Code)
					}
				})
			}
		})
	}
}

func TestChannelMonitorV2GroupByQueryDefaultsAndRejectsInvalid(t *testing.T) {
	groupBy, err := service.ParseChannelMonitorV2GroupBy("")
	require.NoError(t, err)
	require.Equal(t, service.ChannelMonitorV2GroupByPlatformGroup, groupBy)
	_, err = service.ParseChannelMonitorV2GroupBy("invalid")
	require.Error(t, err)
}

func TestChannelMonitorV2MatrixHandlerRejectsInvalidGroupBy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/channel-monitor-v2/matrix?group_by=invalid", nil)
	h := NewChannelMonitorV2Handler(service.NewChannelMonitorV2Service(nil), nil)
	h.Matrix(c)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestChannelMonitorV2ScopeFilterUsesAvailableGroupsForOrdinaryUser(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/channel-monitor-v2/snapshot", nil)
	c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
	authorizer := &channelMonitorV2GroupAuthorizerStub{groups: []service.Group{{ID: 3}, {ID: 7}}}
	h := &ChannelMonitorV2Handler{apiKeyService: authorizer}
	filter := service.ChannelMonitorV2Filter{GroupIDs: []int64{7, 9}}

	require.True(t, h.scopeFilter(c, &filter, false))
	require.True(t, filter.RestrictGroups)
	require.Equal(t, []int64{3, 7}, filter.AllowedGroupIDs)
	require.Equal(t, []int64{42}, authorizer.calls)
}

func TestChannelMonitorV2ScopeFilterPreservesEmptyOrdinaryUserScope(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/channel-monitor-v2/snapshot", nil)
	c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
	h := &ChannelMonitorV2Handler{apiKeyService: &channelMonitorV2GroupAuthorizerStub{}}
	filter := service.ChannelMonitorV2Filter{}

	require.True(t, h.scopeFilter(c, &filter, false))
	require.True(t, filter.RestrictGroups)
	require.Empty(t, filter.AllowedGroupIDs)
}

func TestChannelMonitorV2ScopeFilterLeavesAdminUnrestricted(t *testing.T) {
	authorizer := &channelMonitorV2GroupAuthorizerStub{}
	h := &ChannelMonitorV2Handler{apiKeyService: authorizer}
	filter := service.ChannelMonitorV2Filter{GroupIDs: []int64{9}}

	require.True(t, h.scopeFilter(nil, &filter, true))
	require.False(t, filter.RestrictGroups)
	require.Nil(t, filter.AllowedGroupIDs)
	require.Empty(t, authorizer.calls)
}
