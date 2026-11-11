package service

import (
	"context"
	"crypto/rand"
	"log/slog"
	"math/big"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/google/uuid"
)

const (
	RewardCheckIn = "check_in"
	RewardWheel   = "wheel"
	// Amounts use hundredths of the site's balance unit.
	maxActivityCents = 100_000_000
)

var (
	ErrRewardActivityDisabled = infraerrors.Forbidden("REWARD_ACTIVITY_DISABLED", "This activity is closed")
	ErrRewardAlreadyCheckedIn = infraerrors.Conflict("REWARD_ALREADY_CHECKED_IN", "You have already checked in today")
	ErrRewardSpendRequired    = infraerrors.Conflict("REWARD_SPEND_REQUIRED", "Today's eligible spending is below the required amount")
	ErrRewardDrawLimit        = infraerrors.Conflict("REWARD_DRAW_LIMIT", "Today's draw limit has been reached")
	ErrRewardRequestConflict  = infraerrors.Conflict("REWARD_REQUEST_CONFLICT", "This request ID belongs to a different activity")
)

type RewardCheckInConfig struct {
	Enabled        bool  `json:"enabled"`
	ThresholdCents int64 `json:"threshold_cents"`
	MinRewardCents int64 `json:"min_reward_cents"`
	MaxRewardCents int64 `json:"max_reward_cents"`
}

type RewardPrize struct {
	Name        string `json:"name"`
	RewardCents int64  `json:"reward_cents"`
	Weight      int64  `json:"weight"`
}

type RewardWheelConfig struct {
	Enabled        bool          `json:"enabled"`
	ThresholdCents int64         `json:"threshold_cents"`
	DailyLimit     int           `json:"daily_limit"`
	Prizes         []RewardPrize `json:"prizes"`
}

type RewardActivityConfig struct {
	CheckIn RewardCheckInConfig `json:"check_in"`
	Wheel   RewardWheelConfig   `json:"wheel"`
}

func DefaultRewardActivityConfig() RewardActivityConfig {
	return RewardActivityConfig{
		CheckIn: RewardCheckInConfig{ThresholdCents: 100, MinRewardCents: 10, MaxRewardCents: 100},
		Wheel: RewardWheelConfig{ThresholdCents: 100, DailyLimit: 3, Prizes: []RewardPrize{
			{Weight: 50},
			{RewardCents: 10, Weight: 30},
			{RewardCents: 50, Weight: 15},
			{RewardCents: 100, Weight: 5},
		}},
	}
}

func (c RewardActivityConfig) Validate() error {
	if c.CheckIn.ThresholdCents < 1 || c.CheckIn.ThresholdCents > maxActivityCents ||
		c.Wheel.ThresholdCents < 1 || c.Wheel.ThresholdCents > maxActivityCents {
		return infraerrors.BadRequest("INVALID_REWARD_THRESHOLD", "Each spending threshold must be between 0.01 and 1,000,000")
	}
	if c.CheckIn.MinRewardCents < 1 || c.CheckIn.MaxRewardCents < c.CheckIn.MinRewardCents ||
		c.CheckIn.MaxRewardCents > maxActivityCents {
		return infraerrors.BadRequest("INVALID_CHECK_IN_REWARD", "Set a valid positive check-in reward range")
	}
	if c.Wheel.DailyLimit < 1 || c.Wheel.DailyLimit > 100 || len(c.Wheel.Prizes) < 2 || len(c.Wheel.Prizes) > 12 {
		return infraerrors.BadRequest("INVALID_WHEEL_CONFIG", "Set 1–100 daily draws and 2–12 prize slots")
	}
	var hasReward bool
	for _, prize := range c.Wheel.Prizes {
		if prize.RewardCents < 0 || prize.RewardCents > maxActivityCents ||
			prize.Weight < 1 || prize.Weight > 1_000_000 || utf8.RuneCountInString(prize.Name) > 40 {
			return infraerrors.BadRequest("INVALID_WHEEL_PRIZE", "Check the prize amount, weight, and name length")
		}
		hasReward = hasReward || prize.RewardCents > 0
	}
	if c.Wheel.Enabled && !hasReward {
		return infraerrors.BadRequest("INVALID_WHEEL_PRIZE", "At least one prize must award a positive amount")
	}
	return nil
}

type RewardActivityClaim struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Day         string    `json:"day"`
	Kind        string    `json:"kind"`
	RewardCents int64     `json:"reward_cents"`
	PrizeIndex  int       `json:"prize_index"`
	PrizeName   string    `json:"prize_name"`
	CreatedAt   time.Time `json:"created_at"`
}

type RewardActivityStatus struct {
	Config             RewardActivityConfig  `json:"config"`
	Day                string                `json:"day"`
	SpentCents         int64                 `json:"spent_cents"`
	CheckedIn          bool                  `json:"checked_in"`
	DrawsUsed          int                   `json:"draws_used"`
	DrawSpendUsedCents int64                 `json:"draw_spend_used_cents"`
	DrawsAvailable     int                   `json:"draws_available"`
	History            []RewardActivityClaim `json:"history"`
}

