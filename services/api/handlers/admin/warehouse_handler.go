package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/createlabel"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/order"
	"tebexpressapi/pkg/providers"
	"tebexpressapi/pkg/providers/ibblue"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/dbgorm"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type WarehouseHandler struct {
	Logger *zap.SugaredLogger

	Redis     *redis.Client
	StorageS3 storage.S3
	IBBlue    *ibblue.IBBlue

	CalculatePrice *calculate.CalculatePrice
	CreateLabel    *createlabel.CreateLabel

	UserManager             *sqlmanager.UserManager
	WareHouseManager        *sqlmanager.WareHouseManager
	PackageManager          *sqlmanager.PackageManager
	CheckinManager          *sqlmanager.CheckinManager
	CustomerShipmentManager *sqlmanager.CustomerShipmentManager
	BillManager             *sqlmanager.BillManager
	ServiceManager          *sqlmanager.ServiceManager
	TrackingManager         *sqlmanager.TrackingManager
}

type CheckRelabelWarehouseResponse struct {
	ReLabel bool `json:"re_label"`
}

type CheckRelabelWarehouseForm struct {
	Weight float64 `json:"weight"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Length float64 `json:"length"`
}

type FormCreateTracking struct {
	CheckinRequestID int64   `json:"checkin_request_id"`
	ActualWeight     float64 `json:"actual_weight"`
	ActualLength     float64 `json:"actual_length"`
	ActualWidth      float64 `json:"actual_width"`
	ActualHeight     float64 `json:"actual_height"`
}

type ListWareHouseResponse struct {
	WareHouses []entity.Warehouse `json:"warehouses"`
}

type GetPackagelResponse struct {
	Package interface{} `json:"package"`
}

type CreateTrackingResponse struct {
	StatusCheckin int  `json:"status_checkin"`
	Success       bool `json:"success"`
}

type PackageReturnResponse struct {
	Success bool `json:"success"`
}

func NewWarehouseHandler(l *zap.SugaredLogger, r *redis.Client, s3 storage.S3, calculatePrice *calculate.CalculatePrice, createLabel *createlabel.CreateLabel, wh *sqlmanager.WareHouseManager, pm *sqlmanager.PackageManager, um *sqlmanager.UserManager, cim *sqlmanager.CheckinManager, csm *sqlmanager.CustomerShipmentManager, bm *sqlmanager.BillManager, sm *sqlmanager.ServiceManager, tm *sqlmanager.TrackingManager) *WarehouseHandler {
	return &WarehouseHandler{
		Logger: l,

		Redis:     r,
		StorageS3: s3,

		CalculatePrice: calculatePrice,
		CreateLabel:    createLabel,

		UserManager:             um,
		WareHouseManager:        wh,
		PackageManager:          pm,
		CheckinManager:          cim,
		CustomerShipmentManager: csm,
		BillManager:             bm,
		ServiceManager:          sm,
		TrackingManager:         tm,
	}
}

func (h *WarehouseHandler) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		offset, limit := httputil.GetRequestPaginate(c.Request)
		opts := sqlmanager.OptionWareHouse{
			Status:  cast.ToInt64(c.Request.URL.Query().Get("status")),
			Type:    cast.ToInt64(c.Request.URL.Query().Get("type")),
			Country: cast.ToString(c.Request.URL.Query().Get("country")),
			Limit:   limit,
			Offset:  offset,
		}

		warehouses, err := h.WareHouseManager.GetWareHouses(opts)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get warehouse error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, ListWareHouseResponse{warehouses})
	}
}

func (h *WarehouseHandler) GetPackage() gin.HandlerFunc {
	return func(c *gin.Context) {
		code := strings.TrimSpace(c.Param("code"))
		if code == "" {
			c.JSON(http.StatusBadRequest, "Mã vận đơn không được trống")
			return
		}
		isFinding := false
		pcode, err := h.PackageManager.GetPackageCodeByCode(sqlmanager.PackageCodeQueryOption{
			Codes:  []string{code},
			Status: constant.PackageCodeEnable,
		})

		if err == gorm.ErrRecordNotFound {
			isFinding = true
		}

		h.Logger.Info("waree1: ", code, isFinding, err)
		if err != nil && !isFinding {
			h.Logger.Errorf("Get Package Detail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

			return
		}
		var pkg *entity.Package
		if isFinding {
			result, err := h.PackageManager.GetPackage2(sqlmanager.PackageQueryOption{TrackingNumber: code, Preload: []string{"User", "Service"}})
			h.Logger.Info("waree: ", err)
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, "Mã vận đơn không tồn tại")

				return
			}
			if err != nil {
				h.Logger.Errorf("Get Package Detail %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

				return
			}
			pkg = &result

			pcode, err := h.PackageManager.GetPackageCodeByCode(sqlmanager.PackageCodeQueryOption{
				PackageCodeID: *pkg.PackageCodeID,
				Status:        constant.PackageCodeEnable,
			})
			if err != nil {
				h.Logger.Errorf("Get Package Detail %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

				return
			}

			pkg.PackageCode = pcode
		} else {
			pkg, err = h.PackageManager.GetPackageByPackageCodeID(pcode.ID)
			h.Logger.Info("waree2: ", pcode.ID, err)
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, "Mã vận đơn không tồn tại")

				return
			}
			if err != nil {
				h.Logger.Errorf("Get Package Detail %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

				return
			}

			pkg.PackageCode = pcode
			pkg.User.Password = "*****"
		}

		c.JSON(http.StatusOK, GetPackagelResponse{Package: pkg})
	}
}

func (h *WarehouseHandler) CreateTracking() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

			return
		}

		form := &FormCreateTracking{}
		if err := c.ShouldBindJSON(form); err != nil {
			h.Logger.Error("parse body: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)

			return
		}

		if form.CheckinRequestID < 1 {
			c.JSON(http.StatusBadRequest, "Mã quét vào kho không được trống")
			return
		}

		id := cast.ToInt64(c.Param("package_id"))
		if id < 1 {
			c.JSON(http.StatusBadRequest, "ID đơn hàng không được trống")
			return
		}

		pkg, err := h.PackageManager.GetPackageByPackageID(id)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, fmt.Sprintf("ID đơn hàng %d không tồn tại", id))
			return
		}

		rkey := "package_call_label"
		isCallLabel, err := h.Redis.SIsMember(c, rkey, id).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if isCallLabel {
			if pkg.Service.Code == constant.ServiceACTUSCode {
				err = h.Redis.SRem(c, rkey, id).Err()
				if err != nil {
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}
			} else {
				c.JSON(http.StatusBadRequest, fmt.Sprintf("ID đơn hàng %d đang được tạo mã tracking", id))
				return
			}
		}

		if form.CheckinRequestID > 0 {
			checkin, err := h.CheckinManager.GetCheckinByID(form.CheckinRequestID)
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusBadRequest, "Mã vào kho không tồn tại, vui lòng load lại trang và thao tác lại")
				return
			}

			if err != nil {
				h.Logger.Errorf("Get checkin request: %v", err)
				checkinPackage := entity.CheckinPackage{
					CheckinID: form.CheckinRequestID,
					PackageID: id,
					Status:    constant.CheckinPackageStatusFailed,
				}
				err = h.PackageManager.SaveCheckinPackage(checkinPackage)
				if err != nil {
					h.Logger.Errorf("update packae %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				c.JSON(http.StatusOK, CreateTrackingResponse{StatusCheckin: checkinPackage.Status, Success: true})
				return
			}

			if checkin.Status == constant.CheckinStatusClosed {
				c.JSON(http.StatusBadRequest, "Mã vào kho đã được đóng, vui lòng load lại trang và thao tác lại")
				return
			}
		}

		if err != nil {
			h.Logger.Errorf("Get checkin request: %v", err)
			checkinPackage := entity.CheckinPackage{
				CheckinID: form.CheckinRequestID,
				PackageID: id,
				Status:    constant.CheckinPackageStatusFailed,
			}
			err = h.PackageManager.SaveCheckinPackage(checkinPackage)
			if err != nil {
				h.Logger.Errorf("update packae %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

				return
			}

			c.JSON(http.StatusOK, CreateTrackingResponse{StatusCheckin: checkinPackage.Status, Success: true})
			return
		}

		if pkg.Status >= constant.PackageStatusPicked {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Đơn hàng %s đã được quét vào kho", pkg.PackageCode.Code))
			return
		}

		wID := viper.GetInt64("warehouse_vn_default")
		wh, err := h.PackageManager.GetPackageWarehouse(wID)
		if err != nil {
			h.Logger.Errorf("Get warehouse error %v %v", wID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

			return
		}
		pkg.Warehouse = wh
		// save checkin package with status invalid
		if pkg.Status == constant.PackageStatusCreated ||
			pkg.Service == nil ||
			(pkg.Service.Code != constant.ServiceFBACode && pkg.Service.DomesticCarrier.Code == "") ||
			(pkg.Status == constant.PackageStatusPendingPickup && pkg.Alert == constant.PackageAlertTypeWarehoseReturn) {

			checkinPackage := entity.CheckinPackage{
				CheckinID: form.CheckinRequestID,
				PackageID: id,
				Status:    constant.CheckinPackageStatusInvalid,
			}

			err = h.PackageManager.SaveCheckinPackage(checkinPackage)
			if err != nil {
				h.Logger.Errorf("save checkin %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

				return
			}

			c.JSON(http.StatusOK, CreateTrackingResponse{StatusCheckin: checkinPackage.Status, Success: true})
			return
		}

		form.ActualHeight = utils.Ceil(form.ActualHeight, 2)
		form.ActualWidth = utils.Ceil(form.ActualWidth, 2)
		form.ActualLength = utils.Ceil(form.ActualLength, 2)
		form.ActualWeight = utils.Ceil(form.ActualWeight, 2)

		change := make(map[string]interface{})
		var alogs []entity.PackageAuditLog

		var extraFeePlus float64 = 0
		var shippingFeePlus float64 = 0
		var billID int64

		var priceByWeight bool = true
		var isUpdateVolume, isUpdateLabel bool = false, false

		if form.ActualWeight <= 0 {
			c.JSON(http.StatusBadRequest, "Khối lượng thực > 0")
			return
		}

		if form.ActualLength <= 0 {
			c.JSON(http.StatusBadRequest, "Chiều dài thực > 0")
			return
		}

		if form.ActualWidth <= 0 {
			c.JSON(http.StatusBadRequest, "Chiều rộng thực > 0")
			return
		}

		if form.ActualHeight <= 0 {
			c.JSON(http.StatusBadRequest, "Chiều cao thực > 0")
			return
		}

		if pkg.CountryCode == "AU" {
			_, _, ww := calculate.ParseVolumes(form.ActualLength, form.ActualHeight, form.ActualWidth)
			if ww < 5 {
				c.JSON(http.StatusBadRequest, "Tối thiểu 2 kích thước phải là 5 cm.")
				return
			}
		}

		if pkg.Service.Code == constant.ServiceFBACode {
			valid := order.MakeValidator(nil)
			valid.ValidateFbaServicePackage(&order.PackageResource{
				Country:     pkg.CountryCode,
				Address1:    pkg.Address1,
				Address2:    pkg.Address2,
				City:        pkg.City,
				State:       pkg.StateCode,
				Zipcode:     pkg.Zipcode,
				Weight:      form.ActualWeight,
				Width:       form.ActualWidth,
				Height:      form.ActualHeight,
				Length:      form.ActualLength,
				ServiceCode: pkg.Service.Code,
				Phone:       pkg.PhoneNumber,
			})

			if messages := valid.Errors(); len(messages) > 0 {
				c.JSON(http.StatusBadRequest, messages[0])
				return
			}
		} else {
			valid := order.MakeValidator(nil)
			valid.ValidateVolumes(&order.PackageResource{
				Country:     pkg.CountryCode,
				Weight:      form.ActualWeight,
				Width:       form.ActualWidth,
				Height:      form.ActualHeight,
				Length:      form.ActualLength,
				ServiceCode: pkg.Service.Code,
				Phone:       pkg.PhoneNumber,
			})

			if messages := valid.Errors(); len(messages) > 0 {
				c.JSON(http.StatusBadRequest, messages[0])
				return
			}
		}

		if pkg.ActualWeight == 0 {
			pkg.ActualWeight = pkg.Weight
			change["actual_weight"] = pkg.Weight
		}

		if pkg.ActualLength == 0 {
			pkg.ActualLength = pkg.Length
			change["actual_length"] = pkg.Length
		}

		if pkg.ActualWidth == 0 {
			pkg.ActualWidth = pkg.Width
			change["actual_width"] = pkg.Width
		}

		if pkg.ActualHeight == 0 {
			pkg.ActualHeight = pkg.Height
			change["actual_height"] = pkg.Height
		}

		customer, err := h.UserManager.GetUserByID(pkg.UserID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		var extraPeakFeePlus float64 = 0
		var BatteryFeePlus float64 = 0

		weight := form.ActualWeight
		if pkg.Service.Code != constant.ServiceFBACode {
			weight, err = h.CalculatePrice.PromotionFixWeight(c, pkg.UserID, form.ActualWeight)
			if err != nil {
				h.Logger.Errorf("promotion fix weight: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		}

		if form.ActualWeight > pkg.ActualWeight {
			isUpdateVolume = true
			alogs = append(alogs, entity.PackageAuditLog{
				Model:         dbgorm.Model{CreatedAt: time.Now(), UpdatedAt: time.Now()},
				PackageID:     pkg.ID,
				OldValue:      cast.ToString(pkg.Weight),
				Value:         cast.ToString(form.ActualWeight),
				UpdatedUserID: userID,
				Type:          constant.PackageUpdateTypeWeight,
			})

			extraPeakFeePlus = calculate.PeakFee(weight)
		}

		if pkg.Service.Code == constant.ServiceFBACode {
			extraPeakFeePlus = 0
		}

		extraFees, err := h.PackageManager.GetExtraFeeByPkgID(pkg.ID)
		if err != nil {
			h.Logger.Errorf("Get Package Deliver Logs %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		oldTotalAMount := pkg.ShippingFee
		for _, v := range extraFees {
			oldTotalAMount += v.Amount
		}

		// Check update extra fee
		if form.ActualLength > pkg.ActualLength || form.ActualWidth > pkg.ActualWidth || form.ActualHeight > pkg.ActualHeight {
			ol := pkg.ActualLength
			if pkg.Length > pkg.ActualLength {
				ol = pkg.Length
			}

			ow := pkg.ActualWidth
			if pkg.Width > pkg.ActualWidth {
				ow = pkg.Width
			}

			oh := pkg.ActualHeight
			if pkg.Height > pkg.ActualHeight {
				oh = pkg.Height
			}

			isUpdateVolume = true
			alogs = append(alogs, entity.PackageAuditLog{
				Model:         dbgorm.Model{CreatedAt: time.Now(), UpdatedAt: time.Now()},
				PackageID:     pkg.ID,
				OldValue:      fmt.Sprintf("%vx%vx%v", ol, ow, oh),
				Value:         fmt.Sprintf("%vx%vx%v", form.ActualLength, form.ActualWidth, form.ActualHeight),
				UpdatedUserID: userID,
				Type:          constant.PackageUpdateTypeVolume,
			})
		}

		var isPackageExceed bool = pkg.IsPackageExceed
		var shippingFee float64 = pkg.ShippingFee
		if isUpdateVolume {
			_, priceByWeight = calculate.CalcPriceWeight(weight, form.ActualLength, form.ActualHeight, form.ActualWidth, pkg.ServiceID)
			price, outSizePrice, err := h.CalculatePrice.Price3(c, pkg.UserID, pkg.ServiceID, customer.Class, weight, form.ActualLength, form.ActualHeight, form.ActualWidth, pkg.CountryCode)
			if err == calculate.ErrorNotService {
				c.JSON(http.StatusBadRequest, "Dịch vụ không hợp lệ")
				return
			}

			if pkg.CountryCode == "AU" || pkg.Service.Code == constant.ServiceFBACode || pkg.Service.Code == constant.ServiceINUSCode || pkg.Service.Code == constant.ServiceUS48Code || pkg.Service.Code == constant.ServiceAUCode || pkg.Service.Code == constant.ServiceEUCode {
				if err == calculate.ErrorMaxWeight {
					msg := "Trọng lượng cho phép vượt quá giới hạn"
					if price > 0 {
						msg = fmt.Sprintf("Trọng lượng không được vượt quá %v grams", math.Ceil(price)-1)
					}

					c.JSON(http.StatusBadRequest, msg)
					return
				}

				if err == calculate.ErrorMaxVolume {
					msg := "Kích thước vượt quá giới hạn cho phép"
					if price > 0 {
						msg = fmt.Sprintf("Kích thước không hợp lệ (LxHxW/5 <= %v)", math.Ceil(price)-1)
					}

					c.JSON(http.StatusBadRequest, msg)
					return
				}
			}

			isPackageExceed = false
			if pkg.Service.Code != constant.ServiceFBACode && (err == calculate.ErrorMaxWeight || err == calculate.ErrorMaxVolume) {
				isPackageExceed = true
			} else if err != nil {
				h.Logger.Errorf("parse body: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			if pkg.Service.Code != constant.ServiceFBACode && !isPackageExceed && !pkg.IsPackageExceed {
				shippingFee = price
				if price > pkg.ShippingFee || outSizePrice > 0 {
					extraFeePlus = outSizePrice

					if price > pkg.ShippingFee {
						shippingFeePlus = price - pkg.ShippingFee
					}

					for _, fee := range extraFees {
						if fee.ExtraFeeTypeID == constant.ExtraFeeTypeOutSize {
							extraFeePlus -= fee.Amount
						}

						if fee.ExtraFeeTypeID == constant.ExtraFeeTypeFixVolume || fee.ExtraFeeTypeID == constant.ExtraFeeTypeFixWeight || fee.ExtraFeeTypeID == constant.ExtraFeeService {
							shippingFeePlus -= fee.Amount
						}
					}
				}
			}

			if pkg.Service.Code == constant.ServiceFBACode {
				customerShipment, err := h.CustomerShipmentManager.GetCustomerShipment(sqlmanager.CustomerShipmentOption{ID: utils.Int64Value(pkg.CustomerShipmentID)})
				if err != nil {
					h.Logger.Infof("get customer shipment %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				price = h.CalculatePrice.GetFbaPackagePrice(customerShipment.Price, weight, form.ActualLength, form.ActualHeight, form.ActualWidth)
				shippingFeePlus = price - pkg.ShippingFee
				for _, fee := range extraFees {
					if fee.ExtraFeeTypeID == constant.ExtraFeeTypeFixVolume || fee.ExtraFeeTypeID == constant.ExtraFeeTypeFixWeight || fee.ExtraFeeTypeID == constant.ExtraFeeService {
						shippingFeePlus -= fee.Amount
					}
				}
			}

			if pkg.IncludeBattery {
				BatteryFeePlus = h.CalculatePrice.GetExtraFeeBaterry()
			}
		}

		if extraPeakFeePlus > 0 {
			for _, fee := range extraFees {
				if fee.ExtraFeeTypeID == constant.ExtraFeeTypePeak {
					extraPeakFeePlus -= fee.Amount
				}
			}
		}

		if BatteryFeePlus > 0 {
			for _, fee := range extraFees {
				if fee.ExtraFeeTypeID == constant.ExtraFeeTypeBattery {
					BatteryFeePlus -= fee.Amount
				}
			}
			alogs = append(alogs, entity.PackageAuditLog{
				PackageID:     pkg.ID,
				UpdatedUserID: userID,
				Type:          constant.PackageUpdateExtraFeeTypeBattery,
				Fee:           BatteryFeePlus,
			})
		}

		if form.ActualWeight != pkg.ActualWeight {
			change["actual_weight"] = form.ActualWeight
			pkg.ActualWeight = form.ActualWeight
			isUpdateLabel = true
		}

		if form.ActualLength != pkg.ActualLength {
			change["actual_length"] = form.ActualLength
			pkg.ActualLength = form.ActualLength
			isUpdateLabel = true
		}

		if form.ActualWidth != pkg.ActualWidth {
			change["actual_width"] = form.ActualWidth
			pkg.ActualWidth = form.ActualWidth
			isUpdateLabel = true
		}

		if form.ActualHeight != pkg.ActualHeight {
			change["actual_height"] = form.ActualHeight
			pkg.ActualHeight = form.ActualHeight
			isUpdateLabel = true
		}

		checkinPackage := entity.CheckinPackage{
			CheckinID: form.CheckinRequestID,
			PackageID: id,
			Status:    constant.CheckinPackageStatusSuccess,
		}

		fees := []entity.ExtraFee{}
		if pkg.Service.Code != constant.ServiceFBACode && !isPackageExceed && !pkg.IsPackageExceed {
			fees, err = h.CalculatePrice.PromotionExtras(pkg, pkg.ExtraFee, shippingFee)
			if err != nil {
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		}

		extraFeeService := h.CalculatePrice.GetServiceExtraFeee(form.ActualWidth, form.ActualLength, form.ActualHeight, *pkg.Service)
		if extraFeeService > 0 {
			for _, fee := range extraFees {
				if fee.ExtraFeeTypeID == constant.ExtraFeeTypeService {
					extraFeeService -= fee.Amount
				}
			}
			if extraFeeService > 0 {
				fees = append(fees, entity.ExtraFee{
					Amount:         extraFeeService,
					PackageID:      utils.Int64(pkg.ID),
					ExtraFeeTypeID: constant.ExtraFeeTypeService,
					Status:         constant.ExtraFeeStatusEnable,
				})
			}
		}

		if extraFeePlus > 0 || shippingFeePlus > 0 || extraPeakFeePlus > 0 || len(fees) > 0 || BatteryFeePlus > 0 {
			billID, err = h.BillManager.GetOrCreateNowBillID(pkg.UserID)
			if err != nil {
				h.Logger.Errorf("Get now bill %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		}

		var isCancelLabel bool
		var isPackageExceedOld = pkg.IsPackageExceed
		if isPackageExceed {
			change["is_package_exceed"] = isPackageExceed
			if !pkg.IsPackageExceed {
				isCancelLabel = true
			}
		}

		if pkg.Service.Code == constant.ServiceFBACode {
			change["status"] = constant.PackageStatusWareHouseLabeled
		}

		err = h.PackageManager.WarehousInCheck(checkinPackage, pkg, userID, pkg.UserID, change, alogs, shippingFeePlus, extraFeePlus, BatteryFeePlus, billID, priceByWeight, user, extraPeakFeePlus, fees)
		if err != nil {
			h.Logger.Errorf("update packae %v", err)
			checkinPackage.Status = constant.CheckinPackageStatusFailed
			err = h.PackageManager.SaveCheckinPackage(checkinPackage)
			if err != nil {
				h.Logger.Errorf("update packae %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
			c.JSON(http.StatusOK, CreateTrackingResponse{StatusCheckin: checkinPackage.Status, Success: true})
			return
		}

		if pkg.Service.Code == constant.ServiceFBACode {
			c.JSON(http.StatusOK, CreateTrackingResponse{StatusCheckin: checkinPackage.Status, Success: true})
			return
		}

		if pkg.CountryCode == "AU" {
			if err = h.PackageManager.UpdatePackage(&entity.Package{Status: constant.PackageStatusWareHouseLabeled}, pkg.ID); err != nil {
				h.Logger.Errorf("update packae %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
			c.JSON(http.StatusOK, CreateTrackingResponse{StatusCheckin: checkinPackage.Status, Success: true})
			return
		} else if pkg.Tracking != nil {
			//check if update tracking
			if isUpdateLabel {
				err = h.Redis.SAdd(c, rkey, id).Err()
				if err != nil {
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				pkg, err := h.PackageManager.GetPackageByPackageID(id)
				if err != nil {
					_ = h.Redis.SRem(c, rkey, id).Err()
					h.Logger.Errorf("get package update label err, %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				dbcarrier, err := h.ServiceManager.GetCarrierByID(pkg.Tracking.CarrierID)
				if err != nil {
					_ = h.Redis.SRem(c, rkey, id).Err()
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				carrier := providers.NewCarrier(dbcarrier.Code, pkg.UserID)
				if carrier == nil {
					_ = h.Redis.SRem(c, rkey, id).Err()
					h.Logger.Errorf("New carrier service %s not found", dbcarrier.Code)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				if dbcarrier.Code == providers.CarrierTypeAuspost && shippingFeePlus <= 0 {
					_ = h.Redis.SRem(c, rkey, id).Err()
					if err = h.PackageManager.UpdatePackage(&entity.Package{Status: constant.PackageStatusWareHouseLabeled}, pkg.ID); err != nil {
						h.Logger.Errorf("update packae %v", err)
						c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
						return
					}

					c.JSON(http.StatusOK, CreateTrackingResponse{StatusCheckin: checkinPackage.Status, Success: true})
					return
				}

				if dbcarrier.Code == providers.CarrierTypeShippo && pkg.IsPackageExceed {
					_, _, err := h.CalculatePrice.Price3(c, pkg.UserID, pkg.ServiceID, customer.Class, weight, form.ActualLength, form.ActualHeight, form.ActualWidth, pkg.CountryCode)
					isPackageExceed = false
					if err == calculate.ErrorMaxWeight || err == calculate.ErrorMaxVolume {
						isPackageExceed = true
					}
				}

				if dbcarrier.Code == providers.CarrierTypeShippo && shippingFeePlus <= 0 && !isPackageExceedOld && !isPackageExceed {
					_ = h.Redis.SRem(c, rkey, id).Err()
					if err = h.PackageManager.UpdatePackage(&entity.Package{Status: constant.PackageStatusWareHouseLabeled}, pkg.ID); err != nil {
						h.Logger.Errorf("update packae %v", err)
						c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
						return
					}

					c.JSON(http.StatusOK, CreateTrackingResponse{StatusCheckin: checkinPackage.Status, Success: true})
					return
				}

				labelType := createlabel.LabelTypeUpdate
				if dbcarrier.Code == providers.CarrierTypeShippo && (isPackageExceed || isPackageExceedOld || shippingFeePlus > 0) {
					isCancelLabel = true
					labelType = createlabel.LabelTypeNew
				}

				if !isPackageExceed && !isPackageExceedOld {
					fWeight, _, _, _, err := h.CreateLabel.Fake(c, pkg.ActualWeight, pkg.ActualLength, pkg.ActualHeight, pkg.ActualWidth)
					if err != nil {
						h.Redis.SRem(c, rkey, id).Err()
						h.Logger.Errorf("fake volume %v", err)
						c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
						return
					}

					// kiem tra thay doi service shippo
					if dbcarrier.Code == providers.CarrierTypeShippo && shippingFeePlus > 0 {
						isCancelLabel = true
					}

					if dbcarrier.Code != providers.CarrierTypeAuspost {
						carrierCode, err := h.CreateLabel.GetCarrierCode(c, entity.Package{
							Weight:          pkg.ActualWeight,
							Length:          pkg.ActualLength,
							Height:          pkg.ActualHeight,
							Width:           pkg.ActualWidth,
							IsPackageExceed: pkg.IsPackageExceed,
						}, pkg.UserID, fmt.Sprintf("Zone%d", pkg.Tracking.Zone))

						if err != nil {
							h.Redis.SRem(c, rkey, id).Err()
							h.Logger.Errorf("get carrier code %v", err)
							c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
							return
						}

						if shippingFeePlus > 0 {
							if dbcarrier.Code == providers.CarrierTypeShippo || (carrierCode != dbcarrier.Code && dbcarrier.Code == providers.CarrierTypeShippo) {
								isCancelLabel = true
							}
						}
					}

					if fWeight > pkg.Tracking.Weight && dbcarrier.Code == providers.CarrierTypeIBBlue {
						oOZ := pkg.Tracking.Weight * ibblue.GramToOz
						nOZ := fWeight * ibblue.GramToOz

						if oOZ <= ibblue.FirstClassLimit && nOZ > ibblue.FirstClassLimit {
							isCancelLabel = true
						}
					}
				}

				if isCancelLabel && pkg.CountryCode != "AU" {
					if _, err := order.CancelLabel(carrier, pkg); err != nil {
						h.Redis.SRem(c, rkey, id).Err()
						h.Logger.Errorf("cancel label %v", err)
						c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
						return
					}

					labelType = createlabel.LabelTypeNew
				}

				pkg.IsPackageExceed = isPackageExceed
				tracking, message, err := h.label(c, pkg, nil, carrier, labelType, dbcarrier.Code, pkg.Tracking.Zone)

				_ = h.Redis.SRem(c, rkey, id).Err()
				if message != "" {
					h.Logger.Errorf("Request update tracking: %v", message)
					checkinPackage.Status = constant.CheckinPackageStatusUpdateLabelFailed
					err = h.PackageManager.SaveCheckinPackage(checkinPackage)
					if err != nil {
						h.Logger.Errorf("update packae %v", err)
						c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
						return
					}

					c.JSON(http.StatusOK, CreateTrackingResponse{StatusCheckin: checkinPackage.Status, Success: true})
					return
				}

				if err != nil {
					h.Logger.Errorf("Request update tracking: %v", err)
					checkinPackage.Status = constant.CheckinPackageStatusUpdateLabelFailed
					err = h.PackageManager.SaveCheckinPackage(checkinPackage)
					if err != nil {
						h.Logger.Errorf("update packae %v", err)
						c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
						return
					}

					c.JSON(http.StatusOK, CreateTrackingResponse{StatusCheckin: checkinPackage.Status, Success: true})
					return
				}

				if labelType == createlabel.LabelTypeNew {
					checkinPackage.Status = constant.CheckinPackageStatusChangeLabel
					err = h.PackageManager.SaveCheckinPackage(checkinPackage)
					if err != nil {
						h.Logger.Errorf("update packae %v", err)
						c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
						return
					}
				}

				ext := "png"
				if carrier.GetCode() == providers.CarrierTypeDarius {
					ext = "pdf"
				}

				path, err := order.StoreLabelS3(h.StorageS3, tracking.LabelURL, ext, tracking.TrackingNumber)
				if err != nil {
					h.Logger.Errorf("store label: %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				tracking.LabelURL = path
				if labelType == createlabel.LabelTypeUpdate && tracking.TrackingNumber == pkg.Tracking.TrackingNumber {
					tracking.ID = pkg.Tracking.ID
				} else {
					tracking.UserID = userID
					tracking.Version = viper.GetString("tracking_version")

					if tracking.CarrierID < 1 {
						tracking.CarrierID = pkg.Tracking.CarrierID
					}
				}

				if labelType == createlabel.LabelTypeUpdate && tracking.TrackingNumber != pkg.Tracking.TrackingNumber {
					checkinPackage.Status = constant.CheckinPackageStatusChangeLabel
					err = h.PackageManager.SaveCheckinPackage(checkinPackage)
					if err != nil {
						h.Logger.Errorf("update packae %v", err)
						c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
						return
					}
				}

				if tracking.ID == 0 && oldTotalAMount > 0 {
					billID, err = h.BillManager.GetOrCreateNowBillID(pkg.UserID)
					if err != nil {
						h.Logger.Errorf("Get now bill %v", err)
						c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
						return
					}
				}

				extrafees := []entity.ExtraFee{}
				auditlogs := []entity.PackageAuditLog{}
				if isPackageExceed {
					pw, _ := calculate.CalcPriceWeight(pkg.ActualWeight, pkg.ActualLength, pkg.ActualHeight, pkg.ActualWidth, pkg.ServiceID)
					price, err := h.CalculatePrice.CalculateExceedPackagePrice(pw, tracking.ShipmentCost)
					if err != nil {
						h.Logger.Errorf("calculate exceed package price: %v", err)
						c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
						return
					}

					oldExtraFees, err := h.PackageManager.GetExtraFeeByPkgID(pkg.ID)
					if err != nil {
						h.Logger.Errorf("Get Package Deliver Logs %v", err)
						c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
						return
					}

					totalAMount := pkg.ShippingFee
					for _, v := range oldExtraFees {
						if v.ExtraFeeTypeID == constant.ExtraFeeTypeFixVolume ||
							v.ExtraFeeTypeID == constant.ExtraFeeTypeFixWeight ||
							v.ExtraFeeTypeID == constant.ExtraFeeTypeEditAddress ||
							v.ExtraFeeTypeID == constant.ExtraFeeService {
							totalAMount += v.Amount
						}
					}

					promtionsExtraFees, err := h.CalculatePrice.PromotionExtras(pkg, oldExtraFees, price)
					if err != nil {
						h.Logger.Errorf("PromotionExtras: %v", err)
						c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
						return
					}

					extrafees = append(extrafees, promtionsExtraFees...)

					if price > totalAMount {
						if billID < 1 {
							billID, err = h.BillManager.GetOrCreateNowBillID(pkg.UserID)
							if err != nil {
								h.Logger.Errorf("Get now bill %v", err)
								c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
								return
							}
						}

						extraType := int64(constant.ExtraFeeTypeFixVolume)
						if priceByWeight {
							extraType = constant.ExtraFeeTypeFixWeight
						}

						amount := price - totalAMount
						extrafees = append(extrafees, entity.ExtraFee{
							Model: dbgorm.Model{
								CreatedAt: time.Now(),
								UpdatedAt: time.Now(),
							},
							PackageID:      utils.Int64(pkg.ID),
							BillID:         utils.Int64(billID),
							ExtraFeeTypeID: extraType,
							Status:         constant.ExtraFeeStatusEnable,
							Amount:         utils.Ceil(amount, 2),
						})

						for _, v := range alogs {
							if v.Type == constant.PackageUpdateTypeWeight && priceByWeight {
								auditlogs = append(auditlogs, entity.PackageAuditLog{
									Model:         dbgorm.Model{CreatedAt: time.Now(), UpdatedAt: time.Now()},
									PackageID:     pkg.ID,
									OldValue:      v.OldValue,
									Value:         v.Value,
									UpdatedUserID: userID,
									Type:          constant.PackageUpdateTypeWeight,
									Fee:           utils.Ceil(amount, 2),
								})
							}

							if v.Type == constant.PackageUpdateTypeVolume && !priceByWeight {
								auditlogs = append(auditlogs, entity.PackageAuditLog{
									Model:         dbgorm.Model{CreatedAt: time.Now(), UpdatedAt: time.Now()},
									PackageID:     pkg.ID,
									OldValue:      v.OldValue,
									Value:         v.Value,
									UpdatedUserID: userID,
									Type:          constant.PackageUpdateTypeVolume,
									Fee:           utils.Ceil(amount, 2),
								})
							}
						}
					}
				}

				err = h.TrackingManager.UpdateTracking(billID, tracking, pkg, userID, oldTotalAMount, extrafees, auditlogs)
				if err != nil {
					h.Logger.Errorf("update tracking %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}
				// if order.CheckIsManifestNow() {
				// 	order.SendQueueManifest(h.Producer, []int64{pkg.ID}, true)
				// }
				h.Logger.Info("================", pkg.ID)
			} else {
				if err = h.PackageManager.UpdatePackage(&entity.Package{Status: constant.PackageStatusWareHouseLabeled}, pkg.ID); err != nil {
					h.Logger.Errorf("update packae %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}
			}

			c.JSON(http.StatusOK, CreateTrackingResponse{StatusCheckin: checkinPackage.Status, Success: true})
			return
		} else {
			payload := entity.QueuePushCreateLabel{
				PackageID:       id,
				UserID:          userID,
				Carrier:         pkg.Service.DomesticCarrier.Code,
				IsPackageExceed: isPackageExceed,
			}

			// buf, err := json.Marshal(payload)
			// if err != nil {
			// 	c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			// 	return
			// }

			// if err := h.Producer.PublishSimple(constant.QueueCreateTracking, buf); err != nil {
			// 	c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			// 	return
			// }

			err = h.Redis.SAdd(c, rkey, payload.PackageID).Err()
			if err != nil {
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			c.JSON(http.StatusOK, CreateTrackingResponse{StatusCheckin: checkinPackage.Status, Success: true})
		}
	}
}

func (h *WarehouseHandler) Accept() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		if role == constant.UserRoleCustomer {

			userCustomerForLionnixCreateLabel := viper.GetInt64("label_for_lionnix.user_id")
			if userCustomerForLionnixCreateLabel <= 0 || userID <= 0 ||
				userID != userCustomerForLionnixCreateLabel {
				c.JSON(http.StatusUnauthorized, "Tài khoản ananbay không hợp lệ")
				return
			}
		}

		id := cast.ToInt64(c.Param("package_id"))
		if id < 1 {
			c.JSON(http.StatusBadRequest, "ID vận đơn không đước trống")
			return
		}

		form := &formPackageChecked{}
		if err := c.ShouldBindJSON(form); err != nil {
			h.Logger.Error("parse body: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		pkg, err := h.PackageManager.GetPackageByPackageID(id)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, "Id đơn hàng không tồn tại")
			return
		}
		if err != nil {
			h.Logger.Error("get package: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if pkg.Service != nil && pkg.Service.Code == constant.ServiceFBACode {
			c.JSON(http.StatusBadRequest, "Đơn hàng đi FBA không hợp lệ")
			return
		}

		if pkg.WarehouseID > 0 {
			if user.Role == constant.UserRoleWarehouse && pkg.WarehouseID != user.WarehouseID {
				c.JSON(http.StatusNotFound, "Đơn hàng không nằm trong kho")
			}
		}

		if pkg.Status == constant.PackageStatusCancelled {
			c.JSON(http.StatusBadRequest, "Đơn đã được hủy")
			return
		}

		if pkg.Status == constant.PackageStatusReturned {
			c.JSON(http.StatusBadRequest, "Đơn đã được hoàn trả")
			return
		}

		if pkg.Status != constant.PackageStatusPicked {
			c.JSON(http.StatusBadRequest, "Trạng thái đơn hàng không hợp lệ.")
			return
		}

		if pkg.Tracking != nil {
			c.JSON(http.StatusOK, packageCheckedResponse{
				ID:             pkg.ID,
				TrackID:        pkg.Tracking.ID,
				LabelBase64:    pkg.Tracking.LabelURL,
				LabelURL:       pkg.Tracking.LabelURL,
				TrackingNumber: pkg.Tracking.TrackingNumber,
			})
			return
		}

		customer, err := h.UserManager.GetUserByID(pkg.UserID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		change := make(map[string]interface{})
		var alogs []entity.PackageAuditLog

		var extraFeePlus float64 = 0
		var batteryFeePlus float64 = 0
		var extraPeakFeePlus float64 = 0
		var shippingFeePlus float64 = 0
		var billID int64

		var priceByWeight bool = true
		var isPackageExceed bool = pkg.IsPackageExceed

		if form.Weight <= 0 {
			c.JSON(http.StatusBadRequest, "Khối lượng không hợp lệ")
			return
		}

		if form.Length <= 0 {
			c.JSON(http.StatusBadRequest, "Chiều dài không hợp lệ")
			return
		}

		if form.Width <= 0 {
			c.JSON(http.StatusBadRequest, "Chiều rộng không hợp lệ")
			return
		}

		if form.Height <= 0 {
			c.JSON(http.StatusBadRequest, "Chiều cao không hợp lệ")
			return
		}

		weight, err := h.CalculatePrice.PromotionFixWeight(c, pkg.UserID, form.Weight)
		if err != nil {
			h.Logger.Errorf("promotion fix weight: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		newExtraFees := []entity.ExtraFee{}

		// Check update extra fee
		if form.Weight > pkg.Weight || form.Length > pkg.Length || form.Width > pkg.Width || form.Height > pkg.Height {
			valid := order.MakeValidator(nil)
			valid.ValidateVolumes(&order.PackageResource{
				Country: pkg.CountryCode,
				Weight:  form.Weight,
				Width:   form.Width,
				Height:  form.Height,
				Length:  form.Length,
			})

			if messages := valid.Errors(); len(messages) > 0 {
				c.JSON(http.StatusBadRequest, messages[0])
				return
			}

			if form.Weight > pkg.Weight {
				extraPeakFeePlus = calculate.PeakFee(weight)
			}

			if pkg.Service.Code == constant.ServiceFBACode {
				extraPeakFeePlus = 0
			}

			if (form.Weight != pkg.Weight && pkg.ActualWeight == 0) || (form.Weight != pkg.ActualWeight && pkg.ActualWeight > 0) {
				old := cast.ToString(pkg.Weight)

				if form.Weight != pkg.ActualWeight && pkg.ActualWeight > 0 {
					old = cast.ToString(pkg.ActualWeight)
				}

				alogs = append(alogs, entity.PackageAuditLog{
					Model:         dbgorm.Model{CreatedAt: time.Now(), UpdatedAt: time.Now()},
					PackageID:     pkg.ID,
					OldValue:      old,
					Value:         fmt.Sprintf("%v", utils.Ceil(form.Weight, 2)),
					UpdatedUserID: userID,
					Type:          constant.PackageUpdateTypeWeight,
				})

				pkg.ActualWeight = utils.Ceil(form.Weight, 2)
				//change["weight"] = form.Weight
			}

			if ((form.Length != pkg.Length || form.Width != pkg.Width || form.Height != pkg.Height) && pkg.ActualLength == 0) ||
				((form.Length != pkg.ActualLength || form.Width != pkg.ActualWidth || form.Height != pkg.ActualHeight) && pkg.ActualLength > 0) {
				oldValue := fmt.Sprintf("%vx%vx%v", pkg.Length, pkg.Width, pkg.Height)
				if pkg.ActualLength > 0 {
					oldValue = fmt.Sprintf("%vx%vx%v", pkg.ActualLength, pkg.ActualWidth, pkg.ActualHeight)
				}

				//fix me
				alogs = append(alogs, entity.PackageAuditLog{
					Model:         dbgorm.Model{CreatedAt: time.Now(), UpdatedAt: time.Now()},
					PackageID:     pkg.ID,
					OldValue:      oldValue,
					Value:         fmt.Sprintf("%vx%vx%v", utils.Ceil(form.Length, 2), utils.Ceil(form.Width, 2), utils.Ceil(form.Height, 2)),
					UpdatedUserID: userID,
					Type:          constant.PackageUpdateTypeVolume,
				})

				if form.Length != pkg.Length {
					pkg.ActualLength = utils.Ceil(form.Length, 2)
				}

				if form.Width != pkg.Width {
					pkg.ActualWidth = utils.Ceil(form.Width, 2)
				}

				if form.Height != pkg.Height {
					pkg.ActualHeight = utils.Ceil(form.Height, 2)
				}
			}

			_, priceByWeight = calculate.CalcPriceWeight(weight, form.Length, form.Height, form.Width, pkg.ServiceID)

			price, outSizePrice, err := h.CalculatePrice.Price3(c, pkg.UserID, pkg.ServiceID, customer.Class, weight, form.Length, form.Height, form.Width, pkg.CountryCode)
			if err == calculate.ErrorNotService {
				c.JSON(http.StatusBadRequest, "Dịch vụ không hợp lệ")
				return
			}

			// if pkg.CountryCode == "AU" {
			if err == calculate.ErrorMaxWeight {
				msg := "Trọng lượng cho phép vượt quá giới hạn"
				if price > 0 {
					msg = fmt.Sprintf("Trọng lượng không được vượt quá %v grams", math.Ceil(price)-1)
				}

				c.JSON(http.StatusBadRequest, msg)
				return
			}

			if err == calculate.ErrorMaxVolume {
				msg := "Kích thước vượt quá giới hạn cho phép"
				if price > 0 {
					msg = fmt.Sprintf("Kích thước không hợp lệ (LxHxW/5 <= %v)", math.Ceil(price)-1)
				}

				c.JSON(http.StatusBadRequest, msg)
				return
			}
			// }

			isPackageExceed = false
			if err != nil {
				h.Logger.Errorf("parse body: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			if !isPackageExceed && !pkg.IsPackageExceed {
				fees, err := h.CalculatePrice.PromotionExtras(pkg, pkg.ExtraFee, price)
				if err != nil {
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				newExtraFees = fees
			}

			extraFees, err := h.PackageManager.GetExtraFeeByPkgID(pkg.ID)
			if err != nil {
				h.Logger.Errorf("Get Package Deliver Logs %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			if !isPackageExceed && !pkg.IsPackageExceed && (price > pkg.ShippingFee || outSizePrice > 0) {
				extraFeePlus = outSizePrice

				if price > pkg.ShippingFee {
					shippingFeePlus = price - pkg.ShippingFee
				}

				for _, fee := range extraFees {
					if fee.ExtraFeeTypeID == constant.ExtraFeeTypeOutSize {
						extraFeePlus -= fee.Amount
					}

					if fee.ExtraFeeTypeID == constant.ExtraFeeTypeFixVolume || fee.ExtraFeeTypeID == constant.ExtraFeeTypeFixWeight || fee.ExtraFeeTypeID == constant.ExtraFeeService {
						shippingFeePlus -= fee.Amount
					}
				}
			}

			if pkg.IncludeBattery {
				batteryFeePlus = h.CalculatePrice.GetExtraFeeBaterry()
			}

			if extraPeakFeePlus > 0 {
				for _, fee := range extraFees {
					if fee.ExtraFeeTypeID == constant.ExtraFeeTypePeak {
						extraPeakFeePlus -= fee.Amount
					}
				}
			}

			if batteryFeePlus > 0 {
				for _, fee := range extraFees {
					if fee.ExtraFeeTypeID == constant.ExtraFeeTypeBattery {
						batteryFeePlus -= fee.Amount
					}
				}

				alogs = append(alogs, entity.PackageAuditLog{
					PackageID:     pkg.ID,
					UpdatedUserID: userID,
					Type:          constant.PackageUpdateExtraFeeTypeBattery,
					Fee:           batteryFeePlus,
				})
			}

			extraFeeService := h.CalculatePrice.GetServiceExtraFeee(form.Width, form.Length, form.Height, *pkg.Service)
			if extraFeeService > 0 {
				for _, fee := range extraFees {
					if fee.ExtraFeeTypeID == constant.ExtraFeeTypeService {
						extraFeeService -= fee.Amount
					}
				}
				if extraFeeService > 0 {
					newExtraFees = append(newExtraFees, entity.ExtraFee{
						Amount:         extraFeeService,
						PackageID:      utils.Int64(pkg.ID),
						ExtraFeeTypeID: constant.ExtraFeeTypeService,
						Status:         constant.ExtraFeeStatusEnable,
					})
				}
			}

		} else {
			isExceed, err := h.CalculatePrice.CheckIsExceed(c, pkg.UserID, pkg.ServiceID, customer.Class, weight, form.Length, form.Height, form.Width)
			if err != nil {
				h.Logger.Errorf("CheckIsExceed: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			isPackageExceed = isExceed
		}

		if extraFeePlus > 0 || shippingFeePlus > 0 || extraPeakFeePlus > 0 || len(newExtraFees) > 0 {
			billID, err = h.BillManager.GetOrCreateNowBillID(pkg.UserID)
			if err != nil {
				h.Logger.Errorf("Get now bill %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		}

		if form.Weight > 0 {
			pkg.ActualWeight = utils.Ceil(form.Weight, 2)
			change["actual_weight"] = utils.Ceil(form.Weight, 2)
		}
		if form.Length > 0 {
			pkg.ActualLength = utils.Ceil(form.Length, 2)
			change["actual_length"] = utils.Ceil(form.Length, 2)
		}
		if form.Width > 0 {
			pkg.ActualWidth = utils.Ceil(form.Width, 2)
			change["actual_width"] = utils.Ceil(form.Width, 2)
		}
		if form.Height > 0 {
			pkg.ActualHeight = utils.Ceil(form.Height, 2)
			change["actual_height"] = utils.Ceil(form.Height, 2)
		}

		if pkg.Service == nil {
			c.JSON(http.StatusBadRequest, "Loại vận chuyển không hợp lệ")
			return
		}

		carrier := providers.NewCarrier(pkg.Service.DomesticCarrier.Code, pkg.UserID)
		if carrier == nil {
			c.JSON(http.StatusBadRequest, "Loại vận chuyển không hợp lệ")
			return
		}

		warehouses, err := h.WareHouseManager.GetEstimateCostByWareHouse(pkg.ID)

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get warehouse error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if form.HubID != nil {
			hub, err := h.WareHouseManager.GetWareHouse(sqlmanager.OptionWareHouse{
				ID: utils.Int64Value(form.HubID),
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
			if strings.ToUpper(hub.Country) != strings.ToUpper(pkg.CountryCode) {
				c.JSON(http.StatusBadRequest, "Hub chọn không cùng quốc gia với đơn hàng")
				return
			}
		}

		var msg []string

		if len(warehouses) == 0 || form.HubID != nil {
			h.IBBlue = ibblue.NewIBBlue(nil)
			isWl := utils.IsStateWhiteList(pkg.UserID)
			igState := ""
			if isWl {
				igState = "CA"
			}
			wareHouses, err := h.WareHouseManager.GetWareHouses(sqlmanager.OptionWareHouse{
				Type:        constant.WareHouseTypeInternational,
				IgnoreState: igState,
			})

			if err != nil {
				h.Logger.Errorf("Get warehouses error, %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			if err := h.WareHouseManager.DeactivateOldCost(*pkg); err != nil {
				h.Logger.Errorf("Estimate cost err: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			var wg sync.WaitGroup
			for _, wareHouse := range wareHouses {
				wg.Add(1)

				go func(wareHouse entity.Warehouse) {
					defer wg.Done()

					if wareHouse.Country != pkg.CountryCode {
						return
					}

					cost := &entity.PackageWarehouseCost{
						PackageID: pkg.ID,
						HubID:     wareHouse.ID,
						Warehouse: &wareHouse,
						Cost:      0,
						OrgCost:   0,
					}

					resultOrg, s, err := order.EstimateCost(carrier, pkg, wareHouse)
					if s != "" {
						h.Logger.Errorf("Estimate cost org err: %v", s)
						if pkg.CountryCode == "AU" {
							return
						}
					} else if err != nil {
						h.Logger.Errorf("Estimate cost org err: %v", err)
						if pkg.CountryCode == "AU" {
							return
						}
					} else {
						cost.OrgCost = resultOrg.TotalCost + wareHouse.HandlingFee
						cost.Zone = resultOrg.Zone
						cost.Cost = cost.OrgCost
					}

					if pkg.CountryCode == "US" {
						clone := &entity.Package{}
						if err := utils.DeepCopy(pkg, clone); err != nil {
							h.Logger.Errorf("deep copy: %v", err)
							c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
							return
						}

						//estimate cost fake
						weight, length, height, width, err := h.CreateLabel.Fake(c, clone.ActualWeight, clone.ActualLength, clone.ActualHeight, clone.ActualWidth)
						if err != nil {
							h.Logger.Errorf("fake volume: %v", err)
							return
						}

						clone.ActualWeight = weight
						clone.ActualLength = length
						clone.ActualHeight = height
						clone.ActualWidth = width
						result, s, err := order.EstimateCost(carrier, clone, wareHouse)
						if s != "" {
							msg = append(msg, s)
							h.Logger.Errorf("Estimate cost err: %v", s)
							return
						}

						if err != nil {
							h.Logger.Errorf("Estimate cost err: %v", err)
							return
						}

						cost.Cost = result.TotalCost + wareHouse.HandlingFee
						cost.Zone = result.Zone
					}

					if pkg.Service != nil {
						cost.CarrierID = pkg.Service.DomesticCarrierID
					}

					err = h.WareHouseManager.CreateEstimateCost(cost)
					if err != nil {
						h.Logger.Errorf("Create estimate cost err: %v", err)
						return
					}

					if wareHouse.Status == constant.WareHouseStatusActive {
						warehouses = append(warehouses, *cost)
					}

				}(wareHouse)
			}

			wg.Wait()
		}

		if len(warehouses) == 0 {
			if len(msg) > 0 {
				c.JSON(http.StatusBadRequest, msg[0])
			} else {
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			}
		}

		minOb := warehouses[0]

		for _, wh := range warehouses {
			if form.HubID != nil && wh.HubID == utils.Int64Value(form.HubID) {
				minOb = wh
				break
			}
			if wh.Cost < minOb.Cost {
				minOb = wh
			}
		}

		rKey := "package_call_label_1"
		isCallLabel, err := h.Redis.SIsMember(c, rKey, pkg.ID).Result()
		if err != nil {
			h.Logger.Errorf("Get redis error %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if isCallLabel {
			c.JSON(http.StatusBadRequest, "Đơn hàng đã đang được tạo label, thử lại sau !")
			return
		}

		err = h.Redis.SAdd(c, rKey, pkg.ID).Err()
		if err != nil {
			h.Logger.Errorf("Save redis error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		pkg.IsPackageExceed = isPackageExceed
		//check is white list user
		// isWlUser := false
		// wL := strings.Split(viper.GetString("white_list.label"), ",")
		// if len(wL) > 0 {
		// 	for _, email := range wL {
		// 		if email == customer.Email {
		// 			isWlUser = true
		// 		}
		// 	}
		// }

		// tracking, message, err := h.label(c, pkg, carrier, minOb.Warehouse, form.PostmarkDate, isWlUser, pkg.Service.DomesticCarrier.Code, minOb.Zone)

		tracking, message, err := h.label(c, pkg, minOb.Warehouse, carrier, createlabel.LabelTypeNew, pkg.Service.DomesticCarrier.Code, minOb.Zone)
		if err := h.Redis.SRem(c, rKey, pkg.ID).Err(); err != nil {
			h.Logger.Errorf("Delete redis error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if message != "" {
			c.JSON(http.StatusBadRequest, message)
			return
		}

		if err != nil {
			h.Logger.Errorf("create label %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		ext := "png"
		if carrier.GetCode() == providers.CarrierTypeDarius {
			ext = "pdf"
		}
		path, err := order.StoreLabelS3(h.StorageS3, tracking.LabelURL, ext, tracking.TrackingNumber)
		if err != nil {
			h.Logger.Errorf("store label: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		tracking.LabelURL = path
		tracking.UserID = userID
		tracking.Version = viper.GetString("tracking_version")
		labelBase64 := tracking.LabelURL

		if tracking.CarrierID < 1 {
			tracking.CarrierID = pkg.Service.DomesticCarrierID
		}

		if isPackageExceed {
			if billID < 1 {
				billID, err = h.BillManager.GetOrCreateNowBillID(pkg.UserID)
				if err != nil {
					h.Logger.Errorf("Get now bill %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}
			}

			change["is_package_exceed"] = isPackageExceed

			pw, _ := calculate.CalcPriceWeight(pkg.ActualWeight, pkg.ActualLength, pkg.ActualHeight, pkg.ActualWidth, pkg.ServiceID)
			cost, err := h.CalculatePrice.CalculateExceedPackagePrice(pw, tracking.ShipmentCost)
			if err != nil {
				h.Logger.Errorf("calculate exceed package: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			amount := pkg.ShippingFee
			extraFees, err := h.PackageManager.GetExtraFeeByPkgID(pkg.ID)
			if err != nil {
				h.Logger.Errorf("Get Package Deliver Logs %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			for _, v := range extraFees {
				if v.ExtraFeeTypeID == constant.ExtraFeeTypeFixVolume ||
					v.ExtraFeeTypeID == constant.ExtraFeeTypeFixWeight ||
					v.ExtraFeeTypeID == constant.ExtraFeeService ||
					v.ExtraFeeTypeID == constant.ExtraFeeTypeEditAddress {
					amount += v.Amount
				}
			}

			if cost > amount {
				shippingFeePlus = utils.Ceil(cost-amount, 2)

				fees, err := h.CalculatePrice.PromotionExtras(pkg, extraFees, cost)
				if err != nil {
					h.Logger.Errorf("PromotionExtras: %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				newExtraFees = append(newExtraFees, fees...)
			}
		}

		err = h.PackageManager.WarehousChecked(pkg, userID, pkg.UserID, change, tracking, alogs, shippingFeePlus, extraFeePlus, batteryFeePlus, billID, priceByWeight, extraPeakFeePlus, newExtraFees)
		if err != nil {
			h.Logger.Errorf("update packae %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		// h.Logger.Info("====================ids", pkg.ID)
		// push to webhook url
		// data := struct {
		// 	LionbayTracking  string `json:"lionbay_tracking"`
		// 	LastMileTracking string `json:"last_mile_tracking"`
		// }{
		// 	LionbayTracking:  pkg.PackageCode.Code,
		// 	LastMileTracking: tracking.TrackingNumber,
		// }
		// loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
		// now := time.Now().In(loc).Format("2006-01-02 15:04:05")
		// body := struct {
		// 	UserID    int64       `json:"user_id"`
		// 	EventName string      `json:"event_name"`
		// 	TimeStamp *string     `json:"time_stamp"`
		// 	Data      interface{} `json:"data"`
		// }{
		// 	UserID:    pkg.UserID,
		// 	EventName: constant.EventTrackingCreated,
		// 	TimeStamp: &now,
		// 	Data:      data,
		// }

		// buf, err := json.Marshal(body)

		// if err != nil {
		// 	h.Logger.Errorf("Marshal error %v", err)
		// 	c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
		// 	return
		// }

		// err = h.Producer.PublishSimple(constant.QueuePushWebhook, buf)

		// if err != nil {
		// 	h.Logger.Errorf("Publish to queue error: %v", err)
		// 	c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
		// 	return
		// }

		c.JSON(http.StatusOK, packageCheckedResponse{
			ID:             pkg.ID,
			TrackID:        tracking.ID,
			LabelBase64:    labelBase64,
			LabelURL:       path,
			TrackingNumber: tracking.TrackingNumber,
		})
	}
}

func (h *WarehouseHandler) Return() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))

		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		form := FormPackageReturn{}
		if err := json.NewDecoder(c.Request.Body).Decode(&form); err != nil {
			h.Logger.Error("parse body: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if len(form.IDs) < 1 {
			c.JSON(http.StatusBadRequest, "Missing package id")
			return
		}

		if form.Note == "" {
			c.JSON(http.StatusBadRequest, "Lý do trả hàng không được trống")
			return
		}

		if form.CheckinRequestID > 0 {
			checkin, err := h.CheckinManager.GetCheckinByID(form.CheckinRequestID)
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusBadRequest, "Mã vào kho không tồn tại, vui lòng load lại trang và thao tác lại")
				return
			}

			if err != nil {
				h.Logger.Errorf("Get checkin request: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			if checkin.Status == constant.CheckinStatusClosed {
				c.JSON(http.StatusBadRequest, "Mã vào kho đã được đóng, vui lòng load lại trang và thao tác lại")
				return
			}
		}

		logs := []entity.PackageDeliverLog{}
		for _, id := range form.IDs {
			rkey := fmt.Sprintf("create_label_%d", id)
			result, err := h.Redis.Get(c, rkey).Int()
			if err != nil && err != redis.Nil {
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			if result > 0 {
				c.JSON(http.StatusBadRequest, fmt.Sprintf("ID đơn hàng %d  đang được tạo mã tracking", id))
				return
			}

			pkg, err := h.PackageManager.GetPackageByPackageID(id)
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, constant.MessageNotFound)
				return
			}

			if err != nil {
				h.Logger.Error("get package: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			if pkg.Status != constant.PackageStatusPicked && pkg.Status != constant.PackageStatusPendingPickup {
				c.JSON(http.StatusBadRequest, fmt.Sprintf("Trạng thái đơn hàng: %v không hợp lệ", pkg.PackageCode.Code))
				return
			}

			if pkg.Status == constant.PackageStatusPendingPickup && pkg.Alert == constant.PackageAlertTypeWarehoseReturn {
				c.JSON(http.StatusBadRequest, fmt.Sprintf("Đơn hàng: %v đã trả hàng", pkg.PackageCode.Code))
				return
			}

			wID := viper.GetInt64("warehouse_vn_default")
			if user.Role == constant.UserRoleWarehouse {
				wID = user.WarehouseID
			}

			wh, err := h.PackageManager.GetPackageWarehouse(wID)
			if err != nil {
				h.Logger.Errorf("Get warehouse error %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			log := entity.PackageDeliverLog{
				PackageID:   pkg.ID,
				Location:    fmt.Sprintf("%s, %s", wh.City, wh.Country),
				Description: form.Note,
				Status:      constant.DeliverLogTebexpressReturned,
				Type:        constant.PackageDeliverLogTypeReturned,
				UserID:      &userID,
			}
			logs = append(logs, log)
		}

		if err := h.PackageManager.ReturnPackage(form.IDs, form.CheckinRequestID, logs, nil); err != nil {
			h.Logger.Error("get package: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, PackageReturnResponse{Success: true})
	}
}

func (h *WarehouseHandler) CheckReLabel() gin.HandlerFunc {
	return func(c *gin.Context) {
		form := &CheckRelabelWarehouseForm{}
		if err := c.ShouldBindJSON(form); err != nil {
			h.Logger.Errorf("Bad body request: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		id := cast.ToInt64(c.Param("package_id"))
		if id < 1 || form.Weight == 0 || form.Width == 0 || form.Length == 0 || form.Height == 0 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		weightAfter, _, _, _, _ := h.CreateLabel.Fake(c, form.Weight, form.Length, form.Height, form.Width)

		pkg, err := h.PackageManager.GetPackageByPackageID(id)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, "Id đơn hàng không tồn tại")
			return
		}
		if err != nil {
			h.Logger.Error("get package: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if pkg.Tracking == nil {
			c.JSON(http.StatusOK, CheckRelabelWarehouseResponse{false})
			return
		}

		if pkg.CountryCode == "AU" {
			c.JSON(http.StatusOK, CheckRelabelWarehouseResponse{true})
			return
		}

		weightBefore, _, _, _, _ := h.CreateLabel.Fake(c, pkg.Weight, pkg.Length, pkg.Height, pkg.Width)
		var isRelabel bool
		if weightAfter >= constant.MaxWeightShippingPackage && weightBefore < constant.MaxWeightShippingPackage {
			isRelabel = true
		}

		customer, err := h.UserManager.GetUserByID(pkg.UserID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		price, _, err := h.CalculatePrice.Price3(c, pkg.UserID, pkg.ServiceID, customer.Class, form.Weight, form.Length, form.Height, form.Width, pkg.CountryCode)
		if err == calculate.ErrorNotService {
			c.JSON(http.StatusBadRequest, "Dịch vụ không hợp lệ")
			return
		}

		if err == calculate.ErrorNotPrice {
			c.JSON(http.StatusBadRequest, "Giá không hợp lệ")
			return
		}

		if err == calculate.ErrorMaxWeight || err == calculate.ErrorMaxVolume {
			if !pkg.IsPackageExceed {
				isRelabel = true
			}
		} else if err != nil {
			h.Logger.Errorf("parse body: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		dbcarrier, err := h.ServiceManager.GetCarrierByID(pkg.Tracking.CarrierID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if dbcarrier.Code == providers.CarrierTypeShippo {
			if pkg.IsPackageExceed {
				c.JSON(http.StatusOK, CheckRelabelWarehouseResponse{true})
				return
			}

			extraFees, err := h.PackageManager.GetExtraFeeByPkgID(pkg.ID)
			if err != nil {
				h.Logger.Errorf("Get Package Deliver Logs %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			var shippingFeePlus float64 = price - pkg.ShippingFee
			for _, fee := range extraFees {
				if fee.ExtraFeeTypeID == constant.ExtraFeeTypeFixVolume || fee.ExtraFeeTypeID == constant.ExtraFeeTypeFixWeight || fee.ExtraFeeTypeID == constant.ExtraFeeService {
					shippingFeePlus -= fee.Amount
				}
			}

			if shippingFeePlus > 0 {
				isRelabel = true
			}
		}

		c.JSON(http.StatusOK, CheckRelabelWarehouseResponse{isRelabel})
	}
}

func (h *WarehouseHandler) label(c context.Context, sp *entity.Package, ware *entity.Warehouse, carrier providers.Carrier, labalType int, oldCarrierCode string, zone int) (*entity.Tracking, string, error) {
	body := providers.RequestCreateLabel{
		Company:      sp.Company,
		FirstName:    sp.Recipient,
		LastName:     sp.Recipient,
		FullName:     sp.Recipient,
		City:         sp.City,
		Address1:     sp.Address1,
		Address2:     sp.Address2,
		State:        sp.StateCode,
		Zipcode:      sp.Zipcode,
		Phone:        sp.PhoneNumber,
		Country:      sp.CountryCode,
		Weight:       sp.ActualWeight,
		Height:       sp.ActualHeight,
		Length:       sp.ActualLength,
		Width:        sp.ActualWidth,
		DistanceUnit: "in",
	}

	tracking := &entity.Tracking{
		PackageID: sp.ID,
		Weight:    sp.ActualWeight,
		Length:    sp.ActualLength,
		Width:     sp.ActualWidth,
		Height:    sp.ActualHeight,
	}

	if labalType == createlabel.LabelTypeNew {
		warehouse, err := h.WareHouseManager.GetWareHouse(sqlmanager.OptionWareHouse{ID: utils.Int64Value(&ware.ID), Status: constant.WareHouseStatusActive})
		if err == gorm.ErrRecordNotFound {
			warehouse, err = h.WareHouseManager.GetWareHouseMostPrice(sp.ID)
			if err != nil {
				return nil, "", err
			}
		} else if err != nil {
			return nil, "", err
		}

		body = providers.RequestCreateLabel{
			ID:           sp.ID,
			OrderNumber:  sp.OrderNumber,
			Code:         sp.PackageCode.Code,
			Weight:       sp.ActualWeight,
			Company:      sp.Company,
			FirstName:    sp.Recipient,
			LastName:     sp.Recipient,
			FullName:     sp.Recipient,
			City:         sp.City,
			Address1:     sp.Address1,
			Address2:     sp.Address2,
			State:        sp.StateCode,
			Zipcode:      sp.Zipcode,
			Phone:        sp.PhoneNumber,
			Country:      sp.CountryCode,
			Height:       sp.ActualHeight,
			Length:       sp.ActualLength,
			Width:        sp.ActualWidth,
			DistanceUnit: "in",

			ServiceCode:            sp.Service.Code[0:1],
			DomesticCarrierService: sp.Service.DomesticCarrierService,
			HubStateCode:           warehouse.State,
			DisplayWeight:          sp.Weight,
			LabelTemplate:          ibblue.TemplateTebexpress,
			IsExceedPkg:            sp.IsPackageExceed,
			Zone:                   zone,

			WarehouseCompany:  warehouse.Company,
			WarehouseCity:     warehouse.City,
			WarehouseAddress1: warehouse.Address,
			WarehouseState:    warehouse.State,
			WarehouseZipcode:  warehouse.Zipcode,
			WarehouseCountry:  warehouse.Country,
			WarehousePhone:    warehouse.Phone,
		}

		if sp.LabelPromotion {
			body.LabelTemplate = ibblue.TemplateTebexpress
		}

		tracking.HubID = utils.Int64(warehouse.ID)
		tracking.HandlingFee = warehouse.HandlingFee
		tracking.Status = constant.TrackingStatusSuccess
	}

	if labalType == createlabel.LabelTypeUpdate {
		body.ShipmentID = sp.Tracking.ShipmentID
		body.TrackingNumber = sp.Tracking.TrackingNumber

		warehouse, err := h.WareHouseManager.GetWareHouse(sqlmanager.OptionWareHouse{ID: utils.Int64Value(&ware.ID), Status: constant.WareHouseStatusActive})
		if err == gorm.ErrRecordNotFound {
			warehouse, err = h.WareHouseManager.GetWareHouseMostPrice(sp.ID)
			if err != nil {
				return nil, "", err
			}
		} else if err != nil {
			return nil, "", err
		}

		tracking.HubID = utils.Int64(warehouse.ID)
		tracking.HandlingFee = warehouse.HandlingFee
		tracking.Status = constant.TrackingStatusSuccess
		body.WarehouseCompany = warehouse.Company
		body.WarehouseCity = warehouse.City
		body.WarehouseAddress1 = warehouse.Address
		body.WarehouseState = warehouse.State
		body.WarehouseZipcode = warehouse.Zipcode
		body.WarehouseCountry = warehouse.Country
		body.WarehousePhone = warehouse.Phone
	}

	res, errAudit, err := h.CreateLabel.Request(c, body, carrier, sp.UserID, labalType)
	if err != nil {
		return nil, "", err
	}

	if errAudit != nil {
		return nil, errAudit.Error(), nil
	}

	if res.CarrierCode != "" && res.CarrierCode != oldCarrierCode {
		dbcarrier, err := h.ServiceManager.GetCarrierByCode(res.CarrierCode)
		if err != nil {
			return nil, "", err
		}

		tracking.CarrierID = dbcarrier.ID
	}

	tracking.ShipmentID = res.ShipmentID
	tracking.TrackingNumber = res.TrackingNumber
	tracking.ShipmentCost = res.ShippingFee
	tracking.CarrierService = res.CarrierService
	tracking.LabelURL = res.LabelUrl
	tracking.Zone = res.Zone
	tracking.Weight = res.Weight
	tracking.Length = res.Length
	tracking.Width = res.Width
	tracking.Height = res.Height

	return tracking, "", nil
}
