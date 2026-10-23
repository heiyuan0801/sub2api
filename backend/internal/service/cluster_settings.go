package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

type settingChangeSource interface {
	SubscribeSettingChanges(context.Context) (<-chan struct{}, func(), error)
}

func (s *SettingService) ClusterSettingsStatus() map[string]any {
	last := s.clusterSettingsLastSuccess.Load()
	version, _ := s.clusterSettingsVersion.Load().(string)
	return map[string]any{"last_success_unix": last, "failures": s.clusterSettingsFailures.Load(), "version": version}
}

func (s *SettingService) CheckClusterSettingsFresh(context.Context) error {
	last := s.clusterSettingsLastSuccess.Load()
	if last == 0 || time.Now().Unix()-last > 30 {
		return fmt.Errorf("cluster settings snapshot is stale")
	}
	return nil
}

// InitializeClusterSettings loads worker runtime policies before its listener
// becomes reachable. Failure must prevent the worker from joining the pool.
func (s *SettingService) InitializeClusterSettings(ctx context.Context) error {
	_, err := s.refreshClusterSettings(ctx, [sha256.Size]byte{})
	return err
}

// RunClusterSettingsRefresh keeps worker runtime settings in step with changes
// saved on the primary. Polling also recovers changes missed during disconnects.
// Only digests are retained; settings/credentials are never logged or published.
func (s *SettingService) RunClusterSettingsRefresh(ctx context.Context) {
	var previous [sha256.Size]byte
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	var events <-chan struct{}
	closeEvents := func() {}
	defer func() { closeEvents() }()
	for {
		if events == nil {
			if source, ok := s.settingRepo.(settingChangeSource); ok {
				var err error
				events, closeEvents, err = source.SubscribeSettingChanges(ctx)
				if err != nil {
					slog.Warn("cluster settings subscription failed; using periodic reconciliation")
				}
			}
		}
		refreshCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		next, err := s.refreshClusterSettings(refreshCtx, previous)
		cancel()
		if err == nil {
			previous = next
		} else {
			s.clusterSettingsFailures.Add(1)
			slog.Warn("cluster settings refresh failed", "failures", s.clusterSettingsFailures.Load())
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case _, ok := <-events:
			if !ok {
				closeEvents()
				events = nil
			}
		}
	}
}

func (s *SettingService) refreshClusterSettings(ctx context.Context, previous [sha256.Size]byte) ([sha256.Size]byte, error) {
	settings, err := s.settingRepo.GetAll(ctx)
	if err != nil {
		return previous, err
	}
	encoded, err := json.Marshal(settings)
	if err != nil {
		return previous, err
	}
	next := sha256.Sum256(encoded)
	if next == previous {
		s.clusterSettingsLastSuccess.Store(time.Now().Unix())
		return previous, nil
	}
	s.refreshCachedSettings(s.parseSettings(settings))
	webSearchEmulationSF.Forget(sfKeyWebSearchConfig)
	webSearchEmulationCache.Store(&cachedWebSearchEmulationConfig{})
	s.rebuildWebSearchManager(ctx)
	if err := ctx.Err(); err != nil {
		return previous, err
	}
	s.clusterSettingsVersion.Store(hex.EncodeToString(next[:8]))
	s.clusterSettingsLastSuccess.Store(time.Now().Unix())
	return next, nil
}
