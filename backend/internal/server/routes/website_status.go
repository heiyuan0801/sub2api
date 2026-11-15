package routes

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

const (
	kumaStatusURL    = "https://status.weilai.chat/api/status-page/"
	kumaPageSlug     = "weilai"
	kumaResponseSize = 1 << 20
	kumaMaxBeatAge   = 5 * time.Minute
	kumaTimeout      = 4 * time.Second
)

type kumaStatusClient struct {
	client *http.Client
	mu     sync.Mutex
	status string
	expiry time.Time
}

func newWebsiteStatusHandler() gin.HandlerFunc {
	source := &kumaStatusClient{client: &http.Client{
		Timeout: kumaTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
	return func(c *gin.Context) {
		status := source.statusValue()
		c.Header("Cache-Control", "public, max-age=15")
		response.Success(c, gin.H{"status": status})
	}
}

func (s *kumaStatusClient) statusValue() string {
	// Coalesce refreshes from every console tab on this server.
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Now().Before(s.expiry) {
		return s.status
	}

	ctx, cancel := context.WithTimeout(context.Background(), kumaTimeout)
	defer cancel()
	s.status = s.load(ctx)
	s.expiry = time.Now().Add(30 * time.Second)
	return s.status
}

type kumaStatusPage struct {
	PublicGroupList []struct {
		MonitorList []struct {
			ID int64 `json:"id"`
		} `json:"monitorList"`
	} `json:"publicGroupList"`
}

type kumaHeartbeat struct {
	Status *int   `json:"status"`
	Time   string `json:"time"`
}

func (s *kumaStatusClient) load(ctx context.Context) string {
	var page kumaStatusPage
	if err := s.readJSON(ctx, kumaStatusURL+kumaPageSlug, &page); err != nil {
		return "unknown"
	}
	var heartbeats struct {
		HeartbeatList map[string][]kumaHeartbeat `json:"heartbeatList"`
	}
	if err := s.readJSON(ctx, kumaStatusURL+"heartbeat/"+kumaPageSlug, &heartbeats); err != nil {
		return "unknown"
	}
	return summarizeKumaStatus(page, heartbeats.HeartbeatList, time.Now())
}

func (s *kumaStatusClient) readJSON(ctx context.Context, url string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return errors.New("invalid response status from Uptime Kuma")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, kumaResponseSize+1))
	if err != nil {
		return err
	}
	if len(data) > kumaResponseSize {
		return errors.New("Uptime Kuma response exceeds size limit")
	}
	return json.Unmarshal(data, target)
}

func summarizeKumaStatus(page kumaStatusPage, beats map[string][]kumaHeartbeat, now time.Time) string {
	hasMonitor, hasUnknown, hasMaintenance := false, false, false
	for _, group := range page.PublicGroupList {
		for _, monitor := range group.MonitorList {
			hasMonitor = true
			series := beats[strconv.FormatInt(monitor.ID, 10)]
			if len(series) == 0 {
				hasUnknown = true
				continue
			}
			// Kuma returns heartbeats in chronological order using UTC.
			latest := series[len(series)-1]
			when, err := time.Parse("2006-01-02 15:04:05", latest.Time)
			if latest.Status == nil || err != nil || now.Sub(when) > kumaMaxBeatAge || when.After(now.Add(time.Minute)) {
				hasUnknown = true
				continue
			}
			switch *latest.Status {
			case 0: // DOWN
				return "down"
			case 1: // UP
			case 3: // MAINTENANCE
				hasMaintenance = true
			default: // PENDING or unsupported status
				hasUnknown = true
			}
		}
	}
	if !hasMonitor || hasUnknown {
		return "unknown"
	}
	if hasMaintenance {
		return "maintenance"
	}
	return "up"
}
