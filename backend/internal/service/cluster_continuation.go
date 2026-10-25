package service

import (
	"context"
	"errors"
	"strconv"
)

var errClusterContinuation = errors.New("session connection changed; resend full context without previous_response_id")

func responseOriginKey(groupID int64, responseID string) string {
	return sessionMetadataKey("response-origin", strconv.FormatInt(groupID, 10), responseID)
}

// A socket-only continuation cannot be replayed on a different process. Do not
// silently remove its anchor and submit a delta as if it were the full history.
// Stored upstream responses retain the normal recovery path with shared turn state.
func checkClusterContinuation(ctx context.Context, state OpenAIWSStateStore, groupID int64, responseID string, socketOnly bool) error {
	s, ok := state.(*defaultOpenAIWSStateStore)
	if !ok || s.shared == nil || !socketOnly || responseID == "" {
		return nil
	}
	if _, local := s.GetResponseConn(responseID); local {
		return nil
	}
	cacheCtx, cancel := withOpenAIWSStateStoreRedisTimeout(ctx)
	defer cancel()
	values, err := s.shared.GetSessionValues(cacheCtx, []string{responseOriginKey(groupID, responseID)})
	if err != nil {
		sharedSessionWarning(err)
		return errClusterContinuation
	}
	// Missing ownership (expiry/restart/failed write) is also unsafe for a delta.
	if len(values) == 0 || values[0] == "" || values[0] != s.instanceID {
		return errClusterContinuation
	}
	return nil
}
