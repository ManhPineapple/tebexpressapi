package sqlmanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"reflect"
	"strconv"
	"strings"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/createlabel"
	"tebexpressapi/pkg/dto"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/dbgorm"
	"time"

	"gorm.io/gorm/clause"

	"tebexpressapi/pkg/utils/string_util"

	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type PackageManager struct {
	db *gorm.DB
}
type PackageQueryOption struct {
	ID                   int64
	IDs                  []int64
	Label                string
	Code                 string
	Codes                []string
	Status               int
	StatusArr            []int64
	IgnoreStatusArr      []int64
	StatusString         string
	StartDate            string
	EndDate              string
	UserID               int64
	Preload              []string
	Limit                int
	Offset               int
	Search               string
	SearchBy             string
	Order                string
	IgnoreUsers          []int64
	CodeAndTracking      []string
	QueryAlert           bool
	AlertValue           int
	SupportID            int64
	WarehouseID          int64
	IsWarehouseRole      bool
	HubID                int64
	IsRequestReship      int
	Sort                 string
	OrderNumber          string
	TrackingNumber       string
	TrackingStatus       int64
	IsFba                bool
	ExceptFba            bool
	CustomerShipmentID   int64 `json:"customer_shipment_id"`
	InWarehouseStartDate string
	InWarehouseEndDate   string
	CheckWarehouseSubDay int
	CheckAddPoinDay      int
	DeliveredStartDate   string
	DeliveredEndDate     string
	IsPreloadRefund      bool
	IsBookmark           bool
	LoadShipment         bool
	PartnerID            int64

	ServiceCode        string
	IgnoreServiceCodes []string
	CustomCNBarcode    string
	HasTiktokLabel     bool
	NeedToUploadLabel  bool
	NeedToOcr          bool
	IsEarlyScan        bool
	IsWeightScanned    bool
}

type CouponQueryOption struct {
	Select          string
	Limit           int
	Offset          int
	LoadUserCoupon  bool
	Status          int
	Search          string
	Code            string
	SearchBy        string
	ID              int64
	CustomerID      int64
	IsShow          bool
	IsUseable       bool
	CoponUserID     int64
	Order           string
	Type            []int
	IsActiveNowTime bool
	IsUsed          bool
}

type PackageCodeQueryOption struct {
	ID            int64
	IDs           []int64
	Code          string
	SearchCode    string
	Status        int
	UserID        int64
	Codes         []string
	Select        string
	PackageCodeID int64
}

