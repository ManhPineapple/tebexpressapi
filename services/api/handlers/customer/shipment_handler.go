package customer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"tebexpressapi/pkg/alert"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/createlabel"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/order"
	packageutils "tebexpressapi/pkg/package_utils"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/string_util"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ShipmentHandler struct {
	Logger *zap.SugaredLogger

	CalculatePrice      *calculate.CalculatePrice
	ShipmentCreateLabel *packageutils.CreateLabelHandler

	CustomerShipmentManager *sqlmanager.CustomerShipmentManager
	BillManager             *sqlmanager.BillManager
	ServiceManager          *sqlmanager.ServiceManager
	UserManager             *sqlmanager.UserManager
}

type CountListCustomerShipmentResponse struct {
	Count int64 `json:"count"`
}

type FetchCountItemCustomerShipmentResponse struct {
	Count int64 `json:"count"`
}

type GetListCustomerShipmentHandlerResponse struct {
	Shipments []entity.CustomerShipment `json:"shipments"`
}

type FetchDetailCustomerShipmentHandlerResponse struct {
	Shipment    *entity.CustomerShipment `json:"shipment"`
	TotalAmount float64                  `json:"total_amount"`
}

type cancelResponse struct {
	Success bool `json:"success"`
}

type fulfillCustomerShipmentRequest struct {
	ContainerType constant.ContainerType `json:"container_type"`
}

type fulfillResponse struct {
	Success bool `json:"success"`
}

type CustomerShipmentItem struct {
	ID             int64     `json:"id"`
	Code           string    `json:"code"`
	TrackingNumber string    `json:"tracking_number"`
	OrderNumber    string    `json:"order_number"`
	Label          string    `json:"label"`
	Recipient      string    `json:"recipient"`
	Company        string    `json:"company"`
	PhoneNumber    string    `json:"phone_number"`
	Address1       string    `json:"address_1" `
	Address2       string    `json:"address_2"`
	City           string    `json:"city"`
	State          string    `json:"state"`
	Zipcode        string    `json:"zipcode"`
	Country        string    `json:"country"`
	Detail         string    `json:"detail"`
	Weight         float64   `json:"weight"`
	Length         float64   `json:"length"`
	Height         float64   `json:"height"`
	Width          float64   `json:"width"`
	ActualWeight   float64   `json:"actual_weight"`
	ActualWidth    float64   `json:"actual_width"`
	ActualLength   float64   `json:"actual_length"`
	ActualHeight   float64   `json:"actual_height"`
	Status         int       `json:"status,omitempty"`
	StatusString   string    `json:"status_string"`
	ServiceID      int64     `json:"service_id"`
	ServiceCode    string    `json:"service_code"`
	ServiceName    string    `json:"service_name"`
	ShippingFee    float64   `json:"shipping_fee"`
	ExtraFee       float64   `json:"extra_fee"`
	UpdatedAt      time.Time `json:"updated_at"`
	CreatedAt      time.Time `json:"created_at"`
}

type FetchListItemsCustomerShipmentResponse struct {
	Items []CustomerShipmentItem `json:"items"`
}

func NewShipmentHandler(l *zap.SugaredLogger, r *redis.Client, s3 storage.S3, alert alert.Alert, createLabel *createlabel.CreateLabel, calculatePrice *calculate.CalculatePrice, csm *sqlmanager.CustomerShipmentManager,
	bm *sqlmanager.BillManager, sm *sqlmanager.ServiceManager, um *sqlmanager.UserManager,
	stm *sqlmanager.SettingManager, pm *sqlmanager.PackageManager, whm *sqlmanager.WareHouseManager, srm *sqlmanager.ServiceManager) *ShipmentHandler {
	return &ShipmentHandler{
		Logger: l,

		CalculatePrice:      calculatePrice,
		ShipmentCreateLabel: packageutils.NewCreateLabelHandler(l, r, s3, stm, pm, bm, um, whm, srm, createLabel, alert),

		CustomerShipmentManager: csm,
		BillManager:             bm,
		ServiceManager:          sm,
		UserManager:             um,
	}
}

