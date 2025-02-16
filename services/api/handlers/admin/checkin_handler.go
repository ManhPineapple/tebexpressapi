package admin

import (
	"net/http"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CheckInHandler struct {
	Logger *zap.SugaredLogger

	CheckinManager *sqlmanager.CheckinManager
	UserManager    *sqlmanager.UserManager
}

type FetchCheckinResponse struct {
	Checkin *entity.CheckinRequest `json:"check_in"`
}

type CloseCheckinForm struct {
	ID int64 `json:"id"`
}

type CloseCheckinResponse struct {
	Success bool `json:"success"`
}

func NewCheckInHandler(l *zap.SugaredLogger, cim *sqlmanager.CheckinManager, um *sqlmanager.UserManager) *CheckInHandler {
	return &CheckInHandler{
		Logger: l,

		CheckinManager: cim,
		UserManager:    um,
	}
}

func (h *CheckInHandler) Fetch() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusUnauthorized, constant.MessagePermissionDenied)
			return
		}

		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("Get GetUserByID %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

			return
		}

		if user.ID <= 0 || user.Status != constant.UserStatusActive {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)

			return
		}

		warehouseID := 0

		if user.Role == constant.UserRoleWarehouse {
			warehouseID = int(user.WarehouseID)
		}

		Checkin, err := h.CheckinManager.FetchCheckinRequest(warehouseID)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get checkin request error, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

			return
		}

		for i := range Checkin.CheckinPackage {
			Checkin.CheckinPackage[i].Package.StatusString = constant.MapTextStatusAdminPackage[Checkin.CheckinPackage[i].Package.Status]
		}

		c.JSON(http.StatusOK, FetchCheckinResponse{Checkin})
	}
}

func (h *CheckInHandler) Close() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusUnauthorized, constant.MessagePermissionDenied)
			return
		}

		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("Get GetUserByID %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if user.ID <= 0 || user.Status != constant.UserStatusActive {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		form := &CloseCheckinForm{}
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if form.ID < 1 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		_, err = h.CheckinManager.CloseCheckin(form.ID, userID)
		if err != nil {
			h.Logger.Errorf("Close checkin request  %v error, %v", form.ID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CloseCheckinResponse{Success: true})
	}
}
