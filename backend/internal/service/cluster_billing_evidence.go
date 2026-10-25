package service

import "context"

// ClusterBillingEvidenceWriter persists only already-observed usage before the
// debit transaction. Missing provider outcomes are never synthesized or replayed.
type ClusterBillingEvidenceWriter interface {
	StageClusterBillingEvidence(context.Context, *UsageBillingCommand, *UsageLog, string) error
}
