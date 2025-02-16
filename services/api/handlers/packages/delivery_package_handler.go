package packages

import (
	"encoding/base64"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/helpers/authhelper"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/label"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/order"
	"tebexpressapi/pkg/providers"
	"tebexpressapi/pkg/providers/ibblue"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/dbgorm"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func (h *PackageHandler) Delivery() gin.HandlerFunc {
	return func(c *gin.Context) {
		// check auth
		userId := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		user, err := h.UserManager.GetUserByID(userId)
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
		msg, err := authhelper.CheckCancelLimit2(c, userId, h.UserManager, h.Redis)
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

		// get package from id
		opts := sqlmanager.PackageQueryOption{
			ID:     id,
			UserID: userId,
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

		if pkg.ValidateAddress != constant.PackageValidAddress {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{"Invalid Address"},
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

		// create bill
		bill, err := order.GetOrCreateNowBill(c, h.BillManager, h.Redis, userId)
		if err != nil {
			h.Logger.Errorf("Get bill error: %v", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.APIResponseMessageServerInternalError,
			})

			return
		}

		// calc shipping fee
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

		// check user balance is greater than shipping fee
		if user.Balance < shippingFee && (user.UserInfo == nil || user.UserInfo.DebtMaxAmount <= 0) {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    "Bad request",
				Messages: []string{"The balance in the wallet is not enough. Please top up"},
			})

			return
		}

		if user.Balance-shippingFee < 0 {
			userInfo, err := h.UserManager.GetUserInfoByUserID(user.ID)
			if err != nil && err != gorm.ErrRecordNotFound {
				c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
					Error: constant.APIResponseMessageServerInternalError,
				})

				return
			}
			// check is debt?
			if userInfo != nil && user.Balance < 0 && userInfo.DebtTime.AddDate(0, 0, userInfo.DebtMaxDay).Before(time.Now()) {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    "Bad request",
					Messages: []string{"Your account is overdue. Please top up to continue using the service"},
				})

				return
			}

			if userInfo != nil && math.Abs(user.Balance-shippingFee) > userInfo.DebtMaxAmount {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    "Bad request",
					Messages: []string{"Your account has exceeded the allowable limit. Please top up to continue using the service"},
				})

				return
			}
		}

		lastMileCarrier := ""
		if pkg.Service.Code != constant.ServiceFBACode && pkg.Service.Code != constant.ServiceACTUSCode && pkg.Service.Code != constant.ServiceAUFCode && pkg.Service.Code != constant.ServiceNDCode {
			h.Logger.Info("code: ", pkg.Service.Code)
			start := time.Now()
			var labelBase64 string
			var carrierCode string
			var carrier providers.Carrier = nil
			dbcarrier := &pkg.Service.DomesticCarrier

			if pkg.CountryCode == "AU" {
				carrier = providers.NewCarrier(dbcarrier.Code, pkg.UserID)
			} else {
				carrierCode, err = h.CreateLabel.GetCarrierCode(c, *pkg, pkg.UserID, "")
				if err != nil {
					h.Logger.Errorf("get carrier code %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

					return
				}

				h.Logger.Info("carrierCode: ", carrierCode)
				if carrierCode != "" {
					carrier = providers.NewCarrier(carrierCode, pkg.UserID)
				} else {
					carrier = providers.NewCarrier(dbcarrier.Code, pkg.UserID)
				}
			}

			if carrier == nil {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    "Bad request",
					Messages: []string{"Invalid carrier package id %s", cast.ToString(pkg.ID)},
				})

				return
			}

			if carrierCode != dbcarrier.Code && carrierCode != "" {
				dbcarrier, err = h.ServiceManager.GetCarrierByCode(carrierCode)
				if err != nil {
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

					return
				}
			}

			if dbcarrier != nil {
				lastMileCarrier = dbcarrier.LastMileCarrier
			}

			err, pCodes := h.PackageManager.CreatePackageCodes([]entity.Package{*pkg})
			if err != nil {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    "Bad request",
					Messages: []string{"Failed to create labels"},
				})

				return
			}

			if len(pCodes) == 0 {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    "Bad request",
					Messages: []string{"Failed to create labels"},
				})

				return
			}

			pkg.PackageCode = pCodes[0]

			duration1 := time.Since(start)
			h.Logger.Info(cast.ToString(pkg.ID), " - Duration Create COde Package: ", duration1)
			var template string = ibblue.TemplateTebexpress
			wL := strings.Split(viper.GetString("white_list.label"), ",")
			if len(wL) > 0 {
				for _, email := range wL {
					if email == user.Email {
						template = ibblue.TemplateTebexpress
					}
				}
			}

			h.Logger.Info("pkg.Tracking: ", pkg.Tracking != nil && pkg.Tracking.Status != constant.TrackingStatusCanceled)
			if pkg.Tracking != nil && pkg.Tracking.Status != constant.TrackingStatusCanceled {
				bucket := viper.GetString("bucket.labels")
				h.Logger.Info("pkg.Tracking.LabelURL: ", pkg.Tracking.LabelURL)
				if buf, err := h.StorageS3.ReadFile(pkg.Tracking.LabelURL, bucket); err == nil {
					labelBase64 = base64.StdEncoding.EncodeToString(buf.Bytes())
				} else {
					h.Logger.Errorf("get file s3: %v", err)
				}

				pkg.Tracking.Status = constant.TrackingStatusSuccess
				pkg.Label = pkg.Tracking.LabelURL

				dbcarrier, err = h.ServiceManager.GetCarrierByID(pkg.Tracking.CarrierID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

					return
				}

				lastMileCarrier = dbcarrier.LastMileCarrier
			} else {
				start4 := time.Now()
				warehouse, zone, errEstimate := h.EstimateCostWithPromotionGuest(c, pkg, carrier, dbcarrier)

				duration4 := time.Since(start4)
				h.Logger.Info(cast.ToString(pkg.ID), " - Duration Create Label: ", duration4)

				if errEstimate != nil {
					if err := h.PackageManager.CancelPackageCodes([]entity.Package{*pkg}); err != nil {
						h.Logger.Errorf("Send notification to queue: %v", err)
						c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
							Error:    "Bad request",
							Messages: []string{" Failed to create labels"},
						})

						return
					}

					if cast.ToString(errEstimate) == "AVS04: invalid city" {
						c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
							Error:    "Bad request",
							Messages: []string{"Invalid address"},
						})

						return
					}

					c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
						Error:    "Bad request",
						Messages: []string{" Failed to create labels"},
					})

					return
				}

				start1 := time.Now()

				if pkg.Service == nil || dbcarrier.Code == "" {
					c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
						Error:    "Bad request",
						Messages: []string{" Failed to create labels"},
					})

					return
				}

				tracking, lastMileCarrierName, msg, err := h.label(c, pkg, carrier, warehouse, template, dbcarrier.Code, zone)
				if err != nil {
					h.Logger.Errorf("create label: %v", err)
					c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
						Error:    "Bad request",
						Messages: []string{"Failed to create labels"},
					})

					return
				}

				if lastMileCarrierName != "" {
					lastMileCarrier = lastMileCarrierName
				}

				duration1 := time.Since(start1)
				h.Logger.Info(cast.ToString(pkg.ID), " - Duration Create Label: ", duration1)

				if msg != "" {
					c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
						Error:    "Bad request",
						Messages: []string{msg},
					})

					return
				}

				start2 := time.Now()

				ext := "png"
				if carrier.GetCode() == providers.CarrierTypeDarius {
					ext = "pdf"
				}

				h.Logger.Info("ext: ", ext, carrier.GetCode())
				path, err := order.StoreLabelS3(h.StorageS3, tracking.LabelURL, ext, tracking.TrackingNumber)
				if err != nil {
					h.Logger.Error("StoreLabelS3 fail: ", err)
					c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
						Error:    "Bad request",
						Messages: []string{" Failed to create labels"},
					})

					return
				}

				duration2 := time.Since(start2)
				h.Logger.Info(cast.ToString(pkg.ID), " - Duration Store Label: ", duration2)

				labelBase64 = tracking.LabelURL
				tracking.LabelURL = path
				tracking.UserID = user.ID
				tracking.Version = viper.GetString("tracking_version")
				tracking.Status = constant.TrackingStatusPending
				pkg.Label = path
				pkg.Tracking = tracking

				if tracking.CarrierID < 1 {
					tracking.CarrierID = dbcarrier.ID
				}
			}

			opt := sqlmanager.CreateBillOption{
				Packages:    []entity.Package{*pkg},
				BillID:      bill.ID,
				ShippingFee: shippingFee,
				UserID:      user.ID,
			}

			start3 := time.Now()

			_, err = h.BillManager.CreateBillWithLabelPromotion(opt, user, nil, 0)
			if err != nil {
				h.Logger.Errorf("Error create bill order %v: %v", pkg.ID, err)
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    "Bad request",
					Messages: []string{"Error create bill"},
				})

				return
			}

			duration3 := time.Since(start3)
			h.Logger.Info(cast.ToString(pkg.ID), " - Duration Create Bill: ", duration3)

			// kiem tra so du tk va gui mail neu so du = 0 or so cong no = 0

			// h.SendNotifyPoint(user.ID, point)

			trackingNumber := ""
			if pkg.Tracking != nil && pkg.Tracking.ID > 0 && pkg.Tracking.Status == constant.TrackingStatusPending {
				tk := pkg.Tracking
				tk.Status = constant.TrackingStatusSuccess
				trackingNumber = tk.TrackingNumber

				if err := h.TrackingManager.Update(tk); err != nil {
					h.Logger.Error("update status tracking: ", err)
				}
			}

			c.JSON(http.StatusOK, DeliverPackageResponse{
				Success:         true,
				PackageCode:     pkg.PackageCode.Code,
				BillCode:        bill.Code,
				Base64Label:     labelBase64,
				LastMileCarrier: lastMileCarrier,
				TrackingNumber:  trackingNumber,
			})

			return
		}

		opt := sqlmanager.CreateBillOption{
			Packages:    []entity.Package{*pkg},
			BillID:      bill.ID,
			ShippingFee: shippingFee,
			UserID:      user.ID,
		}

		_, err = h.BillManager.CreateBill(opt, user, nil)
		if err != nil {
			h.Logger.Error("Creat bill err: ", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{Error: constant.APIResponseMessageServerInternalError})

			return
		}

		// preload package code created
		pkg, err = h.PackageManager.GetPackageDetail(opts)
		if err == gorm.ErrRecordNotFound || pkg.ID == 0 {
			h.Logger.Error("Get order detail: ", err)
			c.JSON(http.StatusNotFound, httputil.ErrorResponse{Error: constant.MessageNotFound})

			return
		}

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Error("Get order detail: ", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{Error: constant.MessageServerInternalError})

			return
		}

		var base64 string
		if pkg.Service.Code == constant.ServiceNDCode {
			base, path, err := label.CreateLabel(pkg, h.SettingManager, h.StorageS3)
			if err != nil {
				h.Logger.Errorf("generate label err: %v", err)
				c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{Error: constant.MessageServerInternalError})

				return
			}

			if err = h.PackageManager.UpdatePackage(&entity.Package{Label: path}, pkg.ID); err != nil {
				h.Logger.Errorf("update package err: %v", err)
				c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{Error: constant.MessageServerInternalError})

				return
			}

			base64 = base
		}

		if pkg.Service != nil {
			err = h.ShipmentEstimateCost.Handle(c, []int64{pkg.ID}, 0)
			if err != nil {
				log.Printf("Error publish message queue shipment estimate cost: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		}

		var result = DeliverPackageResponse{
			Success:         true,
			PackageCode:     pkg.PackageCode.Code,
			BillCode:        bill.Code,
			Base64Label:     base64,
			LastMileCarrier: lastMileCarrier,
		}

		c.JSON(http.StatusOK, result)
	}
}
