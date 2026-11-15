package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

func (s *AnnouncementService) Publications(ctx context.Context, ids []int64) (map[int64]AnnouncementPublication, error) {
	if s.deliveryRepo == nil {
		return nil, nil
	}
	return s.deliveryRepo.Publications(ctx, ids)
}

func (s *AnnouncementService) Publish(ctx context.Context, id, actorID int64, startsAt *time.Time) error {
	if s.deliveryRepo == nil {
		return ErrAnnouncementDeliveryDisabled
	}
	if !isJSONTimeInRange(startsAt) {
		return ErrAnnouncementInvalidSchedule
	}
	at := time.Now()
	if startsAt != nil && startsAt.After(at) {
		at = *startsAt
	}
	if err := s.deliveryRepo.Publish(ctx, id, actorID, at); err != nil {
		return err
	}
	if !at.After(time.Now()) {
		// Scheduling is durable even if the immediate attempt is interrupted.
		// Another dispatcher can safely pick it up after this request.
		_, _ = s.deliveryRepo.Dispatch(ctx, id)
	}
	return nil
}

func (s *AnnouncementService) Withdraw(ctx context.Context, id, actorID int64) error {
	if s.deliveryRepo == nil {
		return ErrAnnouncementDeliveryDisabled
	}
	return s.deliveryRepo.Withdraw(ctx, id, actorID)
}

func (s *AnnouncementService) RetryDelivery(ctx context.Context, id, actorID int64) error {
	if s.deliveryRepo == nil {
		return ErrAnnouncementDeliveryDisabled
	}
	if err := s.deliveryRepo.Retry(ctx, id, actorID); err != nil {
		return err
	}
	_, _ = s.deliveryRepo.Dispatch(ctx, id)
	return nil
}

func (s *AnnouncementService) DeliveryStats(ctx context.Context, id int64) (AnnouncementDeliveryStats, error) {
	if s.deliveryRepo == nil {
		return AnnouncementDeliveryStats{}, ErrAnnouncementDeliveryDisabled
	}
	return s.deliveryRepo.Stats(ctx, id)
}

func (s *AnnouncementService) ListDeliveries(ctx context.Context, id int64, params pagination.PaginationParams) ([]AnnouncementDelivery, int64, error) {
	if s.deliveryRepo == nil {
		return nil, 0, ErrAnnouncementDeliveryDisabled
	}
	return s.deliveryRepo.Deliveries(ctx, id, params)
}

func (s *AnnouncementService) ListForUser(ctx context.Context, userID int64, unreadOnly bool) ([]UserAnnouncement, error) {
	if s.deliveryRepo == nil {
		return s.listLegacyForUser(ctx, userID, unreadOnly)
	}
	items, _, _, err := s.listInbox(ctx, userID, true, unreadOnly, 0, 200)
	return items, err
}

func (s *AnnouncementService) HistoryForUser(ctx context.Context, userID int64, unreadOnly bool, params pagination.PaginationParams) ([]UserAnnouncement, int64, int64, error) {
	return s.listInbox(ctx, userID, false, unreadOnly, params.Offset(), params.Limit())
}

func (s *AnnouncementService) listInbox(ctx context.Context, userID int64, activeOnly, unreadOnly bool, offset, limit int) ([]UserAnnouncement, int64, int64, error) {
	legacy, err := s.listLegacyForUser(ctx, userID, false)
	if err != nil {
		return nil, 0, 0, err
	}
	legacyUnread := int64(0)
	matching := make([]UserAnnouncement, 0, len(legacy))
	for _, item := range legacy {
		if item.ReadAt == nil {
			legacyUnread++
		}
		if !unreadOnly || item.ReadAt == nil {
			matching = append(matching, item)
		}
	}
	total, unread := int64(len(matching)), legacyUnread
	if s.deliveryRepo != nil {
		// At most len(matching) legacy entries can precede any offset.
		// Read the last guaranteed-to-be-skipped row as a boundary to
		// remove the legacy entries that sort before it, then merge.
		managedOffset := max(0, offset-len(matching))
		managed, managedTotal, managedUnread, err := s.deliveryRepo.ListForUser(ctx, userID, time.Now(), activeOnly, unreadOnly, max(0, managedOffset-1), limit+len(matching)+1)
		if err != nil {
			return nil, 0, 0, err
		}
		total += managedTotal
		unread += managedUnread
		if managedOffset > 0 {
			if len(managed) == 0 {
				return []UserAnnouncement{}, total, unread, nil
			}
			boundary := managed[0]
			skip := 0
			for skip < len(matching) && userAnnouncementBefore(matching[skip], boundary) {
				skip++
			}
			matching = matching[skip:]
			managed = managed[1:]
			offset -= managedOffset + skip
		}
		matching = append(matching, managed...)
		sortUserAnnouncements(matching)
	}
	if offset >= len(matching) {
		return []UserAnnouncement{}, total, unread, nil
	}
	return matching[offset:min(offset+limit, len(matching))], total, unread, nil
}