func (h *PackageHandler) ImportFBA() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusForbidden, "User id required")
			return
		}

		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		file, handlerFile, err := c.Request.FormFile("file")
		if err != nil {
			h.Logger.Error("Can't access file: ", err)
			c.JSON(http.StatusBadRequest, "Can't access file.")
			return
		}
		defer file.Close()

		if !constant.AllowExtensionFileImportPackage[handlerFile.Header.Get("Content-Type")] {
			h.Logger.Error("Format file is not allow: ", handlerFile.Header.Get("Content-Type"))
			c.JSON(http.StatusBadRequest, "File upload sai định dạng.")
			return
		}

		f, err := handlerFile.Open()
		if err != nil {
			h.Logger.Error("Cant Open file: ", err)
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		usStates, err := h.StateManager.GetStates(sqlmanager.StateOption{
			Countries: []string{"US", "AU"},
		})

		if err != nil {
			h.Logger.Errorf("Error get list state US: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		mapStates := make(map[string]*entity.State)
		for _, state := range usStates {
			key := fmt.Sprintf("%s-%s", state.Country, state.Code)
			mapStates[key] = state

			key = fmt.Sprintf("%s-%s", state.Country, strings.ToUpper(state.Name))
			mapStates[key] = state
		}

		var template *entity.ImportPackageTemplate

		_, msg, packages, importErrors, total, totalWeight, ratePrice, err := h.ImportFBAPackageXlsx(c, f, user, mapStates, template)
		if msg != "" {
			c.JSON(http.StatusBadRequest, msg)
			return
		}

		if err != nil {
			h.Logger.Error("Error while import package: ", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if len(packages) > 0 && len(importErrors) == 0 {
			// extras, err := order.FBARateExtras(c, h.UPS, packages, h.Redis)
			// if err != nil {
			// 	h.Logger.Errorf("estimate cost %v", err)
			// 	c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			// 	return
			// }

			for _, pkgs := range packages {
				_, err = h.PackageManager.CreatePackageFBA(pkgs, totalWeight, ratePrice, nil)
				if err != nil {
					errDetail := strings.Split(cast.ToString(err), ":")
					if errDetail[1] == " Incorrect string value" {
						c.JSON(http.StatusInternalServerError, "Kí tự không hợp lệ")
						return
					}
					h.Logger.Error("Error create shipping package: ", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}
			}
		}

		var successCount int64
		for _, pkgs := range packages {
			successCount += int64(len(pkgs))
		}

		c.JSON(http.StatusOK, ImportPackageResponse{importErrors, total, successCount})
	}
}

func (h *ShipmentHandler) Count() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		status := strings.TrimSpace(c.Request.URL.Query().Get("status"))

		opts := sqlmanager.CustomerShipmentOption{
			InStatus:  constant.MapIntGroupStatusCustomerPackage[status],
			Keyword:   strings.TrimSpace(c.Request.URL.Query().Get("keyword")),
			UserID:    userID,
			StartDate: cast.ToString(c.Request.URL.Query().Get("start_date")),
			EndDate:   cast.ToString(c.Request.URL.Query().Get("end_date")),
		}

		count, err := h.CustomerShipmentManager.CountCustomerShipments(opts)
		if err != nil {
			h.Logger.Errorf("Count list container error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CountListCustomerShipmentResponse{count})
	}
}

func (h *ShipmentHandler) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		status := strings.TrimSpace(c.Request.URL.Query().Get("status"))

		offset, limit := httputil.GetRequestPaginate(c.Request)
		opts := sqlmanager.CustomerShipmentOption{
			UserID:    userID,
			InStatus:  constant.MapIntGroupStatusCustomerPackage[status],
			Keyword:   strings.TrimSpace(c.Request.URL.Query().Get("keyword")),
			StartDate: cast.ToString(c.Request.URL.Query().Get("start_date")),
			EndDate:   cast.ToString(c.Request.URL.Query().Get("end_date")),
			Limit:     limit,
			Offset:    offset,
		}

		shipments, err := h.CustomerShipmentManager.GetCustomerShipments(opts)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get list shipments error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		ids := []int64{}
		for _, v := range shipments {
			ids = append(ids, v.ID)
		}

		mapPrices, err := h.CustomerShipmentManager.GetCustomerShipmentsPrice(ids)
		if err != nil {
			h.Logger.Errorf("Get price shipments error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		for i := range shipments {
			id := shipments[i].ID
			shipments[i].Price = mapPrices[id]
		}

		c.JSON(http.StatusOK, GetListCustomerShipmentHandlerResponse{shipments})
	}
}

func (h *ShipmentHandler) Detail() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		id := cast.ToInt64(c.Param("shipment_id"))
		if id < 1 {
			c.JSON(http.StatusBadRequest, "Shipment id is missing")
			return
		}

		opts := sqlmanager.CustomerShipmentOption{
			ID:              id,
			UserID:          userID,
			ExtraFeePreload: true,
		}

		shipment, err := h.CustomerShipmentManager.GetCustomerShipment(opts)
		if err == gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get list shiment error:, %v", err)
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		totalAmount, err := h.CustomerShipmentManager.GetCustomerShipmentPrice(shipment.ID)
		if err != nil {
			h.Logger.Errorf("get total shipment amount:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		status, err := h.CustomerShipmentManager.GetCustomerShipmentStatus(shipment.ID)
		if err != nil {
			h.Logger.Errorf("Get price shipments error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		weight, err := h.CustomerShipmentManager.GetCustomerShipmentActualWeight(shipment.ID)
		if err != nil {
			h.Logger.Errorf("Get weight shipments error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		shipment.Status = status
		shipment.ActualWeight = weight

		c.JSON(http.StatusOK, FetchDetailCustomerShipmentHandlerResponse{shipment, totalAmount})
	}
}

func (h *ShipmentHandler) Fulfill() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		shipmentID := cast.ToInt64(c.Param("id"))
		if shipmentID < 1 {
			c.JSON(http.StatusBadRequest, "Shipment id is missing")
			return
		}

		shipment, err := h.CustomerShipmentManager.GetCustomerShipment(sqlmanager.CustomerShipmentOption{UserID: userID, ID: shipmentID, ExtraFeePreload: true})
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("get shipment: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		status, err := h.CustomerShipmentManager.GetCustomerShipmentStatus(shipment.ID)
		if err != nil {
			h.Logger.Errorf("Get price shipments error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if status != constant.PackageStatusCreated {
			c.JSON(http.StatusBadRequest, "Lô hàng chỉ tạo tracking khi được ở trạng thái pending")
			return
		}

		packages, err := h.CustomerShipmentManager.GetPackagesByShipment(sqlmanager.CustomerShipmentOption{ID: shipment.ID})
		if err != nil {
			h.Logger.Errorf("get shipment: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		fulfillCustomerShipmentRequest := &fulfillCustomerShipmentRequest{}
		decoder := json.NewDecoder(c.Request.Body)
		if err := decoder.Decode(fulfillCustomerShipmentRequest); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		packageIDs := []int64{}
		var totalWeight float64 = 0
		var totalWeightPrice float64 = 0
		for _, p := range packages {
			packageIDs = append(packageIDs, p.ID)
			totalWeight += p.Weight
			totalWeightPrice += p.Weight
		}

		service, err := h.ServiceManager.GetServiceByCode(packages[0].Service.Code)
		if err != nil {
			h.Logger.Errorf("get service: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		rate, max, err := h.CalculatePrice.GetRatePriceByTotalWeight(c, totalWeightPrice, service.ID, user.Class)
		if err != nil {
			h.Logger.Errorf("Calculate price: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if rate == 0 {
			if max > 0 {
				maxKG := (math.Ceil(max) - 1) / constant.KgToGram
				c.JSON(http.StatusBadRequest, fmt.Sprintf("Khối lượng lô hàng không được vượt quá %v kg", maxKG))
				return
			} else {
				c.JSON(http.StatusBadRequest, "Khối lượng lô hàng không hợp lệ")
				return
			}
		}

		shipment.Weight = totalWeight
		shipment.Price = rate
		shipment.Status = constant.PackageStatusCreated

		var amount float64 = 0
		for i, p := range packages {
			price := h.CalculatePrice.GetFbaPackagePrice(rate, p.Weight, p.Length, p.Height, p.Width)
			packages[i].ShippingFee = price
			amount += price
		}

		for _, v := range shipment.ExtraFees {
			amount += v.Amount
		}

		amount = utils.ToFixed(amount, 2)
		if user.Balance+0.01 < amount && (user.UserInfo == nil || user.UserInfo.DebtMaxAmount <= 0) {
			c.JSON(http.StatusInternalServerError, "Số dư ví không đủ. Vui lòng nạp thêm")
			return
		}

		if user.Balance+0.01-amount < 0 && user.UserInfo != nil && user.UserInfo.DebtMaxAmount > 0 {
			if err != nil && err != gorm.ErrRecordNotFound {
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
			if user.Balance+0.01 < 0 && user.UserInfo.DebtTime != nil && user.UserInfo.DebtTime.AddDate(0, 0, user.UserInfo.DebtMaxDay).Before(time.Now()) {
				c.JSON(http.StatusInternalServerError, "Tài khoản của bạn đã nợ quá thời hạn cho phép. Vui lòng nạp thêm tiền để tiếp tục sử dụng dịch vụ")
				return
			}
			if math.Abs(user.Balance+0.01-amount) > user.UserInfo.DebtMaxAmount {
				c.JSON(http.StatusInternalServerError, "Tài khoản của bạn đã nợ quá giới hạn cho phép. Vui lòng nạp thêm tiền để tiếp tục sử dụng dịch vụ")
				return
			}
		}

		bill, err := h.BillManager.GetOrCreateNowBill(userID)
		if err != nil {
			h.Logger.Errorf("Calculate price: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if err := h.CustomerShipmentManager.Fulfill(shipment, bill, packages, fulfillCustomerShipmentRequest.ContainerType); err != nil {
			h.Logger.Errorf("get shipment: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if len(packageIDs) > 0 {
			err = h.ShipmentCreateLabel.Handle(c, packageIDs, false, shipmentID)
			if err != nil {
				h.Logger.Error("Error publish message queue shipment-create-label: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		}

		c.JSON(http.StatusOK, fulfillResponse{true})
	}
}

func (h *ShipmentHandler) Cancel() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		shipmentID := cast.ToInt64(c.Param("shipment_id"))
		if shipmentID < 1 {
			c.JSON(http.StatusBadRequest, "Shipment id is missing")
			return
		}

		shipment, err := h.CustomerShipmentManager.GetCustomerShipment(sqlmanager.CustomerShipmentOption{UserID: userID,
			ID:              shipmentID,
			ExtraFeePreload: true,
		})
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("get shipment: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		status, err := h.CustomerShipmentManager.GetCustomerShipmentStatus(shipment.ID)
		if err != nil {
			h.Logger.Errorf("Get price shipments error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if status != constant.PackageStatusCreated && status != constant.PackageStatusPendingPickup {
			c.JSON(http.StatusBadRequest, "Chỉ hủy lô hàng ở trạng thái pending hoặc pre-transit")
			return
		}

		bill, err := h.BillManager.GetOrCreateNowBill(userID)
		if err != nil {
			h.Logger.Errorf("Calculate price: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		items, err := h.CustomerShipmentManager.GetCustomerShipmentItems(sqlmanager.CustomerShipmentItemOptions{ShipmentID: shipment.ID})
		if err != nil {
			h.Logger.Errorf("Get list shiment error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		for _, v := range items {
			if v.Status != constant.PackageStatusCreated && v.Status != constant.PackageStatusPendingPickup {
				c.JSON(http.StatusBadRequest, "Lô hàng có chứa đơn hàng khác trạng thái pending và pre-transit không thể hủy được")
				return
			}
		}

		if err := h.CustomerShipmentManager.Cancel(shipment, bill, items); err != nil {
			h.Logger.Errorf("get shipment: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, cancelResponse{true})
	}
}

func (h *ShipmentHandler) Items() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		shipmentID := cast.ToInt64(c.Param("shipment_id"))
		if shipmentID < 1 {
			c.JSON(http.StatusBadRequest, "Shipment id is missing")
			return
		}

		offset, limit := httputil.GetRequestPaginate(c.Request)
		opts := sqlmanager.CustomerShipmentItemOptions{
			ShipmentID: shipmentID,
			UserID:     userID,
			Status:     cast.ToInt(c.Request.URL.Query().Get("status")),
			Keyword:    strings.TrimSpace(c.Request.URL.Query().Get("keyword")),
			Limit:      limit,
			Offset:     offset,
		}

		shipments, err := h.CustomerShipmentManager.GetCustomerShipmentItems(opts)
		if err != nil {
			h.Logger.Errorf("Get list shiment error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		items := []CustomerShipmentItem{}
		for _, v := range shipments {
			var extraFee float64 = 0
			for _, fee := range v.ExtraFee {
				extraFee += fee.Amount
			}

			code := ""
			if v.PackageCode != nil {
				code = v.PackageCode.Code
			}

			tracking := ""
			if v.Tracking != nil {
				tracking = v.Tracking.TrackingNumber
			}

			if v.ActualWeight < v.Weight {
				v.ActualWeight = v.Weight
			}

			if v.ActualHeight*v.ActualWidth*v.ActualLength < v.Height*v.Width*v.Length {
				v.ActualHeight = v.Height
				v.ActualWidth = v.Width
				v.ActualLength = v.Length
			}

			serviceCode := ""
			serviceName := ""
			if v.Service != nil {
				serviceCode = v.Service.Code
				serviceName = v.Service.Name
			}

			items = append(items, CustomerShipmentItem{
				ID:             v.ID,
				Code:           code,
				TrackingNumber: tracking,
				OrderNumber:    v.OrderNumber,
				Label:          v.Label,
				Recipient:      v.Recipient,
				Company:        v.Company,
				PhoneNumber:    v.PhoneNumber,
				Address1:       v.Address1,
				Address2:       v.Address2,
				City:           v.City,
				State:          v.StateCode,
				Zipcode:        v.Zipcode,
				Country:        v.CountryCode,
				Detail:         v.Detail,
				Weight:         v.Weight,
				Length:         v.Length,
				Height:         v.Height,
				Width:          v.Width,
				ActualWeight:   v.ActualWeight,
				ActualWidth:    v.ActualWidth,
				ActualLength:   v.ActualLength,
				ActualHeight:   v.ActualHeight,
				Status:         v.Status,
				StatusString:   constant.MapTextStatusCustomerPackage[v.Status],
				ServiceID:      v.ServiceID,
				ShippingFee:    v.ShippingFee,
				ExtraFee:       extraFee,
				UpdatedAt:      v.UpdatedAt,
				CreatedAt:      v.CreatedAt,
				ServiceCode:    serviceCode,
				ServiceName:    serviceName,
			})
		}

		c.JSON(http.StatusOK, FetchListItemsCustomerShipmentResponse{items})
	}
}

func (h *ShipmentHandler) ItemsCount() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		shipmentID := cast.ToInt64(c.Param("shipment_id"))
		if shipmentID < 1 {
			c.JSON(http.StatusBadRequest, "Shipment id is missing")
			return
		}

		opts := sqlmanager.CustomerShipmentItemOptions{
			ShipmentID: shipmentID,
			Status:     cast.ToInt(c.Request.URL.Query().Get("status")),
			Keyword:    strings.TrimSpace(c.Request.URL.Query().Get("keyword")),
			UserID:     userID,
		}
		count, err := h.CustomerShipmentManager.CountCustomerShipmentItems(opts)
		if err != nil {
			h.Logger.Errorf("Count list container error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		c.JSON(http.StatusOK, FetchCountItemCustomerShipmentResponse{count})
	}
}

func (h *PackageHandler) ImportFBAPackageXlsx(c context.Context, file io.Reader, user *entity.User, mapStates map[string]*entity.State, template *entity.ImportPackageTemplate) (isValidColumn bool, msg string, packages map[string][]*entity.Package, importErrors []ImportPackageError, total int64, totalWeight, ratePrice float64, err error) {
	packages = make(map[string][]*entity.Package, 0)
	importErrors = make([]ImportPackageError, 0)
	isValidColumn = true

	var service *entity.Service

	defer func() {
		if r := recover(); r != nil {
			h.Logger.Errorf("Error when process excel file: %v", r)
			isValidColumn = false
		}
	}()

	columnOrderNumber := 0
	columnReceiveName := 1
	columnReceivePhone := 2
	columnReceiveAddress1 := 3
	columnReceiveAddress2 := 4
	columnCity := 5
	columnStateCode := 6
	columnZipcode := 7
	columnCountry := 8
	// columnPackageName := 9
	columnDetail := 10
	columnWeight := 11
	columnLength := 12
	columnWidth := 13
	columnHeight := 14
	columnService := 15
	// columnCustomTiktokBarcode := 16
	// columnIsTradeMark := 17
	// columnBattery := 18
	// columnIsEarlyScan := 19
	// columnPackageQuantity := 20
	// columnTotalProductPrice := 21
	var total_column = 16

	f, err := excelize.OpenReader(file)
	if err != nil {
		h.Logger.Errorf("Error when process excel file: %v", err)
		return isValidColumn, "", packages, importErrors, total, totalWeight, ratePrice, err
	}

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		h.Logger.Errorf("Error when process excel file: %v", err)
		return isValidColumn, "", packages, importErrors, total, totalWeight, ratePrice, err
	}

	services, err := h.ServiceManager.GetServices(sqlmanager.ServiceQueryOption{
		Status: constant.StatusActive,
	})

	if err != nil {
		h.Logger.Errorf("Get list service error: %v", err)
		return isValidColumn, "", packages, importErrors, total, totalWeight, ratePrice, err
	}

	var mapNameServices, mapCodeServices = make(map[string]*entity.Service), make(map[string]*entity.Service)
	for _, service := range services {
		mapNameServices[strings.ToUpper(service.Name)] = service
		mapCodeServices[strings.ToUpper(service.Code)] = service
	}

	validator := order.MakeValidator(h.StateManager)
	validator.SetStates(mapStates)

	for indexRow, row := range rows {
		h.Logger.Info("len(row): ", indexRow, len(row), total_column)
		if len(row) > total_column || len(row) == 0 {
			continue
		} else if indexRow > 0 && len(row) < total_column {
			for i := len(row); i < total_column; i++ {
				row = append(row, "")
			}
		}
		if indexRow == 0 {
			continue
		}

		if indexRow > 3 {
			return isValidColumn, "Lô hàng FBA chỉ có thể có tối đa 3 kiện hàng.", packages, importErrors, total, totalWeight, ratePrice, ErrorMaxWeight
		}

		total++

		data := &order.PackageResource{}
		data.Recipient = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnReceiveName])
		data.Address1 = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnReceiveAddress1])

		if columnReceiveAddress2 >= 0 {
			data.Address2 = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnReceiveAddress2])
		}

		data.City = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnCity])
		data.Country = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnCountry])
		data.OrderNumber = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnOrderNumber])
		data.State = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnStateCode])
		data.Zipcode = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnZipcode])
		data.Detail = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnDetail])

		if columnReceivePhone >= 0 {
			data.Phone = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnReceivePhone])
		}

		if columnReceiveAddress2 >= 0 {
			data.Address2 = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnReceiveAddress2])
		}

		w := cast.ToFloat64(strings.TrimSpace(row[columnWeight]))
		w = cast.ToFloat64(int(math.Ceil(w / 1000)))
		data.Weight = w * 1000
		data.Length = cast.ToFloat64(int(math.Ceil(cast.ToFloat64(strings.TrimSpace(row[columnLength])))))
		data.Width = cast.ToFloat64(int(math.Ceil(cast.ToFloat64(strings.TrimSpace(row[columnWidth])))))
		data.Height = cast.ToFloat64(int(math.Ceil(cast.ToFloat64(strings.TrimSpace(row[columnHeight])))))
		data.Service = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnService])

		h.Logger.Info("aaa: ", strings.TrimSpace(row[columnWeight]), cast.ToFloat64(int(math.Ceil(cast.ToFloat64(strings.TrimSpace(row[columnWeight]))))))
		var values []string
		var messages []string

		data.Service = strings.ToUpper(data.Service)
		if mapCodeServices[data.Service] != nil {
			service = mapCodeServices[data.Service]
			data.ServiceCode = service.Code
		} else if mapNameServices[data.Service] != nil {
			service = mapNameServices[data.Service]
			data.ServiceCode = service.Code
		} else {
			values = append(values, data.Service)
			messages = append(messages, "Dịch vụ không hợp lệ")
		}

		if len(messages) > 0 && (service == nil || service.ID < 0) {
			for i := range messages {
				messages[i] = fmt.Sprintf("%s\n", messages[i])
			}

			importErrors = append(importErrors, ImportPackageError{
				Line:     int64(indexRow) + 1,
				Value:    values,
				Messages: messages,
			})
			continue
		}

		validator.Reset()
		validator.Validate(data)
		validator.ValidateVolumes(data)
		validator.ValidateFbaServicePackage(data)

		if err := validator.Error(); err != nil {
			h.Logger.Errorf("validate: %v", err)
			return isValidColumn, "", packages, importErrors, total, totalWeight, ratePrice, err
		}

		if errs := validator.Errors(); len(errs) > 0 {
			messages = append(messages, errs...)
			valueErrors := validator.ValueErrors()
			values = append(values, valueErrors...)
		}

		if service.Country != data.Country {
			values = append(values, data.Service)
			messages = append(messages, fmt.Sprintf("Dịch vụ %s không hỗ trợ %s", service.Name, data.Country))
		}
		if service.Code != constant.ServiceFBACode && service.Code != constant.ServiceFastFBACode {
			values = append(values, data.Service)
			messages = append(messages, "Mã dịch vụ không hợp lệ")
		}

		pkg := &entity.Package{
			OrderNumber:     data.OrderNumber,
			Detail:          data.Detail,
			Recipient:       data.Recipient,
			PhoneNumber:     data.Phone,
			Address1:        data.Address1,
			Address2:        data.Address2,
			City:            data.City,
			StateCode:       data.State,
			Zipcode:         data.Zipcode,
			CountryCode:     data.Country,
			Weight:          data.Weight,
			Length:          data.Length,
			Width:           data.Width,
			Height:          data.Height,
			UserID:          user.ID,
			ServiceID:       service.ID,
			Status:          constant.PackageStatusCreated,
			ValidateAddress: constant.PackageValidAddress,
		}

		h.Logger.Infof("fbapkg: %v - %v - %v - %v - %v", pkg.OrderNumber, pkg.Weight, pkg.Height, pkg.Length, pkg.Width)
		key := fmt.Sprintf("%v_%v_%v_%v_%v", pkg.Address1, pkg.City, pkg.StateCode, pkg.Zipcode, pkg.CountryCode)
		if _, ok := packages[key]; !ok {
			packages[key] = make([]*entity.Package, 0)
		}

		// if len(packages) > 0 {
		// 	if pkg.Address1 != packages[0].Address1 {
		// 		values = append(values, pkg.Address1)
		// 		messages = append(messages, fmt.Sprintf("Địa chỉ không trùng nhau"))
		// 	}

		// 	if pkg.City != packages[0].City {
		// 		values = append(values, pkg.City)
		// 		messages = append(messages, fmt.Sprintf("Thành phố không trùng nhau"))
		// 	}

		// 	if pkg.StateCode != packages[0].StateCode {
		// 		values = append(values, pkg.StateCode)
		// 		messages = append(messages, fmt.Sprintf("Mã vùng không trùng nhau"))
		// 	}

		// 	if pkg.Zipcode != packages[0].Zipcode {
		// 		values = append(values, pkg.Zipcode)
		// 		messages = append(messages, fmt.Sprintf("Mã bưu điện không trùng nhau"))
		// 	}

		// 	if pkg.CountryCode != packages[0].CountryCode {
		// 		values = append(values, pkg.CountryCode)
		// 		messages = append(messages, fmt.Sprintf("Mã quốc gia không trùng nhau"))
		// 	}
		// }

		if len(messages) > 0 {
			for i := range messages {
				messages[i] = fmt.Sprintf("%s\n", messages[i])
			}
			importErrors = append(importErrors, ImportPackageError{
				Line:     int64(indexRow) + 1,
				Value:    values,
				Messages: messages,
			})
			continue
		}

		packages[key] = append(packages[key], pkg)
		wp, _ := calculate.CalcFBAPriceWeight(pkg.Weight, pkg.Length, pkg.Height, pkg.Width)
		totalWeight += wp
	}

	if len(packages) > 0 && len(importErrors) < 1 {
		for _, pkgs := range packages {
			var pkgWeights float64
			for _, pkg := range pkgs {
				wp, _ := calculate.CalcFBAPriceWeight(pkg.Weight, pkg.Length, pkg.Height, pkg.Width)
				pkgWeights += wp
			}

			h.Logger.Info("pkgW: ", pkgWeights)
			ratePrice, _, err = h.CalculatePrice.GetRatePriceByTotalWeight(c, pkgWeights, service.ID, user.Class)
			if err != nil {
				return isValidColumn, "", packages, importErrors, total, totalWeight, ratePrice, err
			}
		}

		var maxW float64
		ratePrice, maxW, err = h.CalculatePrice.GetRatePriceByTotalWeight(c, totalWeight, service.ID, user.Class)
		h.Logger.Infof("bbb: %v - %v - %v", ratePrice, maxW, totalWeight)
		if err != nil {
			return isValidColumn, "", packages, importErrors, total, totalWeight, ratePrice, err
		}
		if ratePrice == 0 {
			if maxW > 0 {
				return isValidColumn, fmt.Sprintf("Khối lượng lô hàng không được vượt quá %.2f kg", maxW/constant.KgToGram), packages, importErrors, total, totalWeight, ratePrice, ErrorMaxWeight
			} else {
				return isValidColumn, "Khối lượng lô hàng không hợp lệ", packages, importErrors, total, totalWeight, ratePrice, ErrorMaxWeight
			}
		} else {
			for i, pkgs := range packages {
				for j := range pkgs {
					shippingFee := h.CalculatePrice.GetFbaPackagePrice(ratePrice, packages[i][j].Weight, packages[i][j].Width, packages[i][j].Length, packages[i][j].Height)
					shippingFee = cast.ToFloat64(int(math.Ceil(shippingFee)))
					packages[i][j].ShippingFee = shippingFee
				}
			}
		}
	}

	return isValidColumn, "", packages, importErrors, total, totalWeight, ratePrice, err
}
