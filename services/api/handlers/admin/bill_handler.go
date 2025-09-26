package admin

import (
	"fmt"
	"net/http"
	"strings"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BillHandler struct {
	Logger *zap.SugaredLogger

	BillManager    *sqlmanager.BillManager
	UserManager    *sqlmanager.UserManager
	PackageManager *sqlmanager.PackageManager
}

type GetExtraFeeTypeResponse struct {
	ExtraFeeTypes []entity.ExtraFeeType `json:"extra_fee_types"`
}

type BillListReponse struct {
	Bills []entity.Bill `json:"bills"`
}

type BillCountReponse struct {
	Count int64 `json:"count"`
}

type CreateExtraFeeResponse struct {
	Success bool `json:"success"`
}

type CreateExtraFeeForm struct {
	UserIDs        []int64  `json:"user_id"`
	PackageCodes   []string `json:"package_code"`
	ExtraFeeTypeID int64    `json:"extra_fee_type_id"`
	Amount         float64  `json:"amount"`
	Description    string   `json:"description"`
}

type GetDetailBillResponse struct {
	Bill         entity.Bill `json:"bill"`
	CountPackage int64       `json:"count_package"`
}

type GetBillExtraFeeResponse struct {
	Fees  []entity.ExtraFee `json:"fees"`
	Count int64             `json:"count"`
}

func NewBillHandler(l *zap.SugaredLogger, bm *sqlmanager.BillManager, um *sqlmanager.UserManager, pm *sqlmanager.PackageManager) *BillHandler {
	return &BillHandler{
		Logger: l,

		BillManager:    bm,
		UserManager:    um,
		PackageManager: pm,
	}
}

func (h *BillHandler) FeeTypes() gin.HandlerFunc {
	return func(c *gin.Context) {
		extraFeeTypes, err := h.BillManager.GetAllExtraFeeTypes()

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get bill error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

			return
		}

		c.JSON(http.StatusOK, GetExtraFeeTypeResponse{extraFeeTypes})
	}
}

func (h *BillHandler) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		offset, limit := httputil.GetRequestPaginate(c.Request)

		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
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

		opts := sqlmanager.BillQueryOption{
			Search:             strings.TrimSpace(c.Request.URL.Query().Get("search")),
			SearchBy:           strings.TrimSpace(c.Request.URL.Query().Get("search_by")),
			Status:             cast.ToInt(c.Request.URL.Query().Get("status")),
			Limit:              limit,
			Offset:             offset,
			HidePreloadPackage: true,
			UserID:             cast.ToInt64(c.Request.URL.Query().Get("user_id")),
			PartnerID:          user.PartnerID,
		}
		if role == constant.UserRoleSupport || role == constant.UserRoleSale {
			opts.SupportID = user.ID
		}

		bills, err := h.BillManager.Fetch(opts)
		if err != nil {
			h.Logger.Errorf("count bill: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, BillListReponse{Bills: bills})
	}
}

