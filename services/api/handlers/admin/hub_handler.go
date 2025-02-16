package admin

import (
	"fmt"
	"net/http"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type HubHandler struct {
	Logger *zap.SugaredLogger

	PackageManager *sqlmanager.PackageManager
	UserManager    *sqlmanager.UserManager
	BillManager    *sqlmanager.BillManager
	ServiceManager *sqlmanager.ServiceManager
}

type ScanReturnPackageResponse struct {
	Package *entity.Package `json:"package"`
}

type ScanReturnPackageForm struct {
	Search string `json:"search"`
}

func NewHubHandler(l *zap.SugaredLogger, pm *sqlmanager.PackageManager, um *sqlmanager.UserManager, bm *sqlmanager.BillManager, sm *sqlmanager.ServiceManager) *HubHandler {
	return &HubHandler{
		Logger: l,

		PackageManager: pm,
		UserManager:    um,
		BillManager:    bm,
		ServiceManager: sm,
	}
}

func (h *HubHandler) Return() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		form := &ScanReturnPackageForm{}
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if form.Search == "" {
			c.JSON(http.StatusBadRequest, "Vui lòng nhập/quét mã")
			return
		}

		pkg, err := h.PackageManager.GetPackageReturn(form.Search)
		if err == gorm.ErrRecordNotFound || pkg == nil {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}
		if err != nil {
			h.Logger.Errorf("Get package error : %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if pkg.Status != constant.PackageStatusInTransit && pkg.Status != constant.PackageStatusDelivered {
			c.JSON(http.StatusBadRequest, "Đơn hàng có trạng thái không hợp lệ !")
			return
		}

		if pkg.Alert == constant.PackageAlertTypeHubReturn {
			c.JSON(http.StatusBadRequest, "Đơn hàng đã được quét !")
			return
		}

		service, err := h.ServiceManager.GetServiceByID(pkg.ServiceID)

		if err == gorm.ErrRecordNotFound || pkg == nil {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}
		if err != nil {
			h.Logger.Errorf("Get package error : %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if service.Code == constant.ServiceFBACode {
			c.JSON(http.StatusBadRequest, "Không thể quét đơn FBA")
			return
		}

		if role == constant.UserRoleHub {
			hub, err := h.UserManager.GetWareHouseByUserSupport(userID)
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
				return
			}
			if err != nil {
				h.Logger.Errorf("get user current: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			if hub.Type != constant.WareHouseTypeHub {
				c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
				return
			}

			if utils.Int64Value(pkg.HubID) != hub.ID {
				c.JSON(http.StatusBadRequest, fmt.Sprintf("Đơn hàng không thuộc về Hub %s", hub.Name))
				return
			}
		}

		pkgCodes, err := h.PackageManager.GetPackagesCode(sqlmanager.PackageCodeQueryOption{ID: pkg.ID, Status: constant.PackageCodeEnable})

		if err == gorm.ErrRecordNotFound || len(pkgCodes) == 0 {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}
		if err != nil {
			h.Logger.Errorf("Get package error : %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		billID, err := h.BillManager.GetOrCreateNowBillID(pkg.UserID)
		if err != nil {
			h.Logger.Errorf("Get bill id error %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		pkg.PackageCode = &pkgCodes[0]
		pkg.Alert = constant.PackageAlertTypeHubReturn
		now := time.Now()
		pkg.ReturnedAt = &now
		pkg.UpdatedAt = now
		err = h.PackageManager.SavePackageReturn(pkg, billID, userID)
		if err != nil {
			h.Logger.Errorf("Save package error : %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		c.JSON(http.StatusOK, ScanReturnPackageResponse{pkg})
	}
}
