package admin

import (
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const registrationSourceDateLayout = "2006-01-02"

func parseRegistrationSourceFilter(c *gin.Context, now time.Time) (service.RegistrationSourceFilter, error) {
	var f service.RegistrationSourceFilter
	loc := timezone.Location()
	if zone := strings.TrimSpace(c.Query("timezone")); zone != "" {
		var err error
		loc, err = time.LoadLocation(zone)
		if err != nil {
			return f, errors.New("Invalid timezone")
		}
	}
	// 'Local' is Go's process-local sentinel and is not a Postgres zone name.
	if loc == time.Local && loc.String() == "Local" {
		loc = time.UTC
	}
	f.Timezone = loc.String()
	current := now.In(loc)
	f.TodayStart = time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, loc)
	f.TodayEnd = f.TodayStart.AddDate(0, 0, 1)
	f.End = f.TodayEnd
	switch c.DefaultQuery("range", "30d") {
	case "today":
		f.Start = f.TodayStart
	case "7d":
		f.Start = f.TodayStart.AddDate(0, 0, -6)
	case "30d":
		f.Start = f.TodayStart.AddDate(0, 0, -29)
	case "custom":
		var startErr, endErr error
		f.Start, startErr = time.ParseInLocation(registrationSourceDateLayout, c.Query("start_date"), loc)
		f.End, endErr = time.ParseInLocation(registrationSourceDateLayout, c.Query("end_date"), loc)
		f.End = f.End.AddDate(0, 0, 1)
		if startErr != nil || endErr != nil || !f.End.After(f.Start) || f.End.After(f.Start.AddDate(0, 0, 366)) {
			return f, errors.New("Invalid date range, use at most 366 days")
		}
	default:
		return f, errors.New("Invalid registration range")
	}
	f.Dimension = c.DefaultQuery("dimension", service.RegistrationDimensionSelected)
	if f.Dimension != service.RegistrationDimensionSelected && f.Dimension != service.RegistrationDimensionUTM {
		return f, errors.New("Invalid source dimension")
	}
	f.Source = strings.ToLower(strings.TrimSpace(c.Query("source")))
	if len([]rune(f.Source)) > 120 ||
		(f.Dimension == service.RegistrationDimensionSelected && f.Source != "" && f.Source != service.RegistrationSourceUnknown && !service.IsSelectedRegistrationSource(f.Source)) {
		return f, errors.New("Invalid registration source")
	}
	return f, nil
}

func (h *UserHandler) RegistrationSourceSummary(c *gin.Context) {
	filter, err := parseRegistrationSourceFilter(c, time.Now())
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	summary, err := h.userService.RegistrationSourceSummary(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *UserHandler) ListRegistrationSourceUsers(c *gin.Context) {
	filter, err := parseRegistrationSourceFilter(c, time.Now())
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	page, pageSize := response.ParsePagination(c)
	users, result, err := h.userService.ListRegistrationSourceUsers(c.Request.Context(), filter,
		pagination.PaginationParams{Page: page, PageSize: pageSize})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	response.Paginated(c, users, result.Total, page, pageSize)
}

func (h *UserHandler) ExportRegistrationSourceUsers(c *gin.Context) {
	filter, err := parseRegistrationSourceFilter(c, time.Now())
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	var writer *csv.Writer
	startCSV := func() error {
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Cache-Control", "no-store")
		filename := fmt.Sprintf("registration-sources-%s.csv", filter.TodayStart.Format(registrationSourceDateLayout))
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		// Mark UTF-8 for spreadsheet apps opening the export directly.
		if _, err := c.Writer.WriteString("\uFEFF"); err != nil {
			return err
		}
		writer = csv.NewWriter(c.Writer)
		return writer.Write([]string{
			"id", "email", "username", "created_at", "signup_source",
			"selected_source", "source_note", "utm_source", "utm_medium", "utm_campaign", "utm_content", "utm_term",
			"inviter_id", "promo_code",
		})
	}
	err = h.userService.ExportRegistrationSourceUsers(c.Request.Context(), filter, func(user service.RegistrationSourceUser) error {
		if writer == nil {
			if err := startCSV(); err != nil {
				return err
			}
		}
		inviterID := ""
		if user.InviterID != nil {
			inviterID = strconv.FormatInt(*user.InviterID, 10)
		}
		return writer.Write([]string{
			strconv.FormatInt(user.ID, 10), safeRegistrationCSVCell(user.Email), safeRegistrationCSVCell(user.Username),
			user.CreatedAt.UTC().Format(time.RFC3339), safeRegistrationCSVCell(user.SignupSource),
			safeRegistrationCSVCell(user.SelectedSource), safeRegistrationCSVCell(user.SourceNote), safeRegistrationCSVCell(user.UTMSource),
			safeRegistrationCSVCell(user.UTMMedium), safeRegistrationCSVCell(user.UTMCampaign), safeRegistrationCSVCell(user.UTMContent), safeRegistrationCSVCell(user.UTMTerm),
			inviterID, safeRegistrationCSVCell(user.PromoCode),
		})
	})
	if err != nil {
		if !c.Writer.Written() {
			response.ErrorFrom(c, err)
		} else {
			slog.WarnContext(c.Request.Context(), "registration source export interrupted", "error", err)
		}
		return
	}
	if writer == nil {
		if err := startCSV(); err != nil {
			slog.WarnContext(c.Request.Context(), "registration source export could not start", "error", err)
			return
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		slog.WarnContext(c.Request.Context(), "registration source export could not finish", "error", err)
	}
}

func safeRegistrationCSVCell(value string) string {
	first, _ := utf8.DecodeRuneInString(strings.TrimSpace(value))
	if strings.ContainsRune("=+-@＝＋－＠", first) || strings.HasPrefix(value, "\t") || strings.HasPrefix(value, "\r") || strings.HasPrefix(value, "\n") {
		return "'" + value
	}
	return value
}
