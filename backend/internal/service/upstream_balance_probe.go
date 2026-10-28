package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
)

// UpstreamBalanceSnapshot is a manually queried, key-scoped monetary snapshot.
// A missing value is never interpreted as a zero wallet balance.
type UpstreamBalanceSnapshot struct {
	Status    string    `json:"status"`
	Kind      string    `json:"kind,omitempty"`
	Amount    *float64  `json:"amount,omitempty"`
	Unit      string    `json:"unit,omitempty"`
	QueriedAt time.Time `json:"queried_at"`
}

func parseUpstreamBalance(body []byte) (*UpstreamBalanceSnapshot, error) {
	var data struct {
		Mode      string   `json:"mode"`
		Valid     bool     `json:"isValid"`
		Unit      string   `json:"unit"`
		Balance   *float64 `json:"balance"`
		Remaining *float64 `json:"remaining"`
		Quota     *struct {
			Remaining *float64 `json:"remaining"`
			Unit      string   `json:"unit"`
		} `json:"quota"`
		Subscription json.RawMessage `json:"subscription"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	if !data.Valid {
		return nil, fmt.Errorf("invalid upstream key")
	}
	result := &UpstreamBalanceSnapshot{Status: "ok", Unit: data.Unit}
	switch {
	case data.Mode == "unrestricted" && data.Balance != nil:
		result.Kind, result.Amount = "wallet", data.Balance
	case data.Mode == "quota_limited" && data.Quota != nil:
		result.Kind, result.Amount, result.Unit = "quota", data.Quota.Remaining, data.Quota.Unit
	case data.Mode == "unrestricted" && len(data.Subscription) > 0 && string(data.Subscription) != "null":
		result.Kind, result.Amount = "subscription", data.Remaining
	default:
		return nil, fmt.Errorf("no monetary balance available")
	}
	if result.Amount == nil || result.Unit != "USD" || math.IsNaN(*result.Amount) || math.IsInf(*result.Amount, 0) {
		return nil, fmt.Errorf("invalid monetary balance")
	}
	return result, nil
}

// ProbeAccountBalance uses the same bounded transport and identity-CAS as rate
// probes. It is manual so visiting a table does not fan out upstream usage scans.
func (s *UpstreamBillingProbeService) ProbeAccountBalance(ctx context.Context, id int64) (*UpstreamBillingProbeSnapshot, error) {
	if s == nil || s.accountRepo == nil || s.accountTestService == nil || s.accountTestService.httpUpstream == nil {
		return nil, ErrUpstreamBillingProbeUnavailable
	}
	value, err, _ := s.probeGroup.Do("balance:"+strconv.FormatInt(id, 10), func() (any, error) {
		select {
		case s.probeSlots <- struct{}{}:
			defer func() { <-s.probeSlots }()
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		account, err := s.accountRepo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if !isUpstreamBillingProbeAccount(account) {
			return nil, ErrUpstreamBillingProbeAccountInvalid
		}
		snapshot := decodeUpstreamBillingProbeSnapshot(account.Extra)
		if snapshot == nil {
			snapshot = &UpstreamBillingProbeSnapshot{Status: "unqueried"}
		}
		balance := &UpstreamBalanceSnapshot{Status: "failed", QueriedAt: s.currentTime().UTC()}
		base := account.GetCredential("base_url")
		if account.RoutesProtocolByInbound() && account.IsAdaptiveAPIProtocol() {
			base = account.GetCNProtocolBaseURL(APIProtocolChatCompletions)
		}
		if upstreamBillingProbeTargetIsOfficialAPI(base) {
			balance.Status = "unsupported"
		} else {
			normalized, validationErr := s.accountTestService.validateUpstreamBaseURL(base)
			if validationErr == nil && account.GetCredential("api_key") != "" {
				proxyURL := ""
				if account.ProxyID != nil {
					if account.Proxy == nil || account.Proxy.ID != *account.ProxyID {
						return nil, ErrUpstreamBillingProbeIdentityChanged
					}
					proxyURL = account.Proxy.URL()
				}
				probeCtx, cancel := context.WithTimeout(ctx, upstreamBillingProbeRequestTimeout)
				defer cancel()
				profile := HTTPUpstreamProfileDefault
				if account.Platform == PlatformOpenAI {
					profile = HTTPUpstreamProfileOpenAI
				}
				probeCtx = WithHTTPUpstreamRedirectsDisabled(WithHTTPUpstreamProfile(probeCtx, profile))
				req, buildErr := http.NewRequestWithContext(probeCtx, http.MethodGet, buildOpenAIEndpointURL(normalized, "/v1/usage")+"?days=1", nil)
				if buildErr == nil {
					req.Header.Set("Accept", "application/json")
					req.Header.Set("Authorization", "Bearer "+account.GetCredential("api_key"))
					account.ApplyHeaderOverrides(req.Header)
					var tlsProfile *tlsfingerprint.Profile
					if s.accountTestService.tlsFPProfileService != nil {
						tlsProfile = s.accountTestService.tlsFPProfileService.ResolveTLSProfile(account)
					}
					resp, requestErr := s.accountTestService.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, tlsProfile)
					if resp != nil && resp.Body != nil {
						defer func() { _ = resp.Body.Close() }()
						if requestErr == nil {
							if resp.StatusCode == 404 || resp.StatusCode == 405 {
								balance.Status = "unsupported"
							}
							if resp.StatusCode == http.StatusOK {
								body, readErr := io.ReadAll(io.LimitReader(resp.Body, upstreamBillingProbeMaxBodyBytes+1))
								if readErr == nil && len(body) <= upstreamBillingProbeMaxBodyBytes {
									if parsed, parseErr := parseUpstreamBalance(body); parseErr == nil {
										parsed.QueriedAt = balance.QueriedAt
										balance = parsed
									}
								}
							}
						}
					}
				}
			}
		}
		snapshot.Balance = balance
		if err := s.updateSnapshot(ctx, account, snapshot, nil); err != nil {
			return nil, err
		}
		return snapshot, nil
	})
	if err != nil {
		return nil, err
	}
	snapshot, ok := value.(*UpstreamBillingProbeSnapshot)
	if !ok {
		return nil, ErrUpstreamBillingProbeUnavailable
	}
	return snapshot, nil
}
