package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	"unicode"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	MaxRegistrationAttributionBytes = 2048
	RegistrationSourceUnknown       = "unknown"
	RegistrationDimensionSelected   = "selected"
	RegistrationDimensionUTM        = "utm"
)

// RegistrationAttribution describes how a user reached the registration page.
// User.SignupSource remains the authentication provider used for signup grants.
type RegistrationAttribution struct {
	SelectedSource string `json:"selected_source,omitempty"`
	SourceNote     string `json:"source_note,omitempty"`
	UTMSource      string `json:"utm_source,omitempty"`
	UTMMedium      string `json:"utm_medium,omitempty"`
	UTMCampaign    string `json:"utm_campaign,omitempty"`
	UTMContent     string `json:"utm_content,omitempty"`
	UTMTerm        string `json:"utm_term,omitempty"`
}

func IsSelectedRegistrationSource(source string) bool {
	switch source {
	case "google", "bing", "friend", "telegram", "discord", "github", "youtube", "reddit", "x", "blog_forum", "other":
		return true
	default:
		return false
	}
}

// ParseRegistrationAttribution ignores absent or malformed optional metadata so
// that customers using an older client or a malformed campaign URL can sign up.
func ParseRegistrationAttribution(raw []byte) RegistrationAttribution {
	var attribution RegistrationAttribution
	if len(raw) == 0 || len(raw) > MaxRegistrationAttributionBytes || json.Unmarshal(raw, &attribution) != nil {
		return RegistrationAttribution{}
	}
	attribution.SelectedSource = strings.ToLower(strings.TrimSpace(attribution.SelectedSource))
	if !IsSelectedRegistrationSource(attribution.SelectedSource) {
		attribution.SelectedSource = ""
	}
	if attribution.SelectedSource == "other" {
		attribution.SourceNote = cleanRegistrationAttributionValue(attribution.SourceNote, 240)
	} else {
		attribution.SourceNote = ""
	}
	attribution.UTMSource = strings.ToLower(cleanRegistrationAttributionValue(attribution.UTMSource, 120))
	if attribution.UTMSource == RegistrationSourceUnknown {
		attribution.UTMSource = ""
	}
	attribution.UTMMedium = cleanRegistrationAttributionValue(attribution.UTMMedium, 120)
	attribution.UTMCampaign = cleanRegistrationAttributionValue(attribution.UTMCampaign, 120)
	attribution.UTMContent = cleanRegistrationAttributionValue(attribution.UTMContent, 120)
	attribution.UTMTerm = cleanRegistrationAttributionValue(attribution.UTMTerm, 120)
	return attribution
}

func cleanRegistrationAttributionValue(value string, maxRunes int) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, value)
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > maxRunes {
		runes = runes[:maxRunes]
	}
	return strings.TrimSpace(string(runes))
}

type registrationAttributionKey struct{}

// WithRegistrationAttribution is applied only by registration entry points.
func WithRegistrationAttribution(ctx context.Context, attribution RegistrationAttribution) context.Context {
	if (attribution == RegistrationAttribution{}) {
		return ctx
	}
	return context.WithValue(ctx, registrationAttributionKey{}, attribution)
}

func RegistrationAttributionFromContext(ctx context.Context) (RegistrationAttribution, bool) {
	attribution, ok := ctx.Value(registrationAttributionKey{}).(RegistrationAttribution)
	return attribution, ok
}

type RegistrationSourceFilter struct {
	Start      time.Time
	End        time.Time // Exclusive
	TodayStart time.Time
	TodayEnd   time.Time // Exclusive
	Timezone   string
	Dimension  string
	Source     string
}

type RegistrationSourceChannel struct {
	Source     string  `json:"source"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

type RegistrationSourceDaily struct {
	Date   string `json:"date"`
	Source string `json:"source"`
	Count  int64  `json:"count"`
}

type RegistrationSourceSummary struct {
	TotalRegistrations int64                       `json:"total_registrations"`
	TodayRegistrations int64                       `json:"today_registrations"`
	RangeRegistrations int64                       `json:"range_registrations"`
	UnreportedCount    int64                       `json:"unreported_count"`
	Channels           []RegistrationSourceChannel `json:"channels"`
	Daily              []RegistrationSourceDaily   `json:"daily"`
}

type RegistrationSourceUser struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	CreatedAt    time.Time `json:"created_at"`
	SignupSource string    `json:"signup_source"`
	RegistrationAttribution
	InviterID *int64 `json:"inviter_id,omitempty"`
	PromoCode string `json:"promo_code,omitempty"`
}

type registrationAttributionRepository interface {
	RegistrationSourceSummary(context.Context, RegistrationSourceFilter) (*RegistrationSourceSummary, error)
	ListRegistrationSourceUsers(context.Context, RegistrationSourceFilter, pagination.PaginationParams) ([]RegistrationSourceUser, *pagination.PaginationResult, error)
	ExportRegistrationSourceUsers(context.Context, RegistrationSourceFilter, func(RegistrationSourceUser) error) error
}

func (s *UserService) RegistrationSourceSummary(ctx context.Context, filter RegistrationSourceFilter) (*RegistrationSourceSummary, error) {
	repo, ok := s.userRepo.(registrationAttributionRepository)
	if !ok {
		return nil, ErrServiceUnavailable
	}
	return repo.RegistrationSourceSummary(ctx, filter)
}

func (s *UserService) ListRegistrationSourceUsers(ctx context.Context, filter RegistrationSourceFilter, params pagination.PaginationParams) ([]RegistrationSourceUser, *pagination.PaginationResult, error) {
	repo, ok := s.userRepo.(registrationAttributionRepository)
	if !ok {
		return nil, nil, ErrServiceUnavailable
	}
	return repo.ListRegistrationSourceUsers(ctx, filter, params)
}

func (s *UserService) ExportRegistrationSourceUsers(ctx context.Context, filter RegistrationSourceFilter, write func(RegistrationSourceUser) error) error {
	repo, ok := s.userRepo.(registrationAttributionRepository)
	if !ok {
		return ErrServiceUnavailable
	}
	return repo.ExportRegistrationSourceUsers(ctx, filter, write)
}
