package admin

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"
	"tebexpressapi/pkg/alert"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/createlabel"
	pkg_dto "tebexpressapi/pkg/dto"
	hpkg "tebexpressapi/pkg/helpers/packages"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/models/dto"
	"tebexpressapi/pkg/models/entity"
	packageutils "tebexpressapi/pkg/package_utils"
	"tebexpressapi/pkg/providers"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/dbgorm"
	"tebexpressapi/pkg/utils/string_util"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PackageHandler struct {
	Logger    *zap.SugaredLogger
	Redis     *redis.Client
	StorageS3 storage.S3

	CreateLabel    *createlabel.CreateLabel
	CalculatePrice *calculate.CalculatePrice

	ShipmentEstimateCost       *packageutils.EstimateCost
	PackageRefund              *packageutils.PackageRefund
	ShipmentCancelCarrier      *packageutils.ShipmentCancelCarrier
	ShipmentCreateLabelHandler *packageutils.CreateLabelHandler

	UserManager      *sqlmanager.UserManager
	PackageManager   *sqlmanager.PackageManager
	TrackingManager  *sqlmanager.TrackingManager
	WareHouseManager *sqlmanager.WareHouseManager
	StateManager     *sqlmanager.StateManager
	ServiceManager   *sqlmanager.ServiceManager
	BillManager      *sqlmanager.BillManager
	SettingManager   *sqlmanager.SettingManager
}

type ProcessPackageForm struct {
	Id int64 `json:"id"`
}

type GetPackageByCodeResponse struct {
	Package       interface{} `json:"package"`
	CountTracking int64       `json:"count_tracking"`
}

type GetListPackagesResponse struct {
	Packages interface{} `json:"packages"`
}

type CountListPackagesResponse struct {
	Count       int64                              `json:"count"`
	StatusCount []pkg_dto.CountStatusStringPackage `json:"status_count"`
}

type packageCheckedResponse struct {
	ID             int64  `json:"id"`
	TrackID        int64  `json:"track_id"`
	LabelBase64    string `json:"base64_labels"`
	LabelURL       string `json:"label_url"`
	TrackingNumber string `json:"tracking_number"`
}

type GetPackageDetailResponse struct {
	Package             interface{} `json:"package"`
	DeliverLogs         interface{} `json:"deliver_logs"`
	AuditLogs           interface{} `json:"audit_logs"`
	ExtraFree           interface{} `json:"extra_fee"`
	StatusTicket        bool        `json:"status_ticket"`
	EstimateProcessDate *time.Time  `json:"estimate_process_date"`
}

type FormPackageReturn struct {
	IDs              []int64 `json:"ids"`
	Note             string  `json:"note"`
	CheckinRequestID int64   `json:"checkin_request_id"`
}

type ReshipForm struct {
	Recipient   string `json:"recipient"`
	PhoneNumber string `json:"phone_number"`
	Address1    string `json:"address_1"`
	Address2    string `json:"address_2"`
	City        string `json:"city"`
	StateCode   string `json:"state_code"`
	Zipcode     string `json:"zipcode"`
	CountryCode string `json:"country_code"`
}

type ReshipEstimateCostResponse struct {
	TotalAmount float64 `json:"total_amount"`
	Success     bool    `json:"success"`
}

type UpdateForm struct {
	Sku             string                    `json:"sku"`
	OrderNumber     string                    `json:"-"`
	Recipient       string                    `json:"recipient"`
	PhoneNumber     string                    `json:"phone_number"`
	Address1        string                    `json:"address_1" gorm:"column:address_1"`
	Address2        string                    `json:"address_2" gorm:"column:address_2"`
	City            string                    `json:"city"`
	StateCode       string                    `json:"state_code"`
	Zipcode         string                    `json:"zipcode"`
	CountryCode     string                    `json:"country_code"`
	Detail          string                    `json:"detail"`
	Weight          float64                   `json:"weight"`
	Width           float64                   `json:"width"`
	Length          float64                   `json:"length"`
	Height          float64                   `json:"height"`
	Status          int                       `json:"status"`
	UserID          int64                     `json:"user_id"`
	Service         string                    `json:"service"`
	Note            string                    `json:"note"`
	Description     string                    `json:"description"`
	IncludeBattery  bool                      `json:"include_battery"`
	IsReship        bool                      `json:"is_reship"`
	PackageProducts []*entity.PackageProducts `json:"package_products"`

	PackageName       string  `json:"package_name"`
	PackageQuantity   int64   `json:"package_quantity"`
	TotalProductPrice float64 `json:"product_price"`

	CNProductLink     string   `json:"cn_product_link"`
	CNProductPrice    float64  `json:"cn_product_price"`
	CNShippingFee     float64  `json:"cn_shipping_fee"`
	CNShippingToVNFee *float64 `json:"cn_shipping_to_vn_fee"`
	CNLabelExtraFee   *float64 `json:"cn_label_extra_fee"`
	CustomCNBarcode   *string  `json:"custom_cn_barcode"`
}

