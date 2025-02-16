package customer

import (
	"net/http"
	"strings"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/sqlmanager"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ServiceHandler struct {
	Logger *zap.SugaredLogger

	UserManager    *sqlmanager.UserManager
	ServiceManager *sqlmanager.ServiceManager
}

type GetListServiceResponse struct {
	Services *[]serviceDTO `json:"services"`
}

type serviceDTO struct {
	ID     int64      `json:"id"`
	Name   string     `json:"name"`
	Code   string     `json:"code"`
	Prices []PriceDTO `json:"prices"`
}

type PriceDTO struct {
	ServiceID int64   `json:"service_id"`
	Weight    float64 `json:"weight"`
	Price     float64 `json:"price"`
}

func NewServiceHandler(l *zap.SugaredLogger, um *sqlmanager.UserManager, sm *sqlmanager.ServiceManager) *ServiceHandler {
	return &ServiceHandler{
		Logger: l,

		UserManager:    um,
		ServiceManager: sm,
	}
}

func (h *ServiceHandler) Get() gin.HandlerFunc {
	return func(c *gin.Context) {
		offset, limit := httputil.GetRequestPaginate(c.Request)
		search := c.Request.URL.Query().Get("search")
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))

		if role != constant.UserRoleCustomer {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		user, err := h.UserManager.GetUserByID(userID)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get user %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		hasPrice := strings.TrimSpace(c.Request.URL.Query().Get("has_price"))
		opt := sqlmanager.ServiceQueryOption{
			Status:     constant.StatusActive,
			Limit:      limit,
			Offset:     offset,
			Search:     search,
			IsHasPrice: hasPrice == "yes",
			PriceClass: user.Class,
			PartnerID:  user.PartnerID,
		}

		services, err := h.ServiceManager.GetServices(opt)
		if err != nil && err == gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get user %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		result := &[]serviceDTO{}
		if err := httputil.Transform(services, result); err != nil {
			h.Logger.Errorf("transform service  error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, GetListServiceResponse{Services: result})
	}
}
