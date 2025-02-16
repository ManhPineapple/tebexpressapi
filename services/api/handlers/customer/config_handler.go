package customer

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type ConfigHandler struct {
	Logger *zap.SugaredLogger
}

type GetConfigResponse struct {
	Config Config `json:"config"`
}

type Config struct {
	ExtraFee float64 `json:"extra_fee"`
}

func NewConfigHandler(l *zap.SugaredLogger) *ConfigHandler {
	return &ConfigHandler{
		Logger: l,
	}
}

func (h *ConfigHandler) Get() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusForbidden, "User id required")
			return
		}
		rate := viper.GetFloat64("extra_fees.peak_fee")
		c.JSON(http.StatusOK, GetConfigResponse{Config: Config{ExtraFee: rate}})
	}
}