type UpdatePackageResponse struct {
	Package     interface{} `json:"package"`
	DeliverLogs interface{} `json:"deliver_logs"`
	ExtraFree   interface{} `json:"extra_fee"`
}

type ImportTrackingResponse struct {
	Success bool `json:"success"`
}

type ImportRowForm struct {
	TrackingNumber string
	Label          string
}

type formPackageChecked struct {
	Carrier      string  `json:"carrier"`
	Weight       float64 `json:"weight"`
	Length       float64 `json:"length"`
	Width        float64 `json:"width"`
	Height       float64 `json:"height"`
	PostmarkDate int64   `json:"postmark_date"`
	HubID        *int64  `json:"hub_id"`
}

func NewPackageHandler(l *zap.SugaredLogger, r *redis.Client, s3 storage.S3, calculatePrice *calculate.CalculatePrice,
	createLabel *createlabel.CreateLabel, um *sqlmanager.UserManager, pm *sqlmanager.PackageManager,
	tm *sqlmanager.TrackingManager, wh *sqlmanager.WareHouseManager, stm *sqlmanager.StateManager,
	sm *sqlmanager.ServiceManager, bm *sqlmanager.BillManager, setm *sqlmanager.SettingManager, alert alert.Alert) *PackageHandler {
	return &PackageHandler{
		Logger:    l,
		Redis:     r,
		StorageS3: s3,

		CreateLabel:    createLabel,
		CalculatePrice: calculatePrice,

		ShipmentEstimateCost:       packageutils.NewEstimateCost(l, pm, wh, sm, createLabel),
		PackageRefund:              packageutils.NewPackageRefund(l, pm, bm),
		ShipmentCancelCarrier:      packageutils.NewShipmentCancelCarrier(l, pm, tm),
		ShipmentCreateLabelHandler: packageutils.NewCreateLabelHandler(l, r, s3, setm, pm, bm, um, wh, sm, createLabel, alert),

		UserManager:      um,
		PackageManager:   pm,
		TrackingManager:  tm,
		WareHouseManager: wh,
		StateManager:     stm,
		ServiceManager:   sm,
		BillManager:      bm,
		SettingManager:   setm,
	}
}

