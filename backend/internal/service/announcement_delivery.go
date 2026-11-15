package service

import (
	"context"
	"slices"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	AnnouncementDeliveryDraft     = "draft"
	AnnouncementDeliveryQueued    = "queued"
	AnnouncementDeliveryPublished = "published"
	AnnouncementDeliveryFailed    = "failed"
	AnnouncementDeliveryWithdrawn = "withdrawn"
)

var (
	ErrAnnouncementInvalidPublication = infraerrors.BadRequest("ANNOUNCEMENT_PUBLICATION_INVALID", "invalid announcement type, plan, group or audience")
	ErrAnnouncementPublicationState   = infraerrors.Conflict("ANNOUNCEMENT_PUBLICATION_STATE", "this announcement cannot be changed in its current publication state")
	ErrAnnouncementDeliveryDisabled   = infraerrors.ServiceUnavailable("ANNOUNCEMENT_DELIVERY_UNAVAILABLE", "announcement delivery is unavailable")
)

type AnnouncementAudience struct {
	Mode     string  `json:"mode"`
	GroupIDs []int64 `json:"group_ids,omitempty"`
	UserIDs  []int64 `json:"user_ids,omitempty"`
}

type AnnouncementPublication struct {
	Kind           string               `json:"kind"`
	PlanID         *int64               `json:"plan_id,omitempty"`
	GroupID        *int64               `json:"group_id,omitempty"`
	GroupName      string               `json:"group_name,omitempty"`
	Audience       AnnouncementAudience `json:"audience"`
	DeliveryStatus string               `json:"delivery_status"`
}

type AnnouncementActionTarget struct {
	PlanID  *int64
	GroupID *int64
}

type AnnouncementDeliveryStats struct {
	TargetCount    int64      `json:"target_count"`
	ReadCount      int64      `json:"read_count"`
	UnreadCount    int64      `json:"unread_count"`
	ClickCount     int64      `json:"click_count"`
	DeliveryStatus string     `json:"delivery_status"`
	Attempts       int        `json:"attempts"`
	LastError      string     `json:"last_error,omitempty"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
	NextAttemptAt  *time.Time `json:"next_attempt_at,omitempty"`
}

type AnnouncementDelivery struct {
	UserID      int64      `json:"user_id"`
	Email       string     `json:"email"`
	Username    string     `json:"username"`
	DeliveredAt time.Time  `json:"delivered_at"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
	ClickCount  int64      `json:"click_count"`
}

type AnnouncementDeliveryRepository interface {
	Create(ctx context.Context, a *Announcement, publication AnnouncementPublication) error
	Update(ctx context.Context, a *Announcement, publication AnnouncementPublication) error
	Publications(ctx context.Context, ids []int64) (map[int64]AnnouncementPublication, error)
	Publish(ctx context.Context, id, actorID int64, at time.Time) error
	Withdraw(ctx context.Context, id, actorID int64) error
	Retry(ctx context.Context, id, actorID int64) error
	Dispatch(ctx context.Context, id int64) (bool, error)
	Stats(ctx context.Context, id int64) (AnnouncementDeliveryStats, error)
	Deliveries(ctx context.Context, id int64, params pagination.PaginationParams) ([]AnnouncementDelivery, int64, error)
	ListForUser(ctx context.Context, userID int64, now time.Time, activeOnly, unreadOnly bool, offset, limit int) ([]UserAnnouncement, int64, int64, error)
	GetForUser(ctx context.Context, userID, id int64, now time.Time) (*UserAnnouncement, error)
	MarkAllRead(ctx context.Context, userID int64, legacyIDs []int64, now time.Time) (int64, error)
	Click(ctx context.Context, userID, id int64, now time.Time) (AnnouncementActionTarget, error)
	MutedGroups(ctx context.Context, userID int64) ([]int64, error)
	SetGroupMute(ctx context.Context, userID, groupID int64, mute bool) error
	// MuteStandardGroupIfAvailable handles standard group targets, including denied ones.
	MuteStandardGroupIfAvailable(ctx context.Context, userID, groupID int64) (handled bool, err error)
}

func normalizeAnnouncementPublication(value AnnouncementPublication) (AnnouncementPublication, error) {
	switch value.Kind {
	case "new_group", "group_unlisted":
		if value.GroupID == nil {
			return value, ErrAnnouncementInvalidPublication
		}
	case "price_increase", "price_decrease":
	case "new_plan", "plan_unlisted", "system", "activity":
		if value.GroupID != nil {
			return value, ErrAnnouncementInvalidPublication
		}
	default:
		return value, ErrAnnouncementInvalidPublication
	}
	if value.PlanID != nil && *value.PlanID <= 0 {
		return value, ErrAnnouncementInvalidPublication
	}
	if value.GroupID != nil && (*value.GroupID <= 0 || value.PlanID != nil) {
		return value, ErrAnnouncementInvalidPublication
	}
	switch value.Audience.Mode {
	case "all":
		if len(value.Audience.GroupIDs) != 0 || len(value.Audience.UserIDs) != 0 {
			return value, ErrAnnouncementInvalidPublication
		}
	case "groups":
		if len(value.Audience.GroupIDs) == 0 || len(value.Audience.UserIDs) != 0 {
			return value, ErrAnnouncementInvalidPublication
		}
	case "users":
		if len(value.Audience.UserIDs) == 0 || len(value.Audience.GroupIDs) != 0 {
			return value, ErrAnnouncementInvalidPublication
		}
	default:
		return value, ErrAnnouncementInvalidPublication
	}
	normalized := make([][]int64, 0, 2)
	for _, ids := range [][]int64{value.Audience.GroupIDs, value.Audience.UserIDs} {
		if len(ids) > 5000 {
			return value, ErrAnnouncementInvalidPublication
		}
		for _, id := range ids {
			if id <= 0 {
				return value, ErrAnnouncementInvalidPublication
			}
		}
		ids = slices.Clone(ids)
		slices.Sort(ids)
		normalized = append(normalized, slices.Compact(ids))
	}
	value.Audience.GroupIDs, value.Audience.UserIDs = normalized[0], normalized[1]
	return value, nil
}

type AnnouncementDeliveryWorker struct {
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once
}

func ProvideAnnouncementDeliveryWorker(repo AnnouncementDeliveryRepository) *AnnouncementDeliveryWorker {
	ctx, cancel := context.WithCancel(context.Background())
	worker := &AnnouncementDeliveryWorker{cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(worker.done)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			for i := 0; i < 10 && ctx.Err() == nil; i++ {
				attempt, cancel := context.WithTimeout(ctx, 45*time.Second)
				processed, err := repo.Dispatch(attempt, 0)
				cancel()
				if err != nil {
					logger.LegacyPrintf("service.announcement", "dispatch failed: %v", err)
				}
				if err != nil || !processed {
					break
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return worker
}

func (w *AnnouncementDeliveryWorker) Stop() {
	w.once.Do(w.cancel)
	<-w.done
}
