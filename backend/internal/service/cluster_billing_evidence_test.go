package service

import (
	"context"
	"errors"
	"github.com/Wei-Shaw/sub2api/internal/clusterops"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOpenAIBillingDependenciesPreserveClusterConfiguration(t *testing.T) {
	cfg := &config.Config{}
	cfg.Cluster.Enabled = true
	cfg.Cluster.HAEnabled = true
	s := &OpenAIGatewayService{cfg: cfg}
	require.Same(t, cfg, s.billingDeps().cfg)
	gateway := &GatewayService{cfg: cfg}
	require.Same(t, cfg, gateway.billingDeps().cfg)
}

type evidenceTestRepo struct {
	UsageBillingRepository
	stageErr  error
	staged    bool
	attempted bool
	attempt   string
}

func (r *evidenceTestRepo) StageClusterBillingEvidence(_ context.Context, _ *UsageBillingCommand, _ *UsageLog, attempt string) error {
	r.staged = true
	r.attempt = attempt
	return r.stageErr
}
func (r *evidenceTestRepo) Apply(context.Context, *UsageBillingCommand) (*UsageBillingApplyResult, error) {
	r.attempted = true
	return nil, errors.New("stop before settlement")
}
func TestHABillingRequiresEvidenceBeforeSettlement(t *testing.T) {
	cfg := &config.Config{}
	cfg.Cluster.HAEnabled = true
	p := &postUsageBillingParams{Cost: &CostBreakdown{ActualCost: 1}, User: &User{ID: 1}, APIKey: &APIKey{ID: 2}, Account: &Account{ID: 3}}
	for _, fail := range []bool{true, false} {
		repo := &evidenceTestRepo{}
		if fail {
			repo.stageErr = errors.New("evidence store unavailable")
		}
		_, err := applyUsageBilling(clusterops.WithAttempt(context.Background(), "attempt"), "request", nil, p, &billingDeps{cfg: cfg}, repo)
		require.Error(t, err)
		require.True(t, repo.staged)
		require.Equal(t, "attempt", repo.attempt)
		require.Equal(t, !fail, repo.attempted)
	}
}