func (h *PackageHandler) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		offset, limit := httputil.GetRequestPaginate(c.Request)
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))

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

		opts := sqlmanager.PackageQueryOption{
			Limit:          limit,
			Offset:         offset,
			SearchBy:       cast.ToString(c.Request.URL.Query().Get("search_by")),
			Search:         cast.ToString(c.Request.URL.Query().Get("search")),
			AlertValue:     cast.ToInt(c.Request.URL.Query().Get("alert")),
			UserID:         cast.ToInt64(c.Request.URL.Query().Get("user_id")),
			StartDate:      cast.ToString(c.Request.URL.Query().Get("start_date")),
			EndDate:        cast.ToString(c.Request.URL.Query().Get("end_date")),
			Sort:           cast.ToString(c.Request.URL.Query().Get("sort")),
			ExceptFba:      true,
			Preload:        []string{"Warehouse", "Service"},
			TrackingStatus: constant.TrackingStatusSuccess,
			PartnerID:      user.PartnerID,
		}
		if role == constant.UserRoleSupport || role == constant.UserRoleSale {
			opts.SupportID = user.ID
		}

		var statusString = cast.ToString(c.Request.URL.Query().Get("status"))
		if statusString == constant.PackageStatusAlertText {
			opts.QueryAlert = true
		} else {
			opts.StatusArr = constant.MapIntGroupStatusAdminPackage[statusString]
		}

		if (opts.StartDate != "" && utils.ParseRawDateTime(opts.StartDate) == nil) ||
			(opts.EndDate != "" && utils.ParseRawDateTime(opts.EndDate) == nil) {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		if opts.Sort != "" && opts.Sort != constant.ORDER_ASC && opts.Sort != constant.ORDER_DESC {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		if statusString != "" && len(opts.StatusArr) == 0 && !opts.QueryAlert {
			c.JSON(http.StatusBadRequest, "Invalid status")
			return
		}

		if opts.SearchBy != "" && !utils.ValidSlug(opts.SearchBy) {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		if opts.Search != "" && utils.InvalidTag(opts.Search) {
			c.JSON(http.StatusBadRequest, "Từ khóa không hợp lệ")
			return
		}
		packages := make([]entity.Package, 0)
		packages, err = h.PackageManager.GetPackages(opts)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get packages %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		days := viper.GetInt64("package_code.pending_pickup_max_active") - viper.GetInt64("package_code.package_to_alert") + 1
		pkgDTO := make([]dto.OverPretransitPackageDTO, 0)
		for i, Package := range packages {
			var extraFee float64 = 0
			for _, fee := range Package.ExtraFee {
				extraFee += fee.Amount
			}
			packages[i].ShippingFee = packages[i].ShippingFee + extraFee
			packages[i].StatusString = constant.MapTextStatusAdminPackage[packages[i].Status]
			var dayLeft string
			if Package.AlertAt != nil {
				var d = utils.Floor(Package.AlertAt.Add(time.Duration(days)*24*time.Hour).Sub(time.Now()).Hours()/24, 0)
				// var h = utils.Floor((Package.AlertAt.Add(time.Duration(days)*24*time.Hour).Sub(time.Now()).Hours()/24-d)*24, 0)
				// var m = utils.Floor(((Package.AlertAt.Add(time.Duration(days)*24*time.Hour).Sub(time.Now()).Hours()/24-d)*24-h)*60, 0)
				if d > 0 {
					dayLeft += cast.ToString(d) + " ngày "
				}
				// if h > 0 {
				// 	dayLeft += cast.ToString(h) + " giờ "
				// }
				// if m > 0 {
				// 	dayLeft += cast.ToString(m) + " phút "
				// }
			}
			pkgDTO = append(pkgDTO, dto.OverPretransitPackageDTO{
				Package,
				dayLeft,
			})

			if packages[i].Status == constant.PackageStatusCreated {
				packages[i].PackageCode = nil
				packages[i].Tracking = nil
			}
		}

		if !opts.QueryAlert && opts.AlertValue == constant.PackageAlertTypeOverPretransit {
			c.JSON(http.StatusOK, GetListPackagesResponse{pkgDTO})
			return
		}

		c.JSON(http.StatusOK, GetListPackagesResponse{packages})
	}
}

func (h *PackageHandler) Count() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("Get GetUserByID %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		opts := sqlmanager.PackageQueryOption{
			SearchBy:    cast.ToString(c.Request.URL.Query().Get("search_by")),
			Search:      cast.ToString(c.Request.URL.Query().Get("search")),
			AlertValue:  cast.ToInt(c.Request.URL.Query().Get("alert")),
			ExceptFba:   true,
			UserID:      cast.ToInt64(c.Request.URL.Query().Get("user_id")),
			StartDate:   cast.ToString(c.Request.URL.Query().Get("start_date")),
			EndDate:     cast.ToString(c.Request.URL.Query().Get("end_date")),
			WarehouseID: cast.ToInt64(c.Request.URL.Query().Get("warehouse_id")),
			PartnerID:   user.PartnerID,
		}
		if role == constant.UserRoleSupport || role == constant.UserRoleSale {
			opts.SupportID = user.ID
		}

		if role == constant.UserRoleWarehouse {
			opts.WarehouseID = user.WarehouseID
		}

		var statusString = cast.ToString(c.Request.URL.Query().Get("status"))
		if statusString == constant.PackageStatusAlertText {
			opts.QueryAlert = true
		} else {
			opts.StatusArr = constant.MapIntGroupStatusAdminPackage[statusString]

		}

		if statusString != "" && len(opts.StatusArr) == 0 && !opts.QueryAlert {
			c.JSON(http.StatusBadRequest, "Invalid status")
			return
		}

		if opts.SearchBy != "" && !utils.ValidSlug(opts.SearchBy) {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		if opts.Search != "" && utils.InvalidTag(opts.Search) {
			c.JSON(http.StatusBadRequest, "Từ khóa không hợp lệ")
			return
		}

		Count, err := h.PackageManager.CountPackages(opts)

		if err != nil {
			h.Logger.Errorf("Count package error, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		var countStatusString = make([]pkg_dto.CountStatusStringPackage, 0)
		if opts.AlertValue != constant.PackageAlertTypeOverPretransit { // query slow package
			optsCountAll := sqlmanager.PackageQueryOption{
				SearchBy:    cast.ToString(c.Request.URL.Query().Get("search_by")),
				Search:      cast.ToString(c.Request.URL.Query().Get("search")),
				StartDate:   cast.ToString(c.Request.URL.Query().Get("start_date")),
				EndDate:     cast.ToString(c.Request.URL.Query().Get("end_date")),
				UserID:      cast.ToInt64(c.Request.URL.Query().Get("user_id")),
				WarehouseID: cast.ToInt64(c.Request.URL.Query().Get("warehouse_id")),
				ExceptFba:   true,
				PartnerID:   user.PartnerID,
			}

			var count int64
			if role == constant.UserRoleSupport || role == constant.UserRoleSale {
				optsCountAll.SupportID = user.ID
			}

			if role == constant.UserRoleWarehouse {
				optsCountAll.WarehouseID = user.WarehouseID
			}

			countStatus, err := h.PackageManager.CountAllStatusPackages(optsCountAll)
			if err != nil {
				h.Logger.Errorf("Count all status  error, %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			optsCountAlert := sqlmanager.PackageQueryOption{
				SearchBy:    cast.ToString(c.Request.URL.Query().Get("search_by")),
				Search:      cast.ToString(c.Request.URL.Query().Get("search")),
				StartDate:   cast.ToString(c.Request.URL.Query().Get("start_date")),
				EndDate:     cast.ToString(c.Request.URL.Query().Get("end_date")),
				QueryAlert:  true,
				UserID:      cast.ToInt64(c.Request.URL.Query().Get("user_id")),
				WarehouseID: cast.ToInt64(c.Request.URL.Query().Get("warehouse_id")),
				ExceptFba:   true,
			}
			if role == constant.UserRoleSupport || role == constant.UserRoleSale {
				optsCountAlert.SupportID = user.ID
			}

			if role == constant.UserRoleWarehouse {
				optsCountAlert.WarehouseID = user.WarehouseID
			}

			countAlert, err := h.PackageManager.CountPackages(optsCountAlert)
			if err != nil {
				h.Logger.Errorf("Count all status  error, %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			for _, statusInt := range countStatus {

				var statusConverted string

				statusConverted = constant.MapTextStatusAdminPackage[statusInt.Status]
				if statusConverted == "" {
					continue
				}

				var existing = false
				var index int
				for i, statusString := range countStatusString {
					if statusConverted == statusString.Status {
						existing = true
						index = i
					}
				}

				if !existing {
					countStatusString = append(countStatusString, pkg_dto.CountStatusStringPackage{
						Status: statusConverted,
						Count:  statusInt.Count,
					})
				} else {
					countStatusString[index].Count = countStatusString[index].Count + statusInt.Count
				}

				count = count + statusInt.Count
			}

			countStatusString = append(countStatusString, pkg_dto.CountStatusStringPackage{
				Status: constant.PackageStatusAlertText,
				Count:  countAlert,
			})
		}

		c.JSON(http.StatusOK, CountListPackagesResponse{Count, countStatusString})
	}
}

func (h *PackageHandler) Detail() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusForbidden, "User id required")
			return
		}

		packageID := cast.ToInt64(c.Param("id"))
		if packageID < 1 {
			c.JSON(http.StatusBadRequest, "Missing package id")
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

		options := sqlmanager.PackageQueryOption{
			ID:             packageID,
			TrackingStatus: constant.TrackingStatusSuccess,
		}
		if role == constant.UserRoleSupport || role == constant.UserRoleSale {
			options.SupportID = user.ID
		}

		packages, err := h.PackageManager.GetPackageDetail(options)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get Package Detail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

			return
		}

		if packages.Status == constant.PackageStatusCreated {
			packages.PackageCode = nil
			packages.Tracking = nil
		}

		deliverLogs, err := h.PackageManager.GetDeliverLogsByPkgID(packageID)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Deliver Logs %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

			return
		}

		auditLogs, err := h.PackageManager.GetAuditLogsByPkgID(packageID)
		if err != nil {
			h.Logger.Errorf("Get Package Audit Logs %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

			return
		}

		extraFee, err := h.PackageManager.GetExtraFeeByPkgID(packageID)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Deliver Logs %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

			return
		}

		status_ticket := true

		deliverLogs = pkg_dto.Transfomer3(deliverLogs)

		refundConfigDay := viper.GetInt("package.day_refund_expire_pending")
		pkgRefund, err := h.PackageManager.GetPackakgeRefundByPackageID(packages.ID)

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Deliver Logs %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

			return
		}
		var estimateProcessDate *time.Time
		if err != gorm.ErrRecordNotFound {
			t := pkgRefund.CreatedAt.AddDate(0, 0, refundConfigDay)
			estimateProcessDate = &t
		}

		inStatusEstimateDelivery := map[int]bool{
			constant.PackageStatusPicked:               true,
			constant.PackageStatusWareHouseLabeled:     true,
			constant.PackageStatusWareHouseInContainer: true,
			constant.PackageStatusWareHouseInShipment:  true,
			constant.PackageStatusWareHouseExport:      true,
			constant.PackageStatusImportHub:            true,
			constant.PackageStatusExportHub:            true,
			constant.PackageStatusInTransit:            true,
		}
		if inStatusEstimateDelivery[packages.Status] {
			weight, length, height, width := packages.Weight, packages.Length, packages.Height, packages.Width
			if packages.ActualWeight != 0 {
				weight = packages.ActualWeight
			}

			if packages.ActualWeight != 0 {
				weight = packages.ActualWeight
			}

			if packages.ActualWeight != 0 {
				weight = packages.ActualWeight
			}

			if packages.ActualWeight != 0 {
				weight = packages.ActualWeight
			}

			if packages.CountryCode != "US" {
				estimate, err := hpkg.EstimateDelivery(c, h.Redis, h.CreateLabel, packages.CountryCode, 0, weight, length, height, width, packages.ServiceID)
				if err != nil {
					h.Logger.Error(err)
				}

				packages.EstimateDelivery = estimate
			} else if packages.Tracking != nil {
				estimate, err := hpkg.EstimateDelivery(c, h.Redis, h.CreateLabel, packages.CountryCode, packages.Tracking.Zone, weight, length, height, width, packages.ServiceID)
				if err != nil {
					h.Logger.Error(err)
				}

				packages.EstimateDelivery = estimate
			}
		}

		c.JSON(http.StatusOK, GetPackageDetailResponse{Package: packages, DeliverLogs: deliverLogs, AuditLogs: auditLogs, ExtraFree: extraFee, StatusTicket: status_ticket, EstimateProcessDate: estimateProcessDate})
	}
}

