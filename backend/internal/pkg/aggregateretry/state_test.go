package aggregateretry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPortable(t *testing.T) {
	for _, body := range []string{
		`{"model":"gpt-5","messages":[{"content":"hello"}]}`,
		`{"model":"claude-sonnet-4-5","cache_control":{"type":"ephemeral"}}`,
		`{"model":"gpt-5","previous_response_id":""}`,
	} {
		require.True(t, Portable([]byte(body)), body)
	}
	for _, body := range []string{
		`broken`, `{"previous_response_id":"resp_1"}`, `{"conversation":{"id":"conv_1"}}`,
		`{"cachedContent":"cachedContents/1"}`, `{"input":[{"file_id":"file_1"}]}`,
		`{"previous_response_id":"resp_1","previous_response_id":""}`,
		`{"contents":[{"parts":[{"fileData":{"fileUri":"gs://upload"}}]}]}`,
	} {
		require.False(t, Portable([]byte(body)), body)
	}
}

func TestPlanIsPinnedAndOnlyAdvances(t *testing.T) {
	s := &State{}
	require.Equal(t, []int64{1, 2, 3}, s.Plan([]int64{1, 2, 3}))
	s.Select(2)
	require.Equal(t, []int64{3}, s.Plan([]int64{4, 3, 2, 1}))
	s.Select(1)
	require.Equal(t, int64(2), s.GroupID)
	s.Select(3)
	require.False(t, s.HasNext())
}
