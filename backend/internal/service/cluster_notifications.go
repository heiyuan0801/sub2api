package service

import (
	"context"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/clusterops"
	"strconv"
	"strings"
	"time"
)

// DeliverClusterEvent honors the existing ops recipients, severity, silencing,
// per-hour limiter and notification template delivery deduplication.
func (s *OpsAlertEvaluatorService) DeliverClusterEvent(ctx context.Context, e clusterops.Event) error {
	if s == nil || s.opsService == nil || s.emailService == nil {
		return nil
	}
	cfg, err := s.opsService.GetEmailNotificationConfig(ctx)
	if err != nil {
		return err
	}
	if cfg == nil || !cfg.Alert.Enabled || len(cfg.Alert.Recipients) == 0 {
		return nil
	}
	severity := "P2"
	switch e.Severity {
	case "critical":
		severity = "P0"
	case "warning":
		severity = "P1"
	}
	if !shouldSendOpsAlertEmailByMinSeverity(cfg.Alert.MinSeverity, severity) {
		return nil
	}
	runtime, err := s.opsService.GetOpsAlertRuntimeSettings(ctx)
	if err != nil {
		return err
	}
	rule := &OpsAlertRule{Name: "Cluster " + e.Kind, Severity: severity, MetricType: "cluster", Description: e.Message}
	event := &OpsAlertEvent{ID: e.ID, Status: OpsAlertStatusFiring, FiredAt: e.CreatedAt, Description: e.Message}
	if e.Kind == "alert_resolved" {
		event.Status = OpsAlertStatusResolved
	}
	if runtime != nil && runtime.Silencing.Enabled && isOpsAlertSilenced(time.Now().UTC(), rule, event, runtime.Silencing) {
		return nil
	}
	notifier := s.emailService.notificationEmailService
	if notifier == nil {
		return fmt.Errorf("notification templates unavailable")
	}
	s.emailLimiter.SetLimit(cfg.Alert.RateLimitPerHour)
	for _, to := range cfg.Alert.Recipients {
		if strings.TrimSpace(to) == "" {
			continue
		}
		if !s.emailLimiter.Allow(time.Now().UTC()) {
			return fmt.Errorf("notification rate limited")
		}
		if err = notifier.Send(ctx, NotificationEmailSendInput{Event: NotificationEmailEventOpsAlert, RecipientEmail: to, RecipientName: emailRecipientName(to), SourceType: "cluster_event", SourceID: strconv.FormatInt(e.ID, 10), Variables: opsAlertEmailVariables(rule, event)}); err != nil {
			return err
		}
	}
	return nil
}
