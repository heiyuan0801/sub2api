package routes

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type kumaRoundTripper func(*http.Request) (*http.Response, error)

func (f kumaRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestWebsiteStatusUsesLatestFreshPublicHeartbeats(t *testing.T) {
	fresh := time.Now().UTC().Format("2006-01-02 15:04:05.000")
	stale := time.Now().Add(-time.Hour).UTC().Format("2006-01-02 15:04:05.000")
	for _, tc := range []struct {
		name, last, clock, want string
	}{
		{"recovered", "1", fresh, "up"},
		{"partial outage", "0", fresh, "down"},
		{"planned maintenance", "3", fresh, "maintenance"},
		{"stale success", "1", stale, "unknown"},
		{"missing state", "null", fresh, "unknown"},
		{"pending check", "2", fresh, "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			source := &kumaStatusClient{client: &http.Client{Transport: kumaRoundTripper(func(req *http.Request) (*http.Response, error) {
				requests++
				var body string
				switch req.URL.String() {
				case kumaStatusURL + kumaPageSlug:
					body = `{"publicGroupList":[{"monitorList":[{"id":2},{"id":4}]}]}`
				case kumaStatusURL + "heartbeat/" + kumaPageSlug:
					body = `{"heartbeatList":{"2":[{"status":0,"time":"` + stale + `"},{"status":` + tc.last + `,"time":"` + tc.clock + `"}],"4":[{"status":1,"time":"` + fresh + `"}],"6":[{"status":0,"time":"` + fresh + `"}]}}`
				default:
					t.Errorf("unexpected upstream request: %s", req.URL)
					return &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody}, nil
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
			})}}
			require.Equal(t, tc.want, source.statusValue())
			require.Equal(t, tc.want, source.statusValue())
			require.Equal(t, 2, requests, "the upstream check is shared across console requests")
		})
	}
}