func (s *RewardActivityStatus) SetDrawsAvailable() {
	s.DrawsAvailable = 0
	if !s.Config.Wheel.Enabled || s.DrawsUsed >= s.Config.Wheel.DailyLimit || s.SpentCents <= s.DrawSpendUsedCents {
		return
	}
	remainingSpend := s.SpentCents - s.DrawSpendUsedCents
	remainingDraws := int64(s.Config.Wheel.DailyLimit - s.DrawsUsed)
	s.DrawsAvailable = int(min(remainingDraws, remainingSpend/s.Config.Wheel.ThresholdCents))
}

func (s RewardActivityStatus) PickReward(kind string) (RewardActivityClaim, int64, error) {
	claim := RewardActivityClaim{Kind: kind, Day: s.Day, PrizeIndex: -1}
	if kind == RewardCheckIn {
		config := s.Config.CheckIn
		switch {
		case !config.Enabled:
			return claim, 0, ErrRewardActivityDisabled
		case s.CheckedIn:
			return claim, 0, ErrRewardAlreadyCheckedIn
		case s.SpentCents < config.ThresholdCents:
			return claim, 0, ErrRewardSpendRequired
		}
		draw, err := rand.Int(rand.Reader, big.NewInt(config.MaxRewardCents-config.MinRewardCents+1))
		if err != nil {
			return claim, 0, err
		}
		claim.RewardCents = config.MinRewardCents + draw.Int64()
		return claim, 0, nil
	}

	config := s.Config.Wheel
	switch {
	case !config.Enabled:
		return claim, 0, ErrRewardActivityDisabled
	case s.DrawsUsed >= config.DailyLimit:
		return claim, 0, ErrRewardDrawLimit
	case s.DrawsAvailable == 0:
		return claim, 0, ErrRewardSpendRequired
	}
	var totalWeight int64
	for _, prize := range config.Prizes {
		totalWeight += prize.Weight
	}
	draw, err := rand.Int(rand.Reader, big.NewInt(totalWeight))
	if err != nil {
		return claim, 0, err
	}
	ticket := draw.Int64()
	for index, prize := range config.Prizes {
		if ticket < prize.Weight {
			claim.PrizeIndex, claim.PrizeName, claim.RewardCents = index, prize.Name, prize.RewardCents
			break
		}
		ticket -= prize.Weight
	}
	return claim, config.ThresholdCents, nil
}

type RewardActivityRepository interface {
	GetConfig(context.Context) (RewardActivityConfig, error)
	SaveConfig(context.Context, RewardActivityConfig) error
	Status(context.Context, int64) (*RewardActivityStatus, error)
	Claim(context.Context, int64, string, uuid.UUID) (*RewardActivityClaim, error)
	RecentClaims(context.Context) ([]RewardActivityClaim, error)
}

type RewardActivityService struct {
	repo         RewardActivityRepository
	billingCache *BillingCacheService
	authCache    APIKeyAuthCacheInvalidator
}

func NewRewardActivityService(repo RewardActivityRepository, billingCache *BillingCacheService, authCache APIKeyAuthCacheInvalidator) *RewardActivityService {
	return &RewardActivityService{repo: repo, billingCache: billingCache, authCache: authCache}
}

func (s *RewardActivityService) GetConfig(ctx context.Context) (RewardActivityConfig, error) {
	return s.repo.GetConfig(ctx)
}

func (s *RewardActivityService) SaveConfig(ctx context.Context, config RewardActivityConfig) error {
	for i := range config.Wheel.Prizes {
		config.Wheel.Prizes[i].Name = strings.TrimSpace(config.Wheel.Prizes[i].Name)
	}
	if err := config.Validate(); err != nil {
		return err
	}
	return s.repo.SaveConfig(ctx, config)
}

func (s *RewardActivityService) Status(ctx context.Context, userID int64) (*RewardActivityStatus, error) {
	return s.repo.Status(ctx, userID)
}

func (s *RewardActivityService) RecentClaims(ctx context.Context) ([]RewardActivityClaim, error) {
	return s.repo.RecentClaims(ctx)
}

func (s *RewardActivityService) Claim(ctx context.Context, userID int64, kind, requestID string) (*RewardActivityClaim, error) {
	id, err := uuid.Parse(requestID)
	if err != nil || id == uuid.Nil || (kind != RewardCheckIn && kind != RewardWheel) {
		return nil, infraerrors.BadRequest("INVALID_REWARD_REQUEST", "A valid activity and request ID are required")
	}
	claim, err := s.repo.Claim(ctx, userID, kind, id)
	if err != nil || claim.RewardCents == 0 {
		return claim, err
	}

	// Commit precedes invalidation. Replaying the request also repairs stale cache entries.
	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if s.billingCache != nil {
		if err := s.billingCache.InvalidateUserBalance(cacheCtx, userID); err != nil {
			slog.Warn("reward balance cache invalidation failed", "user_id", userID, "error", err)
		}
	}
	if s.authCache != nil {
		s.authCache.InvalidateAuthCacheByUserID(cacheCtx, userID)
	}
	return claim, nil
}
