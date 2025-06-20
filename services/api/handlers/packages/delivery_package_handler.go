package packages

import (
	"fmt"
	"math"
	"net/http"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/helpers/authhelper"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/dbgorm"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/spf13/cast"
	"gorm.io/gorm"
)

func (h *PackageHandler) Delivery() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: constant.APIResponseMessageParseRequestBody,
			})
		}

		id := cast.ToInt64(c.Param("id"))
		if id < 1 {
			c.JSON(http.StatusUnauthorized, httputil.ErrorResponse{
				Error:   constant.MessageValidateInput,
				Message: "Missing package id",
			})

			return
		}

		// check account package cancel exeeds from user info
		msg, err := authhelper.CheckCancelLimit2(c, userID, h.UserManager, h.Redis)
		if err != nil {
			h.Logger.Error("Get order detail: ", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.MessageServerInternalError,
			})

			return
		}

		if msg != "" {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: "Account exceeds order creation limit",
			})

			return
		}

		// get packages from ids
		opts := sqlmanager.PackageQueryOption{
			ID:     id,
			UserID: userID,
		}

		pkg, err := h.PackageManager.GetPackageDetail(opts)
		if err == gorm.ErrRecordNotFound || pkg.ID == 0 {
			h.Logger.Error("Get order detail: ", err)
			c.JSON(http.StatusNotFound, httputil.ErrorResponse{
				Error: constant.MessageNotFound,
			})

			return
		}

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Error("Get order detail: ", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.MessageServerInternalError,
			})

			return
		}

		rKey := fmt.Sprintf("package_deliver_processing_%d", pkg.ID)
		isProcessing, err := h.Redis.Get(c, rKey).Int()
		if err != nil && err != redis.Nil {
			h.Logger.Errorf("get redis key %s", rKey)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

			return
		}

		if isProcessing > 0 {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Order %s is processing, please try again", pkg.OrderNumber))
			return
		}

		if err := h.Redis.SetNX(c, rKey, 1, 2*time.Minute).Err(); err != nil {
			h.Logger.Errorf("set redis key %s", rKey)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

			return
		}

		defer func(key string) {
			if err := h.Redis.Del(c, key).Err(); err != nil {
				h.Logger.Errorf("del redis key %s: %v", key, err)
			}
		}(rKey)

		// check package is created
		if pkg.Status != constant.PackageStatusCreated {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{"Can't update order when status different pending"},
			})

			return
		}

		// check package is exceed
		if pkg.IsPackageExceed && pkg.ShippingFee == 0 {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{"Exceed order are being calculated price. Try again later!"},
			})

			return
		}

		if pkg.PackageCode != nil && pkg.PackageCode.Status == constant.PackageCodeDisable {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Mã vận đơn %s đã bị hủy", pkg.PackageCode.Code))
			return
		}

		if pkg.ValidateAddress != constant.PackageValidAddress {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{"Invalid Address"},
			})
			return
		}

		if pkg.Service.Code == constant.ServiceCNCode && pkg.CustomCNBarcode == nil {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{"CN Service requires custom barcode before delivering"},
			})
			return
		}

		if pkg.Service.Code == constant.ServiceFBACode {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{fmt.Sprintf("%s package is not support", constant.ServiceFBACode)},
			})
			return
		}

		// if pkg.Service.Code != constant.ServiceFBACode {
		// 	isCallLabel, err := h.Redis.SIsMember(c, rkey, pkg.ID).Result()
		// 	if err != nil {
		// 		c.JSON(http.StatusBadRequest, constant.MessageServerInternalError)
		// 		return
		// 	}

		// 	if isCallLabel {
		// 		c.JSON(http.StatusBadRequest, fmt.Sprintf("Đơn hàng #%s đang được tạo mã tracking", pkg.OrderNumber))
		// 		return
		// 	}
		// }

		// create bill
		bill, err := h.BillManager.GetOrCreateNowBill(user.ID)
		if err != nil {
			h.Logger.Errorf("Get bill error: %v", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.APIResponseMessageServerInternalError,
			})
			return
		}

		peakFee, err := h.BillManager.GetExtraFeeTypeByID(constant.ExtraFeeTypePeak)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("get extra peak fee: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if peakFee != nil {
			amount := calculate.PeakFee(pkg.Weight)
			if amount > 0 {
				pkg.ExtraFee = append(pkg.ExtraFee, entity.ExtraFee{
					Model: dbgorm.Model{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					BillID:         utils.Int64(bill.ID),
					PackageID:      utils.Int64(pkg.ID),
					ExtraFeeTypeID: peakFee.ID,
					Description:    peakFee.Name,
					Amount:         amount,
					Status:         constant.ExtraFeeStatusEnable,
				})
			}

		}

		var shippingFee float64 = 0
		var extraFee float64 = 0
		for _, fee := range pkg.ExtraFee {
			extraFee += fee.Amount
		}
		shippingFee += pkg.ShippingFee + extraFee
		shippingFee = utils.ToFixed(shippingFee, 2)

		// check user balance is greater than shipping fee
		if user.Balance < shippingFee && (user.UserInfo == nil || user.UserInfo.DebtMaxAmount <= 0) {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    "Bad request",
				Messages: []string{"The balance in the wallet is not enough. Please top up"},
			})
			return
		}

		if user.Balance-shippingFee < 0 && user.UserInfo != nil && user.UserInfo.DebtMaxAmount > 0 {
			if err != nil && err != gorm.ErrRecordNotFound {
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
			if user.Balance < 0 && user.UserInfo.DebtTime != nil && user.UserInfo.DebtTime.AddDate(0, 0, user.UserInfo.DebtMaxDay).Before(time.Now()) {
				c.JSON(http.StatusInternalServerError, "Tài khoản của bạn đã nợ quá thời hạn cho phép. Vui lòng nạp thêm tiền để tiếp tục sử dụng dịch vụ")
				return
			}
			if math.Abs(user.Balance-shippingFee) > user.UserInfo.DebtMaxAmount {
				c.JSON(http.StatusInternalServerError, "Tài khoản của bạn đã nợ quá giới hạn cho phép. Vui lòng nạp thêm tiền để tiếp tục sử dụng dịch vụ")
				return
			}
		}

		// if len(fbaPkgIDs) > 0 {
		// 	opt := sqlmanager.CreateBillOption{
		// 		Packages:    pkgs,
		// 		BillID:      bill.ID,
		// 		ShippingFee: shippingFee,
		// 		UserID:      userID,
		// 	}

		// 	_, err = h.BillManager.CreateBill(opt, user, nil)
		// 	if err != nil {
		// 		c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
		// 		return
		// 	}

		// 	err = h.EstimateCost.Handle(c, fbaPkgIDs, 0)
		// 	if err != nil {
		// 		log.Printf("Error publish message queue shipment estimate cost: %v", err)
		// 		c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
		// 		return
		// 	}

		// 	err = h.ShipmentCreateLabelHandler.Handle(c, fbaPkgIDs, false, false, 0)
		// 	if err != nil {
		// 		c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
		// 		return
		// 	}
		// }

		if pkg.Service.Code == constant.ServiceTiktokCode || pkg.CustomTiktokBarcode != nil {
			err, packageCodes := h.PackageManager.CreatePackageCodes([]entity.Package{*pkg})
			if err != nil {
				h.Logger.Errorf("Error create package code: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
			pkg.PackageCode = packageCodes[0]

			price := pkg.ShippingFee
			for _, fee := range pkg.ExtraFee {
				price += fee.Amount
			}
			billID, _ := h.BillManager.GetOrCreateNowBillID(pkg.UserID)
			opt := sqlmanager.CreateBillOption{
				Packages:    []entity.Package{*pkg},
				BillID:      billID,
				ShippingFee: price,
				UserID:      pkg.UserID,
			}

			user, err := h.UserManager.GetUserByID(pkg.UserID)
			if err != nil {
				h.Logger.Errorf("Error get user: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			var refundCoupon *entity.ExtraFee
			_, err = h.BillManager.CreateBillWithLabelPromotion(opt, user, refundCoupon, 0)
			if err != nil {
				h.Logger.Errorf("Error create bill: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		} else {
			isPackageCN := pkg.Service.Code == constant.ServiceCNCode
			err = h.ShipmentCreateLabelHandler.Handle(c, []int64{pkg.ID}, true, isPackageCN, 0)
			if err != nil {
				h.Logger.Error("Error publish message queue shipment-create-label: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		}

		var result = DeliverPackageResponse{
			Success:     true,
			PackageCode: pkg.PackageCode.Code,
			BillCode:    bill.Code,
			// Base64Label:     base64,
			// LastMileCarrier: lastMileCarrier,
		}

		c.JSON(http.StatusOK, result)
	}
}