func (h *BillHandler) Detail() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		offset, limit := httputil.GetRequestPaginate(c.Request)

		if role == "" {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}
		if userID < 1 {
			c.JSON(http.StatusForbidden, "User id required")
			return
		}

		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("Get GetUserByID %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if user.Status != constant.UserStatusActive {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		billCode := cast.ToString(c.Param("bill_code"))
		if len(billCode) == 0 {
			c.JSON(http.StatusBadRequest, "Missing bill code")
			return
		}

		opts := sqlmanager.BillQueryOption{
			Limit:    limit,
			Offset:   offset,
			BillCode: billCode,
		}
		if role == constant.UserRoleSupport || role == constant.UserRoleSale {
			opts.SupportID = user.ID
		}

		bill, err := h.BillManager.GetBill(opts)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
		}

		if err != nil {
			h.Logger.Errorf("Get bill error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		opts.ID = bill.ID
		countPackages, err := h.BillManager.CountBillPackages(opts)
		if err != nil {
			h.Logger.Errorf("Get bill error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, GetDetailBillResponse{bill, countPackages})
	}
}

func (h *BillHandler) DetailFee() gin.HandlerFunc {
	return func(c *gin.Context) {
		offset, limit := httputil.GetRequestPaginate(c.Request)

		opts := sqlmanager.BillFeeQueryOption{
			BillCode: cast.ToString(c.Param("bill_code")),
			Type:     cast.ToInt(c.Request.URL.Query().Get("type")),
			Limit:    limit,
			Offset:   offset,
		}
		if len(opts.BillCode) == 0 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		extraFees, err := h.BillManager.GetBillAdminExtraFee(opts)

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get bill error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		opts.Limit = 0
		opts.Offset = 0

		Count, err := h.BillManager.CountBillAdminExtraFee(opts)

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get bill error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, GetBillExtraFeeResponse{extraFees, Count})
	}
}

func (h *BillHandler) Count() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
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

		opts := sqlmanager.BillQueryOption{
			Search:    strings.TrimSpace(c.Request.URL.Query().Get("search")),
			SearchBy:  strings.TrimSpace(c.Request.URL.Query().Get("search_by")),
			Status:    cast.ToInt(c.Request.URL.Query().Get("status")),
			UserID:    cast.ToInt64(c.Request.URL.Query().Get("user_id")),
			PartnerID: user.PartnerID,
		}
		if role == constant.UserRoleSupport || role == constant.UserRoleSale {
			opts.SupportID = user.ID
		}

		count, err := h.BillManager.Count(opts)
		if err != nil {
			h.Logger.Errorf("count bill: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, BillCountReponse{Count: count})
	}
}

func (h *BillHandler) ExtraFee() gin.HandlerFunc {
	return func(c *gin.Context) {
		form := &CreateExtraFeeForm{}
		adminID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if len(form.UserIDs) <= 0 || form.ExtraFeeTypeID <= 0 || len(form.PackageCodes) == 0 || form.Amount <= 0 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		if form.ExtraFeeTypeID == constant.ExtraFeeTypeRefund && form.Amount > constant.MaxRefundFee {
			c.JSON(http.StatusForbidden, "Phí hoàn tiền tối đa là $50")
			return
		}

		extraFeeType, err := h.BillManager.GetExtraFeeTypeByID(form.ExtraFeeTypeID)

		if err != nil {
			h.Logger.Errorf("Get extra fee type error %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		countSuccess := 0
		for index, value := range form.PackageCodes {
			pcode, err := h.PackageManager.GetPackageCodeByCode(sqlmanager.PackageCodeQueryOption{
				SearchCode: value,
			})

			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, map[string]interface{}{
					"error":         "Mã vận đơn không tồn tại",
					"count_success": countSuccess,
				})
				return
			}

			if err != nil {
				h.Logger.Errorf("Get Package Detail %v", err)
				c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error":         constant.MessageServerInternalError,
					"count_success": countSuccess,
				})
				return
			}

			Package, err := h.PackageManager.GetPackageByPackageCodeID(pcode.ID)

			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, map[string]interface{}{
					"error":         "Không tìm thấy mã vận đơn",
					"count_success": countSuccess,
				})
				return
			}

			if err != nil {
				h.Logger.Errorf("Get package error %v", err)
				c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error":         constant.MessageServerInternalError,
					"count_success": countSuccess,
				})
				return
			}

			user, err := h.UserManager.GetUserByID(form.UserIDs[index])
			if err != nil {
				h.Logger.Errorf("Get user error %v", err)
				c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error":         constant.MessageServerInternalError,
					"count_success": countSuccess,
				})
				return
			}

			if user.Role != constant.UserRoleCustomer || user.Status != constant.UserStatusActive {
				c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error":         "Khách hàng không hợp lệ",
					"count_success": countSuccess,
				})
				return
			}

			if Package.UserID != user.ID {
				c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":         fmt.Sprintf("Mã vận đơn không thuộc %s", user.FullName),
					"count_success": countSuccess,
				})
				return
			}

			if Package.Status == constant.PackageStatusCreated {
				c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":         "Đơn ở trạng thái tạo mới không được tạo phí phát sinh",
					"count_success": countSuccess,
				})
				return
			}

			billID, err := h.BillManager.GetOrCreateNowBillID(form.UserIDs[index])

			if err != nil {
				h.Logger.Errorf("Get bill id error %v", err)
				c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error":         constant.MessageServerInternalError,
					"count_success": countSuccess,
				})
				return
			}

			var amount = form.Amount
			if extraFeeType.IsRefund {
				amount = -amount
			}

			extraFee := &entity.ExtraFee{
				PackageID:      utils.Int64(Package.ID),
				BillID:         &billID,
				Amount:         amount,
				Description:    form.Description,
				ExtraFeeTypeID: extraFeeType.ID,
				Status:         constant.ExtraFeeStatusEnable,
			}

			err = h.BillManager.CreateExtraFee(extraFee, user.ID, adminID)
			if err != nil {
				h.Logger.Errorf("Save extra fee error %v", err)
				c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error":         constant.MessageServerInternalError,
					"count_success": countSuccess,
				})
				return
			}

			Package.ExtraFee = append(Package.ExtraFee, *extraFee)
			_, _, updateVatExtrafeeOpt := utils.CreateOrUpdateVat(Package, billID, adminID)
			err = h.BillManager.UpdateVatAfterPretransit(*updateVatExtrafeeOpt)
			if err != nil {
				h.Logger.Errorf("Save extra fee error %v", err)
				c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error":         constant.MessageServerInternalError,
					"count_success": countSuccess,
				})
				return
			}

			countSuccess++
		}

		c.JSON(http.StatusOK, CreateExtraFeeResponse{true})
	}
}