func (h *PackageHandler) DetailByCode() gin.HandlerFunc {
	return func(c *gin.Context) {
		// userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		// user, err := h.UserManager.GetUserByID(userID)
		// if err != nil {
		// 	h.Logger.Errorf("get user: %v", err)
		// 	c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
		// 	return
		// }
		code := strings.TrimSpace(c.Param("code"))
		if code == "" {
			c.JSON(http.StatusBadRequest, "Mã vận đơn không được trống")
			return
		}

		pcode, err := h.PackageManager.GetPackageCodeByCode(sqlmanager.PackageCodeQueryOption{
			Code:   code,
			Status: constant.PackageCodeEnable,
		})

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, "Mã vận đơn không tồn tại")
			return
		}

		if err != nil {
			h.Logger.Errorf("Get Package Detail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		pkg, err := h.PackageManager.GetPackageByPackageCodeID(pcode.ID)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, "Mã vận đơn không tồn tại")
			return
		}

		if err != nil {
			h.Logger.Errorf("Get Package Detail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		// if pkg.WarehouseID > 0 {
		// 	wareHouse, err := h.WareHouseManager.GetWareHouse(sqlmanager.OptionWareHouse{
		// 		ID:     user.WarehouseID,
		// 		Status: constant.WareHouseStatusActive,
		// 	})

		// 	if err != nil {
		// 		h.Logger.Errorf("Get warehouse error: %s", err)
		// 		c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
		// 		return
		// 	}
		// 	// if user.Role == constant.UserRoleWarehouse && pkg.WarehouseID != user.WarehouseID {
		// 	// 	c.JSON(http.StatusNotFound, fmt.Sprintf("Đơn hàng không nằm trong kho %v", wareHouse.Name))
		// 	// }
		// }

		pkg.PackageCode = pcode
		extraFees, err := h.PackageManager.GetExtraFeeByPkgID(pkg.ID)
		if err != nil {
			h.Logger.Errorf("Get Package Deliver Logs %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		tracking, err := h.TrackingManager.GetTracking(sqlmanager.TrackingOption{
			PackageID: pkg.ID,
			Status:    constant.TrackingStatusSuccess,
		})
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Deliver Logs %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if err != gorm.ErrRecordNotFound {
			pkg.Tracking = tracking
		}

		countTracking, err := h.TrackingManager.CountTrackings(sqlmanager.TrackingOption{
			PackageID: pkg.ID,
		})
		if err != nil {
			h.Logger.Errorf("Count tracking %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		pkg.ExtraFee = extraFees

		c.JSON(http.StatusOK, GetPackageByCodeResponse{Package: pkg, CountTracking: countTracking})
	}
}

func (h *PackageHandler) EstimateCostReship() gin.HandlerFunc {
	return func(c *gin.Context) {
		packageID := cast.ToInt64(c.Param("package_id"))

		if packageID < 1 {
			c.JSON(http.StatusBadRequest, "Invalid order id")
			return
		}

		pkg, err := h.PackageManager.GetPackageByPackageID(packageID)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get Package Detail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if !strings.EqualFold(strings.ToLower(pkg.Service.Code), strings.ToLower(constant.ServiceExpressCode)) && !strings.EqualFold(strings.ToLower(pkg.Service.Code), strings.ToLower(constant.ServiceUSCode)) {
			c.JSON(http.StatusBadRequest, "Service not supported")
			return
		}

		form := &ReshipForm{}
		if err := c.ShouldBindJSON(form); err != nil {
			h.Logger.Errorf("Parse body request %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		message, err := h.validate(form)
		if err != nil {
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if message != "" {
			c.JSON(http.StatusBadRequest, message)
			return
		}

		if form.Recipient != pkg.Recipient {
			pkg.Recipient = form.Recipient
		}

		if form.PhoneNumber != pkg.PhoneNumber {
			pkg.PhoneNumber = form.PhoneNumber
		}

		if form.Address1 != pkg.Address1 {
			pkg.Address1 = form.Address1
		}

		if form.Address2 != pkg.Address2 {
			pkg.Address2 = form.Address2
		}

		if form.City != pkg.City {
			pkg.City = form.City
		}

		if form.StateCode != pkg.StateCode {
			pkg.StateCode = form.StateCode
		}

		if form.Zipcode != pkg.Zipcode {
			pkg.Zipcode = form.Zipcode
		}

		if form.CountryCode != pkg.CountryCode {
			pkg.CountryCode = form.CountryCode
		}

		cost, message, err := h.EstimateCost(c, pkg)
		if err != nil {
			h.Logger.Errorf("estimate cost: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if message != "" {
			c.JSON(http.StatusBadRequest, message)
			return
		}

		cost = utils.Ceil(cost+viper.GetFloat64("extra_fees.reship_fee"), 2)

		c.JSON(http.StatusOK, ReshipEstimateCostResponse{Success: true, TotalAmount: cost})
	}
}

func (h *PackageHandler) ImportTracking() gin.HandlerFunc {
	return func(c *gin.Context) {
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

		// Upload to s3
		f, err := handlerFile.Open()
		if err != nil {
			h.Logger.Error("Cant Open file: ", err)
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		fileReader, err := excelize.OpenReader(f)
		if err != nil {
			h.Logger.Errorf("Error when process excel file: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		rows, err := fileReader.GetRows("Sheet1")
		if err != nil {
			h.Logger.Errorf("Error when process excel file: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		var codes []string
		data := make(map[string]*ImportRowForm)
		var deliverdPackageCodes []string
		for _, row := range rows {
			if len(row) < 3 {
				continue
			}
			code := string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[0])
			tracking := string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[1])
			label := string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[2])

			if len(row) >= 4 {
				value := string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[3])
				if strings.ToUpper(value) == "YES" {
					deliverdPackageCodes = append(deliverdPackageCodes, code)
					continue
				}
			}

			h.Logger.Info("code: ", code)
			codes = append(codes, code)
			data[code] = &ImportRowForm{
				TrackingNumber: tracking,
				Label:          label,
			}
		}

		if len(codes) == 0 && len(deliverdPackageCodes) == 0 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		if len(deliverdPackageCodes) > 0 {
			err = h.PackageManager.UpdateDeliveriedPackage(deliverdPackageCodes)
			if err != nil {
				h.Logger.Errorf("UpdateDeliveriedPackage error: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			c.JSON(http.StatusOK, ImportTrackingResponse{true})
			return
		}

		if len(codes) > 0 {
			var packages []struct {
				ID          int64   `json:"id"`
				Code        string  `json:"code"`
				Weight      float64 `json:"weight"`
				Width       float64 `json:"width"`
				Height      float64 `json:"height"`
				Length      float64 `json:"length"`
				Label       string  `json:"label"`
				CountryCode string  `json:"country_code"`
			}
			err = h.PackageManager.GetNonTrackingPackageByCodes(codes, &packages)
			if err != nil {
				h.Logger.Errorf("Error get packages: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			if len(packages) == 0 {
				c.JSON(http.StatusNotFound, constant.MessageNotFound)
				return
			}

			var trackings []entity.Tracking
			var trackingIds []int64
			for _, pkg := range packages {
				wh, err := h.WareHouseManager.GetWareHouse(sqlmanager.OptionWareHouse{
					Country: pkg.CountryCode,
					Limit:   1,
					Status:  constant.WareHouseStatusActive,
				})
				if err != nil {
					h.Logger.Errorf("Error get packages: %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				trackingIds = append(trackingIds, pkg.ID)
				h.Logger.Info("entity.Tracking: ", pkg.ID, pkg.CountryCode, wh.ID)
				trackings = append(trackings, entity.Tracking{
					PackageID:      pkg.ID,
					TrackingNumber: data[pkg.Code].TrackingNumber,
					Status:         constant.TrackingStatusSuccess,
					LabelURL:       data[pkg.Code].Label,
					Weight:         pkg.Weight,
					Height:         pkg.Height,
					Width:          pkg.Width,
					HubID:          &wh.ID,
					Length:         pkg.Length,
				})
			}

			if len(trackings) > 0 {
				err = h.TrackingManager.CreateTrackingLabeled(trackings)
				if err != nil {
					h.Logger.Errorf("Save tracking error: %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}
			}
		}

		c.JSON(http.StatusOK, ImportTrackingResponse{true})
	}
}

func (h *PackageHandler) ProcessCNPackage() gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if adminID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		formData := &ProcessPackageForm{}
		if err := c.ShouldBindJSON(formData); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if formData.Id == 0 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		pkg, err := h.PackageManager.GetPackageByPackageID(formData.Id)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, constant.MessageNotFound)
			return
		}

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Detail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		rkey := "package_call_label"
		pkgIDs := []int64{pkg.ID}

		// check package is created / purchased
		if pkg.Status != constant.PackageStatusCreated && pkg.Status != constant.PackageStatusCNPurchased {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		if pkg.Service.Code != constant.ServiceCNCode {
			c.JSON(http.StatusBadRequest, "Chỉ có thể vận đơn cho đơn CN.")
			return
		}

		if pkg.Service.Code == constant.ServiceCNCode && pkg.Status != constant.PackageStatusCNPurchased {
			c.JSON(http.StatusBadRequest, "Gói hàng này chưa được thanh toán, hãy thanh toán giá sản phẩm trước.")
			return
		}

		// check package is exceed
		if pkg.IsPackageExceed && pkg.ShippingFee == 0 {
			c.JSON(http.StatusBadRequest, "Đơn hàng quá cỡ đang được tính giá.Thử lại sau !")
			return
		}

		if pkg.PackageCode != nil && pkg.PackageCode.Status == constant.PackageCodeDisable {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Mã vận đơn %s đã bị hủy", pkg.PackageCode.Code))
			return
		}

		if pkg.ValidateAddress != constant.PackageValidAddress {
			c.JSON(http.StatusBadRequest, "Địa chỉ không hợp lệ")
			return
		}

		if pkg.Service.Code == constant.ServiceCNCode && pkg.CustomCNBarcode == nil {
			c.JSON(http.StatusBadRequest, "Đơn hàng CN Exclusive cần bổ sung mã vạch tự tạo, hãy cập nhật thông tin đơn hàng")
			return
		}

		if pkg.Service.Code != constant.ServiceFBACode {
			isCallLabel, err := h.Redis.SIsMember(c, rkey, pkg.ID).Result()
			if err != nil {
				c.JSON(http.StatusBadRequest, constant.MessageServerInternalError)
				return
			}

			if isCallLabel {
				c.JSON(http.StatusBadRequest, fmt.Sprintf("Đơn hàng #%s đang được tạo mã tracking", pkg.OrderNumber))
				return
			}
		}

		user, err := h.UserManager.GetUserByID(pkg.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		// create bill
		bill, err := h.BillManager.GetOrCreateNowBill(user.ID)
		if err != nil {
			h.Logger.Errorf("Get bill error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.APIResponseMessageServerInternalError)
			return
		}

		peakFee, err := h.BillManager.GetExtraFeeTypeByID(constant.ExtraFeeTypePeak)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("get extra peak fee: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		var shippingFee float64 = 0
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

		var extraFee float64 = 0
		for _, fee := range pkg.ExtraFee {
			extraFee += fee.Amount
		}

		shippingFee += pkg.ShippingFee + extraFee
		shippingFee = utils.ToFixed(shippingFee, 2)

		// check user balance is greater than shipping fee
		isPackageCN := pkg.Service.Code == constant.ServiceCNCode
		if isPackageCN {
			if user.Balance < shippingFee {
				c.JSON(http.StatusInternalServerError, "Số dư ví không đủ. Vui lòng nạp thêm")
				return
			}
		} else {
			if user.Balance < shippingFee && (user.UserInfo == nil || user.UserInfo.DebtMaxAmount <= 0) {
				c.JSON(http.StatusInternalServerError, "Số dư ví không đủ. Vui lòng nạp thêm")
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
		}

		if len(pkgIDs) > 0 {
			for _, id := range pkgIDs {
				_ = h.Redis.SAdd(c, rkey, id).Err()
			}

			err = h.ShipmentCreateLabelHandler.Handle(c, pkgIDs, true, isPackageCN, 0)
			if err != nil {
				h.Logger.Error("Error publish message queue shipment-create-label: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"success":        true,
			"promotionLabel": false,
		})
	}
}

func (h *PackageHandler) EstimateCost(c context.Context, sp *entity.Package) (float64, string, error) {
	if sp.Tracking == nil || sp.Tracking.CarrierID < 1 {
		return 0, "Loại vận chuyển không hợp lệ", nil
	}

	carrierDB, err := h.TrackingManager.GetCarrierByID(sp.Tracking.CarrierID)
	if err != nil {
		h.Logger.Errorf("Get carrier from db error %v", err)
		return 0, "", err
	}

	var carrier = providers.NewCarrier(carrierDB.Code, sp.UserID)
	if carrier == nil {
		return 0, "Loại vận chuyển không hợp lệ", nil
	}

	warehouse, err := h.WareHouseManager.GetWareHouse(sqlmanager.OptionWareHouse{ID: utils.Int64Value(sp.Tracking.HubID)})
	if err != nil {
		h.Logger.Errorf("get package detail: %v", err)
		return 0, "", err
	}

	address1 := sp.Address1
	if address1 == "" && sp.Address2 != "" {
		address1 = sp.Address2
	}

	body := providers.RequestCreateLabel{
		ID:                     sp.ID,
		OrderNumber:            sp.OrderNumber,
		Code:                   sp.PackageCode.Code,
		Weight:                 sp.ActualWeight,
		Company:                sp.Company,
		FirstName:              sp.Recipient,
		LastName:               sp.Recipient,
		FullName:               sp.Recipient,
		City:                   sp.City,
		Address1:               address1,
		Address2:               sp.Address2,
		State:                  sp.StateCode,
		Zipcode:                sp.Zipcode,
		Phone:                  sp.PhoneNumber,
		Country:                sp.CountryCode,
		Height:                 sp.ActualHeight,
		Length:                 sp.ActualLength,
		Width:                  sp.ActualWidth,
		IsExceedPkg:            sp.IsPackageExceed,
		DistanceUnit:           "in",
		DomesticCarrierService: sp.Service.DomesticCarrierService,

		WarehouseCompany:  warehouse.Company,
		WarehouseCity:     warehouse.City,
		WarehouseAddress1: warehouse.Address,
		WarehouseState:    warehouse.State,
		WarehouseZipcode:  warehouse.Zipcode,
		WarehouseCountry:  warehouse.Country,
		WarehousePhone:    warehouse.Phone,
	}

	if body.Weight == 0 {
		body.Weight = sp.Weight
	}
	if body.Width == 0 {
		body.Width = sp.Width
	}
	if body.Height == 0 {
		body.Height = sp.Height
	}
	if body.Length == 0 {
		body.Length = sp.Length
	}

	if body.Country == "US" {
		weight, length, height, width, err := h.CreateLabel.Fake(c, body.Weight, body.Length, body.Height, body.Width)
		if err != nil {
			return 0, "", err
		}

		body.Weight = weight
		body.Length = length
		body.Height = height
		body.Width = width
	}

	res, errAudit, err := carrier.EstimateCost(body)
	if errAudit != nil {
		return 0, errAudit.Error(), nil
	}
	if err != nil {
		return 0, "", err
	}

	return res.TotalCost, "", nil
}

// validate kiem tra du lieu dau vao
func (h *PackageHandler) validate(form *ReshipForm) (string, error) {
	form.Recipient = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Recipient)
	if form.Recipient == "" {
		return "Tên người nhận không để trống", nil
	}

	form.PhoneNumber = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.PhoneNumber)
	var regexPhone = regexp.MustCompile("^[0-9 +()-]*$")
	if form.PhoneNumber != "" && !regexPhone.MatchString(form.PhoneNumber) {
		return "Số điện thoại người nhận không đúng format", nil
	}

	if form.Address1 = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Address1); form.Address1 == "" {
		return "Địa chỉ người nhận không để trống", nil
	}

	if len(form.Address1) > 200 {
		return "Địa chỉ người nhận không được vượt quá 200 ký tự", nil
	}

	form.Address2 = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Address2)
	if form.Address2 != "" && len(form.Address2) > 200 {
		return "Địa chỉ người nhận phụ không được vượt quá 200 ký tự", nil
	}

	form.CountryCode = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.CountryCode)
	if form.CountryCode == "" {
		return "Mã quốc gia không để trống", nil
	}

	form.CountryCode = strings.ToUpper(form.CountryCode)
	if form.CountryCode == "AU" || form.CountryCode == "AUSTRALIA" {
		form.CountryCode = "AU"
	} else {
		if form.CountryCode == "UNITED STATES" {
			form.CountryCode = "US"
		}
		if form.CountryCode != "US" {
			return "Mã quốc gia chỉ chấp nhận US(United States) hoặc AU(Australia)", nil
		}
	}

	form.City = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.City)
	if form.City == "" {
		return "Thành phố không để trống", nil
	}

	if len(form.City) > 50 {
		return "Thành phố không được vượt quá 50 ký tự", nil
	}

	form.StateCode = strings.ToUpper(form.StateCode)

	form.StateCode = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.StateCode)
	if form.StateCode == "" {
		return "Mã vùng không để trống", nil
	}

	state, err := h.StateManager.GetState(sqlmanager.StateOption{Country: form.CountryCode, Code: form.StateCode})
	if err == gorm.ErrRecordNotFound {
		return "Mã vùng không hợp lệ", nil
	}

	if err != nil {
		h.Logger.Errorf("Error get list state US: %v", err)
		return "", err
	}

	if state.Status != constant.StatusActive {
		return "Mã vùng không hỗ trợ ship", nil
	}

	if state.Code != "" {
		form.StateCode = state.Code
	}

	form.Zipcode = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Zipcode)
	if form.Zipcode == "" {
		return "Mã bưu điện không để trống", nil
	}

	if len(form.Zipcode) > 15 {
		return "Mã bưu điện không được vượt quá 15 ký tự", nil
	}

	var zipcodeRegex = regexp.MustCompile("^[0-9-]*$")
	if !zipcodeRegex.MatchString(form.Zipcode) {
		return "Mã bưu điện không đúng format", nil
	}

	return "", nil
}
