package dto

import (
	"fmt"
	"strings"
	"tebexpressapi/pkg/constant"
	"time"
)

type CountStatusPackage struct {
	Status int   `json:"status"`
	Count  int64 `json:"count"`
}

type UserPackageStatus struct {
	UserID int64 `json:"user_id"`
	Type   int   `json:"type"`
	Count  int64 `json:"count"`
}

type CountStatusStringPackage struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type PackageAuditLogDTO struct {
	ID              int64   `json:"id"`
	PackageID       int64   `json:"package_id"`
	OrderNumber     string  `json:"order_number"`
	Code            string  `json:"code"`
	OldValue        string  `json:"old_value"`
	Value           string  `json:"value"`
	UpdatedUserID   int64   `json:"updated_user_id"`
	Type            int     `json:"type"`
	UpdatedUserName string  `json:"updated_user_name"`
	Fee             float64 `json:"fee"`
	ExtraFee        float64 `json:"extra_fee"`
	ExtraFeeStatus  int64   `json:"extra_fee_status"`
	UpdatedUserRole string  `json:"updated_user_role"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MissingPackageDTO struct {
	ID int64 `json:"id"`
}
type PackageDetailDTO struct {
	ID                  int64                `json:"id"`
	OrderNumber         string               `json:"order_number"`
	Label               string               `json:"label"`
	Recipient           string               `json:"recipient"`
	Company             string               `json:"company"`
	PhoneNumber         string               `json:"phone_number"`
	Address1            string               `json:"address_1" `
	Address2            string               `json:"address_2"`
	City                string               `json:"city"`
	StateCode           string               `json:"state_code"`
	Zipcode             string               `json:"zipcode"`
	CountryCode         string               `json:"country_code"`
	Detail              string               `json:"detail"`
	Weight              float64              `json:"weight"`
	Width               float64              `json:"width"`
	Length              float64              `json:"length"`
	Height              float64              `json:"height"`
	ActualWeight        float64              `json:"actual_weight"`
	ActualWidth         float64              `json:"actual_width"`
	ActualLength        float64              `json:"actual_length"`
	ActualHeight        float64              `json:"actual_height"`
	Status              int                  `json:"status,omitempty"`
	StatusString        string               `json:"status_string"`
	ServiceID           int64                `json:"service_id"`
	Note                string               `json:"note"`
	ServiceName         string               `json:"service_name"`
	ServiceCode         string               `json:"service_code"`
	CNProductLink       string               `json:"cn_product_link"`
	CNProductPrice      float64              `json:"cn_product_price"`
	CNShippingFee       float64              `json:"cn_shipping_fee"`
	CustomCNBarcode     string               `json:"custom_cn_barcode"`
	TrackingNumber      string               `json:"tracking_number"`
	CodePackage         string               `json:"code_package"`
	ShippingFee         float64              `json:"shipping_fee"`
	CreatedAt           time.Time            `json:"created_at"`
	Alert               int64                `json:"alert"`
	IsInsured           bool                 `json:"is_insured"`
	EstimateDateProcess *time.Time           `json:"estimate_date_process"`
	IsPackageExceed     bool                 `json:"is_package_exceed"`
	IncludeBattery      bool                 `json:"include_battery"`
	PackageProducts     []PackageProductsDTO `json:"package_products"`
	EstimateDelivery    float64              `json:"estimate_delivery"`
	IsBookmark          bool                 `json:"is_bookmark"`
}

type PackageProductsDTO struct {
	ID        int64     `json:"id"`
	PackageID int64     `json:"package_id"`
	ProductID int64     `json:"product_id"`
	Name      string    `json:"name"`
	SKU       string    `json:"sku"`
	Status    int       `json:"status"`
	Quantity  int64     `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PackageDeliverLogDTO struct {
	ID              int64  `json:"id"`
	PackageID       int64  `json:"package_id"`
	Location        string `json:"location"`
	Description     string `json:"description"`
	Status          string `json:"status"`
	Type            int    `json:"type"`
	Code            string `json:"code"`
	UpdatedUserRole string `json:"updated_user_role,omitempty"`
	UpdatedUserName string `json:"updated_user_name,omitempty"`

	ShipTime  time.Time `json:"ship_time"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TrackDeDeliverDTO struct {
	Location    string    `json:"location"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	ShipTime    time.Time `json:"ship_time"`
}

type TrackingDTO struct {
	ID                 int64               `json:"id"`
	Tracking           *TrackingInfoDTO    `json:"tracking"`
	CountryCode        string              `json:"country_code"`
	PackageCode        *PackageCodeInfoDTO `json:"package_code"`
	StatusString       string              `gorm:"-" json:"status_string"`
	DeliveredAt        *time.Time          `json:"delivered_at"`
	CheckinWarehouseAt *time.Time          `json:"checkin_warehouse_at"`
	Alert              int64               `json:"alert"`
}

type TrackingInfoDTO struct {
	PackageID      int64  `json:"package_id"`
	TrackingNumber string `json:"tracking_number"`
}

type PackageCodeInfoDTO struct {
	Code string `json:"code"`
}

type PartnerEvent struct {
	Time     string `json:"time"`
	Location string `json:"location"`
	Content  string `json:"content"`
}

func Transfomer(logs []PackageDeliverLogDTO) []PackageDeliverLogDTO {
	var nLogs []PackageDeliverLogDTO

	skipLogExportWarehouse := false

	n := len(logs)
	for i := n - 1; i >= 0; i-- {
		log := logs[i]

		if strings.ToLower(log.Description) == "received data" || log.Type == constant.PackageStatusCreated || log.Type == constant.PackageDeliverLogTypeExportHub || log.Type == constant.PackageDeliverLogTypeImportHub {
			continue
		}

		if skipLogExportWarehouse && (log.Location == "Hanoi, VN" || log.Type <= constant.PackageStatusWareHouseExport) {
			continue
		}

		if log.Type > constant.PackageStatusWareHouseExport && log.Location != "Hanoi, VN" {
			skipLogExportWarehouse = true
		}

		if log.Type == constant.PackageStatusCancelled {
			log.Description = constant.MapStatusDescriptionUS[log.Type]
		}

		if log.Type == constant.PackageStatusReturned {
			log.Description = constant.MapStatusDescriptionUS[log.Type]
		}

		if log.Type == constant.PackageDeliverLogTypeReship {
			log.Description = constant.MapStatusDescriptionUS[log.Type]
		}

		if log.Type > 0 && log.Description == "" {
			if constant.MapStatusDescriptionUS[log.Type] != "" {
				log.Description = constant.MapStatusDescriptionUS[log.Type]
			}
		}

		if log.Description == "Accepted at USPS Origin Facility" {
			log.Description = "USPS‘s pickup awaiting"
		}

		log.UpdatedUserName = ""
		log.UpdatedUserRole = ""
		nLogs = append(nLogs, log)
	}

	for i, j := 0, len(nLogs)-1; i < j; i, j = i+1, j-1 {
		nLogs[i], nLogs[j] = nLogs[j], nLogs[i]
	}

	return nLogs
}

func Transfomer2(logs []PackageDeliverLogDTO) []PackageDeliverLogDTO {
	var nLogs []PackageDeliverLogDTO

	skipLogExportWarehouse := false

	n := len(logs)
	for i := n - 1; i >= 0; i-- {
		log := logs[i]

		if strings.ToLower(log.Description) == "received data" || log.Type == constant.PackageStatusCreated || log.Type == constant.PackageDeliverLogTypeExportHub || log.Type == constant.PackageDeliverLogTypeImportHub {
			continue
		}

		if skipLogExportWarehouse && (log.Location == "Hanoi, VN" || log.Type <= constant.PackageStatusWareHouseExport) {
			continue
		}

		if log.Type > constant.PackageStatusWareHouseExport && log.Location != "Hanoi, VN" {
			skipLogExportWarehouse = true
		}

		if log.Type == constant.PackageStatusCancelled {
			if log.UpdatedUserRole == constant.UserRoleCustomer {
				log.Description = fmt.Sprintf("%s by %v", constant.MapStatusDescriptionUS[log.Type], log.UpdatedUserName)
			} else {
				log.Description = constant.MapStatusDescriptionUS[log.Type]
			}
		}

		if log.Type == constant.PackageStatusArchived {
			if log.UpdatedUserRole == constant.UserRoleCustomer {
				log.Description = fmt.Sprintf("%s by %v", constant.MapStatusDescriptionUS[log.Type], log.UpdatedUserName)
			} else {
				log.Description = constant.MapStatusDescriptionUS[log.Type]
			}
		}

		if log.Description == "Accepted at USPS Origin Facility" {
			log.Description = "USPS‘s pickup awaiting"
		}

		if log.Type == constant.PackageStatusReturned {
			log.Description = fmt.Sprintf("%s, reason: %s", constant.MapStatusDescriptionUS[log.Type], log.Description)
		}

		if log.Type == constant.PackageDeliverLogTypeReship {
			log.Description = constant.MapStatusDescriptionUS[log.Type]
		}

		if log.Type > 0 && log.Description == "" {
			if constant.MapStatusDescriptionUS[log.Type] != "" {
				log.Description = constant.MapStatusDescriptionUS[log.Type]
			}
		}

		log.UpdatedUserName = ""
		log.UpdatedUserRole = ""
		nLogs = append(nLogs, log)
	}

	for i, j := 0, len(nLogs)-1; i < j; i, j = i+1, j-1 {
		nLogs[i], nLogs[j] = nLogs[j], nLogs[i]
	}

	return nLogs
}

func Transfomer3(logs []PackageDeliverLogDTO) []PackageDeliverLogDTO {
	var nLogs []PackageDeliverLogDTO
	skipLogExportWarehouse := false

	n := len(logs)
	for i := n - 1; i >= 0; i-- {
		log := logs[i]

		if strings.ToLower(log.Description) == "received data" || log.Type == constant.PackageStatusCreated {
			continue
		}

		if skipLogExportWarehouse && (log.Location == "Hanoi, VN" || log.Type <= constant.PackageStatusWareHouseExport) {
			continue
		}

		if log.Type > constant.PackageStatusWareHouseExport && log.Location != "Hanoi, VN" {
			skipLogExportWarehouse = true
		}

		if log.Type == constant.PackageStatusCancelled {
			userName := log.UpdatedUserName
			if userName == "" {
				userName = "Ananbay"
			}

			if log.UpdatedUserRole == constant.UserRoleCustomer {
				log.Description = fmt.Sprintf("%s by %v (KH)", constant.MapStatusDescriptionUS[log.Type], userName)
			} else if log.UpdatedUserRole == constant.UserRoleSupport || log.UpdatedUserRole == constant.UserRoleSale {
				log.Description = fmt.Sprintf("%s by %v (CS)", constant.MapStatusDescriptionUS[log.Type], userName)
			} else {
				log.Description = fmt.Sprintf("%s by %v", constant.MapStatusDescriptionUS[log.Type], userName)
			}
		}

		if log.Type == constant.PackageStatusArchived {
			userName := log.UpdatedUserName
			if userName == "" {
				userName = "Ananbay"
			}

			if log.UpdatedUserRole == constant.UserRoleCustomer {
				log.Description = fmt.Sprintf("%s by %v (KH)", constant.MapStatusDescriptionUS[log.Type], userName)
			} else if log.UpdatedUserRole == constant.UserRoleSupport || log.UpdatedUserRole == constant.UserRoleSale {
				log.Description = fmt.Sprintf("%s by %v (CS)", constant.MapStatusDescriptionUS[log.Type], userName)
			} else {
				log.Description = fmt.Sprintf("%s by %v", constant.MapStatusDescriptionUS[log.Type], userName)
			}
		}

		if log.Description == "Accepted at USPS Origin Facility" {
			log.Description = "USPS‘s pickup awaiting"
		}

		if log.Type == constant.PackageDeliverLogTypeReship {
			log.Description = fmt.Sprintf("%s by %s", constant.MapStatusDescriptionUS[log.Type], log.UpdatedUserName)
		}

		if log.Type == constant.PackageStatusReturned {
			log.Description = fmt.Sprintf("%s, reason: %s", constant.MapStatusDescriptionUS[log.Type], log.Description)
		}

		if log.Type > 0 && log.Description == "" {
			if constant.MapStatusDescriptionUS[log.Type] != "" {
				log.Description = constant.MapStatusDescriptionUS[log.Type]
			}
		}

		if log.Type == constant.PackageDeliverLogTypeExportHub {

			log.Description = fmt.Sprintf("%s by %s", constant.MapStatusDescriptionUS[log.Type], log.UpdatedUserName)
			fmt.Println(log.Description)
		}

		if log.Type == constant.PackageDeliverLogTypeImportHub {
			log.Description = fmt.Sprintf("%s by %s", constant.MapStatusDescriptionUS[log.Type], log.UpdatedUserName)
		}

		log.UpdatedUserName = ""
		log.UpdatedUserRole = ""
		nLogs = append(nLogs, log)
	}

	for i, j := 0, len(nLogs)-1; i < j; i, j = i+1, j-1 {
		nLogs[i], nLogs[j] = nLogs[j], nLogs[i]
	}

	return nLogs
}

func PartnerTransfomer(logs []PackageDeliverLogDTO) []PartnerEvent {
	var events []PartnerEvent

	skipLogExportWarehouse := false

	n := len(logs)
	for i := n - 1; i >= 0; i-- {
		log := logs[i]
		if strings.ToLower(log.Description) == "received data" ||
			log.Type == constant.PackageStatusCreated ||
			log.Type == constant.PackageDeliverLogTypeExportHub ||
			log.Type == constant.PackageDeliverLogTypeImportHub {
			continue
		}

		if skipLogExportWarehouse && (log.Location == "Hanoi, VN" || log.Type <= constant.PackageStatusWareHouseExport) {
			continue
		}

		if log.Type > constant.PackageStatusWareHouseExport && log.Location != "Hanoi, VN" {
			skipLogExportWarehouse = true
		}

		if log.Type == constant.PackageStatusCancelled {
			log.Description = constant.MapStatusDescriptionUS[log.Type]
		}

		if log.Type == constant.PackageStatusReturned {
			log.Description = constant.MapStatusDescriptionUS[log.Type]
		}

		if log.Type == constant.PackageDeliverLogTypeReship {
			log.Description = constant.MapStatusDescriptionUS[log.Type]
		}

		if log.Type > 0 && log.Description == "" {
			if constant.MapStatusDescriptionUS[log.Type] != "" {
				log.Description = constant.MapStatusDescriptionUS[log.Type]
			}
		}

		if log.Description == "Accepted at USPS Origin Facility" {
			log.Description = "USPS‘s pickup awaiting"
		}

		t := log.ShipTime.Add(7 * time.Hour).Format("2006-01-02 15:04:05")
		events = append(events, PartnerEvent{
			Time:     t,
			Content:  log.Description,
			Location: log.Location,
		})
	}

	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}

	return events
}
