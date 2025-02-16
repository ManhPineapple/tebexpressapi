package customer

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"tebexpressapi/pkg/alert"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/createlabel"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/models/entity"
	packageutils "tebexpressapi/pkg/package_utils"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"
	"tebexpressapi/pkg/utils"
	"time"

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

func (h *ShipmentHandler) Fullfill() gin.HandlerFunc {
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

		packageIDs := []int64{}
		var totalWeight float64 = 0
		var totalWeightPrice float64 = 0
		for _, p := range packages {
			packageIDs = append(packageIDs, p.ID)
			totalWeight += p.Weight
			totalWeightPrice += p.Weight
		}

		service, err := h.ServiceManager.GetServiceByCode(constant.ServiceFBACode)
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
		if user.Balance < amount && (user.UserInfo == nil || user.UserInfo.DebtMaxAmount <= 0) {
			c.JSON(http.StatusInternalServerError, "Số dư ví không đủ. Vui lòng nạp thêm")
			return
		}

		if user.Balance-amount < 0 && user.UserInfo != nil && user.UserInfo.DebtMaxAmount > 0 {
			if err != nil && err != gorm.ErrRecordNotFound {
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
			if user.Balance < 0 && user.UserInfo.DebtTime != nil && user.UserInfo.DebtTime.AddDate(0, 0, user.UserInfo.DebtMaxDay).Before(time.Now()) {
				c.JSON(http.StatusInternalServerError, "Tài khoản của bạn đã nợ quá thời hạn cho phép. Vui lòng nạp thêm tiền để tiếp tục sử dụng dịch vụ")
				return
			}
			if math.Abs(user.Balance-amount) > user.UserInfo.DebtMaxAmount {
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

		if err := h.CustomerShipmentManager.Fulfill(shipment, bill, packages); err != nil {
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
