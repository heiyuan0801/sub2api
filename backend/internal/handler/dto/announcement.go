package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type Announcement struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Status     string `json:"status"`
	NotifyMode string `json:"notify_mode"`

	Targeting      service.AnnouncementTargeting `json:"targeting"`
	Kind           string                        `json:"kind,omitempty"`
	PlanID         *int64                        `json:"plan_id,omitempty"`
	GroupID        *int64                        `json:"group_id,omitempty"`
	GroupName      string                        `json:"group_name,omitempty"`
	Audience       *service.AnnouncementAudience `json:"audience,omitempty"`
	DeliveryStatus string                        `json:"delivery_status,omitempty"`

	StartsAt *time.Time `json:"starts_at,omitempty"`
	EndsAt   *time.Time `json:"ends_at,omitempty"`

	CreatedBy *int64 `json:"created_by,omitempty"`
	UpdatedBy *int64 `json:"updated_by,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserAnnouncement struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	NotifyMode string `json:"notify_mode"`
	Kind       string `json:"kind,omitempty"`
	PlanID     *int64 `json:"plan_id,omitempty"`
	GroupID    *int64 `json:"group_id,omitempty"`
	GroupName  string `json:"group_name,omitempty"`
	ActionURL  string `json:"action_url,omitempty"`

	StartsAt *time.Time `json:"starts_at,omitempty"`
	EndsAt   *time.Time `json:"ends_at,omitempty"`

	ReadAt *time.Time `json:"read_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func AnnouncementFromService(a *service.Announcement) *Announcement {
	if a == nil {
		return nil
	}
	return &Announcement{
		ID:         a.ID,
		Title:      a.Title,
		Content:    a.Content,
		Status:     a.Status,
		NotifyMode: a.NotifyMode,
		Targeting:  a.Targeting,
		StartsAt:   a.StartsAt,
		EndsAt:     a.EndsAt,
		CreatedBy:  a.CreatedBy,
		UpdatedBy:  a.UpdatedBy,
		CreatedAt:  a.CreatedAt,
		UpdatedAt:  a.UpdatedAt,
	}
}

func (a *Announcement) ApplyPublication(p service.AnnouncementPublication) {
	a.Kind = p.Kind
	a.PlanID = p.PlanID
	a.GroupID = p.GroupID
	a.GroupName = p.GroupName
	a.Audience = &p.Audience
	a.DeliveryStatus = p.DeliveryStatus
	a.Targeting = service.AnnouncementTargeting{}
}

func UserAnnouncementFromService(a *service.UserAnnouncement) *UserAnnouncement {
	if a == nil {
		return nil
	}
	var actionGroupID *int64
	if a.GroupActionAvailable {
		actionGroupID = a.GroupID
	}
	return &UserAnnouncement{
		ID:         a.Announcement.ID,
		Title:      a.Announcement.Title,
		Content:    a.Announcement.Content,
		NotifyMode: a.Announcement.NotifyMode,
		Kind:       a.Kind,
		PlanID:     a.PlanID,
		GroupID:    a.GroupID,
		GroupName:  a.GroupName,
		ActionURL:  service.AnnouncementActionURL(a.PlanID, actionGroupID),
		StartsAt:   a.Announcement.StartsAt,
		EndsAt:     a.Announcement.EndsAt,
		ReadAt:     a.ReadAt,
		CreatedAt:  a.Announcement.CreatedAt,
		UpdatedAt:  a.Announcement.UpdatedAt,
	}
}
