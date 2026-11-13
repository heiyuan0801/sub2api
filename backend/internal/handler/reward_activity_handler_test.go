package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type rewardActivityStatusRepo struct {
	service.RewardActivityRepository
	status service.RewardActivityStatus
}

func (r rewardActivityStatusRepo) Status(context.Context, int64) (*service.RewardActivityStatus, error) {
	return &r.status, nil
}

func (r rewardActivityStatusRepo) GetConfig(context.Context) (service.RewardActivityConfig, error) {
	return r.status.Config, nil
}

func TestRewardActivityStatusDoesNotExposePrizeWeights(t *testing.T) {
	config := service.DefaultRewardActivityConfig()
	config.Wheel.Enabled = true
	config.Wheel.Prizes = []service.RewardPrize{
		{Name: "No reward", Weight: 90},
		{Name: "Balance credit", RewardCents: 50, Weight: 10},
	}
	repo := rewardActivityStatusRepo{status: service.RewardActivityStatus{Config: config, DrawsAvailable: 2}}
	h := NewRewardActivityHandler(service.NewRewardActivityService(repo, nil, nil))

	user := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(user)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/reward-activities", nil)
	c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1})
	h.Status(c)
	require.Equal(t, http.StatusOK, user.Code)
	var response struct {
		Data struct {
			Config struct {
				Wheel struct {
					Prizes json.RawMessage `json:"prizes"`
				} `json:"wheel"`
			} `json:"config"`
			DrawsAvailable int `json:"draws_available"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(user.Body.Bytes(), &response))
	require.Equal(t, 2, response.Data.DrawsAvailable)
	require.JSONEq(t, `[{"name":"No reward","reward_cents":0},{"name":"Balance credit","reward_cents":50}]`, string(response.Data.Config.Wheel.Prizes))

	admin := httptest.NewRecorder()
	c, _ = gin.CreateTestContext(admin)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/reward-activities", nil)
	h.GetConfig(c)
	require.Equal(t, http.StatusOK, admin.Code)
	var adminResponse struct {
		Data service.RewardActivityConfig `json:"data"`
	}
	require.NoError(t, json.Unmarshal(admin.Body.Bytes(), &adminResponse))
	require.Equal(t, config.Wheel.Prizes, adminResponse.Data.Wheel.Prizes)
}
