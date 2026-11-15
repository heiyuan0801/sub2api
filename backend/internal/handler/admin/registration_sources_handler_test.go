package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegistrationSourceFilterUsesCalendarDaysAcrossDST(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet,
		"/?range=custom&start_date=2025-03-09&end_date=2025-03-09&timezone=America%2FNew_York&dimension=utm&source=GitHub", nil)
	filter, err := parseRegistrationSourceFilter(c, time.Date(2025, 3, 9, 14, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Equal(t, 23*time.Hour, filter.End.Sub(filter.Start))
	require.Equal(t, filter.TodayStart, filter.Start)
	require.Equal(t, "github", filter.Source)

	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/?range=custom&start_date=2022-01-01&end_date=2025-01-01", nil)
	_, err = parseRegistrationSourceFilter(c, time.Now())
	require.Error(t, err)
}

func TestRegistrationSourceCSVCellsDoNotExecuteFormulas(t *testing.T) {
	for _, value := range []string{"=1+1", "+123", "-6", "@cmd", "  =SUM()", "＝1+1", "\tplain", "\rplain", "\nplain"} {
		require.Equal(t, "'"+value, safeRegistrationCSVCell(value))
	}
	require.Equal(t, "中文论坛", safeRegistrationCSVCell("中文论坛"))
	require.Equal(t, "name@example.com", safeRegistrationCSVCell("name@example.com"))
}

func TestRegistrationSourcePromotionAllowsExplicitZeroBonus(t *testing.T) {
	for _, tc := range []struct {
		body string
		ok   bool
	}{{`{"bonus_amount":0}`, true}, {`{}`, false}, {`{"bonus_amount":-1}`, false}} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
		var req CreatePromoCodeRequest
		if tc.ok {
			require.NoError(t, c.ShouldBindJSON(&req))
			require.Zero(t, *req.BonusAmount)
		} else {
			require.Error(t, c.ShouldBindJSON(&req))
		}
	}
}