func sortUserAnnouncements(items []UserAnnouncement) {
	sort.Slice(items, func(i, j int) bool {
		return userAnnouncementBefore(items[i], items[j])
	})
}

func userAnnouncementBefore(a, b UserAnnouncement) bool {
	if (a.ReadAt == nil) != (b.ReadAt == nil) {
		return a.ReadAt == nil
	}
	return a.Announcement.ID > b.Announcement.ID
}

func (s *AnnouncementService) GetForUser(ctx context.Context, userID, id int64) (*UserAnnouncement, error) {
	if s.deliveryRepo != nil {
		item, err := s.deliveryRepo.GetForUser(ctx, userID, id, time.Now())
		if err == nil || !errors.Is(err, ErrAnnouncementNotFound) {
			return item, err
		}
	}
	a, err := s.announcementRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !a.IsActiveAt(time.Now()) {
		return nil, ErrAnnouncementNotFound
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	subs, err := s.userSubRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list active subscriptions: %w", err)
	}
	groupIDs := make(map[int64]struct{}, len(subs))
	for _, sub := range subs {
		groupIDs[sub.GroupID] = struct{}{}
	}
	if !a.Targeting.Matches(user.Balance, groupIDs) {
		return nil, ErrAnnouncementNotFound
	}
	readMap, err := s.readRepo.GetReadMapByUser(ctx, userID, []int64{id})
	if err != nil {
		return nil, err
	}
	result := &UserAnnouncement{Announcement: *a}
	if read, ok := readMap[id]; ok {
		result.ReadAt = &read
	}
	return result, nil
}

func (s *AnnouncementService) MarkAllRead(ctx context.Context, userID int64) (int64, error) {
	if s.deliveryRepo == nil {
		return 0, ErrAnnouncementDeliveryDisabled
	}
	legacy, err := s.listLegacyForUser(ctx, userID, true)
	if err != nil {
		return 0, err
	}
	ids := make([]int64, 0, len(legacy))
	for _, item := range legacy {
		ids = append(ids, item.Announcement.ID)
	}
	return s.deliveryRepo.MarkAllRead(ctx, userID, ids, time.Now())
}

func AnnouncementActionURL(planID, groupID *int64) string {
	if planID != nil {
		return "/purchase?plan_id=" + strconv.FormatInt(*planID, 10)
	}
	if groupID != nil {
		return "/available-channels?group_id=" + strconv.FormatInt(*groupID, 10)
	}
	return ""
}

func (s *AnnouncementService) RecordClick(ctx context.Context, userID, id int64) (string, error) {
	if s.deliveryRepo == nil {
		return "", ErrAnnouncementNotFound
	}
	target, err := s.deliveryRepo.Click(ctx, userID, id, time.Now())
	if err != nil {
		return "", err
	}
	return AnnouncementActionURL(target.PlanID, target.GroupID), nil
}

func (s *AnnouncementService) MutedGroups(ctx context.Context, userID int64) ([]int64, error) {
	if s.deliveryRepo == nil {
		return nil, ErrAnnouncementDeliveryDisabled
	}
	return s.deliveryRepo.MutedGroups(ctx, userID)
}

func (s *AnnouncementService) SetGroupSubscription(ctx context.Context, userID, groupID int64, subscribed bool) error {
	if s.deliveryRepo == nil {
		return ErrAnnouncementDeliveryDisabled
	}
	if subscribed {
		// A user may clear their own preference even when a plan is delisted.
		return s.deliveryRepo.SetGroupMute(ctx, userID, groupID, false)
	}
	if standardGroup, err := s.deliveryRepo.MuteStandardGroupIfAvailable(ctx, userID, groupID); err != nil || standardGroup {
		return err
	}
	subs, err := s.userSubRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return err
	}
	for _, item := range subs {
		if item.GroupID == groupID {
			return s.deliveryRepo.SetGroupMute(ctx, userID, groupID, true)
		}
	}
	if s.paymentPlans != nil {
		plans, err := s.paymentPlans.ListPlansForSale(ctx, userID)
		if err != nil {
			return err
		}
		for _, plan := range plans {
			if plan.GroupID == groupID {
				return s.deliveryRepo.SetGroupMute(ctx, userID, groupID, true)
			}
		}
	}
	return ErrAnnouncementNotFound
}
