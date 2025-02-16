package customer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/dto"
	"tebexpressapi/pkg/sqlmanager"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"go.uber.org/zap"
)

type AnalyticHandler struct {
	Logger  *zap.SugaredLogger
	Redis   *redis.Client
	OnCache bool

	AnalyticsManager *sqlmanager.AnalyticsManager
	PackageManager   *sqlmanager.PackageManager
}

type DashboardResponse struct {
	Logs             interface{} `json:"logs"`
	PackageAnalytics interface{} `json:"package_analytics"`
}

func NewAnalyticHandler(l *zap.SugaredLogger, r *redis.Client, am *sqlmanager.AnalyticsManager, pm *sqlmanager.PackageManager) *AnalyticHandler {
	return &AnalyticHandler{
		Logger: l,
		Redis:  r,

		AnalyticsManager: am,
		PackageManager:   pm,
	}
}

func (h *AnalyticHandler) Get() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		opts := sqlmanager.PackageAnalyticsOptions{
			UserID:    userID,
			StartDate: strings.TrimSpace(c.Request.URL.Query().Get("start_date")),
			EndDate:   strings.TrimSpace(c.Request.URL.Query().Get("end_date")),
			// Status: cast.ToInt(r.URL.Query().Get("status")),
		}

		if opts.StartDate != "" {
			if _, err := time.Parse("2006-01-02", opts.StartDate); err != nil {
				c.JSON(http.StatusBadRequest, "Thời gian không đúng định dạng.")
				return
			}
		}

		if opts.EndDate != "" {
			if _, err := time.Parse("2006-01-02", opts.EndDate); err != nil {
				c.JSON(http.StatusBadRequest, "Thời gian không đúng định dạng.")
				return
			}
		}

		if opts.StartDate != "" && opts.EndDate != "" {
			st, _ := time.Parse("2006-01-02", opts.StartDate)
			et, _ := time.Parse("2006-01-02", opts.EndDate)
			diff := et.Sub(st) / (31 * 24 * time.Hour)
			if diff > 1 {
				c.JSON(http.StatusBadRequest, "Khoảng thời gian thống kê tối đa 31 ngày.")
				return
			}
		}

		analytics, err := h.packageAnalytics(c, opts)
		if err != nil {
			h.Logger.Errorf("analytics package: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		var logs []dto.PackageAuditLogDTO
		err = h.AnalyticsManager.GetLogs(&logs, userID, 10)
		if err != nil {
			h.Logger.Errorf("analytics package audit logs: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		response := DashboardResponse{
			PackageAnalytics: analytics,
			Logs:             logs,
		}

		c.JSON(http.StatusOK, response)
	}
}

func (h *AnalyticHandler) packageAnalytics(c context.Context, opts sqlmanager.PackageAnalyticsOptions) ([]dto.PackageAnalyticsStatus, error) {
	var analytics []dto.PackageAnalyticsStatus

	key := fmt.Sprintf("%d_analytics", opts.UserID)
	if h.OnCache {
		res, err := h.Redis.Get(c, key).Result()

		if err == nil && res != "" {
			err = json.Unmarshal([]byte(res), &analytics)
			if err == nil {
				return analytics, nil
			}

			h.Logger.Errorf("package analytics cache: %v", err)
		}
	}

	now := time.Now()
	opts.EndDate = now.Format("2006-01-02")
	opts.StartDate = now.Add(-30 * 24 * time.Hour).Format("2006-01-02")

	if err := h.AnalyticsManager.PackageAnalyticsStatus(opts, &analytics); err != nil {
		return analytics, err
	}

	if b, err := json.Marshal(analytics); err != nil {
		now := time.Now()
		end, _ := time.Parse("2006-01-02", now.AddDate(0, 0, 1).Format("2006-01-02"))
		expire := end.Sub(now)

		h.Redis.Set(c, key, string(b), expire)
	}

	return analytics, nil
}