type ExportForm struct {
	UserID          int64    `json:"user_id"`
	Status          []string `json:"status"`
	StatusInt       []int64  `json:"status_int"`
	StartDate       string   `json:"start_date"`
	EndDate         string   `json:"end_date"`
	IgnoreUsers     []int64
	WarehouseID     int64 `json:"warehouse_id"`
	IsWarehouseRole bool
}
type ExportPackageWrongWeightForm struct {
	FullName  string `json:"full_name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

func NewPackageManager(db *gorm.DB) *PackageManager {
	return &PackageManager{
		db: db,
	}
}

func (m PackageManager) BuildPackageCodeQuery(opts PackageCodeQueryOption) *gorm.DB {
	db := m.db
	if opts.Select != "" {
		db = db.Select(opts.Select)
	}

	if opts.ID != 0 {
		db = db.Joins("JOIN packages on packages.package_code_id = package_codes.id").Where("packages.id = ?", opts.ID)
	}

	if len(opts.IDs) > 0 {
		db = db.Joins("JOIN packages on packages.package_code_id = package_codes.id").Where("packages.id IN (?)", opts.IDs)
	}

	if len(opts.Codes) > 0 {
		db = db.Joins("JOIN packages on packages.package_code_id = package_codes.id")
		db = db.Where("package_codes.code IN (?) OR packages.custom_cn_barcode IN (?)", opts.Codes, opts.Codes)
	}

	if opts.Code != "" {
		db = db.Joins("JOIN packages on packages.package_code_id = package_codes.id")
		db = db.Joins("LEFT JOIN trackings ON trackings.package_id = packages.id").
			Where("trackings.status != ?", constant.TrackingStatusCanceled).
			Where("(package_codes.code = ? OR packages.id = (?) OR packages.custom_cn_barcode = ? OR packages.order_number = ?)",
				opts.Code,
				m.db.Model(&entity.Tracking{}).
					Select("package_id").
					Limit(1).
					Where("tracking_number = ?", opts.Code),
				opts.Code,
				opts.Code,
			)
	}

	if len(opts.SearchCode) > 0 {
		db = db.Where("package_codes.code = ?", opts.SearchCode)
	}

	if opts.Status > 0 {
		db = db.Where("package_codes.status=?", opts.Status)
	}

	if opts.UserID > 0 {
		db = db.Where("package_codes.user_id=?", opts.UserID)
	}

	if opts.PackageCodeID > 0 {
		db = db.Where("package_codes.id = ?", opts.PackageCodeID)
	}

	return db
}

func (m PackageManager) BuildPackageQuery(opts PackageQueryOption) *gorm.DB {
	db := m.db

	if len(opts.IgnoreUsers) > 0 {
		db = db.Where("packages.user_id NOT IN (?)", opts.IgnoreUsers)
	}

	if opts.ID > 0 {
		db = db.Where("packages.id = ?", opts.ID)
	}

	if opts.PartnerID > 0 {
		db = db.Where("packages.partner_id = ?", opts.PartnerID)
	}

	if opts.Label != "" {
		db = db.Where("packages.label = ?", opts.Label)
	}

	if opts.OrderNumber != "" {
		db = db.Where("packages.order_number = ?", opts.OrderNumber)
	}

	if opts.Code != "" {
		db = db.Joins("LEFT JOIN package_codes on package_codes.id = packages.package_code_id")
		db = db.Joins("LEFT JOIN trackings ON trackings.package_id = packages.id AND trackings.status != ? ", constant.TrackingStatusCanceled)
		db = db.Where("((package_codes.code = ? AND package_codes.status = ? ) OR packages.order_number = ? OR trackings.tracking_number = ?) ", opts.Code, constant.PackageCodeEnable, opts.Code, opts.Code)
	}

	if opts.CustomCNBarcode != "" {
		db = db.Where("packages.custom_cn_barcode = ?", opts.CustomCNBarcode)
	}

	if opts.HasTiktokLabel {
		db = db.Where("packages.custom_tiktok_barcode != ''")
	}

	if opts.NeedToUploadLabel {
		db = db.Where("packages.label IS NOT NULL AND packages.label LIKE 'http%'")
	}

	if opts.NeedToOcr {
		db = db.
			Joins("LEFT JOIN trackings ON trackings.package_id = packages.id AND trackings.status != ?", constant.TrackingStatusCanceled).
			Where("trackings.package_id IS NULL")
	}

	if opts.IsEarlyScan {
		db = db.Where("packages.is_early_scan = ?", opts.IsEarlyScan)
	}

	if opts.ServiceCode != "" || len(opts.IgnoreServiceCodes) > 0 || opts.ExceptFba {
		db = db.Joins("JOIN services ON services.id = packages.service_id")

		if opts.ServiceCode != "" {
			db = db.Where("services.code = ?", opts.ServiceCode)
		}
		if len(opts.IgnoreServiceCodes) > 0 {
			db = db.Where("services.code NOT IN ?", opts.IgnoreServiceCodes)
		}
		if opts.ExceptFba {
			db = db.Where("services.code != ?", constant.ServiceFBACode)
		}
	}

	if opts.Status > 0 {
		db = db.Where("packages.status = ?", opts.Status)
	}

	if len(opts.StatusArr) > 0 {
		db = db.Where("packages.status IN (?)", opts.StatusArr)
	}

	if len(opts.IgnoreStatusArr) > 0 {
		db = db.Where("packages.status NOT IN ?", opts.IgnoreStatusArr)
	}

	if len(opts.StartDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(packages.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)", opts.StartDate)
	}

	if len(opts.EndDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(packages.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)", opts.EndDate)
	}

	if opts.IsWeightScanned {
		db = db.Where("packages.scan_weight_at IS NOT NULL")
	}

	if len(opts.InWarehouseStartDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(packages.checkin_warehouse_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)", opts.InWarehouseStartDate)
	}

	if len(opts.InWarehouseEndDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(packages.checkin_warehouse_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)", opts.InWarehouseEndDate)
	}

	if opts.CheckWarehouseSubDay > 0 {
		db = db.Where("DATE(convert_tz(packages.checkin_warehouse_at, @@session.time_zone,'+07:00')) = CURDATE() - INTERVAL ? DAY  ", opts.CheckWarehouseSubDay)
		db = db.Where("packages.status <> ?", constant.PackageStatusCancelled)
	}

	if opts.CheckAddPoinDay > 0 {
		db = db.Where("id IN (?)", m.db.Model(&entity.PackageDeliverLog{}).Select("package_id").Where("status = ? AND type = ? AND DATE(convert_tz(created_at, @@session.time_zone,'+07:00'))  > (NOW() - INTERVAL ? DAY) AND DATE(convert_tz(created_at,@@session.time_zone,'+07:00')) < CURDATE()", constant.DeliverLogTebexpressInTransit, constant.PackageDeliverLogTypeInTransit, opts.CheckAddPoinDay))
	}

	if opts.UserID > 0 {
		db = db.Where("packages.user_id = ?", opts.UserID)
	}
	if opts.SupportID > 0 {
		db = db.Joins("JOIN users aliasuser ON aliasuser.id=packages.user_id")
		db = db.Joins("JOIN user_permissions  ON aliasuser.id=user_permissions.customer_id")
		db = db.Where("user_permissions.support_id = ?", opts.SupportID)

	}

	if len(opts.IDs) > 0 {
		db = db.Where("packages.id IN (?)", opts.IDs)
	}

	if len(opts.Codes) > 0 {
		db = db.Joins("LEFT JOIN package_codes on package_codes.id = packages.package_code_id")
		db = db.Where("package_codes.code IN (?)", opts.Codes)
	}

	if len(opts.CodeAndTracking) > 0 {
		db = db.Joins("LEFT JOIN package_codes on package_codes.id = packages.package_code_id")
		db = db.Joins("LEFT JOIN trackings on trackings.package_id = packages.id AND trackings.status!=?", constant.TrackingStatusCanceled)
		db = db.Where("package_codes.code IN (?) OR trackings.tracking_number IN (?)", opts.CodeAndTracking, opts.CodeAndTracking)
	}

	if len(opts.TrackingNumber) > 0 {
		db = db.Where("id IN (?)", m.db.Model(&entity.Tracking{}).Select("package_id").Where("tracking_number = ? AND status != ?", opts.TrackingNumber, constant.TrackingStatusCanceled))
	}

	if opts.CustomerShipmentID > 0 {
		db = db.Where("customer_shipment_id = ? ", opts.CustomerShipmentID)
	}

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	} else {
		db = db.Limit(200)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	if opts.Search != "" {
		switch opts.SearchBy {
		case "code":
			db = db.Joins("LEFT JOIN package_codes on package_codes.id = packages.package_code_id").
				Joins("LEFT JOIN trackings ON trackings.package_id = packages.id AND trackings.status != ? ", constant.TrackingStatusCanceled).
				Where("((package_codes.code = ? AND package_codes.status = ? ) OR packages.order_number = ? OR trackings.tracking_number = ?) ", opts.Search, constant.PackageCodeEnable, opts.Search, opts.Search)
		case "recipient":
			db = db.Where("packages.recipient LIKE (?)", fmt.Sprintf("%%%s%%", opts.Search))
		case "phone":
			db = db.Where("packages.phone_number LIKE (?)", fmt.Sprintf("%%%s%%", opts.Search))
		case "order_number":
			db = db.Where("packages.order_number = ?", opts.Search)
		case "state_code":
			db = db.Where("packages.state_code = ?", opts.Search)
		case "zipcode":
			db = db.Where("packages.zipcode = ?", opts.Search)
		case "sku":
			db = db.Joins("LEFT JOIN package_products ON package_products.package_id = packages.id")
			db = db.Joins("LEFT JOIN products ON products.id = package_products.product_id")
			db = db.Where("products.sku LIKE (?) AND package_products.status = ? AND products.status = ?", fmt.Sprintf("%%%s%%", opts.Search), constant.StatusActive, constant.StatusActive)
		case "account":
			db = db.Joins("JOIN users ON users.id = packages.user_id")
			if strings.Contains(opts.Search, "@") {
				db = db.Where("users.email = ?", opts.Search)
			} else {
				db = db.Where("users.phone_number = ? OR users.full_name LIKE ?", opts.Search, fmt.Sprintf("%%%s%%", opts.Search))
			}
		case "tracking":
			db = db.Joins("JOIN trackings ON trackings.package_id = packages.id").
				Where("trackings.tracking_number = ? AND trackings.status = ?", opts.Search, constant.TrackingStatusSuccess)

		case "container":
			db = db.Joins("JOIN container_items ON container_items.package_id = packages.id").
				Joins("JOIN containers ON containers.id = container_items.container_id").
				Where("containers.code = ?", opts.Search)
		default:
			db = db.Joins("JOIN package_codes on package_codes.id = packages.package_code_id").Where("package_codes.code = ?", opts.Search)
		}
	}

	if opts.QueryAlert {
		db = db.Where("packages.alert > ?", constant.PackageAlertTypeDisable)
	}

	if opts.AlertValue > 0 {
		db = db.Where("packages.alert = ?", opts.AlertValue)

		if opts.AlertValue == constant.PackageAlertTypeOverPretransit {
			if len(opts.Sort) > 0 {
				db = db.Order("alert_at " + opts.Sort)
			}
		}
	}

	if len(opts.Preload) > 0 {
		for _, col := range opts.Preload {
			db = db.Preload(col)
		}
	}

	if opts.WarehouseID > 0 {
		if opts.IsWarehouseRole {
			subquery := m.db.Model(&entity.Warehouse{}).Where("type = ?", constant.WareHouseTypeInternational).Where("warehouses.status = ?", constant.WareHouseStatusActive).Select("id")
			db = db.Where("packages.warehouse_id IN (?) or packages.warehouse_id = ?", subquery, opts.WarehouseID)
		} else {
			db = db.Where("packages.warehouse_id = ?", opts.WarehouseID)
		}
	}

	if opts.IsFba {
		db = db.Where("packages.service_id IN (?)", m.db.Model(&entity.Service{}).Select("id").Where("code = ?", constant.ServiceFBACode))
	}

	if opts.IsBookmark {
		db = db.Where("packages.is_bookmark = ?", opts.IsBookmark)
	}

	if opts.IsPreloadRefund {
		db = db.Preload("PackageRefunds")
	}
	return db
}

func (m PackageManager) BuildPackageRefundQuery(opts PackageQueryOption) *gorm.DB {
	db := m.db
	db = db.Joins("LEFT JOIN packages on packages.id = package_refunds.package_id")

	if opts.Search != "" {
		db = db.Joins("JOIN package_codes on package_codes.id = packages.package_code_id").Where("package_codes.code = ?", opts.Search)
	}

	if opts.Status > 0 {
		db = db.Where("package_refunds.status = ?", opts.Status)
	}
	if opts.ID > 0 {
		db = db.Where("package_refunds.package_id = ?", opts.ID)
	}

	if len(opts.StartDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(package_refunds.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)", opts.StartDate)
	}

	if len(opts.EndDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(package_refunds.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)", opts.EndDate)
	}

	if opts.UserID > 0 {
		db = db.Where("packages.user_id = ?", opts.UserID)
	}

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	return db
}

func (m PackageManager) GetPackagesInCustomerShipment(opts PackageQueryOption) ([]entity.Package, error) {
	db := m.BuildPackageQuery(opts)
	packages := []entity.Package{}
	db = db.Preload("Tracking", func(db *gorm.DB) *gorm.DB {
		db = db.Joins("JOIN packages ON packages.id = trackings.package_id")
		db = db.Where("trackings.status != ?", constant.TrackingStatusCanceled)
		return db
	})
	db = db.Preload("PackageCode", func(db *gorm.DB) *gorm.DB {
		db = db.Select("code", "id")
		return db
	})
	db = db.Preload("ExtraFee", func(db *gorm.DB) *gorm.DB {
		db = db.Where("extra_fees.status = ?", constant.ExtraFeeStatusEnable)
		return db
	})

	db = db.Preload("Service")
	db = db.Order("id DESC")
	db = db.Find(&packages)
	return packages, db.Error
}

func (m PackageManager) GetPackagesContainer(opts PackageQueryOption) ([]entity.Package, error) {
	db := m.BuildPackageQuery(opts)
	db = db.Model(&entity.Package{})
	db = db.Preload("ExtraFee", func(db *gorm.DB) *gorm.DB {
		db = db.Where("extra_fees.status = ?", constant.ExtraFeeStatusEnable)
		return db
	})
	db = db.Preload("Tracking", func(db *gorm.DB) *gorm.DB {
		db = db.Joins("JOIN packages ON packages.id = trackings.package_id")
		db = db.Where("trackings.status != ?", constant.TrackingStatusCanceled)
		return db
	})
	db = db.Preload("PackageCode")
	db = db.Preload("User")

	db = db.Preload("Service.DomesticCarrier")
	var order = "FIELD(packages.id"
	for _, id := range opts.IDs {
		order += "," + cast.ToString(id)
	}
	order += ")"
	db = db.Order(order)

	packages := []entity.Package{}
	db = db.Find(&packages)
	return packages, db.Error
}

func (m PackageManager) GetPackages(opts PackageQueryOption) ([]entity.Package, error) {
	db := m.BuildPackageQuery(opts)
	db = db.Model(&entity.Package{})
	db = db.Preload("ExtraFee", func(db *gorm.DB) *gorm.DB {
		db = db.Where("extra_fees.status = ?", constant.ExtraFeeStatusEnable)
		return db
	})
	db = db.Preload("Tracking", func(db *gorm.DB) *gorm.DB {
		db = db.Joins("JOIN packages ON packages.id = trackings.package_id")

		if opts.TrackingStatus > 0 {
			db = db.Where("trackings.status = ?", opts.TrackingStatus)
		} else {
			db = db.Where("trackings.status != ?", constant.TrackingStatusCanceled)
		}
		return db
	})
	db = db.Preload("PackageCode")
	db = db.Preload("User")
	db = db.Preload("PackageProducts")
	db = db.Preload("ContainerItem.Container")
	db = db.Preload("Service.DomesticCarrier")

	if opts.Order != "" {
		db = db.Order(opts.Order)
	} else {
		db = db.Order("packages.id DESC")
	}

	packages := []entity.Package{}
	db = db.Find(&packages)
	return packages, db.Error
}

func (m PackageManager) GetNonTrackingPackageByCodes(codes []string, result interface{}) error {
	var statusScan = []int{
		constant.PackageStatusPendingPickup,
		constant.PackageStatusPicked,
		constant.PackageStatusWareHouseLabeled,
		constant.PackageStatusWareHouseInContainer,
		constant.PackageStatusWareHouseInShipment,
		constant.PackageStatusWareHouseExport,
		constant.PackageStatusImportHub,
		constant.PackageStatusExportHub,
	}
	db := m.db
	db = db.Model(&entity.Package{}).Joins("INNER JOIN package_codes ON package_codes.id = packages.package_code_id")
	db = db.Joins("LEFT JOIN trackings ON trackings.package_id = packages.id AND trackings.status = ?", constant.TrackingStatusSuccess)
	db = db.Where("package_codes.code IN (?)", codes)
	db = db.Where("packages.status IN (?)", statusScan)
	db = db.Where("trackings.tracking_number IS NULL")
	db = db.Select("packages.id,package_codes.code,packages.weight,packages.width,packages.height,packages.length,packages.label,packages.country_code")
	db = db.Scan(result)
	return db.Error
}

func (m PackageManager) GetNonTrackingAuPackages(result interface{}, opts PackageQueryOption) error {
	var statusScan = []int{
		constant.PackageStatusPendingPickup,
		constant.PackageStatusPicked,
		constant.PackageStatusWareHouseLabeled,
		constant.PackageStatusWareHouseInContainer,
		constant.PackageStatusWareHouseInShipment,
		constant.PackageStatusWareHouseExport,
		constant.PackageStatusImportHub,
		constant.PackageStatusExportHub,
	}
	db := m.db
	db = db.Model(&entity.Package{}).
		Joins("INNER JOIN package_codes ON package_codes.id = packages.package_code_id AND package_codes.status = ?", constant.PackageCodeEnable).
		Joins("INNER JOIN users ON users.id = packages.user_id").
		Joins("LEFT JOIN trackings ON trackings.package_id = packages.id AND trackings.status = ?", constant.TrackingStatusSuccess)
	db = db.Where("packages.status IN (?) AND packages.country_code = ?", statusScan, "AU")
	db = db.Where("trackings.tracking_number IS NULL")

	if len(opts.StartDate) > 0 {
		db = db.Where(`DATE_FORMAT(convert_tz(package_codes.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)`, opts.StartDate)
	}

	if len(opts.EndDate) > 0 {
		db = db.Where(`DATE_FORMAT(convert_tz(package_codes.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)`, opts.EndDate)
	}

	db = db.Select("packages.order_number,packages.detail,users.full_name,package_codes.code,packages.weight,packages.width,packages.height,packages.length,packages.actual_weight,packages.actual_width,packages.actual_height,packages.actual_length,packages.recipient,packages.company,packages.phone_number,packages.address_1,packages.address_2,packages.city,packages.state_code,packages.zipcode,packages.country_code")
	db = db.Scan(result)
	return db.Error
}

func (m PackageManager) buildQueryPackageReturn(opts PackageQueryOption) *gorm.DB {
	db := m.db

	if opts.Search != "" {
		switch opts.SearchBy {
		case "code":
			db = db.Where("package_codes.code = ?", opts.Search)
		case "recipient":
			db = db.Where("packages.recipient LIKE (?)", fmt.Sprintf("%%%s%%", opts.Search))
		case "phone":
			db = db.Where("packages.phone_number LIKE (?)", fmt.Sprintf("%%%s%%", opts.Search))
		case "order_number":
			db = db.Where("packages.order_number = ?", opts.Search)
		case "state_code":
			db = db.Where("packages.state_code = ?", opts.Search)
		case "zipcode":
			db = db.Where("packages.zipcode = ?", opts.Search)
		case "account":
			if strings.Contains(opts.Search, "@") {
				db = db.Where("users.email = ?", opts.Search)
			} else {
				db = db.Where("users.phone_number = ?", opts.Search)
			}
		case "tracking":
			db = db.Where("trackings.tracking_number = ? AND trackings.status = ?", opts.Search, constant.TrackingStatusSuccess)
		case "customer":
			db = db.Where("users.full_name LIKE (?)", fmt.Sprintf("%%%s%%", opts.Search))
		default:
			db = db.Where("package_codes.code = ? OR packages.id = ? OR packages.order_number = ? OR (trackings.tracking_number = ? AND trackings.status = ?)", opts.Search, opts.Search, opts.Search, opts.Search, constant.TrackingStatusSuccess)
		}
	}

	if opts.IsRequestReship > 0 {
		db = db.Where("packages.request_reship", true)
	}
	if opts.IsRequestReship < 0 {
		db = db.Where("packages.request_reship", false)
	}

	if len(opts.StartDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(packages.returned_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)", opts.StartDate)
	}

	if len(opts.EndDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(packages.returned_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)", opts.EndDate)
	}

	if opts.UserID > 0 {
		db = db.Where("packages.user_id = ?", opts.UserID)
	}

	db = db.Where("packages.alert = ?", opts.AlertValue)
	return db
}

func (m PackageManager) GetPackagesReturn(opts PackageQueryOption, dtoPackage interface{}) error {
	db := m.buildQueryPackageReturn(opts)

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	db = db.Model(&entity.Package{})
	db = db.Select(`
		packages.id as package_id,
		packages.returned_at,
		packages.alert,
		packages.label,
		packages.order_number,
		packages.hub_imported_at,
		users.full_name,
		package_codes.code as package_code,
		package_returns.id as package_return_id,
		trackings.tracking_number,
		packages.shipping_fee,
		packages.request_reship
	`)

	db = db.Joins("LEFT JOIN users ON packages.user_id = users.id")
	db = db.Joins("LEFT JOIN package_codes ON packages.package_code_id = package_codes.id")
	db = db.Joins("LEFT JOIN trackings ON packages.id = trackings.package_id AND trackings.status = ?", constant.TrackingStatusSuccess)
	db = db.Joins("LEFT JOIN package_returns ON packages.id = package_returns.package_id")
	db = db.Order("packages.id DESC")

	if err := db.Scan(dtoPackage).Error; err != nil {
		return db.Error
	}

	if reflect.TypeOf(dtoPackage).Kind() == reflect.Slice || (reflect.TypeOf(dtoPackage).Kind() == reflect.Ptr && reflect.TypeOf(dtoPackage).Elem().Kind() == reflect.Slice) {
		arr := reflect.ValueOf(dtoPackage)
		if arr.Kind() == reflect.Ptr {
			arr = arr.Elem()
		}
		ids := make([]int64, 0)
		for i := 0; i < arr.Len(); i++ {
			v := arr.Index(i)
			if v.Type().Kind() != reflect.Struct && v.Type().Kind() == reflect.Ptr && v.Type().Elem().Kind() != reflect.Struct {
				break
			}

			rid := arr.Index(i).FieldByName("PackageID")
			des := arr.Index(i).FieldByName("Description")

			if rid.Kind() == 0 || des.Kind() == 0 {
				break
			}

			if rid.Type().Name() == "int64" && des.Type().Name() == "string" {
				id := rid.Interface().(int64)
				ids = append(ids, id)
			}

		}
		sub := m.db.Table("package_deliver_logs")
		sub = sub.Select("package_id, description")
		sub = sub.Where("description like '%return%' AND package_id IN (?)", ids)
		sub = sub.Order("id ASC")

		var logs = []entity.PackageDeliverLog{}
		if err := sub.Find(&logs).Error; err != nil {
			return db.Error
		}

		mapLogs := make(map[int64]string)
		for _, v := range logs {
			mapLogs[v.PackageID] = v.Description
		}

		fee := viper.GetFloat64("extra_fees.reship_fee")
		for i := 0; i < arr.Len(); i++ {
			v := arr.Index(i)
			if v.Type().Kind() != reflect.Struct && v.Type().Kind() == reflect.Ptr && v.Type().Elem().Kind() != reflect.Struct {
				break
			}

			rid := arr.Index(i).FieldByName("PackageID")
			des := arr.Index(i).FieldByName("Description")

			if rid.Kind() == 0 || des.Kind() == 0 {
				break
			}

			if rid.Type().Name() == "int64" && des.Type().Name() == "string" {
				id := rid.Interface().(int64)
				des.Set(reflect.ValueOf(mapLogs[id]))
			}

			reshipExtraFee := arr.Index(i).FieldByName("ReshipExtraFee")
			if reshipExtraFee.Kind() != 0 && reshipExtraFee.Type().Name() == "float64" {
				reshipExtraFee.Set(reflect.ValueOf(fee))
			}
		}
	}

	return nil
}

func (m PackageManager) CountPackagesReturn(opts PackageQueryOption) (int64, error) {
	db := m.buildQueryPackageReturn(opts)

	db = db.Joins("LEFT JOIN users ON packages.user_id = users.id")
	db = db.Joins("LEFT JOIN package_codes ON packages.package_code_id = package_codes.id")
	db = db.Joins("LEFT JOIN trackings ON packages.id = trackings.package_id AND trackings.status = ?", constant.TrackingStatusSuccess)
	db = db.Joins("LEFT JOIN package_returns ON packages.id = package_returns.package_id")

	db = db.Model(&entity.Package{})

	var count int64
	db = db.Count(&count)

	return count, db.Error
}

func (m PackageManager) GetPackage(opts PackageQueryOption) (entity.Package, error) {
	db := m.BuildPackageQuery(opts)

	pkg := entity.Package{}
	db = db.Preload("PackageCode")
	db = db.Preload("Tracking", func(db *gorm.DB) *gorm.DB {
		db = db.Joins("JOIN packages ON packages.id = trackings.package_id")
		db = db.Where("trackings.status != ?", constant.TrackingStatusCanceled)
		return db
	})

	db = db.Preload("Service").Preload("Service.DomesticCarrier")
	db = db.Find(&pkg)
	return pkg, db.Error
}

func (m PackageManager) FetchPackage(opts PackageQueryOption, field string, result interface{}) error {
	db := m.BuildPackageQuery(opts)
	db = db.Model(&entity.Package{}).Select(field)
	db = db.Scan(result)
	return db.Error
}

func (m PackageManager) GetPackage2(opts PackageQueryOption) (entity.Package, error) {
	db := m.BuildPackageQuery(opts)
	pkg := entity.Package{}
	db = db.First(&pkg)
	return pkg, db.Error
}

func (m PackageManager) GetPackageReturn(search string) (*entity.Package, error) {
	db := m.db.Model(&entity.Package{})
	Q1 := m.db.Model(&entity.PackageCode{}).Select("id").Where("code = ?", search)
	Q2 := m.db.Model(&entity.Tracking{}).Select("package_id").Where("status <> ? AND tracking_number = ?", constant.TrackingStatusCanceled, search)
	db = db.Where("package_code_id IN ( ? ) OR id IN( ? )", Q1, Q2)
	db = db.Select("id,status,hub_id ,order_number,alert,returned_at,package_code_id,country_code,user_id,service_id")
	var pkg *entity.Package
	db = db.Find(&pkg)
	return pkg, db.Error
}
func (m PackageManager) SavePackageReturn(pkg *entity.Package, billId int64, userID int64) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	err := tx.Model(entity.Package{}).Where("id = ?", pkg.ID).UpdateColumns(map[string]interface{}{
		"alert":       pkg.Alert,
		"returned_at": pkg.ReturnedAt,
		"updated_at":  time.Now(),
	}).Error
	if err != nil {
		tx.Rollback()
		return err
	}
	if pkg.CountryCode == "AU" {
		fee := entity.ExtraFee{
			PackageID:      utils.Int64(pkg.ID),
			BillID:         &billId,
			ExtraFeeTypeID: constant.ExtraFeeTypeReturn,
			Description:    "Phí return đơn hàng",
			Amount:         cast.ToFloat64(viper.GetString("fee.returned")),
			Status:         constant.ExtraFeeStatusEnable,
		}
		err = tx.Create(&fee).Error
		if err != nil {
			tx.Rollback()
			return err
		}

		log := &entity.PackageAuditLog{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			ExtraFeeID:    fee.ID,
			PackageID:     utils.Int64Value(fee.PackageID),
			Type:          constant.PackageUpdateReturnPackage,
			Fee:           fee.Amount,
			Value:         fmt.Sprintf("Phí return cho đơn %s", pkg.PackageCode.Code),
			UpdatedUserID: userID,
			Description:   fee.Description,
		}
		if err := tx.Model(&entity.PackageAuditLog{}).Create(&log).Error; err != nil {
			tx.Rollback()
			return err
		}

		transaction := &entity.Transaction{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			UserID: pkg.UserID,
			Type:   constant.TransactionLogTypePay,
			Status: constant.TransactionStatusSuccess,
			Amount: fee.Amount,
			BillID: &billId,
		}

		err := tx.Create(&transaction).Error
		if err != nil {
			fmt.Errorf("Create transaction error %v", err)
			tx.Rollback()
			return err
		}

		transactionLog := &entity.TransactionLog{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			UserID:        pkg.UserID,
			TransactionID: transaction.ID,
			Amount:        transaction.Amount,
			Type:          transaction.Type,
			Status:        transaction.Status,
			BillID:        transaction.BillID,
		}

		if err = tx.Model(entity.TransactionLog{}).Create(transactionLog).Error; err != nil {
			tx.Rollback()
			return err
		}

		info := &entity.UserInfo{}
		if err := tx.First(info).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				tx.Rollback()
				return err
			}

			now := time.Now()
			info = &entity.UserInfo{UserID: pkg.UserID, UpdatedAt: &now, CancelMaxAmount: constant.DefaultCancelMaxAMount}
			if err := tx.Create(info).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		var balanceType string
		if pkg.Service.Code == constant.ServiceCNCode {
			// balanceType = "balance_china" // remove china wallet
			balanceType = "balance"
		} else {
			balanceType = "balance"
		}

		sqlString := fmt.Sprintf("UPDATE users SET %s = %s - ?, updated_at = ? WHERE id = ?", balanceType, balanceType)
		if err := tx.Exec(sqlString, fee.Amount, time.Now(), pkg.UserID).Error; err != nil {
			tx.Rollback()
			return err
		}

		sqlString = `UPDATE user_infos SET debt_time = ? WHERE user_id = ? AND debt_time IS NULL AND (SELECT balance FROM users WHERE id = ? limit 1) < 0`
		if err := tx.Exec(sqlString, time.Now(), pkg.UserID, pkg.UserID).Error; err != nil {
			tx.Rollback()
			return err
		}

		sqlString = `UPDATE bills SET extra_fee = extra_fee + ? WHERE id = ?`
		if err := tx.Exec(sqlString, fee.Amount, billId).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func (m PackageManager) GetPackagesCode(opts PackageCodeQueryOption) ([]entity.PackageCode, error) {
	db := m.BuildPackageCodeQuery(opts)

	packages := []entity.PackageCode{}
	db = db.Find(&packages).Select("package_codes.id, package_codes.code")
	return packages, db.Error
}

func (m PackageManager) GetContainerPackages(IDs []int64) ([]entity.ContainerItem, error) {
	db := m.db
	containers := make([]entity.ContainerItem, 0)
	db = db.Model(&entity.ContainerItem{}).Where("package_id IN (?) AND status = ?", IDs, constant.ContainerItemActive).Find(&containers)
	return containers, db.Error
}

func (m PackageManager) SavePackages(packages []entity.Package, logs []entity.PackageDeliverLog) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	err := tx.Save(&packages).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	//delete in checkin_package
	for _, pkg := range packages {
		if err := tx.Model(entity.CheckinPackage{}).Where("package_id = ? ", pkg.ID).Delete(entity.CheckinPackage{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	err = tx.Save(&logs).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m PackageManager) CancelPackages(packages []entity.Package, logs []entity.PackageDeliverLog, ids []int64, refunds []entity.PackageRefund) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	err := tx.Omit(clause.Associations).Save(&packages).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	if err = tx.Model(&entity.Tracking{}).Where("package_id IN (?)", ids).
		Update("status", constant.TrackingStatusCanceled).Error; err != nil {
		fmt.Errorf("update Tracking error %v", err)
		tx.Rollback()
		return err
	}

	cIDs := []int64{}
	for _, p := range packages {
		if p.PackageCodeID != nil {
			cIDs = append(cIDs, *p.PackageCodeID)
		}

	}
	if len(cIDs) > 0 {
		if err = tx.Model(&entity.PackageCode{}).Where("id IN (?) and status = ?", cIDs, constant.PackageCodeTemp).
			Update("status", constant.PackageCodeStatusInActive).Error; err != nil {
			fmt.Errorf("update Package Code 	 error %v", err)
			tx.Rollback()
			return err
		}
	}

	err = tx.Save(&logs).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	if len(refunds) > 0 {
		if err := tx.Omit("Package").Create(&refunds).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m *PackageManager) CreatePackageCodes(pkgs []entity.Package) (error, []*entity.PackageCode) {
	tx := m.db.Begin()
	pCodes := make([]*entity.PackageCode, len(pkgs))
	for i, pkg := range pkgs {
		// gen package code
		if pkg.PackageCode != nil && pkg.PackageCode.Status == constant.PackageCodeTemp {
			pCodes[i] = pkg.PackageCode
			continue
		}

		PackageIDGenCode := pkg.ID
		if pkg.ID > constant.MaxPackageIDGenerate {
			result := &entity.PackageCode{}
			if err := tx.Where(
				"status = ? AND package_id_generate IS NOT NULL AND (user_id <> ? OR (user_id=? AND service_id != ?))",
				constant.PackageCodeDisable,
				pkg.UserID,
				pkg.UserID,
				pkg.ServiceID,
			).First(result).Error; err != nil {
				tx.Rollback()
				return err, nil
			}

			if result.ID == 0 {
				tx.Rollback()
				return errors.New("can not generate code packages"), nil
			}

			PackageIDGenCode = result.PackageIDGenerate
		}

		env := viper.GetString("env")

		NumGen := fmt.Sprintf("%05d%02d%08d", pkg.UserID, pkg.ServiceID, PackageIDGenCode)

		prefixCode := entity.PartnerPrefixCodeMap[pkg.PartnerID]
		Code := fmt.Sprintf("%v%05d%02d%08d%d", prefixCode, pkg.UserID, pkg.ServiceID, PackageIDGenCode, m.GenerateLastDigitCode(NumGen))

		if env == "development" {
			Code = fmt.Sprintf("%s%s", Code, "DEV")
		}

		var count int64
		if err := tx.Model(&entity.PackageCode{}).Where("code=? AND (user_id = ? OR status=?)", Code, pkg.UserID, constant.PackageCodeEnable).Count(&count).Error; err != nil {
			tx.Rollback()
			return err, nil
		}

		if count > 0 {
			return errors.New("create package code duplicate or is active"), nil
		}

		packageCode := entity.PackageCode{
			UserID:            pkg.UserID,
			Code:              Code,
			Status:            constant.PackageCodeTemp,
			ServiceID:         pkg.ServiceID,
			PackageIDGenerate: PackageIDGenCode,
		}

		if err := tx.Create(&packageCode).Error; err != nil {
			tx.Rollback()
			return err, nil
		}

		mapPkg := make(map[string]interface{})
		mapPkg["updated_at"] = time.Now()
		mapPkg["package_code_id"] = packageCode.ID
		err := tx.Model(&entity.Package{}).Where("id = ?", pkg.ID).UpdateColumns(mapPkg).Error
		if err != nil {
			fmt.Errorf("Update package error %v", err)
			tx.Rollback()
			return err, nil
		}
		pCodes[i] = &packageCode
	}

	return tx.Commit().Error, pCodes
}

func (m *PackageManager) CancelPackageCodes(pkgs []entity.Package) error {
	Ids := make([]int64, 0)
	for _, pkg := range pkgs {
		Ids = append(Ids, pkg.PackageCode.ID)
	}
	err := m.db.Model(&entity.PackageCode{}).Where("id IN (?)", Ids).UpdateColumns(
		map[string]interface{}{
			"updated_at": time.Now(),
			"status":     constant.PackageCodeTemp,
		},
	).Error
	return err
}

func (m PackageManager) GetPackageDetail(opts PackageQueryOption) (*entity.Package, error) {
	db := m.BuildPackageQuery(opts)
	packages := &entity.Package{}
	db = db.Preload("Service.DomesticCarrier")
	db = db.Preload("Tracking", func(db *gorm.DB) *gorm.DB {
		db = db.Joins("JOIN packages ON packages.id = trackings.package_id")

		if opts.TrackingStatus > 0 {
			db = db.Where("trackings.status = ?", opts.TrackingStatus)
		} else {
			db = db.Where("trackings.status != ?", constant.TrackingStatusCanceled)
		}

		return db
	})
	db = db.Preload("PackageCode")
	db = db.Preload("User")
	db = db.Preload("PackageReturn")
	db = db.Preload("PackageProducts")

	db = db.Preload("ExtraFee", func(db *gorm.DB) *gorm.DB {
		db = db.Where("extra_fees.status = ?", constant.ExtraFeeStatusEnable)
		return db
	})

	db = db.First(&packages)
	return packages, db.Error
}

func (m PackageManager) CountPackages(opts PackageQueryOption) (int64, error) {
	db := m.BuildPackageQuery(opts)
	var count int64
	db = db.Model(&entity.Package{}).Count(&count)
	return count, db.Error
}

func (m PackageManager) CountAllStatusPackages(opts PackageQueryOption) ([]dto.CountStatusPackage, error) {
	opts.Status = 0
	db := m.BuildPackageQuery(opts)
	db = db.Model(&entity.Package{}).Select("packages.status, COUNT(packages.id) as count")
	db = db.Group("packages.status")
	countStatus := []dto.CountStatusPackage{}
	db = db.Scan(&countStatus)

	return countStatus, db.Error
}

func (m PackageManager) GetDeliverLogsByPkgID(packageID int64) ([]dto.PackageDeliverLogDTO, error) {
	deliverLogs := []dto.PackageDeliverLogDTO{}

	sql := `
		SELECT 
			package_deliver_logs.location,
			package_deliver_logs.status,
			package_deliver_logs.type,
			"" AS code,
			package_deliver_logs.created_at AS ship_time,
			package_deliver_logs.description,
			users.full_name AS updated_user_name,
			users.role AS updated_user_role
		FROM
			package_deliver_logs
		LEFT JOIN users on users.id = package_deliver_logs.user_id
		WHERE package_id=? AND package_deliver_logs.description not in (?, ?)
		UNION
		SELECT 
			location,
			ANY_VALUE(status),
			ANY_VALUE(type),
			ANY_VALUE(code),
			ANY_VALUE(ship_time) AS ship_time,
			ANY_VALUE(description),
			ANY_VALUE(updated_user_name),
			ANY_VALUE(updated_user_role)
		FROM
			(SELECT
				location,
				status,
				30 AS type,
				code,
				ship_time,
				description,
				"" AS updated_user_name,
				"" AS updated_user_role
			FROM
				container_deliver_logs
			WHERE
				code NOT IN (?)
				AND
				container_id=(SELECT container_id FROM container_items WHERE package_id=? AND status=? limit 1)
			ORDER by ship_time DESC, type DESC) AS logs
			GROUP BY logs.location ORDER BY ship_time DESC
	`

	db := m.db.Raw(sql, packageID, "Shipping Label Created, USPS Awaiting Item", "Received data", constant.UPSIgnoreCodeLogs, packageID, constant.ContainerItemActive).Scan(&deliverLogs)
	return deliverLogs, db.Error
}

func (m PackageManager) GetAuditLogsByPkgID(packageID int64) ([]dto.PackageAuditLogDTO, error) {
	auditLogs := []dto.PackageAuditLogDTO{}
	db := m.db
	db = db.Table("package_audit_logs")
	db = db.Select("package_audit_logs.*, users.full_name as updated_user_name,users.role as updated_user_role, extra_fees.amount as extra_fee, extra_fees.status as extra_fee_status")
	db = db.Where("package_audit_logs.package_id = ?", packageID)
	db = db.Joins("left join users on users.id = package_audit_logs.updated_user_id")
	db = db.Joins("left join extra_fees on extra_fees.id = package_audit_logs.extra_fee_id")
	db = db.Order("package_audit_logs.id DESC").Scan(&auditLogs)
	return auditLogs, db.Error
}

func (m PackageManager) GetExtraFeeByPkgID(packageID int64) ([]entity.ExtraFee, error) {
	extraFee := []entity.ExtraFee{}
	db := m.db
	db = db.Where("package_id = ? and extra_fees.status = ? ", packageID, constant.ExtraFeeStatusEnable)
	db = db.Preload("ExtraFeeType")
	db = db.Joins("left join extra_fee_types on extra_fee_types.id = extra_fees.extra_fee_type_id")
	db = db.Where("extra_fee_types.status=?", constant.ExtraFeeStatusEnable)
	db = db.Find(&extraFee)
	return extraFee, db.Error
}
func (m *PackageManager) GetTotalExtrafee(packageID int64) (float64, error) {
	sql := `SELECT SUM(extra_fees.amount) as total FROM extra_fees
	LEFT JOIN extra_fee_types on extra_fee_types.id = extra_fees.extra_fee_type_id
	WHERE extra_fees.package_id = ? and extra_fees.status = ? and extra_fee_types.status= ? `
	var result = struct {
		Total float64 `json:"total"`
	}{}
	db := m.db.Raw(sql, packageID, constant.ExtraFeeStatusEnable, constant.ExtraFeeStatusEnable).Scan(&result)
	return result.Total, db.Error
}

func (m *PackageManager) GetPackageByPackageID(id int64) (*entity.Package, error) {
	packages := &entity.Package{}
	db := m.db.Where("packages.id = ?", id).Preload("PackageCode").Preload("Service").Preload("Service.DomesticCarrier").Preload("ExtraFee")
	db = db.Preload("Tracking", func(db *gorm.DB) *gorm.DB {
		db = db.Joins("JOIN packages ON packages.id = trackings.package_id")
		db = db.Where("trackings.status != ?", constant.TrackingStatusCanceled)
		return db
	})

	db = db.First(packages)
	return packages, db.Error
}

func (m *PackageManager) GetPackageWarehouse(warehouseID int64) (*entity.Warehouse, error) {
	warehouse := &entity.Warehouse{}
	db := m.db.Where("id = ?", warehouseID).First(warehouse)
	return warehouse, db.Error
}

func (m *PackageManager) GetPackageTicketByPackageCode(code string) (*entity.Package, error) {
	packages := &entity.Package{}
	db := m.db.Where("package_codes.code=?", code).
		Joins("LEFT JOIN package_codes ON package_codes.id=packages.package_code_id and package_codes.user_id = packages.user_id").
		Select("packages.*, package_codes.code as code").Preload("PackageCode").
		Preload("Service").First(packages)

	return packages, db.Error
}

func (m *PackageManager) GetPackageByPackageCode(code string) (*entity.Package, error) {
	packages := &entity.Package{}
	db := m.db.Where("(package_codes.code = ? AND package_codes.status = ?) OR (packages.order_number = ? AND packages.status IN ?)", code, constant.PackageCodeEnable, code, []int64{constant.PackageStatusWareHouseLabeled, constant.PackageStatusPicked}).
		Joins("LEFT JOIN package_codes ON package_codes.id=packages.package_code_id and package_codes.user_id = packages.user_id").
		Select("packages.*, package_codes.code as code").Preload("PackageCode").
		Preload("Service").Preload("Tracking", func(db *gorm.DB) *gorm.DB {
		db = db.Where("trackings.status != ?", constant.TrackingStatusCanceled)
		return db
	}).First(packages)

	return packages, db.Error
}

func (m *PackageManager) GetPackageByPackageTrackingNumber(code string) (*entity.Package, error) {
	packages := &entity.Package{}
	db := m.db.Where("trackings.tracking_number = ?", code).Where("packages.status IN ?", []int64{constant.PackageStatusWareHouseLabeled, constant.PackageStatusPicked}).
		Joins("JOIN trackings ON trackings.package_id = packages.id").
		Preload("Service").Preload("Tracking", func(db *gorm.DB) *gorm.DB {
		db = db.Where("trackings.status != ?", constant.TrackingStatusCanceled)
		return db
	}).First(packages)
	return packages, db.Error
}

func (m PackageManager) CreatePackages(packages []*entity.Package, userID int64) ([]int64, error) {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			fmt.Errorf("Panic create packages", r)
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return []int64{}, err
	}

	var packageIDCreated []int64

	for _, Package := range packages {

		err := tx.Omit("Service", "ExtraFee", "Tracking", "PackageCode", "User").Create(&Package).Error
		if err != nil {
			tx.Rollback()
			return []int64{}, err
		}

		deliverLog := &entity.PackageDeliverLog{
			PackageID: Package.ID,
			Status:    constant.DeliverLogTebexpressCreated,
			Type:      constant.PackageDeliverLogTypeCreated,
		}

		if err := tx.Create(&deliverLog).Error; err != nil {
			tx.Rollback()
			return []int64{}, err
		}

		model := dbgorm.Model{CreatedAt: time.Now(), UpdatedAt: time.Now()}
		for _, fee := range Package.ExtraFee {
			if fee.Amount == 0 {
				continue
			}

			extraFee := &entity.ExtraFee{
				Model:          model,
				PackageID:      utils.Int64(Package.ID),
				ExtraFeeTypeID: fee.ExtraFeeTypeID,
				Amount:         fee.Amount,
				Status:         constant.ExtraFeeStatusEnable,
				Description:    fee.Description,
			}

			if err := tx.Save(extraFee).Error; err != nil {
				tx.Rollback()
				return []int64{}, err
			}
		}

		packageIDCreated = append(packageIDCreated, Package.ID)
	}
	return packageIDCreated, tx.Commit().Error
}

func (m PackageManager) GenerateLastDigitCode(numGen string) int64 {
	number := cast.ToFloat64(numGen)
	var total float64
	mapRatio := map[int]float64{
		1:  3,
		2:  8,
		3:  7,
		4:  6,
		5:  5,
		6:  4,
		7:  3,
		8:  2,
		9:  1,
		10: 1,
		11: 2,
		12: 3,
		13: 3,
		14: 2,
		15: 1,
	}
	for i := 1; i <= 15; i++ {
		mod := math.Mod(number, 10)
		total += mod * mapRatio[i]
		number = (number - mod) / 10
	}

	return cast.ToInt64(math.Mod(cast.ToFloat64(total), 10))
}

func (m PackageManager) UpdatePackage(packages *entity.Package, id int64) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			return
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	mapPackageEntity := m.buildMapPackageQuery(packages)
	if err := tx.Model(&entity.Package{}).Where("id = ?", id).UpdateColumns(mapPackageEntity).Error; err != nil {
		fmt.Errorf("Update package error %v", err)
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m *PackageManager) buildMapPackageQuery(packages *entity.Package) map[string]interface{} {
	mapEntity := make(map[string]interface{})
	mapEntity["updated_at"] = time.Now()

	//if packages.Code != "" {
	//	mapEntity["code"] = packages.Code
	//}

	if packages.OrderNumber != "" {
		mapEntity["order_number"] = packages.OrderNumber
	}

	if packages.Detail != "" {
		mapEntity["detail"] = packages.Detail
	}

	if packages.Label != "" {
		mapEntity["label"] = packages.Label
	}

	if packages.Recipient != "" {
		mapEntity["recipient"] = packages.Recipient
	}

	if packages.Company != "" {
		mapEntity["company"] = packages.Company
	}

	if packages.PhoneNumber != "" {
		mapEntity["phone_number"] = packages.PhoneNumber
	}

	if packages.Address1 != "" {
		mapEntity["address_1"] = packages.Address1
	}

	if packages.Address2 != "" {
		mapEntity["address_2"] = packages.Address2
	}

	if packages.City != "" {
		mapEntity["city"] = packages.City
	}

	if packages.StateCode != "" {
		mapEntity["state_code"] = packages.StateCode
	}

	if packages.Zipcode != "" {
		mapEntity["zipcode"] = packages.Zipcode
	}

	if packages.CountryCode != "" {
		mapEntity["country_code"] = packages.CountryCode
	}

	if packages.Weight > 0 {
		mapEntity["weight"] = packages.Weight
	}

	if packages.Width > 0 {
		mapEntity["width"] = packages.Width
	}

	if packages.Length > 0 {
		mapEntity["length"] = packages.Length
	}

	if packages.Height > 0 {
		mapEntity["height"] = packages.Height
	}

	if packages.Status > 0 {
		mapEntity["status"] = packages.Status
	}

	if packages.ServiceID > 0 {
		mapEntity["service_id"] = packages.ServiceID
	}

	if packages.ActualWidth > 0 {
		mapEntity["actual_width"] = packages.ActualWidth
	}

	if packages.ActualHeight > 0 {
		mapEntity["actual_height"] = packages.ActualHeight
	}

	if packages.ActualLength > 0 {
		mapEntity["actual_length"] = packages.ActualLength
	}

	if packages.ActualWeight > 0 {
		mapEntity["actual_weight"] = packages.ActualWeight
	}

	if packages.ShippingFee > 0 {
		mapEntity["shipping_fee"] = packages.ShippingFee
	}

	if packages.LastPrintLabelAt != nil {
		mapEntity["last_print_label_at"] = packages.LastPrintLabelAt
	}

	if packages.ScanWeightAt != nil {
		mapEntity["scan_weight_at"] = packages.ScanWeightAt
	}

	return mapEntity
}

func (m PackageManager) SavePackageAuditLog(packageID int64, typeUpdate int, oldValue string, newValue string, userID int64, fee float64) error {
	packageLog := &entity.PackageAuditLog{
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		PackageID:     packageID,
		Type:          typeUpdate,
		OldValue:      oldValue,
		Value:         newValue,
		UpdatedUserID: userID,
		Fee:           fee,
	}

	db := m.db.Model(&entity.PackageAuditLog{}).Create(&packageLog)
	return db.Error
}

func (m PackageManager) SaveCheckAddressLog(packageID int64, oldValue string, newValue string) error {
	packageLog := &entity.PackageAuditLog{
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		PackageID: packageID,
		Type:      constant.PackageCheckAddressType,
		OldValue:  oldValue,
		Value:     newValue,
		//Description: description,
	}

	db := m.db.Model(&entity.PackageAuditLog{}).Create(&packageLog)
	return db.Error
}

func (m PackageManager) SaveExtraFee(packageID int64, typeUpdate int64, amount float64) error {
	extraFee := &entity.ExtraFee{
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		PackageID:      utils.Int64(packageID),
		ExtraFeeTypeID: typeUpdate,
		Amount:         amount,
		Status:         constant.ExtraFeeStatusEnable,
	}

	db := m.db.Model(&entity.ExtraFee{}).Create(&extraFee)
	return db.Error
}

func (m PackageManager) UpdateExtraFee(packageID int64, priceOutSize float64, userID int64, mapChange []entity.PackageAuditLog) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			return
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	for _, audit := range mapChange {
		audit.UpdatedUserID = userID
		audit.PackageID = packageID

		extraFee := &entity.ExtraFee{}

		if audit.Type == constant.PackageUpdateTypeVolume {

			extraFee = &entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				PackageID:      utils.Int64(packageID),
				ExtraFeeTypeID: constant.ExtraFeeTypeOutSize,
				Amount:         priceOutSize,
				Status:         constant.ExtraFeeStatusEnable,
			}
			extraFeeMap := map[string]interface{}{
				"Status":    constant.ExtraFeeStatusDisable,
				"UpdatedAt": time.Now(),
			}

			if err := tx.Model(&entity.ExtraFee{}).Where("package_id = ? AND extra_fee_type_id = ?", packageID, constant.ExtraFeeTypeOutSize).UpdateColumns(extraFeeMap).Error; err != nil {
				tx.Rollback()
				return err
			}

			if priceOutSize > 0 {

				if err := tx.Model(&entity.ExtraFee{}).Create(&extraFee).Error; err != nil {
					tx.Rollback()
					return err
				}

			}

			if extraFee.ID > 0 {
				audit.ExtraFeeID = extraFee.ID
			}

		}

		if audit.Type == constant.PackageUpdateExtraFeeCNProduct {
			newProductPrice, err := strconv.ParseFloat(audit.Value, 64)
			if err != nil {
				return err
			}
			var cnPricePercentage float64
			if newProductPrice > 200 {
				cnPricePercentage = viper.GetFloat64("extra_fees.cn_high_price_percentage")
			} else {
				cnPricePercentage = viper.GetFloat64("extra_fees.cn_low_price_percentage")
			}
			minProxyPrice := viper.GetFloat64("extra_fees.cn_min_proxy_buying_fee")

			extraFeeMap := map[string]interface{}{
				"amount": newProductPrice,
			}

			result := tx.Model(&entity.ExtraFee{}).
				Where("package_id = ? AND extra_fee_type_id = ?", packageID, constant.ExtraFeeTypeChinaProduct).
				UpdateColumns(extraFeeMap)

			if result.Error != nil {
				tx.Rollback()
				return result.Error
			}

			// If no record was updated, insert a new record
			if result.RowsAffected == 0 {
				newFee := &entity.ExtraFee{
					PackageID:      utils.Int64(packageID),
					ExtraFeeTypeID: constant.ExtraFeeTypeChinaProduct,
					Amount:         newProductPrice,
					Status:         constant.ExtraFeeStatusEnable,
				}
				if err := tx.Save(newFee).Error; err != nil {
					tx.Rollback()
					return err
				}
			}

			// Handle the percentage fee
			var extraFee entity.ExtraFee
			newAmount := math.Max(newProductPrice*cnPricePercentage, minProxyPrice)

			err = tx.Where("package_id = ? AND extra_fee_type_id = ?", packageID, constant.ExtraFeeTypeChinaProductPercentage).
				First(&extraFee).Error

			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					// If not found, create a new record
					extraFee = entity.ExtraFee{
						PackageID:      utils.Int64(packageID),
						ExtraFeeTypeID: constant.ExtraFeeTypeChinaProductPercentage,
						Amount:         newAmount,
						Status:         constant.ExtraFeeStatusEnable,
					}
					if err := tx.Create(&extraFee).Error; err != nil {
						tx.Rollback()
						return err
					}
				} else {
					// Some other error occurred
					tx.Rollback()
					return err
				}
			} else {
				// If found, check if the amount needs an update
				if extraFee.Amount != newAmount {
					if err := tx.Model(&extraFee).Update("amount", newAmount).Error; err != nil {
						tx.Rollback()
						return err
					}
				}
			}
		}

		if audit.Type == constant.PackageUpdateExtraFeeCNShipping {
			valueFloat, err := strconv.ParseFloat(audit.Value, 64)
			if err != nil {
				return err
			}

			extraFeeMap := map[string]interface{}{
				"amount": valueFloat,
			}

			result := tx.Model(&entity.ExtraFee{}).
				Where("package_id = ? AND extra_fee_type_id = ?", packageID, constant.ExtraFeeTypeChinaShipping).
				UpdateColumns(extraFeeMap)

			if result.Error != nil {
				tx.Rollback()
				return result.Error
			}

			if result.RowsAffected == 0 {
				newFee := entity.ExtraFee{
					PackageID:      utils.Int64(packageID),
					ExtraFeeTypeID: constant.ExtraFeeTypeChinaShipping,
					Amount:         valueFloat,
					Status:         constant.ExtraFeeStatusEnable,
				}
				if err := tx.Save(&newFee).Error; err != nil {
					tx.Rollback()
					return err
				}
			}
		}

		if audit.Type == constant.PackageUpdateExtraFeeCNShippingToVN {
			valueFloat, err := strconv.ParseFloat(audit.Value, 64)
			if err != nil {
				return err
			}

			extraFeeMap := map[string]interface{}{
				"amount": valueFloat,
			}

			result := tx.Model(&entity.ExtraFee{}).
				Where("package_id = ? AND extra_fee_type_id = ?", packageID, constant.ExtraFeeTypeCNShippingToVN).
				UpdateColumns(extraFeeMap)

			if result.Error != nil {
				tx.Rollback()
				return result.Error
			}

			if result.RowsAffected == 0 {
				newFee := entity.ExtraFee{
					PackageID:      utils.Int64(packageID),
					ExtraFeeTypeID: constant.ExtraFeeTypeCNShippingToVN,
					Amount:         valueFloat,
					Status:         constant.ExtraFeeStatusEnable,
				}
				if err := tx.Save(&newFee).Error; err != nil {
					tx.Rollback()
					return err
				}
			}
		}

		if audit.Type == constant.PackageUpdateExtraFeeCNLabel {
			valueFloat, err := strconv.ParseFloat(audit.Value, 64)
			if err != nil {
				return err
			}

			extraFeeMap := map[string]interface{}{
				"amount": valueFloat,
			}

			result := tx.Model(&entity.ExtraFee{}).
				Where("package_id = ? AND extra_fee_type_id = ?", packageID, constant.ExtraFeeTypeHandling).
				UpdateColumns(extraFeeMap)

			if result.Error != nil {
				tx.Rollback()
				return result.Error
			}

			if result.RowsAffected == 0 {
				newFee := entity.ExtraFee{
					PackageID:      utils.Int64(packageID),
					ExtraFeeTypeID: constant.ExtraFeeTypeHandling,
					Amount:         valueFloat,
					Status:         constant.ExtraFeeStatusEnable,
				}
				if err := tx.Save(&newFee).Error; err != nil {
					tx.Rollback()
					return err
				}
			}
		}

		if audit.Type == constant.PackageUpdateExtraFeeVatTax {
			valueFloat, err := strconv.ParseFloat(audit.Value, 64)
			if err != nil {
				return err
			}

			extraFeeMap := map[string]interface{}{
				"amount": valueFloat,
			}

			result := tx.Model(&entity.ExtraFee{}).
				Where("package_id = ? AND extra_fee_type_id = ?", packageID, constant.ExtraFeeTypeVatTax).
				UpdateColumns(extraFeeMap)

			if result.Error != nil {
				tx.Rollback()
				return result.Error
			}

			if result.RowsAffected == 0 {
				newFee := entity.ExtraFee{
					PackageID:      utils.Int64(packageID),
					ExtraFeeTypeID: constant.ExtraFeeTypeVatTax,
					Amount:         valueFloat,
					Status:         constant.ExtraFeeStatusEnable,
				}
				if err := tx.Save(&newFee).Error; err != nil {
					tx.Rollback()
					return err
				}
			}
		}

		if err := tx.Model(&entity.PackageAuditLog{}).Create(&audit).Error; err != nil {
			tx.Rollback()
			return err
		}

	}

	return tx.Commit().Error

}

func (m PackageManager) BuildPackageQueryForCustomer(opts PackageQueryOption) *gorm.DB {
	db := m.db

	if len(opts.IDs) > 0 {
		db = db.Where("packages.id IN (?)", opts.IDs)
	}

	if opts.Status > 0 {
		db = db.Where("packages.status = ?", opts.Status)
	}

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	if opts.ID > 0 {
		db = db.Where("packages.id = ?", opts.ID)
	}

	if opts.Code != "" {
		db = db.Where("package_codes.code = ? OR packages.order_number = ? AND package_codes.status = ?", opts.Code, opts.Code, constant.PackageCodeEnable)
	}

	if len(opts.Codes) > 0 {
		db = db.Where("package_codes.code IN (?)", opts.Codes)
	}
	if len(opts.StatusArr) > 0 {
		db = db.Where("packages.status IN (?)", opts.StatusArr)
	}

	if opts.AlertValue > 0 {
		db = db.Where("packages.alert = ?", opts.AlertValue)
	}

	if opts.OrderNumber != "" {
		db = db.Where("packages.order_number = ?", opts.OrderNumber)
	}

	if opts.CustomerShipmentID > 0 {
		db = db.Where("packages.customer_shipment_id = ?", opts.CustomerShipmentID)
	}

	db = db.Joins("LEFT JOIN package_codes on package_codes.id = packages.package_code_id")
	return db
}

func (m PackageManager) GetPackagesForCustomer(opts PackageQueryOption) ([]entity.PackageCustomer, error) {
	db := m.BuildPackageQueryForCustomer(opts)
	db = db.Table("packages").
		Select(`packages.id, packages.label,
				package_codes.code, 
				package_codes.status as p_code_status,
				packages.order_number, 
				packages.recipient,
				packages.company, 
				packages.phone_number, 
				packages.address_1, 
				packages.address_2,
				packages.city, 
				packages.state_code, 
				packages.zipcode, 
				packages.country_code,
				packages.detail, 
				packages.weight, 
				packages.width, 
				packages.length,
				packages.height, 
				packages.user_id, 
				packages.created_at, 
				packages.updated_at,
				packages.shipping_fee,
				packages.order_id,
				packages.is_package_exceed, 
				packages.include_battery,
				bills.code as bill_code,
			packages.status,
			users.phone_number user_phone_number, 
			users.email user_email, users.full_name user_full_name, 
			services.code service_code`,
		).
		Preload("ExtraFees", func(db *gorm.DB) *gorm.DB {
			return db.Table("extra_fees").
				Select("extra_fees.package_id, extra_fees.id, extra_fees.extra_fee_type_id, extra_fee_types.name extra_fee_type, extra_fees.amount, extra_fees.description").
				Joins("LEFT JOIN extra_fee_types on extra_fees.extra_fee_type_id = extra_fee_types.id").
				Where("extra_fees.status = ?", constant.ExtraFeeStatusEnable).
				Where("extra_fee_types.status = ?", constant.ExtraFeeStatusEnable)
		}).Preload("Tracking", func(db *gorm.DB) *gorm.DB {
		return db.Table("trackings").Select("trackings.package_id,trackings.tracking_number, carriers.last_mile_carrier").
			Joins("JOIN packages ON packages.id = trackings.package_id").
			Joins("JOIN carriers ON carriers.id = trackings.carrier_id").
			Where("trackings.status != ?", constant.TrackingStatusCanceled)

	}).
		Joins("LEFT JOIN services ON packages.service_id = services.id").
		Joins("LEFT JOIN users ON packages.user_id = users.id").
		Joins("LEFT JOIN bills ON bills.id = packages.bill_id").
		Where("packages.user_id = ?", opts.UserID).
		Order("id DESC")
	packages := []entity.PackageCustomer{}
	db = db.Find(&packages)
	return packages, db.Error
}

func (m PackageManager) GetPackageDetailForCustomer(opts PackageQueryOption) (entity.PackageCustomer, error) {
	db := m.BuildPackageQueryForCustomer(opts)
	db = db.Table("packages").
		Select(`packages.id,package_codes.code,package_codes.status as p_code_status, packages.order_number, packages.recipient,
					packages.company, packages.phone_number, packages.address_1, packages.address_2,
					packages.city, packages.state_code, packages.zipcode, packages.country_code,
					packages.detail, packages.weight, packages.width, packages.length,
					packages.height, packages.user_id, packages.include_battery,packages.created_at, packages.updated_at,
					packages.shipping_fee, bills.code as bill_code,
				packages.status, packages.label,
				packages.order_id,
				packages.package_name,
				packages.package_quantity,
				packages.total_product_price,
				packages.custom_cn_barcode,
				users.phone_number user_phone_number, 
				users.email user_email, users.full_name user_full_name, 
				services.code service_code`,
		).
		Preload("ExtraFees", func(db *gorm.DB) *gorm.DB {
			return db.Table("extra_fees").
				Select("extra_fees.package_id, extra_fees.id, extra_fees.extra_fee_type_id, extra_fee_types.name extra_fee_type, extra_fees.amount, extra_fees.description").
				Joins("LEFT JOIN extra_fee_types on extra_fees.extra_fee_type_id = extra_fee_types.id").
				Where("extra_fees.status = ?", constant.ExtraFeeStatusEnable).
				Where("extra_fee_types.status = ?", constant.ExtraFeeStatusEnable)
		}).
		Preload("Tracking", func(db *gorm.DB) *gorm.DB {
			return db.Table("trackings").Select("trackings.package_id,trackings.tracking_number,carriers.last_mile_carrier").
				Joins("JOIN packages ON packages.id = trackings.package_id").
				Joins("LEFT JOIN carriers ON carriers.id = trackings.carrier_id").
				Where("trackings.status != ?", constant.TrackingStatusCanceled)

		}).
		Joins("LEFT JOIN services ON packages.service_id = services.id").
		Joins("LEFT JOIN users ON packages.user_id = users.id").
		Joins("LEFT JOIN bills ON bills.id = packages.bill_id").
		Where("packages.user_id = ?", opts.UserID).
		Order("id DESC")

	pkg := entity.PackageCustomer{}
	db = db.Find(&pkg)
	return pkg, db.Error
}
func (m PackageManager) SaveExceedPackage(pkg *entity.Package) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			fmt.Errorf("Panic save exceed pkg", r)
			tx.Rollback()
			return
		}
	}()

	data := make(map[string]interface{})
	data["shipping_fee"] = pkg.ShippingFee
	data["updated_at"] = time.Now()
	if err := tx.Model(&entity.Package{}).Where("id = ?", pkg.ID).UpdateColumns(data).Error; err != nil {
		tx.Rollback()
		return err
	}
	for _, fee := range pkg.ExtraFee {
		if fee.Amount == 0 {
			continue
		}
		extraFee := &entity.ExtraFee{
			PackageID:      utils.Int64(pkg.ID),
			ExtraFeeTypeID: fee.ExtraFeeTypeID,
			Amount:         fee.Amount,
			Status:         constant.ExtraFeeStatusEnable,
			Description:    fee.Description,
		}

		if err := tx.Save(extraFee).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func (m PackageManager) SaveUpdatePackage2(id, userID int64, mapchange map[string]interface{}, logs []entity.PackageAuditLog, priceOutSize float64, extraFees []entity.ExtraFee, oldStatus int) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			return
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Table("packages").Where("id=?", id).UpdateColumns(mapchange).Error; err != nil {
		tx.Rollback()
		return err
	}

	if oldStatus == constant.PackageStatusCreated {
		disStatusList := []int64{constant.ExtraFeeTypeOversize, constant.ExtraFeeTypeService, constant.ExtraFeeTypeDiscount, constant.ExtraFeeTypeBattery}
		if err := tx.Table("extra_fees").
			Where("extra_fee_type_id IN (?) AND package_id=?", disStatusList, id).
			Update("status", constant.ExtraFeeStatusDisable).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	a, _ := json.Marshal(extraFees)
	log.Println("extraFees: ", string(a))
	if len(extraFees) > 0 {
		if err := tx.Create(&extraFees).Error; err != nil {
			tx.Rollback()
			return err
		}

		if oldStatus != constant.PackageStatusCreated {
			for _, fee := range extraFees {
				auditLog := entity.PackageAuditLog{
					Model: dbgorm.Model{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					ExtraFeeID:    fee.ID,
					PackageID:     id,
					UpdatedUserID: userID,
					Type:          constant.MapTypeAuditLogByExtraFeeTypeID[fee.ExtraFeeTypeID],
					Fee:           fee.Amount,
				}

				logs = append(logs, auditLog)
			}
		}
	}

	currentExtraFee := []entity.ExtraFee{}
	if err := tx.Table("extra_fees").
		Where("package_id = ?", id).
		Where("status = ?", constant.ExtraFeeStatusEnable).Find(&currentExtraFee).Error; err != nil {
		tx.Rollback()
		return err
	}

	var oldOutSize float64 = 0

	for _, fee := range currentExtraFee {
		if fee.ExtraFeeTypeID == constant.ExtraFeeTypeOutSize {
			oldOutSize += fee.Amount
		}
	}

	priceOutSize = priceOutSize - oldOutSize

	for i, audit := range logs {
		logs[i].PackageID = id
		logs[i].UpdatedUserID = userID

		extraFee := &entity.ExtraFee{}
		if audit.Type == constant.PackageUpdateTypeVolume {

			extraFee = &entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				PackageID:      utils.Int64(id),
				ExtraFeeTypeID: constant.ExtraFeeTypeOutSize,
				Amount:         priceOutSize,
				Status:         constant.ExtraFeeStatusEnable,
			}

			if priceOutSize != 0 {
				if err := tx.Model(&entity.ExtraFee{}).Create(&extraFee).Error; err != nil {
					tx.Rollback()
					return err
				}

				log := entity.PackageAuditLog{
					Model: dbgorm.Model{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					ExtraFeeID:    extraFee.ID,
					PackageID:     id,
					UpdatedUserID: userID,
					OldValue:      logs[i].OldValue,
					Value:         logs[i].Value,
					Type:          constant.PackageUpdateExtraFeeTypeOutSize,
					Fee:           priceOutSize,
				}
				if extraFee.ID > 0 {
					log.ExtraFeeID = extraFee.ID
				}

				logs = append(logs, log)
			}
		}
	}

	if len(logs) > 0 {
		if err := tx.Model(&entity.PackageAuditLog{}).Create(&logs).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m PackageManager) SaveUpdatePackage(id, userID int64, mapchange map[string]interface{}, changeProduct []*entity.PackageProducts, logs []entity.PackageAuditLog, priceOutSize float64, extraFees []entity.ExtraFee, oldStatus int) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			return
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Table("packages").Where("id=?", id).UpdateColumns(mapchange).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, prod := range changeProduct {
		prodChange := make(map[string]interface{})
		prodChange["quantity"] = prod.Quantity
		prodChange["status"] = prod.Status

		if err := tx.Table("package_products").Where("product_id=? and package_id=?", prod.ProductID, id).First(&entity.PackageProducts{}).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := tx.Create(&prod).Error; err != nil {
					tx.Rollback()
					return err
				}
			} else {
				tx.Rollback()
				return err
			}
		} else {
			if err := tx.Table("package_products").Where("product_id=? and package_id=?", prod.ProductID, id).UpdateColumns(prodChange).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	if oldStatus == constant.PackageStatusCreated {
		disStatusList := []int64{}
		for _, v := range extraFees {
			if v.ExtraFeeTypeID == constant.ExtraFeeTypeOversize {
				disStatusList = append(disStatusList, constant.ExtraFeeTypeOversize)
			}

			if v.ExtraFeeTypeID == constant.ExtraFeeTypeDiscount {
				disStatusList = append(disStatusList, constant.ExtraFeeTypeDiscount)
			}
		}

		if err := tx.Table("extra_fees").
			Where("extra_fee_type_id IN (?) AND package_id=?", disStatusList, id).
			Update("status", constant.ExtraFeeStatusDisable).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if len(extraFees) > 0 {
		if err := tx.Create(&extraFees).Error; err != nil {
			tx.Rollback()
			return err
		}

		if oldStatus != constant.PackageStatusCreated {
			for _, fee := range extraFees {
				auditLog := entity.PackageAuditLog{
					Model: dbgorm.Model{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					ExtraFeeID:    fee.ID,
					PackageID:     id,
					UpdatedUserID: userID,
					Type:          constant.MapTypeAuditLogByExtraFeeTypeID[fee.ExtraFeeTypeID],
					Fee:           fee.Amount,
				}

				logs = append(logs, auditLog)
			}
		}
	}

	currentExtraFee := []entity.ExtraFee{}
	if err := tx.Table("extra_fees").
		Where("package_id = ?", id).
		Where("status = ?", constant.ExtraFeeStatusEnable).Find(&currentExtraFee).Error; err != nil {
		tx.Rollback()
		return err
	}

	var oldOutSize float64 = 0

	for _, fee := range currentExtraFee {
		if fee.ExtraFeeTypeID == constant.ExtraFeeTypeOutSize {
			oldOutSize += fee.Amount
		}
	}

	priceOutSize = priceOutSize - oldOutSize

	for i, audit := range logs {
		logs[i].PackageID = id
		logs[i].UpdatedUserID = userID

		extraFee := &entity.ExtraFee{}
		if audit.Type == constant.PackageUpdateTypeVolume {

			extraFee = &entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				PackageID:      utils.Int64(id),
				ExtraFeeTypeID: constant.ExtraFeeTypeOutSize,
				Amount:         priceOutSize,
				Status:         constant.ExtraFeeStatusEnable,
			}

			if priceOutSize != 0 {
				if err := tx.Model(&entity.ExtraFee{}).Create(&extraFee).Error; err != nil {
					tx.Rollback()
					return err
				}

				log := entity.PackageAuditLog{
					Model: dbgorm.Model{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					ExtraFeeID:    extraFee.ID,
					PackageID:     id,
					UpdatedUserID: userID,
					OldValue:      logs[i].OldValue,
					Value:         logs[i].Value,
					Type:          constant.PackageUpdateExtraFeeTypeOutSize,
					Fee:           priceOutSize,
				}
				if extraFee.ID > 0 {
					log.ExtraFeeID = extraFee.ID
				}

				logs = append(logs, log)
			}
		}
	}

	if len(logs) > 0 {
		if err := tx.Model(&entity.PackageAuditLog{}).Create(&logs).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m PackageManager) SaveUpdatePackageAdmin(id, userID int64, mapchange map[string]interface{}, changeProduct []*entity.PackageProducts, logs []entity.PackageAuditLog, priceOutSize float64, priceByWeight bool, billID *int64) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			fmt.Printf("Panic: %v", r)
			return
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	currentPkg := entity.Package{}
	if err := tx.Table("packages").Preload("Service").Where("id=?", id).Find(&currentPkg).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, prod := range changeProduct {

		prodChange := make(map[string]interface{})
		prodChange["quantity"] = prod.Quantity
		prodChange["status"] = prod.Status

		if err := tx.Table("package_products").Where("product_id=? and package_id=?", prod.ProductID, id).First(&entity.PackageProducts{}).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := tx.Create(&prod).Error; err != nil {
					tx.Rollback()
					return err
				}
			} else {
				tx.Rollback()
				return err
			}
		} else {
			if err := tx.Table("package_products").Where("product_id=? and package_id=?", prod.ProductID, id).UpdateColumns(prodChange).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	currentExtraFee := []entity.ExtraFee{}
	if err := tx.Table("extra_fees").
		Where("package_id = ? ", id).
		Where("status=? ", constant.ExtraFeeStatusEnable).
		Find(&currentExtraFee).Error; err != nil {
		tx.Rollback()
		return err
	}

	var oldOutSize float64 = 0
	var oldFixShippingFee float64 = 0

	for _, fee := range currentExtraFee {
		if fee.ExtraFeeTypeID == constant.ExtraFeeTypeOutSize {
			oldOutSize += fee.Amount
		}
		if fee.ExtraFeeTypeID == constant.ExtraFeeTypeFixWeight || fee.ExtraFeeTypeID == constant.ExtraFeeTypeFixVolume || fee.ExtraFeeTypeID == constant.ExtraFeeService {
			oldFixShippingFee += fee.Amount
		}
	}

	oldFixShippingFee += currentPkg.ShippingFee

	newShippingFee := cast.ToFloat64(mapchange["shipping_fee"])
	mapchange["shipping_fee"] = currentPkg.ShippingFee

	if err := tx.Table("packages").Where("id=?", id).UpdateColumns(mapchange).Error; err != nil {
		tx.Rollback()
		return err
	}

	extraOutSize := priceOutSize - oldOutSize
	if extraOutSize < 0 {
		extraOutSize = 0
	}

	extraFeeWeight := newShippingFee - oldFixShippingFee
	if extraFeeWeight < 0 {
		extraFeeWeight = 0
	}

	var balanceType string
	// if currentPkg.Service.Code == constant.ServiceCNCode {
	// balanceType = "balance_china" // remove china wallet
	// balanceType = "balance"
	// } else {
	balanceType = "balance"
	// }

	sqlString := fmt.Sprintf("UPDATE users SET %s = %s - ? - ?, updated_at = ? WHERE id = ?", balanceType, balanceType)
	if err := tx.Exec(sqlString, extraOutSize, extraFeeWeight, time.Now(), currentPkg.UserID).Error; err != nil {
		tx.Rollback()
		return err
	}

	amount := extraOutSize + extraFeeWeight

	extraFeePlusByService := false

	for i, audit := range logs {
		logs[i].PackageID = id
		logs[i].UpdatedUserID = userID

		if audit.Type == constant.PackageUpdateTypeService && extraFeeWeight > 0 {
			extFee := entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				PackageID:      utils.Int64(id),
				ExtraFeeTypeID: constant.ExtraFeeService,
				Amount:         extraFeeWeight,
				Status:         constant.ExtraFeeStatusEnable,
				BillID:         billID,
			}

			if err := tx.Create(&extFee).Error; err != nil {
				tx.Rollback()
				return err
			}

			logs[i].Fee = extraFeeWeight
			logs[i].ExtraFeeID = extFee.ID

			extraFeePlusByService = true
		}

		if audit.Type == constant.PackageUpdateTypeWeight && !extraFeePlusByService && extraFeeWeight > 0 && priceByWeight {
			extFee := entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				PackageID:      utils.Int64(id),
				ExtraFeeTypeID: constant.ExtraFeeTypeFixWeight,
				Amount:         extraFeeWeight,
				Status:         constant.ExtraFeeStatusEnable,
				BillID:         billID,
			}

			if err := tx.Create(&extFee).Error; err != nil {
				tx.Rollback()
				return err
			}

			logs[i].Fee = extraFeeWeight
			logs[i].ExtraFeeID = extFee.ID
		}

		if audit.Type == constant.PackageUpdateTypeVolume && !extraFeePlusByService && extraFeeWeight > 0 && !priceByWeight {

			extFee := entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				PackageID:      utils.Int64(id),
				ExtraFeeTypeID: constant.ExtraFeeTypeFixVolume,
				Amount:         extraFeeWeight,
				Status:         constant.ExtraFeeStatusEnable,
				BillID:         billID,
			}

			if err := tx.Create(&extFee).Error; err != nil {
				tx.Rollback()
				return err
			}

			logs[i].Fee = extraFeeWeight
			logs[i].ExtraFeeID = extFee.ID

		}

		if audit.Type == constant.PackageUpdateTypeVolume && extraOutSize > 0 {

			extFee := entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				PackageID:      utils.Int64(id),
				ExtraFeeTypeID: constant.ExtraFeeTypeOutSize,
				Amount:         extraOutSize,
				Status:         constant.ExtraFeeStatusEnable,
				BillID:         billID,
			}

			if err := tx.Create(&extFee).Error; err != nil {
				tx.Rollback()
				return err
			}

			log := &entity.PackageAuditLog{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				ExtraFeeID:    extFee.ID,
				PackageID:     id,
				UpdatedUserID: userID,
				OldValue:      logs[i].OldValue,
				Value:         logs[i].Value,
				Type:          constant.PackageUpdateExtraFeeTypeOutSize,
				Fee:           extraOutSize,
			}
			logs = append(logs, *log)
		}

	}

	if len(logs) > 0 {
		if err := tx.Model(&entity.PackageAuditLog{}).Create(&logs).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if amount > 0 {
		transaction := &entity.Transaction{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			UserID:  currentPkg.UserID,
			AdminID: userID,
			Type:    constant.TransactionLogTypePay,
			Status:  constant.TransactionStatusSuccess,
			Amount:  amount,
			BillID:  billID,
		}

		err := tx.Create(&transaction).Error
		if err != nil {
			fmt.Errorf("Create transaction error %v", err)
			tx.Rollback()
			return err
		}

		transactionLog := &entity.TransactionLog{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			UserID:        currentPkg.UserID,
			AdminID:       userID,
			TransactionID: transaction.ID,
			Amount:        transaction.Amount,
			Type:          transaction.Type,
			Status:        transaction.Status,
			BillID:        transaction.BillID,
		}

		if err = tx.Model(entity.TransactionLog{}).Create(transactionLog).Error; err != nil {
			tx.Rollback()
			return err
		}

		if currentPkg.IsPackageExceed {
			extFee := entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				PackageID:      utils.Int64(id),
				ExtraFeeTypeID: constant.ExtraFeeTypeEditAddress,
				Description:    "Phí chênh lệch sửa đơn quá cỡ",
				Amount:         amount,
				Status:         constant.ExtraFeeStatusEnable,
				BillID:         billID,
			}
			if err := tx.Create(&extFee).Error; err != nil {
				tx.Rollback()
				return err
			}

			log := &entity.PackageAuditLog{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				ExtraFeeID:    extFee.ID,
				PackageID:     id,
				UpdatedUserID: userID,
				Type:          constant.PackageUpdateAddressExceed,
				Fee:           amount,
				Description:   "Phí chênh lệch sửa đơn quá cỡ",
			}
			if err := tx.Model(&entity.PackageAuditLog{}).Create(&log).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		info := &entity.UserInfo{}
		if err := tx.First(info).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				tx.Rollback()
				return err
			}

			now := time.Now()
			info = &entity.UserInfo{UserID: currentPkg.UserID, UpdatedAt: &now, CancelMaxAmount: constant.DefaultCancelMaxAMount}
			if err := tx.Create(info).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		sqlString := `UPDATE user_infos SET debt_time = ? WHERE user_id = ? AND debt_time IS NULL AND (SELECT balance FROM users WHERE id = ? limit 1) < 0`
		if err := tx.Exec(sqlString, time.Now(), currentPkg.UserID, currentPkg.UserID).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if billID != nil && utils.Int64Value(billID) > 0 {
		var extraFeeOfBill float64
		var shippingFeeOfBill float64
		var packagesInBill []entity.Package
		var extraFeeInBill []entity.ExtraFee

		if err := tx.Where("bill_id=?", billID).Find(&packagesInBill).Error; err != nil {
			tx.Rollback()
			return err
		}

		for _, packageInBill := range packagesInBill {
			shippingFeeOfBill += packageInBill.ShippingFee
		}

		if err := tx.Table("extra_fees").
			Where("bill_id = ? ", billID).
			Where("status=? ", constant.ExtraFeeStatusEnable).
			Find(&extraFeeInBill).Error; err != nil {
			tx.Rollback()
			return err
		}

		for _, fee := range extraFeeInBill {
			extraFeeOfBill += fee.Amount
		}

		if err := tx.Exec("UPDATE bills SET extra_fee = ?, shipping_fee = ? WHERE id=?",
			extraFeeOfBill, shippingFeeOfBill, billID).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m PackageManager) UpdatePackageFiled(id int64, filed string, value interface{}) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			return
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := m.db.Table("packages").Where("id=?", id).UpdateColumn(filed, value).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m PackageManager) UpdateDeliveriedPackage(codes []string) error {
	sql := `
    update packages 
		left join package_codes on package_codes.package_id_generate = packages.id 
		set packages.status = 60, packages.updated_at = NOW(), packages.delivered_at = NOW() 
		where package_codes.code in (?)`

	// Execute the raw SQL query
	result := m.db.Exec(sql, codes)
	return result.Error

}
func (m PackageManager) GetPackageByPackageCodeID(id int64) (*entity.Package, error) {
	pkg := &entity.Package{}
	db := m.db.
		Where("package_code_id = ?", id).
		Preload("User").
		Preload("Service").
		Preload("Tracking").
		Preload("Tracking.Warehouse").
		Preload("ExtraFee").
		First(pkg)
	return pkg, db.Error
}

func (m *PackageManager) UpdateStatus(id int64, status int, log *entity.PackageDeliverLog, alog *entity.PackageAuditLog) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	change := map[string]interface{}{"updated_at": time.Now(), "status": status}

	if status == constant.PackageStatusPicked {
		change["checkin_warehouse_at"] = time.Now()
	}

	if err := tx.Model(&entity.Package{}).Where("id=?", id).UpdateColumns(change).Error; err != nil {
		tx.Rollback()
		return err
	}

	if status == constant.PackageStatusReturned {
		if err := tx.Model(&entity.ContainerItem{}).Where("package_id = ?", id).UpdateColumns(map[string]interface{}{
			"updated_at": time.Now(),
			"status":     constant.ContainerItemInactive,
		}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if log != nil {
		if err := tx.Create(log).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if alog != nil {
		alog.UpdatedAt = time.Now()
		alog.CreatedAt = alog.UpdatedAt
		if err := tx.Create(alog).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m *PackageManager) WarehousChecked(pkg *entity.Package, userID, customerID int64, change map[string]interface{}, tk *entity.Tracking, alogs []entity.PackageAuditLog, shippingFeePlus, outsizePlus, batteryFeePlus float64, billID int64, priceByWeight bool, extraPeakFeePlus float64, fees []entity.ExtraFee) error {
	id := pkg.ID
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	change["updated_at"] = time.Now()
	change["status"] = constant.PackageStatusWareHouseLabeled
	if err := tx.Model(&entity.Package{}).Where("id=?", id).UpdateColumns(change).Error; err != nil {
		tx.Rollback()
		return err
	}

	if tk != nil {
		tk.CreatedAt = time.Now()
		tk.UpdatedAt = tk.CreatedAt
		if err := tx.Create(tk).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if shippingFeePlus > 0 || outsizePlus > 0 || extraPeakFeePlus > 0 || len(fees) > 0 || batteryFeePlus > 0 {
		var amount float64 = 0
		var extraFees []entity.ExtraFee

		for _, fee := range fees {
			amount += fee.Amount
			extraFees = append(extraFees, entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				BillID:         &billID,
				PackageID:      utils.Int64(id),
				ExtraFeeTypeID: fee.ExtraFeeTypeID,
				Amount:         fee.Amount,
				Status:         constant.ExtraFeeStatusEnable,
			})

			if fee.ExtraFeeTypeID != constant.ExtraFeeTypeFixVolume && fee.ExtraFeeTypeID != constant.ExtraFeeTypeFixWeight && fee.ExtraFeeTypeID != constant.ExtraFeeTypeBattery {
				auditLog := entity.PackageAuditLog{
					Model: dbgorm.Model{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					PackageID:     id,
					UpdatedUserID: userID,
					Type:          constant.MapTypeAuditLogByExtraFeeTypeID[fee.ExtraFeeTypeID],
					Fee:           fee.Amount,
				}

				alogs = append(alogs, auditLog)
			}
		}

		if priceByWeight && shippingFeePlus > 0 {
			amount += shippingFeePlus
			extraFees = append(extraFees, entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				BillID:         &billID,
				PackageID:      utils.Int64(id),
				ExtraFeeTypeID: constant.ExtraFeeTypeFixWeight,
				Amount:         shippingFeePlus,
				Status:         constant.ExtraFeeStatusEnable,
			})
		}

		if !priceByWeight && shippingFeePlus > 0 {
			amount += shippingFeePlus
			extraFees = append(extraFees, entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				BillID:         &billID,
				PackageID:      utils.Int64(id),
				ExtraFeeTypeID: constant.ExtraFeeTypeFixVolume,
				Amount:         shippingFeePlus,
				Status:         constant.ExtraFeeStatusEnable,
			})
		}

		if outsizePlus > 0 {
			amount += outsizePlus
			extraFees = append(extraFees, entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				BillID:         &billID,
				PackageID:      utils.Int64(id),
				ExtraFeeTypeID: constant.ExtraFeeTypeOutSize,
				Amount:         outsizePlus,
				Status:         constant.ExtraFeeStatusEnable,
			})
		}

		if extraPeakFeePlus > 0 {
			amount += extraPeakFeePlus
			extraFees = append(extraFees, entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				BillID:         &billID,
				PackageID:      utils.Int64(id),
				ExtraFeeTypeID: constant.ExtraFeeTypePeak,
				Amount:         extraPeakFeePlus,
				Status:         constant.ExtraFeeStatusEnable,
			})
		}

		if batteryFeePlus > 0 {
			amount += batteryFeePlus
			extraFees = append(extraFees, entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				BillID:         &billID,
				PackageID:      utils.Int64(id),
				ExtraFeeTypeID: constant.ExtraFeeTypeBattery,
				Amount:         batteryFeePlus,
				Status:         constant.ExtraFeeStatusEnable,
			})
		}

		if len(extraFees) > 0 {
			if err := tx.Create(&extraFees).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		if err := tx.Exec("UPDATE bills SET extra_fee=extra_fee+? WHERE id=?", amount, billID).Error; err != nil {
			tx.Rollback()
			return err
		}

		var balanceType string
		if pkg.Service.Code == constant.ServiceCNCode {
			// balanceType = "balance_china" // remove china wallet
			balanceType = "balance"
		} else {
			balanceType = "balance"
		}

		sqlString := fmt.Sprintf("UPDATE users SET %s = %s - ?, updated_at = ? WHERE id = ?", balanceType, balanceType)
		if err := tx.Exec(sqlString, amount, time.Now(), customerID).Error; err != nil {
			tx.Rollback()
			return err
		}

		if amount > 0 {
			info := &entity.UserInfo{}
			if err := tx.First(info).Error; err != nil {
				if err != gorm.ErrRecordNotFound {
					tx.Rollback()
					return err
				}

				now := time.Now()
				info = &entity.UserInfo{UserID: customerID, UpdatedAt: &now, CancelMaxAmount: constant.DefaultCancelMaxAMount}
				if err := tx.Create(info).Error; err != nil {
					tx.Rollback()
					return err
				}
			}

			strsql := "UPDATE user_infos SET debt_time = ? WHERE user_id = ? AND debt_time IS NULL AND (SELECT balance FROM users WHERE id = ? limit 1) < 0"
			if err := tx.Exec(strsql, time.Now(), customerID, customerID).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		transaction := entity.Transaction{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			UserID:  customerID,
			AdminID: userID,
			Type:    constant.TransactionLogTypePay,
			Status:  constant.TransactionStatusSuccess,
			Amount:  -amount,
			BillID:  &billID,
		}

		if err := tx.Create(&transaction).Error; err != nil {
			tx.Rollback()
			return err
		}

		transactionLog := entity.TransactionLog{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			UserID:        customerID,
			AdminID:       userID,
			TransactionID: transaction.ID,
			Amount:        transaction.Amount,
			Type:          transaction.Type,
			Status:        transaction.Status,
			BillID:        transaction.BillID,
		}

		if err := tx.Create(&transactionLog).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if extraPeakFeePlus < 0 {
		extraPeakFeePlus = 0
	}

	if len(alogs) > 0 {
		for i, audit := range alogs {
			if shippingFeePlus < 0 {
				shippingFeePlus = 0
			}

			if audit.Type == constant.PackageUpdateTypeWeight {
				alogs[i].Fee = extraPeakFeePlus
				if priceByWeight {
					alogs[i].Fee += shippingFeePlus
				}
			}

			if audit.Type == constant.PackageUpdateTypeVolume && !priceByWeight {
				alogs[i].Fee = shippingFeePlus
			}

			if audit.Type == constant.PackageUpdateTypeVolume && outsizePlus > 0 {

				log := &entity.PackageAuditLog{
					Model: dbgorm.Model{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					PackageID:     id,
					UpdatedUserID: userID,
					OldValue:      alogs[i].OldValue,
					Value:         alogs[i].Value,
					Type:          constant.PackageUpdateExtraFeeTypeOutSize,
					Fee:           outsizePlus,
				}
				alogs = append(alogs, *log)
			}

		}

		if err := tx.Create(&alogs).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m PackageManager) GetPackageCodeByCode(opts PackageCodeQueryOption) (*entity.PackageCode, error) {
	db := m.BuildPackageCodeQuery(opts)

	packages := &entity.PackageCode{}
	db = db.First(packages)
	return packages, db.Error
}

func (m PackageManager) SaveDeliverLogPackage(logs []entity.PackageDeliverLog, pkg *entity.Package, isReturnPackage bool, billId int64) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	err := tx.Create(&logs).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	pkg.UpdatedAt = time.Now()
	err = tx.Save(&pkg).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	if isReturnPackage && pkg.CountryCode == "AU" {
		fee := entity.ExtraFee{
			PackageID:      utils.Int64(pkg.ID),
			BillID:         &billId,
			ExtraFeeTypeID: constant.ExtraFeeTypeReturn,
			Description:    "Phí return đơn hàng",
			Amount:         cast.ToFloat64(viper.GetString("fee.returned")),
			Status:         constant.ExtraFeeStatusEnable,
		}
		if err := tx.Create(&fee).Error; err != nil {
			tx.Rollback()
			return err
		}

		log := &entity.PackageAuditLog{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			ExtraFeeID:  fee.ID,
			PackageID:   utils.Int64Value(fee.PackageID),
			Type:        constant.PackageUpdateReturnPackage,
			Fee:         fee.Amount,
			Value:       fmt.Sprintf("Phí return cho đơn %s", pkg.PackageCode.Code),
			Description: fee.Description,
		}
		if err := tx.Model(&entity.PackageAuditLog{}).Create(&log).Error; err != nil {
			tx.Rollback()
			return err
		}

		transaction := &entity.Transaction{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			UserID: pkg.UserID,
			Type:   constant.TransactionLogTypePay,
			Status: constant.TransactionStatusSuccess,
			Amount: fee.Amount,
			BillID: &billId,
		}

		err := tx.Create(&transaction).Error
		if err != nil {
			fmt.Errorf("Create transaction error %v", err)
			tx.Rollback()
			return err
		}

		transactionLog := &entity.TransactionLog{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			UserID:        pkg.UserID,
			TransactionID: transaction.ID,
			Amount:        transaction.Amount,
			Type:          transaction.Type,
			Status:        transaction.Status,
			BillID:        transaction.BillID,
		}

		if err = tx.Model(entity.TransactionLog{}).Create(transactionLog).Error; err != nil {
			tx.Rollback()
			return err
		}

		var balanceType string
		if pkg.Service.Code == constant.ServiceCNCode {
			// balanceType = "balance_china" // remove china wallet
			balanceType = "balance"
		} else {
			balanceType = "balance"
		}

		sqlString := fmt.Sprintf("UPDATE users SET %s = %s - ?, updated_at = ? WHERE id = ?", balanceType, balanceType)
		if err := tx.Exec(sqlString, fee.Amount, time.Now(), pkg.UserID).Error; err != nil {
			tx.Rollback()
			return err
		}

		info := &entity.UserInfo{}
		if err := tx.First(info).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				tx.Rollback()
				return err
			}

			now := time.Now()
			info = &entity.UserInfo{UserID: pkg.UserID, UpdatedAt: &now, CancelMaxAmount: constant.DefaultCancelMaxAMount}
			if err := tx.Create(info).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		sqlString = `UPDATE user_infos SET debt_time = ? WHERE user_id = ? AND debt_time IS NULL AND (SELECT balance FROM users WHERE id = ? limit 1) < 0`
		if err := tx.Exec(sqlString, time.Now(), pkg.UserID, pkg.UserID).Error; err != nil {
			tx.Rollback()
			return err
		}

		sqlString = `UPDATE bills SET extra_fee = extra_fee + ? WHERE id = ?`
		if err := tx.Exec(sqlString, fee.Amount, billId).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func (m PackageManager) UpdatePackageDeliverLog(log *entity.PackageDeliverLog, t time.Time, isAlertReturn bool) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	log.UpdatedAt = t
	log.CreatedAt = t
	if err := tx.Create(log).Error; err != nil {
		tx.Rollback()
		return err
	}

	change := map[string]interface{}{"updated_at": time.Now(), "tracking_time": t}
	if log.Status == constant.DeliverLogTebexpressDelivered {
		change["status"] = constant.PackageStatusDelivered
		change["delivered_at"] = log.CreatedAt
	}

	if isAlertReturn {
		change["alert"] = constant.PackageAlertTypeHubReturn
		change["returned_at"] = time.Now()
	}

	if err := tx.Model(&entity.Package{}).Where("id=?", log.PackageID).UpdateColumns(change).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m PackageManager) GetDeliverPackageLogs(code []string) ([]dto.PackageDeliverLogDTO, error) {
	deliverLogs := []dto.PackageDeliverLogDTO{}
	sql := `
			SELECT 
			package_deliver_logs.package_id,
			package_deliver_logs.location,
			package_deliver_logs.status,
			package_deliver_logs.type,
			"" AS code,
			package_deliver_logs.created_at AS ship_time,
			package_deliver_logs.description,
			users.full_name AS updated_user_name,
			users.role AS updated_user_role
		FROM
			package_deliver_logs
			LEFT JOIN users ON users.id = package_deliver_logs.user_id
			LEFT JOIN packages ON packages.id = package_deliver_logs.package_id
			LEFT JOIN package_codes ON package_codes.id = packages.package_code_id
			LEFT JOIN trackings ON trackings.package_id = packages.id
		WHERE
			package_codes.code IN (?)
			AND package_deliver_logs.description not in (?, ?)
	
		UNION

		SELECT
			packages.id,
			location,
			container_deliver_logs.status,
			30 AS type,
			container_deliver_logs.code,
			ship_time,
			container_deliver_logs.description,
			"" AS updated_user_name,
			"" AS updated_user_role
		FROM container_deliver_logs
			LEFT JOIN container_items on container_items.container_id = container_deliver_logs.container_id
			LEFT JOIN packages on packages.id = container_items.package_id
			LEFT JOIN package_codes ON package_codes.id = packages.package_code_id
			LEFT JOIN trackings ON trackings.package_id = packages.id
		WHERE container_deliver_logs.code NOT IN (?)
			AND package_codes.code in (?)
			AND container_items.status = ?
		ORDER by ship_time DESC, type DESC
	`

	db := m.db.Raw(sql, code, "Shipping Label Created, USPS Awaiting Item", "Received data", constant.UPSIgnoreCodeLogs, code, constant.ContainerItemActive).Scan(&deliverLogs)
	return deliverLogs, db.Error
}

func (m PackageManager) GetDeliverPackageLogsByIDs(ids []int64) ([]dto.PackageDeliverLogDTO, error) {
	deliverLogs := []dto.PackageDeliverLogDTO{}
	sql := `
			SELECT 
			package_deliver_logs.package_id,
			package_deliver_logs.location,
			package_deliver_logs.status,
			package_deliver_logs.type,
			"" AS code,
			package_deliver_logs.created_at AS ship_time,
			package_deliver_logs.description,
			users.full_name AS updated_user_name,
			users.role AS updated_user_role
		FROM
			package_deliver_logs
			LEFT JOIN users ON users.id = package_deliver_logs.user_id
			LEFT JOIN packages ON packages.id = package_deliver_logs.package_id
			LEFT JOIN trackings ON trackings.package_id = packages.id
		WHERE
			packages.id IN (?)
			AND package_deliver_logs.description NOT IN (?, ?)
	
		UNION

		SELECT
			packages.id,
			location,
			container_deliver_logs.status,
			30 AS type,
			container_deliver_logs.code,
			ship_time,
			container_deliver_logs.description,
			"" AS updated_user_name,
			"" AS updated_user_role
		FROM container_deliver_logs
			LEFT JOIN container_items on container_items.container_id = container_deliver_logs.container_id
			LEFT JOIN packages on packages.id = container_items.package_id
			LEFT JOIN trackings ON trackings.package_id = packages.id
		WHERE container_deliver_logs.code NOT IN (?)
			AND packages.id in (?)
			AND container_items.status = ?
		ORDER by ship_time DESC, type DESC
	`

	db := m.db.Raw(sql, ids, "Shipping Label Created, USPS Awaiting Item", "Received data", constant.UPSIgnoreCodeLogs, ids, constant.ContainerItemActive).Scan(&deliverLogs)
	return deliverLogs, db.Error
}

func (m PackageManager) ValidateAddressOrder(packageID int64, userID int64, resultValidate int, description string, saveLog bool) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	mapString := map[string]interface{}{
		"UpdatedAt":       time.Now(),
		"ValidateAddress": resultValidate,
	}

	if err := tx.Model(&entity.Package{}).
		Where("id = ?", packageID).
		UpdateColumns(mapString).Error; err != nil {
		tx.Rollback()
		return err
	}

	if saveLog {
		log := &entity.PackageAuditLog{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			PackageID:     packageID,
			UpdatedUserID: userID,
			Type:          constant.PackageCheckAddressType,
			Description:   description,
		}

		if err := tx.Create(&log).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m PackageManager) IgnoreValidateAddressOrder(packageID int64, userID int64, description string) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	mapString := map[string]interface{}{
		"UpdatedAt":       time.Now(),
		"ValidateAddress": constant.PackageValidAddress,
	}

	if err := tx.Model(&entity.Package{}).
		Where("id = ?", packageID).
		UpdateColumns(mapString).Error; err != nil {
		tx.Rollback()
		return err
	}

	log := &entity.PackageAuditLog{
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		PackageID:     packageID,
		UpdatedUserID: userID,
		Type:          constant.PackageIgnoreAddressType,
		Description:   description,
	}

	if err := tx.Create(&log).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m *PackageManager) BuildQueryExportPackage(sqlString string, opts *ExportForm) string {
	sqlString += ` (
		SELECT 
		bills.code as bill_code,
		DATE_FORMAT(convert_tz(bills.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d %H:%i:%s') as bill_create,
		packages.status,
		packages.order_number,
		DATE_FORMAT(convert_tz(packages.updated_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d %H:%i:%s') as pkg_updated_at,
		package_codes.code AS package_code,
		users.full_name,
		users.class,
		DATE_FORMAT(convert_tz(packages.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d %H:%i:%s') AS created_at,
		trackings.tracking_number,
		services.name AS service_name,
		(CASE
			WHEN packages.address_1 IS NULL THEN packages.address_2
			ELSE packages.address_1
		END) AS address,
		packages.city,
		packages.state_code,
		packages.zipcode,
		packages.country_code,
		packages.detail,
		packages.shipping_fee,
		DATE_FORMAT(convert_tz(packages.delivered_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d %H:%i:%s') as delivered_at,
		0 AS extra_fee,
		packages.shipping_fee + COALESCE(e.extra_fee_total, 0) AS total_fee,
		'' AS extra_fee_type,
		'' AS extra_fee_created_at,
		'' AS description,
		packages.weight,
		packages.length,
		packages.width,
		packages.height,
		trackings.weight AS weight_real,
		trackings.length AS length_real,
		trackings.width AS width_real,
		trackings.height AS height_real,
		carriers.name AS carrier_name,
		trackings.carrier_service,
		warehouses.name AS warehouse,
		trackings.shipment_cost,
		trackings.handling_fee,
		users.username,
		(SELECT rate FROM exchange_rate_logs WHERE  
			exchange_rate_logs.created_at <= packages.created_at 
			ORDER BY exchange_rate_logs.id DESC
		LIMIT 1) as rate
	FROM
		packages
			LEFT JOIN
		users ON users.id = packages.user_id
			LEFT JOIN
		bills ON bills.id = packages.bill_id
			LEFT JOIN
		package_codes ON package_codes.id = packages.package_code_id
			LEFT JOIN
		trackings ON trackings.package_id = packages.id AND trackings.status = ?
			LEFT JOIN
		services ON services.id = packages.service_id
			LEFT JOIN
		extra_fees ON extra_fees.package_id = packages.id AND extra_fees.status = ? AND extra_fees.bill_id = packages.bill_id
			LEFT JOIN
		extra_fee_types ON extra_fee_types.id = extra_fees.extra_fee_type_id
			LEFT JOIN
		(SELECT 
			package_id,
				SUM(COALESCE(extra_fees.amount, 0)) AS extra_fee_total
		FROM
			extra_fees
		WHERE
			extra_fees.status = ?
		GROUP BY package_id) e ON e.package_id = packages.id
			LEFT JOIN
		carriers ON carriers.id = trackings.carrier_id
			LEFT JOIN
		warehouses ON warehouses.id = trackings.hub_id
	WHERE
		packages.bill_id > 0`
	if opts.UserID > 0 {
		sqlString += fmt.Sprintf(` AND users.ID = %s`, string_util.Int64ToString(opts.UserID))
	}

	if len(opts.StatusInt) > 0 {
		sqlString += fmt.Sprintf(` AND packages.status IN (%s)`, string_util.JoinInt64Array(opts.StatusInt, ","))
	}

	if len(opts.StartDate) > 0 {
		sqlString += fmt.Sprintf(` AND DATE_FORMAT(convert_tz(bills.created_at, @@session.time_zone,'+07:00') ,'%%Y-%%m-%%d') >= DATE('%s')`, opts.StartDate)
	}

	if len(opts.EndDate) > 0 {
		sqlString += fmt.Sprintf(` AND DATE_FORMAT(convert_tz(bills.created_at, @@session.time_zone,'+07:00') ,'%%Y-%%m-%%d') <= DATE('%s')`, opts.EndDate)
	}

	if len(opts.IgnoreUsers) > 0 {
		sqlString += fmt.Sprintf(` AND packages.user_id NOT IN (%s)`, string_util.JoinInt64Array(opts.IgnoreUsers, ","))
	}

	sqlString += `
		UNION SELECT 
		bills.code as bill_code,
		DATE_FORMAT(convert_tz(bills.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d %H:%i:%s') as bill_create,
		packages.status,
		packages.order_number,
		DATE_FORMAT(convert_tz(packages.updated_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d %H:%i:%s') as pkg_updated_at,
		package_codes.code AS package_code,
		users.full_name,
		users.class,
		DATE_FORMAT(CONVERT_TZ(packages.created_at,
						@@SESSION .time_zone,
						'+07:00'),
				'%Y-%m-%d %H:%i:%s') AS created_at,
		trackings.tracking_number,
		services.name AS service_name,
		(CASE
			WHEN packages.address_1 IS NULL THEN packages.address_2
			ELSE packages.address_1
		END) AS address,
		packages.city,
		packages.state_code,
		packages.zipcode,
		packages.country_code,
		packages.detail,
		0 AS shipping_fee,
		DATE_FORMAT(convert_tz(packages.delivered_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d %H:%i:%s') as delivered_at,
		extra_fees.amount AS extra_fee,
		packages.shipping_fee + COALESCE(e.extra_fee_total, 0) AS total_fee,
		extra_fee_types.name AS extra_fee_type,
		DATE_FORMAT(CONVERT_TZ(extra_fees.created_at,
						@@SESSION .time_zone,
						'+07:00'),
				'%Y-%m-%d %H:%i:%s') AS extra_fee_created_at,
		extra_fees.description,
		packages.weight,
		packages.length,
		packages.width,
		packages.height,
		trackings.weight AS weight_real,
		trackings.length AS length_real,
		trackings.width AS width_real,
		trackings.height AS height_real,
		carriers.name AS carrier_name,
		trackings.carrier_service,
		warehouses.name AS warehouse,
		trackings.shipment_cost,
		trackings.handling_fee,
		users.username,
		(SELECT rate FROM exchange_rate_logs WHERE  
			exchange_rate_logs.created_at <= packages.created_at 
			ORDER BY exchange_rate_logs.id DESC
		LIMIT 1) as rate
	FROM
		extra_fees
			LEFT JOIN
		packages ON extra_fees.package_id = packages.id
			LEFT JOIN
		bills ON extra_fees.bill_id = bills.id
			LEFT JOIN
		users ON users.id = packages.user_id
			LEFT JOIN
		package_codes ON package_codes.id = packages.package_code_id
			LEFT JOIN
		trackings ON trackings.package_id = packages.id AND trackings.status = ?
			LEFT JOIN
		services ON services.id = packages.service_id
			LEFT JOIN
		extra_fee_types ON extra_fee_types.id = extra_fees.extra_fee_type_id
			LEFT JOIN
		(SELECT 
			package_id,
				SUM(COALESCE(extra_fees.amount, 0)) AS extra_fee_total
		FROM
			extra_fees
		WHERE
			extra_fees.status = ?
		GROUP BY package_id) e ON e.package_id = packages.id
			LEFT JOIN
		carriers ON carriers.id = trackings.carrier_id
			LEFT JOIN
		warehouses ON warehouses.id = trackings.hub_id
	WHERE
		 extra_fees.status = ?`

	if opts.UserID > 0 {
		sqlString += fmt.Sprintf(` AND bills.user_id = %s`, string_util.Int64ToString(opts.UserID))
	}

	if len(opts.StatusInt) > 0 {
		sqlString += fmt.Sprintf(` AND packages.status IN (%s)`, string_util.JoinInt64Array(opts.StatusInt, ","))
	}

	if len(opts.StartDate) > 0 {
		sqlString += fmt.Sprintf(` AND DATE_FORMAT(convert_tz(bills.created_at, @@session.time_zone,'+07:00') ,'%%Y-%%m-%%d') >= DATE('%s')`, opts.StartDate)
	}

	if len(opts.EndDate) > 0 {
		sqlString += fmt.Sprintf(` AND DATE_FORMAT(convert_tz(bills.created_at, @@session.time_zone,'+07:00') ,'%%Y-%%m-%%d') <= DATE('%s')`, opts.EndDate)
	}

	if len(opts.IgnoreUsers) > 0 {
		sqlString += fmt.Sprintf(` AND (packages.user_id NOT IN (%s) OR packages.user_id IS NULL)`, string_util.JoinInt64Array(opts.IgnoreUsers, ","))
	}

	sqlString += `) A ORDER BY package_code , created_at`
	return sqlString
}

func (m *PackageManager) CountPackageExport(result interface{}, opts *ExportForm) error {
	db := m.db
	sqlString := `SELECT COUNT(*) as count FROM`
	sqlString = m.BuildQueryExportPackage(sqlString, opts)
	db = db.Raw(sqlString, constant.TrackingStatusSuccess, constant.ExtraFeeStatusEnable, constant.ExtraFeeStatusEnable, constant.TransactionStatusSuccess, constant.ExtraFeeStatusEnable, constant.ExtraFeeStatusEnable).Scan(result)
	return db.Error
}

func (m *PackageManager) GetPackageExport(result interface{}, opts *ExportForm) error {
	db := m.db
	sqlString := `SELECT * FROM`
	sqlString = m.BuildQueryExportPackage(sqlString, opts)
	db = db.Raw(sqlString, constant.TrackingStatusSuccess, constant.ExtraFeeStatusEnable, constant.ExtraFeeStatusEnable, constant.TransactionStatusSuccess, constant.ExtraFeeStatusEnable, constant.ExtraFeeStatusEnable).Scan(result)
	return db.Error
}

func (m *PackageManager) GetPackageWrongWeightExport(result interface{}, opts *ExportPackageWrongWeightForm) error {
	db := m.db
	sqlString := `select 
		u.full_name as full_name, 
		b.code as package_code, 
		c.tracking_number as tracking_number, 
		a.order_number as order_number, 
		a.recipient as recipient , 
		a.company as company, 
		a.phone_number as phone, 
		replace(a.address_1, '\n', ' ') as address, 
		a.address_2 as address_extra, 
		a.city as city, 
		a.state_code as state_code, 
		a.zipcode as zipcode, 
		a.country_code as country_code , 
		a.weight as weight, 
		a.length as length, 
		a.width as width, 
		a.height as height, 
		a.actual_weight as weight_real, 
		a.actual_length as length_real,
		a.actual_width as width_real,
		a.actual_height as height_real, 
		e.code  as container_code, 
		DATE_FORMAT(CONVERT_TZ(a.created_at,@@SESSION .time_zone,'+07:00'),'%Y-%m-%d %H:%i:%s') as  created_at, 
		DATE_FORMAT(CONVERT_TZ(a.checkin_warehouse_at,@@SESSION .time_zone,'+07:00'),'%Y-%m-%d %H:%i:%s') as checkin_warehouse from packages a
		left join package_codes b on a.package_code_id = b.id
		left join trackings c on c.package_id = a.id and c.status = ?
		left join container_items d on d.package_id = a.id and d.status = ?
		left join containers e on e.id = d.container_id
		left join users u on u.id = a.user_id
		where u.full_name = ?
	`

	if len(opts.StartDate) > 0 {
		sqlString += fmt.Sprintf(` AND DATE_FORMAT(convert_tz(a.checkin_warehouse_at, @@session.time_zone,'+07:00') ,'%%Y-%%m-%%d') >= DATE('%s')`, opts.StartDate)
	}

	if len(opts.EndDate) > 0 {
		sqlString += fmt.Sprintf(` AND DATE_FORMAT(convert_tz(a.checkin_warehouse_at, @@session.time_zone,'+07:00') ,'%%Y-%%m-%%d') <= DATE('%s')`, opts.EndDate)
	}

	db = db.Raw(sqlString, constant.TrackingStatusSuccess, constant.ContainerItemActive, opts.FullName).Scan(result)
	return db.Error
}
func (m *PackageManager) GetExportPackagesCheckedInWarehouse(options ExportForm) ([]entity.Package, error) {
	db := m.db
	db = db.Where("packages.status IN (?)", constant.WarehouseStatus)
	if len(options.StartDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(packages.checkin_warehouse_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)", options.StartDate)
	}
	if len(options.EndDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(packages.checkin_warehouse_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)", options.EndDate)
	}
	if len(options.IgnoreUsers) > 0 {
		db = db.Where("packages.user_id NOT IN (?)", options.IgnoreUsers)
	}
	if options.WarehouseID > 0 {
		if options.IsWarehouseRole {
			db = db.Joins("LEFT JOIN warehouses ON warehouses.id = packages.warehouse_id")
			db = db.Where("warehouses.status = ?", constant.WareHouseStatusActive)
			db = db.Where("packages.warehouse_id = ?", options.WarehouseID)
		}
	}

	db = db.Preload("User").Preload("PackageCode")
	var packages []entity.Package
	db = db.Find(&packages)
	return packages, db.Error
}

func (m *PackageManager) WarehousLabelPDF(id int64, tk *entity.Tracking, change map[string]interface{}, user *entity.User) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	change["updated_at"] = time.Now()
	change["status"] = constant.PackageStatusWareHouseLabeled
	change["alert"] = constant.PackageAlertTypeDisable
	if user.Role == constant.UserRoleWarehouse {
		change["warehouse_id"] = user.WarehouseID
	} else {
		change["warehouse_id"] = viper.GetInt64("warehouse_vn_default")
	}
	if err := tx.Model(&entity.Package{}).Where("id=?", id).UpdateColumns(change).Error; err != nil {
		tx.Rollback()
		return err
	}

	tk.CreatedAt = time.Now()
	tk.UpdatedAt = tk.CreatedAt
	if err := tx.Create(tk).Error; err != nil {
		tx.Rollback()
		return err
	}

	// log := &entity.PackageDeliverLog{
	// 	PackageID: id,
	// 	Location:  constant.DeliverLogLocalLocation,
	// 	Status:    constant.DeliverLogTebexpressPicked,
	// 	Type:      constant.PackageDeliverLogTypeInWareHouse,
	// 	UserID:    utils.Int64(tk.UserID),
	// 	Model:     dbgorm.Model{CreatedAt: time.Now(), UpdatedAt: time.Now()},
	// }

	// if err := tx.Create(&log).Error; err != nil {
	// 	tx.Rollback()
	// 	return err
	// }

	return tx.Commit().Error
}

func (m *PackageManager) DeactivatePackageCodes(pendingPickUpMaxActive int64) ([]int64, error) {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return nil, err
	}

	sql := `
		UPDATE
			package_codes pkg_codes
		JOIN
			packages pkg ON pkg.package_code_id = pkg_codes.id
		SET
			pkg_codes.status = ?,
			pkg_codes.updated_at = ?
		WHERE
			pkg.status IN (?)
		AND 
			pkg_codes.status IN (?)
		AND 
			pkg.updated_at < NOW() - INTERVAL ? DAY
	`

	if err := tx.Exec(sql, constant.PackageCodeDisable,
		time.Now(),
		[]int64{constant.PackageStatusPendingPickup},
		[]int64{constant.PackageCodeEnable, constant.PackageCodeTemp},
		pendingPickUpMaxActive,
	).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	sql = `
		UPDATE
			packages pkg
		SET
			alert = ?, alert_at = ?
		WHERE
			pkg.status IN (?)
		AND 
			pkg.alert NOT IN (?)
		AND 
			pkg.updated_at < NOW() - INTERVAL ? DAY
	`
	if err := tx.Exec(sql, constant.PackageAlertTypeOverPretransit, time.Now(),
		[]int64{constant.PackageStatusPendingPickup}, []int64{constant.PackageAlertTypeWarehoseReturn, constant.PackageAlertTypeOverPretransit},
		pendingPickUpMaxActive,
	).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	var packages []entity.Package
	if err := tx.Model(&entity.Package{}).
		Where("status=?", constant.PackageStatusPendingPickup).
		Where("updated_at < NOW() - INTERVAL ? DAY", pendingPickUpMaxActive).
		Select("id, shipping_fee").
		Find(&packages).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if len(packages) < 1 {
		return nil, tx.Commit().Error
	}

	var ids []int64 // ids packages expired
	for _, p := range packages {
		ids = append(ids, p.ID)
	}

	sql = `
		UPDATE
			packages pkg
		SET
			status = ?,
			alert = ?,
			updated_at = ?,
			order_id = ?
		WHERE
			pkg.id IN (?)
	`
	if err := tx.Exec(sql, constant.PackageStatusExpired, constant.PackageAlertTypeDisable, time.Now(), nil, ids).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	//add timeline package expired
	logs := []entity.PackageDeliverLog{}
	for _, id := range ids {
		logs = append(logs, entity.PackageDeliverLog{
			PackageID: id,
			Status:    constant.DeliverLogTebexpressExpired,
			Type:      constant.PackageDeliverLogTypeExpired,
		})
	}

	if err := tx.Save(&logs).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	return ids, tx.Commit().Error
}

func (m *PackageManager) GetPackageDeliverLogs(pkg_id int64) ([]entity.PackageDeliverLog, error) {
	logs := []entity.PackageDeliverLog{}
	db := m.db.Where("package_id = ?", pkg_id).Find(&logs)
	return logs, db.Error
}

func (m *PackageManager) Save17TrackDataWebhook(pkg *entity.Package, logs []*entity.PackageDeliverLog, isPkgReturn bool, billId int64) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	for _, log := range logs {
		if err := tx.Save(log).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := tx.Omit(clause.Associations).Save(pkg).Error; err != nil {
		tx.Rollback()
		return err
	}

	if pkg.CountryCode == "AU" && isPkgReturn {
		fee := entity.ExtraFee{
			PackageID:      utils.Int64(pkg.ID),
			BillID:         &billId,
			ExtraFeeTypeID: constant.ExtraFeeTypeReturn,
			Description:    "Phí return đơn hàng",
			Amount:         cast.ToFloat64(viper.GetString("fee.returned")),
			Status:         constant.ExtraFeeStatusEnable,
		}
		if err := tx.Create(&fee).Error; err != nil {
			tx.Rollback()
			return err
		}

		log := &entity.PackageAuditLog{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			ExtraFeeID:  fee.ID,
			PackageID:   utils.Int64Value(fee.PackageID),
			Type:        constant.PackageUpdateReturnPackage,
			Fee:         fee.Amount,
			Description: fee.Description,
		}
		if err := tx.Model(&entity.PackageAuditLog{}).Create(&log).Error; err != nil {
			tx.Rollback()
			return err
		}

		transaction := &entity.Transaction{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			UserID: pkg.UserID,
			Type:   constant.TransactionLogTypePay,
			Status: constant.TransactionStatusSuccess,
			Amount: fee.Amount,
			BillID: &billId,
		}

		err := tx.Create(&transaction).Error
		if err != nil {
			fmt.Errorf("Create transaction error %v", err)
			tx.Rollback()
			return err
		}

		transactionLog := &entity.TransactionLog{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			UserID:        pkg.UserID,
			TransactionID: transaction.ID,
			Amount:        transaction.Amount,
			Type:          transaction.Type,
			Status:        transaction.Status,
			BillID:        transaction.BillID,
		}

		if err = tx.Model(entity.TransactionLog{}).Create(transactionLog).Error; err != nil {
			tx.Rollback()
			return err
		}

		var balanceType string
		if pkg.Service.Code == constant.ServiceCNCode {
			// balanceType = "balance_china" // remove china wallet
			balanceType = "balance"
		} else {
			balanceType = "balance"
		}

		sqlString := fmt.Sprintf("UPDATE users SET %s = %s - ?, updated_at = ? WHERE id = ?", balanceType, balanceType)
		if err := tx.Exec(sqlString, fee.Amount, time.Now(), pkg.UserID).Error; err != nil {
			tx.Rollback()
			return err
		}

		info := &entity.UserInfo{}
		if err := tx.First(info).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				tx.Rollback()
				return err
			}

			now := time.Now()
			info = &entity.UserInfo{UserID: pkg.UserID, UpdatedAt: &now, CancelMaxAmount: constant.DefaultCancelMaxAMount}
			if err := tx.Create(info).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		sqlString = `UPDATE user_infos SET debt_time = ? WHERE user_id = ? AND debt_time IS NULL AND (SELECT balance FROM users WHERE id = ? limit 1) < 0`
		if err := tx.Exec(sqlString, time.Now(), pkg.UserID, pkg.UserID).Error; err != nil {
			tx.Rollback()
			return err
		}

		sqlString = `UPDATE bills SET extra_fee = extra_fee + ? WHERE id = ?`
		if err := tx.Exec(sqlString, fee.Amount, billId).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

// 05/03/2025 this func wasnt be used, so balance_china wasnt be updated here. Update it when use
func (m *PackageManager) InWarehouse(tracking *entity.Tracking, extrafees []entity.ExtraFee, auditlogs []entity.PackageAuditLog, customerID, userID int64) error {
	tracking.CreatedAt = time.Now()
	tracking.UpdatedAt = tracking.CreatedAt

	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			fmt.Errorf("recover: %v", r)
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Create(tracking).Error; err != nil {
		tx.Rollback()
		return err
	}

	mchange := map[string]interface{}{
		"updated_at": time.Now(),
		"status":     constant.PackageStatusWareHouseLabeled,
	}

	if err := tx.Model(&entity.Package{}).Where("id=?", tracking.PackageID).UpdateColumns(mchange).Error; err != nil {
		tx.Rollback()
		return err
	}

	if len(extrafees) > 0 {
		if err := tx.Create(&extrafees).Error; err != nil {
			tx.Rollback()
			return err
		}

		var amount float64 = 0
		for _, v := range extrafees {
			amount += v.Amount
		}

		billID := utils.Int64Value(extrafees[0].BillID)
		if err := tx.Exec("UPDATE bills SET extra_fee=extra_fee+? WHERE id=?", amount, billID).Error; err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Exec("UPDATE users SET balance=balance-? WHERE id=?", amount, customerID).Error; err != nil {
			tx.Rollback()
			return err
		}

		if amount > 0 {
			info := &entity.UserInfo{}
			if err := tx.First(info).Error; err != nil {
				if err != gorm.ErrRecordNotFound {
					tx.Rollback()
					return err
				}

				now := time.Now()
				info = &entity.UserInfo{UserID: customerID, UpdatedAt: &now, CancelMaxAmount: constant.DefaultCancelMaxAMount}
				if err := tx.Create(info).Error; err != nil {
					tx.Rollback()
					return err
				}
			}

			strsql := "UPDATE user_infos SET debt_time = ? WHERE user_id = ? AND debt_time IS NULL AND (SELECT balance FROM users WHERE id = ? limit 1) < 0"
			if err := tx.Exec(strsql, time.Now(), customerID, customerID).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		transaction := entity.Transaction{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			UserID:  customerID,
			AdminID: userID,
			Type:    constant.TransactionLogTypePay,
			Status:  constant.TransactionStatusSuccess,
			Amount:  -amount,
			BillID:  &billID,
		}

		if err := tx.Create(&transaction).Error; err != nil {
			tx.Rollback()
			return err
		}

		transactionLog := entity.TransactionLog{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			UserID:        customerID,
			AdminID:       userID,
			TransactionID: transaction.ID,
			Amount:        transaction.Amount,
			Type:          transaction.Type,
			Status:        transaction.Status,
			BillID:        transaction.BillID,
		}

		if err := tx.Create(&transactionLog).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if len(auditlogs) > 0 {
		if err := tx.Create(&auditlogs).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m *PackageManager) WarehousInCheck(checkinPackage entity.CheckinPackage, pkg *entity.Package, userID, customerID int64, change map[string]interface{}, alogs []entity.PackageAuditLog, shippingFeePlus, outsizePlus, BatteryFeePlus float64, billID int64, priceByWeight bool, user *entity.User, extraPeakFeePlus float64, fees []entity.ExtraFee) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			fmt.Errorf("panic save warehouse checkin")
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	change["updated_at"] = time.Now()
	change["checkin_warehouse_at"] = time.Now()
	change["alert"] = constant.PackageAlertTypeDisable
	if user.Role == constant.UserRoleWarehouse {
		change["warehouse_id"] = user.WarehouseID
	} else {
		change["warehouse_id"] = viper.GetInt64("warehouse_vn_default")
	}
	change["user_id_imported"] = user.ID

	if _, ok := change["status"]; !ok {
		change["status"] = constant.PackageStatusPicked
	}

	if err := tx.Model(&entity.Package{}).Where("id=?", pkg.ID).UpdateColumns(change).Error; err != nil {
		tx.Rollback()
		return err
	}

	if tx.Model(&entity.CheckinPackage{}).Where("package_id=? AND status !=?", checkinPackage.PackageID, constant.CheckinPackageStatusSuccess).Updates(&checkinPackage).RowsAffected == 0 {
		tx.Create(checkinPackage)
	}

	if tx.Error != nil {
		tx.Rollback()
		return tx.Error
	}

	if shippingFeePlus > 0 || outsizePlus > 0 || extraPeakFeePlus > 0 || len(fees) > 0 || BatteryFeePlus > 0 {
		var amount float64 = 0
		var extraFees []entity.ExtraFee

		for _, fee := range fees {
			amount += fee.Amount
			extraFees = append(extraFees, entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				BillID:         &billID,
				PackageID:      utils.Int64(pkg.ID),
				ExtraFeeTypeID: fee.ExtraFeeTypeID,
				Amount:         fee.Amount,
				Status:         constant.ExtraFeeStatusEnable,
			})

			if fee.ExtraFeeTypeID == constant.ExtraFeeTypeBattery {
				continue
			}

			auditLog := entity.PackageAuditLog{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				PackageID:     pkg.ID,
				UpdatedUserID: userID,
				Type:          constant.MapTypeAuditLogByExtraFeeTypeID[fee.ExtraFeeTypeID],
				Fee:           fee.Amount,
			}

			alogs = append(alogs, auditLog)
		}

		if priceByWeight && shippingFeePlus > 0 {
			amount += shippingFeePlus
			extraFees = append(extraFees, entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				BillID:         &billID,
				PackageID:      utils.Int64(pkg.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeFixWeight,
				Amount:         shippingFeePlus,
				Status:         constant.ExtraFeeStatusEnable,
			})
		}

		if !priceByWeight && shippingFeePlus > 0 {
			amount += shippingFeePlus
			extraFees = append(extraFees, entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				BillID:         &billID,
				PackageID:      utils.Int64(pkg.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeFixVolume,
				Amount:         shippingFeePlus,
				Status:         constant.ExtraFeeStatusEnable,
			})
		}

		if outsizePlus > 0 {
			amount += outsizePlus
			extraFees = append(extraFees, entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				BillID:         &billID,
				PackageID:      utils.Int64(pkg.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeOutSize,
				Amount:         outsizePlus,
				Status:         constant.ExtraFeeStatusEnable,
			})
		}

		if BatteryFeePlus > 0 {
			amount += BatteryFeePlus
			extraFees = append(extraFees, entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				BillID:         &billID,
				PackageID:      utils.Int64(pkg.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeBattery,
				Amount:         BatteryFeePlus,
				Status:         constant.ExtraFeeStatusEnable,
			})
		}

		if extraPeakFeePlus > 0 {
			amount += extraPeakFeePlus
			extraFees = append(extraFees, entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				BillID:         &billID,
				PackageID:      utils.Int64(pkg.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypePeak,
				Amount:         extraPeakFeePlus,
				Status:         constant.ExtraFeeStatusEnable,
			})
		}

		if len(extraFees) > 0 {
			if err := tx.Create(&extraFees).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		if err := tx.Exec("UPDATE bills SET extra_fee=extra_fee+? WHERE id=?", amount, billID).Error; err != nil {
			tx.Rollback()
			return err
		}

		var balanceType string
		if pkg.Service.Code == constant.ServiceCNCode {
			// balanceType = "balance_china" // remove china wallet
			balanceType = "balance"
		} else {
			balanceType = "balance"
		}

		sqlString := fmt.Sprintf("UPDATE users SET %s = %s - ?, updated_at = ? WHERE id = ?", balanceType, balanceType)
		if err := tx.Exec(sqlString, amount, time.Now(), customerID).Error; err != nil {
			tx.Rollback()
			return err
		}

		if amount > 0 {
			info := &entity.UserInfo{}
			if err := tx.First(info).Error; err != nil {
				if err != gorm.ErrRecordNotFound {
					tx.Rollback()
					return err
				}

				now := time.Now()
				info = &entity.UserInfo{UserID: customerID, UpdatedAt: &now, CancelMaxAmount: constant.DefaultCancelMaxAMount}
				if err := tx.Create(info).Error; err != nil {
					tx.Rollback()
					return err
				}
			}

			strsql := "UPDATE user_infos SET debt_time = ? WHERE user_id = ? AND debt_time IS NULL AND (SELECT balance FROM users WHERE id = ? limit 1) < 0"
			if err := tx.Exec(strsql, time.Now(), customerID, customerID).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		transaction := entity.Transaction{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			UserID:  customerID,
			AdminID: userID,
			Type:    constant.TransactionLogTypePay,
			Status:  constant.TransactionStatusSuccess,
			Amount:  -amount,
			BillID:  &billID,
		}

		if err := tx.Create(&transaction).Error; err != nil {
			tx.Rollback()
			return err
		}

		transactionLog := entity.TransactionLog{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			UserID:        customerID,
			AdminID:       userID,
			TransactionID: transaction.ID,
			Amount:        transaction.Amount,
			Type:          transaction.Type,
			Status:        transaction.Status,
			BillID:        transaction.BillID,
		}

		if err := tx.Create(&transactionLog).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if extraPeakFeePlus < 0 {
		extraPeakFeePlus = 0
	}

	if len(alogs) > 0 {
		for i, audit := range alogs {
			if shippingFeePlus < 0 {
				shippingFeePlus = 0
			}

			if audit.Type == constant.PackageUpdateTypeWeight {
				alogs[i].Fee = extraPeakFeePlus
				if priceByWeight {
					alogs[i].Fee += shippingFeePlus
				}
			}

			if audit.Type == constant.PackageUpdateTypeVolume && !priceByWeight {
				alogs[i].Fee = shippingFeePlus
			}

			if audit.Type == constant.PackageUpdateTypeVolume && outsizePlus > 0 {

				log := &entity.PackageAuditLog{
					Model: dbgorm.Model{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					PackageID:     pkg.ID,
					UpdatedUserID: userID,
					OldValue:      alogs[i].OldValue,
					Value:         alogs[i].Value,
					Type:          constant.PackageUpdateExtraFeeTypeOutSize,
					Fee:           outsizePlus,
				}
				alogs = append(alogs, *log)
			}

		}

		if err := tx.Create(&alogs).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	dlog := &entity.PackageDeliverLog{
		PackageID: pkg.ID,
		Location:  fmt.Sprintf("%s, %s", pkg.Warehouse.City, pkg.Warehouse.Country),
		Status:    constant.DeliverLogTebexpressPicked,
		Type:      constant.PackageDeliverLogTypeInWareHouse,
		UserID:    &userID,
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	if err := tx.Create(dlog).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m *PackageManager) ReturnPackage(ids []int64, checkinID int64, logs []entity.PackageDeliverLog, alog *entity.PackageAuditLog) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	change := map[string]interface{}{
		"updated_at":  time.Now(),
		"returned_at": time.Now(),
		"status":      constant.PackageStatusPendingPickup,
		"alert":       constant.PackageAlertTypeWarehoseReturn,
	}

	if checkinID > 0 {
		change["checkin_warehouse_at"] = time.Now()
	}

	for _, id := range ids {
		if err := tx.Model(&entity.Package{}).Where("id=?", id).UpdateColumns(change).Error; err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Model(&entity.ContainerItem{}).Where("package_id = ?", id).UpdateColumns(map[string]interface{}{
			"updated_at": time.Now(),
			"status":     constant.ContainerItemInactive,
		}).Error; err != nil {
			tx.Rollback()
			return err
		}

		checkinPackage := entity.CheckinPackage{
			CheckinID: checkinID,
			PackageID: id,
			Status:    constant.CheckinPackageStatusSuccess,
		}
		if tx.Model(&entity.CheckinPackage{}).Where("package_id=? AND status !=?", checkinPackage.PackageID, constant.CheckinPackageStatusSuccess).Updates(&checkinPackage).RowsAffected == 0 {
			tx.Create(checkinPackage)
		}
		if tx.Error != nil {
			tx.Rollback()
			return tx.Error
		}
	}

	if len(logs) > 0 {
		if err := tx.Create(&logs).Error; err != nil {
			tx.Rollback()
			return err
		}

	}

	if alog != nil {
		alog.UpdatedAt = time.Now()
		alog.CreatedAt = alog.UpdatedAt
		if err := tx.Create(alog).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m *PackageManager) GetPackageIntransit() ([]dto.UserPackageStatus, error) {

	sdb := m.db.Table("package_deliver_logs")
	sdb = sdb.Select("package_id, min(package_deliver_logs.created_at) AS created_at")
	sdb = sdb.Where("type=?", constant.PackageStatusInTransit)
	sdb = sdb.Group("package_id")
	sdb = sdb.Having("created_at > (NOW() - INTERVAL 1 DAY)")

	db := m.db.Table("(?) as logs", sdb)
	db = db.Select("packages.user_id, COUNT(packages.id) as count")
	db = db.Joins("INNER JOIN packages ON packages.id=logs.package_id")
	db = db.Group("packages.user_id")

	var result []dto.UserPackageStatus
	db = db.Scan(&result)

	return result, db.Error
}

func (m *PackageManager) GetPackageDelivered(date string) ([]dto.UserPackageStatus, error) {
	db := m.db.Table("package_deliver_logs")
	db = db.Select("packages.user_id, COUNT(packages.id) as count")
	db = db.Joins("INNER JOIN packages ON packages.id=package_deliver_logs.package_id")
	db = db.Where("package_deliver_logs.type=?", constant.PackageStatusDelivered)
	db = db.Where(`DATE_FORMAT(convert_tz(package_deliver_logs.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') = DATE(?)`, date)
	db = db.Group("packages.user_id")

	var result []dto.UserPackageStatus
	db = db.Scan(&result)

	return result, db.Error
}

func (m *PackageManager) GetPackageRefund(date string) ([]dto.UserPackageStatus, error) {
	sdb := m.db.Table("extra_fees")
	sdb = sdb.Where("extra_fee_type_id=?", constant.ExtraFeeTypeRefund)
	sdb = sdb.Where("status=?", constant.ExtraFeeStatusEnable)
	sdb = sdb.Where(`DATE_FORMAT(convert_tz(created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') = DATE(?)`, date)
	sdb = sdb.Group("extra_fees.package_id").Select("package_id")

	db := m.db.Table("(?) AS ef", sdb)
	db = db.Select("packages.user_id, COUNT(packages.id) as count")
	db = db.Joins("INNER JOIN packages ON packages.id=ef.package_id")
	db = db.Group("packages.user_id")

	var result []dto.UserPackageStatus
	db = db.Scan(&result)

	return result, db.Error
}

func (m *PackageManager) ReshipCustomer(id int64, mapchange map[string]interface{}, logs []entity.PackageAuditLog) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			return
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if mapchange == nil {
		mapchange = make(map[string]interface{})
	}

	mapchange["updated_at"] = time.Now()
	mapchange["request_reship"] = true

	if err := tx.Table("packages").Where("id=?", id).UpdateColumns(mapchange).Error; err != nil {
		tx.Rollback()
		return err
	}

	if len(logs) > 0 {
		if err := tx.Create(&logs).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m *PackageManager) CheckPermissionUserPackages(userID int64, packageIDs []int64) (bool, error) {
	db := m.db.Model(&entity.Package{})
	db = db.Joins("LEFT JOIN user_permissions ON user_permissions.customer_id=packages.user_id")
	db = db.Where("packages.id IN (?)", packageIDs)
	db = db.Where("user_permissions.support_id=?", userID)

	var count int64
	db = db.Count(&count)

	return count == int64(len(packageIDs)), db.Error
}

func (m *PackageManager) Reship(id int64, isPackageCN bool, mapchange map[string]interface{}, tracking *entity.Tracking, userID, customerID, billID int64, amount float64, description string, logs []entity.PackageAuditLog) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			return
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	err := tx.Model(&entity.Tracking{}).Where("package_id=?", id).
		Update("status", constant.TrackingStatusCanceled).Error

	if err != nil {
		tx.Rollback()
		return err
	}

	tracking.CreatedAt = time.Now()
	tracking.UpdatedAt = tracking.CreatedAt
	if err := tx.Create(tracking).Error; err != nil {
		tx.Rollback()
		return err
	}

	mapchange["updated_at"] = time.Now()
	mapchange["alert"] = constant.PackageAlertTypeDisable
	mapchange["status"] = constant.PackageStatusReship
	mapchange["reship_at"] = time.Now()
	mapchange["request_reship"] = false

	if err := tx.Table("packages").Where("id=?", id).UpdateColumns(mapchange).Error; err != nil {
		tx.Rollback()
		return err
	}

	if amount >= 0 {
		extraFee := &entity.ExtraFee{
			BillID:         &billID,
			PackageID:      utils.Int64(id),
			ExtraFeeTypeID: constant.ExtraFeeTypeReship,
			Amount:         amount,
			Status:         constant.ExtraFeeStatusEnable,
			Description:    description,
		}

		if err := tx.Create(extraFee).Error; err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Exec("UPDATE bills SET extra_fee=extra_fee+? WHERE id=?", amount, billID).Error; err != nil {
			tx.Rollback()
			return err
		}

		var balanceType string
		if isPackageCN {
			// balanceType = "balance_china" // remove china wallet
			balanceType = "balance"
		} else {
			balanceType = "balance"
		}

		sqlString := fmt.Sprintf("UPDATE users SET %s = %s - ?, updated_at = ? WHERE id = ?", balanceType, balanceType)
		if err := tx.Exec(sqlString, amount, time.Now(), customerID).Error; err != nil {
			tx.Rollback()
			return err
		}

		info := &entity.UserInfo{}
		if err := tx.First(info).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				tx.Rollback()
				return err
			}

			now := time.Now()
			info = &entity.UserInfo{UserID: customerID, UpdatedAt: &now, CancelMaxAmount: constant.DefaultCancelMaxAMount}
			if err := tx.Create(info).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		strsql := "UPDATE user_infos SET debt_time = ? WHERE user_id = ? AND debt_time IS NULL AND (SELECT balance FROM users WHERE id = ? limit 1) < 0"
		if err := tx.Exec(strsql, time.Now(), customerID, customerID).Error; err != nil {
			tx.Rollback()
			return err
		}

		transaction := &entity.Transaction{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			UserID:  customerID,
			AdminID: userID,
			Type:    constant.TransactionLogTypePay,
			Status:  constant.TransactionStatusSuccess,
			Amount:  -amount,
			BillID:  &billID,
		}

		if err := tx.Create(transaction).Error; err != nil {
			tx.Rollback()
			return err
		}

		transactionLog := &entity.TransactionLog{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			UserID:        customerID,
			AdminID:       userID,
			TransactionID: transaction.ID,
			Amount:        transaction.Amount,
			Type:          transaction.Type,
			Status:        transaction.Status,
			BillID:        transaction.BillID,
		}

		if err := tx.Create(transactionLog).Error; err != nil {
			tx.Rollback()
			return err
		}

		for i := 0; i < len(logs); i++ {
			logs[i].PackageID = id
			logs[i].UpdatedUserID = userID
		}

		log := &entity.PackageAuditLog{
			PackageID:     id,
			Type:          constant.PackageUpdateExtraFeeTypeReship,
			UpdatedUserID: userID,
			Fee:           extraFee.Amount,
			Value:         description,
			Description:   description,
		}
		logs = append(logs, *log)
		if err := tx.Create(&logs).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	deliverLog := &entity.PackageDeliverLog{
		PackageID:   id,
		Status:      constant.DeliverLogTebexpressReship,
		Type:        constant.PackageDeliverLogTypeReship,
		Description: description,
		UserID:      &userID,
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	if err := tx.Create(deliverLog).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
func (m *PackageManager) CheckRefundfeeByPkgID(packageID int64) (bool, error) {
	var count int64

	db := m.db.Model(&entity.ExtraFee{})
	db = db.Where("package_id = ? and extra_fee_type_id = ? and extra_fees.status = ? ", packageID, constant.ExtraFeeTypeRefund, constant.ExtraFeeStatusEnable)
	db = db.Preload("ExtraFeeType")
	db = db.Joins("left join extra_fee_types on extra_fee_types.id = extra_fees.extra_fee_type_id")
	db = db.Where("extra_fee_types.status=?", constant.ExtraFeeStatusEnable)
	db = db.Count(&count)

	return count > 0, db.Error
}

func (m *PackageManager) GetPackageIntransitExpire(day int, limit, offset int) ([]int64, error) {
	var ids []int64

	db := m.db.Model(&entity.Package{})
	db = db.Where("status=?", constant.PackageStatusInTransit)
	db = db.Where("updated_at <= DATE_SUB(NOW(), INTERVAL ? DAY)", day)
	db = db.Where("alert=0 OR alert is NULL")

	if limit > 0 {
		db = db.Limit(limit)
	}

	if offset > 0 {
		db = db.Offset(offset)
	}

	db = db.Pluck("id", &ids)
	return ids, db.Error
}

func (m *PackageManager) CountPackageIntransitExpire(day int) (int64, error) {
	var count int64

	db := m.db.Model(&entity.Package{})
	db = db.Where("status=?", constant.PackageStatusInTransit)
	db = db.Where("updated_at <= DATE_SUB(NOW(), INTERVAL ? DAY)", day)
	db = db.Where("alert=0 OR alert is NULL")
	db = db.Count(&count)

	return count, db.Error
}

func (m *PackageManager) UndeliveredPackageIntransitExpire(day int) error {
	mapChange := map[string]interface{}{
		"status":     constant.PackageStatusUndelivered,
		"alert":      constant.PackageAlertTypeDisable,
		"updated_at": time.Now(),
	}

	statusList := constant.MapIntGroupStatusAdminPackage["in-transit"]

	db := m.db.Model(&entity.Package{}).
		Where("status IN (?)", statusList).
		Where("updated_at <= DATE_SUB(NOW(), INTERVAL ? DAY)", day).
		Updates(mapChange)

	return db.Error
}

func (m *PackageManager) CancelPackageProcessingExpire(day int, limit int) ([]int64, error) {
	statusList := constant.MapIntGroupStatusAdminPackage["processing"]

	var ids []int64
	if err := m.db.Model(&entity.Package{}).
		Where("status IN (?)", statusList).
		Where("updated_at <= DATE_SUB(NOW(), INTERVAL ? DAY)", day).
		Select("id").
		Limit(limit).
		Pluck("id", &ids).Error; err != nil {
		return ids, err
	}

	if len(ids) < 1 {
		return ids, nil
	}

	mapChange := map[string]interface{}{
		"order_id":   nil,
		"status":     constant.PackageStatusCancelled,
		"alert":      constant.PackageAlertTypeDisable,
		"updated_at": time.Now(),
	}

	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return ids, err
	}

	if err := tx.Model(&entity.Package{}).
		Where("id IN (?)", ids).
		Updates(mapChange).Error; err != nil {
		tx.Rollback()
		return ids, err
	}

	var logs []entity.PackageDeliverLog
	for _, id := range ids {
		logs = append(logs, entity.PackageDeliverLog{
			PackageID: id,
			Status:    constant.DeliverLogTebexpressCanceled,
			Type:      constant.PackageDeliverLogTypeCancelled,
		})
	}

	if err := tx.Create(&logs).Error; err != nil {
		tx.Rollback()
		return ids, err
	}

	return ids, tx.Commit().Error
}

func (m PackageManager) GetPackagesRefundExpiredPending(day int) ([]entity.PackageRefund, error) {
	var refunds []entity.PackageRefund
	db := m.db.Where("created_at <= DATE_SUB(NOW(), INTERVAL ? DAY)", day)
	db = db.Where("created_at > ?", "2024-10-01")
	db = db.Where("status", constant.PackageRefundPending)
	db = db.Preload("Package", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, user_id")
	})
	db = db.Find(&refunds)
	return refunds, db.Error
}

func (m PackageManager) GetPackakgeRefundByPackageID(pkgID int64) (*entity.PackageRefund, error) {
	var pkgRefund *entity.PackageRefund
	db := m.db.Where("package_id = ? ", pkgID)
	db = db.Where("status", constant.PackageRefundPending)
	db = db.First(&pkgRefund)
	return pkgRefund, db.Error
}

func (m PackageManager) GetUserIDsOfPackages(ids []int64) ([]int64, error) {
	var userIDs []int64

	db := m.db.Model(&entity.Package{})
	db = db.Where("id IN (?)", ids)
	db = db.Pluck("user_id", &userIDs)

	return userIDs, db.Error
}

func (m PackageManager) GetListPackagesRefund(opts PackageQueryOption) ([]entity.PackageRefund, error) {
	db := m.BuildPackageRefundQuery(opts)
	db = db.Model(&entity.PackageRefund{})
	db = db.Preload("Package").Preload("Package.PackageCode")

	packages := []entity.PackageRefund{}
	db = db.Find(&packages)
	return packages, db.Error
}

func (m PackageManager) CountPackagesRefund(opts PackageQueryOption) (int64, error) {
	db := m.BuildPackageRefundQuery(opts)
	var count int64
	db = db.Model(&entity.PackageRefund{}).Count(&count)
	return count, db.Error
}

func (m PackageManager) GetProducts(userID, packageID int64, v interface{}) error {
	db := m.db.Model(&entity.PackageProducts{})
	db = db.Joins("LEFT JOIN products ON package_products.product_id=products.id")
	db = db.Select(`
		package_products.*,
		products.user_id,
		products.name,
		products.sku,
		products.weight,
		products.width,
		products.length,
		products.height,
		products.detail,
		products.material,
		products.status AS product_status
	`)

	db = db.Where("products.user_id=?", userID)
	db = db.Where("package_products.package_id=?", packageID)
	db = db.Where("package_products.status=?", constant.PackageProductsStatusActive)
	db = db.Find(v)

	return db.Error
}

func (m PackageManager) SaveCheckinPackage(checkinPackage entity.CheckinPackage) error {
	db := m.db.Model(&entity.CheckinPackage{})
	if db.Where("package_id=?", checkinPackage.PackageID).Updates(&checkinPackage).RowsAffected == 0 {
		db.Create(checkinPackage)
	}
	return db.Error
}

func (m PackageManager) GetListPackagesRefundByPackageID(packageID int64, status int) ([]entity.PackageRefund, error) {
	packages := []entity.PackageRefund{}
	db := m.db.Where("package_id=?", packageID)

	if status > 0 {
		db = db.Where("status=?", status)
	}

	db = db.Find(&packages)
	return packages, db.Error
}

func (m PackageManager) PartnerGetPackage(code string) (*entity.Package, error) {
	pCode := &entity.PackageCode{}
	db := m.db.Where("code=?", code).Select("id,code,status")
	if err := db.First(pCode).Error; err != nil {
		return nil, err
	}

	db1 := m.db.Where("package_code_id=?", pCode.ID)
	db1 = db1.Select("id,created_at,package_code_id")

	db1 = db1.Preload("Tracking", func(db *gorm.DB) *gorm.DB {
		db = db.Select("id,tracking_number,package_id")
		db = db.Where("trackings.status != ?", constant.TrackingStatusCanceled)
		return db
	})

	// db1 = db1.Preload("Service.DomesticCarrier")

	pkg := &entity.Package{}
	db1 = db1.First(&pkg)

	pkg.PackageCode = pCode
	return pkg, db1.Error
}

func (m PackageManager) PartnerGetDeliverPackageLogs(code string) ([]dto.PackageDeliverLogDTO, error) {
	deliverLogs := []dto.PackageDeliverLogDTO{}
	sql := `
		SELECT 
			package_deliver_logs.package_id,
			package_deliver_logs.location,
			package_deliver_logs.status,
			package_deliver_logs.type,
			"" AS code,
			package_deliver_logs.created_at AS ship_time,
			package_deliver_logs.description
		FROM
			package_deliver_logs
			INNER JOIN packages ON packages.id = package_deliver_logs.package_id
			INNER JOIN package_codes ON package_codes.id = packages.package_code_id
		WHERE
			package_codes.code =?
			AND
			package_deliver_logs.description not in (?)
			AND
			package_deliver_logs.type >= ?
		UNION
		SELECT
			packages.id AS package_id,
			location,
			container_deliver_logs.status,
			30 AS type,
			container_deliver_logs.code,
			ship_time,
			container_deliver_logs.description
		FROM
			container_deliver_logs
			INNER JOIN container_items on container_items.container_id = container_deliver_logs.container_id
			INNER JOIN packages on packages.id = container_items.package_id
			INNER JOIN package_codes ON package_codes.id = packages.package_code_id
		WHERE
			container_deliver_logs.code NOT IN (?) AND package_codes.code=? AND container_items.status = ?
		ORDER BY ship_time DESC, type DESC
	`

	ignoreDescription := []string{"Shipping Label Created, USPS Awaiting Item", "Received data"}
	db := m.db.Raw(sql, code, ignoreDescription, constant.PackageStatusPicked, constant.UPSIgnoreCodeLogs, code, constant.ContainerItemActive).Scan(&deliverLogs)
	return deliverLogs, db.Error
}

func (m PackageManager) QueryCountUserPackageAlert(opts PackageQueryOption) (int64, error) {
	db := m.BuildPackageQuery(opts)
	var count int64
	db = db.Select("COUNT(DISTINCT(user_id))").Model(&entity.Package{}).Count(&count)
	return count, db.Error
}

func (m PackageManager) QueryListUserPackageAlert(opts PackageQueryOption) ([]dto.UserAlertPackge, error) {
	db := m.BuildPackageQuery(opts).Model(&entity.Package{})
	db = db.Joins("LEFT JOIN users ON users.id = packages.user_id")
	db = db.Group("users.id")
	var users []dto.UserAlertPackge
	db = db.Select("users.id,users.email,users.full_name,COUNT(packages.id) as count").Scan(&users)
	return users, db.Error
}

func (m PackageManager) QueryTrackingUpdateLogs(days int64) (int64, error) {
	sql := `SELECT COUNT(*)  FROM (SELECT  MAX(package_deliver_logs.created_at) as maximum,package_id  FROM package_deliver_logs 
			INNER JOIN packages ON packages.id = package_deliver_logs.package_id AND packages.status = ?
			GROUP BY package_id) temp WHERE temp.maximum < CURRENT_TIMESTAMP()  - INTERVAL ? DAY`
	var count int64
	db := m.db.Raw(sql, constant.PackageStatusInTransit, days).Count(&count)
	return count, db.Error
}

func (m PackageManager) QueryPackageErrorIntransit() ([]dto.MissingPackageDTO, error) {
	db := m.db.Model(&entity.PackageDeliverLog{}).Joins("INNER JOIN packages ON packages.id = package_deliver_logs.package_id")
	db = db.Where("( package_deliver_logs.type IN (?) AND package_deliver_logs.user_id IS NULL ) AND packages.status IN (?)", []int64{constant.PackageStatusInTransit, constant.PackageStatusDelivered}, []int{constant.PackageStatusCreated, constant.PackageStatusPendingPickup, constant.PackageStatusPicked, constant.PackageStatusWareHouseLabeled, constant.PackageStatusWareHouseInContainer, constant.PackageStatusWareHouseInShipment, constant.PackageStatusWareHouseExport})
	var pkgs []dto.MissingPackageDTO
	db = db.Select("DISTINCT(package_deliver_logs.package_id ) as id").Scan(&pkgs)
	return pkgs, db.Error
}

func (m *PackageManager) FetchPackagesReturnExpire(day int, limit int) ([]entity.Package, error) {
	packages := []entity.Package{}

	db := m.db.Where("returned_at <= CURRENT_TIMESTAMP() - INTERVAL ? DAY", day)
	db = db.Where("alert=?", constant.PackageAlertTypeHubReturn)
	db = db.Preload("User").Preload("PackageCode")
	db = db.Order("user_id ASC")

	if limit > 0 {
		db = db.Limit(limit)
	}

	db = db.Find(&packages)
	return packages, db.Error
}

func (m *PackageManager) CountPackagesReturnExpire(day int) (int64, error) {
	var count int64
	db := m.db.Where("returned_at <= CURRENT_TIMESTAMP() - INTERVAL ? DAY", day)
	db = db.Where("alert=?", constant.PackageAlertTypeHubReturn)
	db = db.Model(&entity.Package{}).Count(&count)
	return count, db.Error
}

func (m *PackageManager) CancelPackagesReturnExpire(ids []int64) error {
	mapChange := map[string]interface{}{
		"order_id":   nil,
		"status":     constant.PackageStatusCancelled,
		"alert":      constant.PackageAlertTypeDisable,
		"updated_at": time.Now(),
	}

	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Model(&entity.Package{}).Where("id IN (?)", ids).Updates(mapChange).Error; err != nil {
		tx.Rollback()
		return err
	}

	var logs []entity.PackageDeliverLog
	for _, id := range ids {
		logs = append(logs, entity.PackageDeliverLog{
			PackageID: id,
			Status:    constant.DeliverLogTebexpressCanceled,
			Type:      constant.PackageDeliverLogTypeCancelled,
		})
	}

	if err := tx.Create(&logs).Error; err != nil {
		tx.Rollback()
		return err
	}

	mapChangeTrack := map[string]interface{}{
		"status":     constant.TrackingStatusCanceled,
		"updated_at": time.Now(),
	}
	if err := tx.Model(&entity.Tracking{}).Where("package_id IN (?)", ids).Updates(mapChangeTrack).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m PackageManager) SavePackage(pkg entity.Package) error {
	db := m.db
	db = db.Save(&pkg)
	return db.Error
}

func (m PackageManager) UpdateBookmark(pkg entity.Package) error {
	db := m.db
	db = db.Model(&entity.Package{}).Where("id=?", pkg.ID).Updates(map[string]interface{}{
		"is_bookmark": pkg.IsBookmark,
		"updated_at":  time.Now(),
	})
	return db.Error
}

func (m PackageManager) SavePackageDeliveryLog(log entity.PackageDeliverLog) error {
	db := m.db
	db = db.Save(&log)
	return db.Error
}

func (m PackageManager) CreatePackageFBA(packages []*entity.Package, totalWeight, ratePrice float64, extras []*entity.ExtraFee) (*entity.CustomerShipment, error) {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			fmt.Errorf("Panic create packages", r)
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return nil, err
	}

	cShipment := &entity.CustomerShipment{
		Weight: totalWeight,
		UserID: packages[0].UserID,
		Price:  ratePrice,
		Status: constant.PackageStatusCreated,
	}

	if err := tx.Create(&cShipment).Error; err != nil {
		fmt.Errorf("Error create shipment", err)
		tx.Rollback()
		return nil, err
	}

	for _, Package := range packages {
		Package.CustomerShipmentID = &cShipment.ID
		err := tx.Omit("Service", "ExtraFee", "Tracking", "PackageCode", "User").Create(&Package).Error
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		cShipment.Packages = append(cShipment.Packages, *Package)
	}

	if len(extras) > 0 {
		now := time.Now()
		for i := range extras {
			extras[i].CreatedAt = now
			extras[i].UpdatedAt = now
			extras[i].CustomerShipmentID = utils.Int64(cShipment.ID)
		}

		if err := tx.Omit("Package", "CustomerShipment", "ExtraFeeType").Create(&extras).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	return cShipment, tx.Commit().Error
}

func (m PackageManager) SaveUpdatePackageFBA(id, userID int64, mapchange map[string]interface{}, logs []entity.PackageAuditLog, changePackages []map[string]interface{}, shipment *entity.CustomerShipment, changeProducts []*entity.PackageProducts) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			return
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Table("packages").Where("id=?", id).UpdateColumns(mapchange).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, prod := range changeProducts {
		prodChange := make(map[string]interface{})
		prodChange["quantity"] = prod.Quantity
		prodChange["status"] = prod.Status

		if err := tx.Table("package_products").Where("product_id=? and package_id=?", prod.ProductID, id).First(&entity.PackageProducts{}).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := tx.Create(&prod).Error; err != nil {
					tx.Rollback()
					return err
				}
			} else {
				tx.Rollback()
				return err
			}
		} else {
			if err := tx.Table("package_products").Where("product_id=? and package_id=?", prod.ProductID, id).UpdateColumns(prodChange).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	for i := range logs {
		logs[i].PackageID = id
		logs[i].UpdatedUserID = userID
	}

	if len(logs) > 0 {
		if err := tx.Model(&entity.PackageAuditLog{}).Create(&logs).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if len(changePackages) > 0 {
		for _, change := range changePackages {
			if err := tx.Table("packages").Where("id=?", change["id"]).UpdateColumns(change).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	if shipment != nil {
		shipment.UpdatedAt = time.Now()
		if err := tx.Save(shipment).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m PackageManager) CreateAppendPackageFBA(userID int64, pkg *entity.Package, changePackages []map[string]interface{}, shipment *entity.CustomerShipment) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			return
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	pkg.CreatedAt = time.Now()
	pkg.UpdatedAt = pkg.CreatedAt
	if err := tx.Omit("Service", "ExtraFee", "Tracking", "PackageCode", "User").Create(pkg).Error; err != nil {
		tx.Rollback()
		return err
	}

	if len(changePackages) > 0 {
		for _, change := range changePackages {
			if err := tx.Table("packages").Where("id=?", change["id"]).UpdateColumns(change).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	if shipment != nil {
		shipment.UpdatedAt = time.Now()
		if err := tx.Save(shipment).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m PackageManager) GetCustomerPackagesInfo(customerIDs []int64, result interface{}) error {
	db := m.db.Raw(`SELECT SUM(shipping_fee + amount) as revenue,a.user_id as customer_id, total_package FROM
	(SELECT SUM(shipping_fee) as shipping_fee ,COUNT(packages.id) as total_package,user_id  FROM packages WHERE user_id  IN (?) AND status NOT IN (?)   GROUP BY  packages.user_id) as a
	INNER JOIN 
	(SELECT
			COALESCE(SUM(ef.amount),0) as amount,bills.user_id FROM
			bills LEFT JOIN extra_fees ef  ON
			ef.bill_id = bills.id
			AND ef.status = 1 WHERE
			bills.user_id IN (?)
			GROUP BY  bills.user_id
			) b ON b.user_id = a.user_id
			GROUP BY a.user_id`, customerIDs, []int{constant.PackageStatusArchived, constant.PackageStatusCreated}, customerIDs)
	db = db.Scan(result)
	return db.Error
}

func (m PackageManager) GetTotalRevenueSupport(supportID int64) (float64, error) {
	db := m.db.Model(entity.Bill{})
	db = db.Where("bills.user_id IN (?)", m.db.Model(&entity.UserPermission{}).Select("customer_id").Where("support_id = ?", supportID))
	db = db.Joins("INNER JOIN users ON users.id = bills.user_id  AND users.status = ?", constant.UserStatusActive)
	db = db.Select("sum(bills.shipping_fee + bills.extra_fee) as revenue")
	var result struct {
		Revenue float64 `json:"revenue"`
	}
	db = db.Scan(&result)
	return result.Revenue, db.Error
}

func (m PackageManager) GetMonthRevenue(supportID int64, startDate, endDate string, result interface{}) error {
	db := m.db.Model(entity.Bill{})
	db = db.Where("bills.user_id IN (?)", m.db.Model(&entity.UserPermission{}).Select("customer_id").Where("support_id = ?", supportID))
	db = db.Joins("INNER JOIN users ON users.id = bills.user_id  AND users.status = ?", constant.UserStatusActive)
	db = db.Select("sum(bills.shipping_fee + bills.extra_fee) as revenue,DATE_FORMAT(bills.created_at,'%m/%Y') as month")
	db = db.Where("DATE(bills.created_at) between ? AND ?", startDate, endDate)
	db = db.Group("DATE_FORMAT(bills.created_at,'%m/%Y')")
	db = db.Scan(result)
	return db.Error
}

func (m PackageManager) GetTopCustomer(supportID int64, topCount int, result interface{}) error {
	db := m.db.Raw(`SELECT SUM(shipping_fee+extra_fee) as revenue,users.full_name FROM bills
					INNER JOIN users ON users.id = bills.user_id AND users.status = ?
					WHERE user_id  IN ((SELECT customer_id FROM user_permissions WHERE support_id = ?))
					GROUP BY user_id
					ORDER BY revenue DESC
					LIMIT ?`, constant.UserStatusActive, supportID, topCount)
	db = db.Scan(result)
	return db.Error
}

func (m *PackageManager) EstimateDelivery(opts PackageQueryOption, v interface{}) error {
	sdb := m.db.Table("packages")
	sdb = sdb.Joins("INNER JOIN trackings ON trackings.package_id=packages.id")

	sdb = sdb.Select(`
		trackings.zone,
		packages.country_code AS country,
		IF(trackings.weight > trackings.height * trackings.width * trackings.length / 5, trackings.weight, trackings.height * trackings.width * trackings.length / 5) AS volume,
		(86400 * 5 * (DATEDIFF(packages.delivered_at, packages.checkin_warehouse_at) DIV 7) + 86400 * MID('0123455501234445012333450122234501101234000123450', 7 * WEEKDAY(packages.checkin_warehouse_at) + WEEKDAY(packages.delivered_at) + 1, 1) + TIMESTAMPDIFF(SECOND, DATE(packages.delivered_at), packages.delivered_at) - TIMESTAMPDIFF(SECOND, DATE(packages.checkin_warehouse_at), packages.checkin_warehouse_at)) AS duration
	`)

	if opts.StartDate != "" {
		sdb = sdb.Where("DATE_FORMAT(convert_tz(packages.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)", opts.StartDate)
	}

	if opts.EndDate != "" {
		sdb = sdb.Where("DATE_FORMAT(convert_tz(packages.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)", opts.EndDate)
	}

	if opts.InWarehouseStartDate != "" {
		sdb = sdb.Where("DATE_FORMAT(convert_tz(packages.checkin_warehouse_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)", opts.InWarehouseStartDate)
	}

	if opts.InWarehouseEndDate != "" {
		sdb = sdb.Where("DATE_FORMAT(convert_tz(packages.checkin_warehouse_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)", opts.InWarehouseEndDate)
	}

	if opts.DeliveredStartDate != "" {
		sdb = sdb.Where("DATE_FORMAT(convert_tz(packages.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)", opts.StartDate)
	}

	if opts.DeliveredEndDate != "" {
		sdb = sdb.Where("DATE_FORMAT(convert_tz(packages.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)", opts.EndDate)
	}

	if opts.Status > 0 {
		sdb = sdb.Where("packages.status=?", opts.Status)
	}

	if len(opts.StatusArr) > 0 {
		sdb = sdb.Where("packages.status IN (?)", opts.StatusArr)
	}

	sdb = sdb.Where("packages.checkin_warehouse_at IS NOT NULL")
	sdb = sdb.Where("packages.delivered_at IS NOT NULL")
	sdb = sdb.Where("packages.checkin_warehouse_at < packages.delivered_at")
	sdb = sdb.Where("trackings.status=?", constant.TrackingStatusSuccess)
	weights := createlabel.GetWeightStep()
	txtSelect := ""
	for i, w := range weights {
		if i == 0 {
			txtSelect += fmt.Sprintf(`
						CASE WHEN volume < %.0f THEN 1
                      `, w/createlabel.GramToOz)
		} else {
			txtSelect += fmt.Sprintf(`
											WHEN volume < %.0f THEN %d
										`, w/createlabel.GramToOz, i+1)
		}
	}
	txtSelect += fmt.Sprintf(`ELSE %d END AS point, AVG(duration) as duration,country,zone
								`, len(weights)+1)
	db := m.db.Select(txtSelect)

	db = db.Table("(?) AS custom", sdb)
	db = db.Group("zone, point, country")
	db = db.Order("country, point, zone DESC")
	db = db.Scan(v)
	return db.Error
}

func (m PackageManager) BuildCouponQuery(opts CouponQueryOption) *gorm.DB {
	db := m.db
	db = db.Model(&entity.Coupon{})
	if opts.Search != "" {
		switch opts.SearchBy {
		case "code":
			db = db.Where("code LIKE (?)", fmt.Sprintf("%%%s%%", opts.Search))
			break
		case "customer_name":
			db = db.Joins("JOIN users ON users.id = coupons.customer_id")
			db = db.Where("users.full_name LIKE (?)", fmt.Sprintf("%%%s%%", opts.Search))
			break
		}
	}

	if opts.ID > 0 {
		db = db.Where("coupons.id = ?", opts.ID)
	}

	if opts.CustomerID > 0 || opts.CoponUserID > 0 {
		db = db.Joins("JOIN coupon_users ON coupon_users.coupon_id = coupons.id")
		if opts.CustomerID > 0 {
			db = db.Where("coupon_users.customer_id = ?", opts.CustomerID)
			if opts.IsUsed {
				db = db.Where("coupon_users.quantity = coupon_users.used")
			}
			if opts.IsUseable {
				db = db.Where("coupon_users.quantity > coupon_users.used")
			}
		}
		if opts.CoponUserID > 0 {
			db = db.Where("coupon_users.id = ?", opts.CoponUserID)
		}
	}

	if opts.Code != "" {
		db = db.Where("BINARY coupons.code = ?", opts.Code)
	}

	if opts.Status > 0 {
		db = db.Where("coupons.status = ?", opts.Status)
	}

	if opts.IsActiveNowTime {
		db = db.Where("coupons.start_date >= NOW()")
	}

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}
	if opts.Select != "" {
		db = db.Select(opts.Select)
	}

	if len(opts.Type) > 0 {
		db = db.Where("type IN (?)", opts.Type)
	}

	if opts.Order != "" {
		db = db.Order(opts.Order)
	} else {
		db = db.Order("id DESC")
	}

	if opts.IsShow {
		db = db.Where("is_show = ?", opts.IsShow)
	}

	return db
}

func (m *PackageManager) SaveCoupon(coupon *entity.Coupon) error {
	db := m.db.Save(&coupon)
	return db.Error
}

func (m *PackageManager) SaveUserCoupon(data []*entity.CouponUser) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		return err
	}

	for _, item := range data {
		if item.ID > 0 {
			if err := tx.Model(&entity.CouponUser{}).Where("id = ?", item.ID).Update("quantity", item.Quantity).Error; err != nil {
				tx.Rollback()
				return err
			}
		} else {
			if err := tx.Model(&entity.CouponUser{}).Create(item).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	return tx.Commit().Error
}

func (m *PackageManager) CreateCoupon(coupon *entity.Coupon) error {
	db := m.db.Model(&entity.Coupon{}).Create(coupon)
	return db.Error
}

func (m PackageManager) GetUserCoupons(opts CouponQueryOption, result interface{}) error {
	db := m.BuildCouponQuery(opts)

	db = db.Scan(result)
	return db.Error
}

func (m PackageManager) CountCoupons(opts CouponQueryOption) (int64, error) {
	db := m.BuildCouponQuery(opts)
	db = db.Model(&entity.Coupon{})
	var count int64
	db = db.Count(&count)
	return count, db.Error
}

func (m PackageManager) GetCoupon(opts CouponQueryOption) (*entity.Coupon, error) {
	var coupon *entity.Coupon
	db := m.BuildCouponQuery(opts)
	db = db.First(&coupon)
	return coupon, db.Error
}

func (m PackageManager) GetCouponUser(opts CouponQueryOption, result interface{}) error {
	db := m.BuildCouponQuery(opts)
	db = db.Scan(result)
	return db.Error
}

func (m PackageManager) GetCouponUsers(opts CouponQueryOption, result interface{}) error {
	db := m.db.Model(&entity.CouponUser{})
	db = db.Joins("INNER JOIN users ON users.id = coupon_users.customer_id")
	db = db.Select("users.email,users.full_name,coupon_users.id,coupon_users.customer_id,coupon_users.quantity,coupon_users.used")
	db = db.Where("coupon_id = ?", opts.ID)
	db = db.Scan(result)
	return db.Error
}

// 21/02/2025 this func wasnt be used, so balance_china wasnt be updated here. Update it when use
func (m PackageManager) UseCoupon(data interface{}, billID int64) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}
	var cpUser entity.CouponUser
	_ = utils.TypeConverter(data, &cpUser)

	cpUser.Used++
	if err := tx.Model(&entity.CouponUser{}).Where("id = ?", cpUser.ID).UpdateColumns(map[string]interface{}{
		"used":       cpUser.Used,
		"updated_at": time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	var coupon entity.Coupon
	_ = utils.TypeConverter(data, &coupon)
	extra := &entity.ExtraFee{
		CouponID:       utils.Int64(coupon.ID),
		BillID:         utils.Int64(billID),
		Amount:         -coupon.Value,
		Description:    fmt.Sprintf("Coupon %s", coupon.Code),
		ExtraFeeTypeID: constant.ExtraFeeTypeRefund,
		Status:         constant.ExtraFeeStatusEnable,
	}

	extra.CreatedAt = time.Now()
	extra.UpdatedAt = extra.CreatedAt

	if err := tx.Save(extra).Error; err != nil {
		tx.Rollback()
		return err
	}

	sqlString := `UPDATE bills SET extra_fee = extra_fee - ?, updated_at = ? WHERE id = ?`
	if err := tx.Exec(sqlString, math.Abs(extra.Amount), time.Now(), billID).Error; err != nil {
		tx.Rollback()
		return err
	}

	sqlString = `UPDATE users SET balance = balance + ?, updated_at = ? WHERE id = ?`
	if err := tx.Exec(sqlString, math.Abs(extra.Amount), time.Now(), cpUser.CustomerID).Error; err != nil {
		tx.Rollback()
		return err
	}

	info := &entity.UserInfo{}
	if err := tx.First(info).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			tx.Rollback()
			return err
		}

		now := time.Now()
		info = &entity.UserInfo{UserID: cpUser.CustomerID, UpdatedAt: &now, CancelMaxAmount: constant.DefaultCancelMaxAMount}
		if err := tx.Create(info).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	sqlString = `UPDATE user_infos SET debt_time = NULL WHERE user_id = ? AND debt_time IS NOT NULL AND (SELECT balance FROM users WHERE id = ? limit 1) >= 0`
	if err := tx.Exec(sqlString, cpUser.CustomerID, cpUser.CustomerID).Error; err != nil {
		tx.Rollback()
		return err
	}

	transaction := &entity.Transaction{
		UserID:      cpUser.CustomerID,
		Type:        constant.TransactionLogTypeRefund,
		Status:      constant.TransactionStatusSuccess,
		Amount:      math.Abs(extra.Amount),
		BillID:      extra.BillID,
		Description: fmt.Sprintf("Coupon %s", coupon.Code),
	}

	err := tx.Create(&transaction).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	transactionLog := &entity.TransactionLog{
		UserID:        cpUser.CustomerID,
		TransactionID: transaction.ID,
		Amount:        transaction.Amount,
		Type:          transaction.Type,
		Status:        transaction.Status,
		BillID:        transaction.BillID,
		Description:   transaction.Description,
	}

	if err = tx.Model(entity.TransactionLog{}).Create(transactionLog).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m PackageManager) GetCoupons(opts CouponQueryOption) ([]entity.Coupon, error) {
	db := m.BuildCouponQuery(opts)
	coupons := []entity.Coupon{}
	if opts.LoadUserCoupon {
		db = db.Preload("Users")
	}
	db = db.Find(&coupons)
	return coupons, db.Error
}

func (m PackageManager) BuyCoupon(userID int64, coupon *entity.Coupon, quantity int) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
			tx.Rollback()
		}
	}()

	var user entity.User
	ucp := entity.CouponUser{
		CouponID:   coupon.ID,
		CustomerID: userID,
	}
	if err := tx.Model(&entity.User{}).Where("id = ?", userID).First(&user).Error; err != nil {
		tx.Rollback()
		return err
	}
	if user.Point < coupon.Point*quantity {
		tx.Rollback()
		return errors.New("missing point to buy coupon")
	}
	if err := tx.Model(&entity.CouponUser{}).Where("customer_id = ? AND coupon_id = ?", userID, coupon.ID).FirstOrCreate(&ucp).Error; err != nil {
		tx.Rollback()
		return err
	}
	var sql string
	sql = "UPDATE coupon_users SET quantity = quantity + ? WHERE id = ?"
	if err := tx.Exec(sql, quantity, ucp.ID).Error; err != nil {
		tx.Rollback()
		return err
	}
	sql = "UPDATE users SET point = point - ? WHERE id = ?"
	if err := tx.Exec(sql, coupon.Point*quantity, user.ID).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(entity.PointLog{}).Create(&entity.PointLog{
		Point:       -coupon.Point * quantity,
		UserID:      user.ID,
		Description: fmt.Sprintf("Trừ điểm khi mua coupon %s", coupon.Code),
		Status:      constant.StatusActive,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m *PackageManager) GetRelabelPackages(result interface{}) error {
	sql := `SELECT rs.id as package_id,rs.user_id,users.email,package_codes.code,trackings.tracking_number FROM (SELECT
					packages.id,
					packages.user_id,
					count(trackings.id),
					packages.package_code_id
				FROM
					packages
				INNER JOIN trackings ON trackings.package_id = packages.id
			WHERE package_id IN (SELECT
								package_id
							FROM
								trackings
							WHERE
								created_at >= now() - INTERVAL 1 DAY AND status = ?)
			GROUP BY package_id
			HAVING count(trackings.id) > 1) as rs
			INNER JOIN package_codes ON package_codes.id  = rs.package_code_id
			INNER JOIN trackings ON trackings.package_id = rs.id AND trackings.status = ?
			INNER JOIN users ON users.id = rs.user_id
			ORDER BY rs.user_id DESC`
	db := m.db.Raw(sql, constant.TrackingStatusSuccess, constant.TrackingStatusSuccess).Scan(result)
	return db.Error
}

func (m *PackageManager) GetReturnPackages(result interface{}) error {
	sql := `SELECT
				packages.id,
				users.id as user_id,
				code,
				users.email,
				trackings.tracking_number
			FROM
				packages
			INNER JOIN users ON
				users.id = packages.user_id
			LEFT JOIN trackings ON trackings.package_id  = packages.id AND trackings.status = 2
			LEFT JOIN package_codes ON packages.package_code_id = package_codes.id 
			WHERE
				returned_at >= now() - INTERVAL 1 DAY
			ORDER BY user_id DESC`
	db := m.db.Raw(sql).Scan(result)
	return db.Error
}

func (m *PackageManager) CountDeliveryPackage(opts PackageQueryOption, result interface{}) error {
	db := m.BuildPackageQuery(opts)
	db = db.Where("delivered_at IS NOT NULL")
	db = db.Model(&entity.Package{})
	db = db.Select("IF(DATEDIFF(delivered_at,IFNULL(checkin_warehouse_at,created_at)) < 1,concat(HOUR(TIMEDIFF(delivered_at,IFNULL(checkin_warehouse_at,created_at))),' H'),concat(DATEDIFF(delivered_at,IFNULL(checkin_warehouse_at,created_at)),' D')) as time_diff,COUNT(packages.id) as count")
	db = db.Group("time_diff")
	db = db.Scan(result)
	return db.Error
}
