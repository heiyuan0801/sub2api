package service

import (
	"context"
	"log/slog"
	"time"
)

type subscriptionRevisionReader interface {
	SubscriptionCacheRevision(context.Context) (string, error)
}

func (s *SubscriptionService) startSubscriptionInvalidation(ctx context.Context) {
	s.subInvalidationOnce.Do(func() {
		ctx, s.subInvalidationCancel = context.WithCancel(ctx)
		s.subInvalidationWG.Add(1)
		go func() {
			defer s.subInvalidationWG.Done()
			reader, _ := s.billingCacheService.cache.(subscriptionRevisionReader)
			var last string
			initialized, subscribed := false, false
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for {
				if !subscribed {
					err := s.billingCacheService.SubscribeSubscriptionCacheInvalidation(ctx, s.invalidateSubCacheKeySync)
					subscribed = err == nil
					if err != nil {
						slog.Warn("subscription invalidation subscription unavailable; retrying")
					}
				}
				if reader != nil {
					readCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
					revision, err := reader.SubscriptionCacheRevision(readCtx)
					cancel()
					if err == nil {
						if !initialized || revision != last {
							s.clearSubscriptionL1()
						}
						last, initialized = revision, true
					} else {
						slog.Warn("subscription cache revision check failed")
					}
				}
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}()
	})
}

func (s *SubscriptionService) clearSubscriptionL1() {
	s.subCacheWriteMu.Lock()
	defer s.subCacheWriteMu.Unlock()
	s.subCacheEpoch.Add(1)
	if s.subCacheL1 != nil {
		s.subCacheL1.Clear()
	}
}
