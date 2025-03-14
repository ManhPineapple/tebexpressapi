package entity

import (
	"tebexpressapi/pkg/utils/dbgorm"
	"time"
)

// Package
type Package struct {
	dbgorm.Model
	OrderNumber        string     `json:"order_number"`
	Label              string     `json:"label"`
	Recipient          string     `json:"recipient"`
	Company            string     `json:"company"`
	PhoneNumber        string     `json:"phone_number"`
	Address1           string     `json:"address_1" gorm:"column:address_1"`
	Address2           string     `json:"address_2" gorm:"column:address_2"`
	City               string     `json:"city"`
	StateCode          string     `json:"state_code"`
	Zipcode            string     `json:"zipcode"`
	CountryCode        string     `json:"country_code"`
	Detail             string     `json:"detail"`
	Weight             float64    `json:"weight"`
	Width              float64    `json:"width"`
	Length             float64    `json:"length"`
	Height             float64    `json:"height"`
	ActualWeight       float64    `json:"actual_weight"`
	ActualWidth        float64    `json:"actual_width"`
	ActualLength       float64    `json:"actual_length"`
	ActualHeight       float64    `json:"actual_height"`
	Status             int        `json:"status,omitempty"`
	StatusString       string     `gorm:"-" json:"status_string"`
	UserID             int64      `json:"user_id"`
	ServiceID          int64      `json:"service_id"`
	Note               string     `json:"note"`
	ShippingFee        float64    `json:"shipping_fee"`
	BillID             int64      `json:"bill_id" gorm:"default:NULL"`
	ValidateAddress    int        `json:"validate_address"`
	DeliveredAt        *time.Time `json:"delivered_at"`
	TrackingTime       *time.Time `json:"tracking_time"`
	PackageCodeID      *int64     `json:"package_code_id"`
	CheckinWarehouseAt *time.Time `json:"checkin_warehouse_at"`
	Alert              int64      `json:"alert" gorm:"default:0"`
	HubExportedAt      *time.Time `json:"hub_exported_at"`
	HubImportedAt      *time.Time `json:"hub_imported_at"`
	WarehouseID        int64      `json:"warehouse_id"`
	HubID              *int64     `json:"hub_id"`
	UserIDImported     int64      `json:"user_id_imported"`
	ReturnedAt         *time.Time `json:"returned_at"`
	ReshipAt           *time.Time `json:"reship_at"`
	RequestReship      bool       `json:"request_reship"`
	LabelPromotion     bool       `json:"label_promotion"`
	AlertAt            *time.Time `json:"alert_at"`
	IsPackageExceed    bool       `json:"is_package_exceed"`
	OrderID            *int64     `json:"order_id"`
	CustomerShipmentID *int64     `json:"customer_shipment_id"`
	IncludeBattery     bool       `json:"include_battery"`
	IsInsured          bool       `json:"-"`
	IsBookmark         bool       `json:"is_bookmark"`
	PartnerID          int64      `json:"partner_id"`

	PackageCode      *PackageCode      `json:"package_code"`
	ContainerItem    *ContainerItem    `json:"container_item"`
	User             *User             `json:"user" gorm:"save_associations:false"`
	Tracking         *Tracking         `json:"tracking"`
	ExtraFee         []ExtraFee        `json:"extra_fee"`
	Service          *Service          `json:"service" gorm:"save_associations:false"`
	PackageReturn    *PackageReturn    `json:"package_return"`
	CustomerShipment *CustomerShipment `json:"customer_shipment"`
	Warehouse        *Warehouse        `json:"warehouse" gorm:"foreignKey:WarehouseID;"`
	EstimateDelivery float64           `json:"estimate_delivery" gorm:"-"`
	PackageRefunds   []PackageRefund   `json:"package_refunds"`

	// For Tiktok and early scan
	CustomTiktokBarcode *string `json:"custom_tiktok_barcode" gorm:"default:NULL"`
	IsEarlyScan         bool    `json:"is_early_scan,omitempty"`
	// For customs (Hải quan) check
	PackageName       string  `json:"package_name"`
	PackageQuantity   int64   `json:"package_quantity"`
	TotalProductPrice float64 `json:"product_price"`

	CNIsPurchased   bool    `json:"cn_is_purchased"`
	CNProductLink   string  `json:"cn_product_link"`
	CNProductPrice  float64 `json:"cn_product_price"`
	CNInvoiceImage  string  `json:"cn_invoice_image"`
	CNShippingFee   float64 `json:"cn_shipping_fee"`
	CustomCNBarcode *string `json:"custom_cn_barcode" gorm:"uniqueIndex"`
}

type PackageCode struct {
	dbgorm.Model
	UserID            int64  `json:"user_id"`
	Code              string `json:"code"`
	PackageIDGenerate int64  `json:"package_id_generate"`
	ServiceID         int64  `json:"service_id"`
	Status            int    `json:"status"`
}
type PackageDeliverLog struct {
	dbgorm.Model
	PackageID   int64  `json:"package_id"`
	Location    string `json:"location"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Type        int    `json:"type"`
	UserID      *int64 `json:"user_id" gorm:"default:NULL"`
	User        *User  `json:"user" gorm:"foreignKey:UserID;"`
}

type PackageAuditLog struct {
	dbgorm.Model
	PackageID     int64   `json:"package_id"`
	OldValue      string  `json:"old_value"`
	Value         string  `json:"value"`
	UpdatedUserID int64   `json:"updated_user_id" gorm:"default:NULL"`
	Type          int     `json:"type"`
	Fee           float64 `json:"fee"`
	Description   string  `json:"description"`
	ExtraFeeID    int64   `json:"extra_fee_id" gorm:"default:NULL"`
}

type ExtraFee struct {
	dbgorm.Model
	PackageID          *int64            `json:"package_id"`
	CouponID           *int64            `json:"coupon_id"`
	CustomerShipmentID *int64            `json:"customer_shipment_id"`
	BillID             *int64            `json:"bill_id"`
	ExtraFeeTypeID     int64             `json:"extra_fee_type_id"`
	Amount             float64           `json:"amount"`
	Description        string            `json:"description"`
	Status             int               `json:"status"`
	Package            *Package          `json:"package" gorm:"save_associations:false"`
	CustomerShipment   *CustomerShipment `json:"CustomerShipment,omitempty" gorm:"save_associations:false"`
	ExtraFeeType       *ExtraFeeType     `json:"extra_fee_types" gorm:"save_associations:false"`
	Coupon             *Coupon           `json:"coupon"`
}

type ExtraFeeType struct {
	dbgorm.Model
	Name     string  `json:"name"`
	Status   int     `json:"status"`
	ParentID int64   `json:"parent_id"`
	IsRefund bool    `json:"is_refund"`
	Fee      float64 `json:"Fee"`
	IsShow   bool    `json:"is_show"`
}

func (PackageAuditLog) TableName() string {
	return "package_audit_logs"
}

type PackageCustomer struct {
	ID                int64                   `json:"id,omitempty"`
	Label             string                  `json:"label,omitempty"`
	CreatedAt         time.Time               `json:"created_at"`
	UpdatedAt         time.Time               `json:"updated_at"`
	Code              string                  `json:"code" gorm:"default:NULL"`
	PCodeStatus       int64                   `json:"p_code_status"`
	OrderNumber       string                  `json:"order_number"`
	Recipient         string                  `json:"recipient"`
	Company           string                  `json:"company"`
	PhoneNumber       string                  `json:"phone_number"`
	Address1          string                  `json:"address_1" gorm:"column:address_1"`
	Address2          string                  `json:"address_2" gorm:"column:address_2"`
	City              string                  `json:"city"`
	StateCode         string                  `json:"state_code"`
	Zipcode           string                  `json:"zipcode"`
	CountryCode       string                  `json:"country_code"`
	Detail            string                  `json:"detail"`
	Weight            float64                 `json:"weight"`
	Width             float64                 `json:"width"`
	Length            float64                 `json:"length"`
	Height            float64                 `json:"height"`
	Status            int                     `json:"status"`
	UserID            int64                   `json:"user_id"`
	UserFullName      string                  `json:"user_full_name"`
	UserEmail         string                  `json:"user_email"`
	UserPhoneNumber   string                  `json:"user_phone_number"`
	ShippingFee       float64                 `json:"shipping_fee"`
	BillCode          string                  `json:"bill_code"`
	ServiceCode       string                  `json:"service_code"`
	CustomCNBarcode   string                  `json:"custom_cn_barcode"`
	IsPackageExceed   bool                    `json:"is_package_exceed"`
	OrderID           *int64                  `json:"order_id"`
	IncludeBattery    bool                    `json:"include_battery"`
	ExtraFees         []ExtraFeeCustom        `json:"extra_fees" gorm:"foreignKey:PackageID;"`
	Tracking          TrackingCustom          `json:"tracking" gorm:"foreignKey:PackageID;references:ID"`
	PackageProducts   []PackageProductsCustom `json:"package_products" gorm:"foreignKey:PackageID"`
	PackageName       string                  `json:"package_name"`
	PackageQuantity   int64                   `json:"package_quantity"`
	TotalProductPrice float64                 `json:"total_product_price"`
}

type PackageProductsCustom struct {
	PackageID int64  `json:"-"`
	Name      string `json:"name"`
	SKU       string `json:"sku"`
	Quantity  int64  `json:"quantity"`
}

type ExtraFeeCustom struct {
	PackageID    int64   `json:"-"`
	ExtraFeeType string  `json:"extra_fee_type"`
	Amount       float64 `json:"amount"`
	Description  string  `json:"description"`
}
type TrackingCustom struct {
	PackageID       int64  `json:"-"`
	TrackingNumber  string `json:"tracking_number"`
	LabelURL        string `json:"label_url"`
	CarrierID       int64  `json:"-"`
	LastMileCarrier string `json:"last_mile_carrier"`
}

type PackageInWarehouse struct {
	ID                      int64      `json:"id,omitempty"`
	Code                    string     `json:"code" gorm:"default:NULL"`
	OrderNumber             string     `json:"order_number"`
	Recipient               string     `json:"recipient"`
	Company                 string     `json:"company"`
	PhoneNumber             string     `json:"phone_number"`
	Address1                string     `json:"address_1" gorm:"column:address_1"`
	Address2                string     `json:"address_2" gorm:"column:address_2"`
	City                    string     `json:"city"`
	StateCode               string     `json:"state_code"`
	Zipcode                 string     `json:"zipcode"`
	ServiceCode             string     `json:"service_code"`
	ServiceName             string     `json:"service_name"`
	Label                   string     `json:"label"`
	Tracking                []Tracking `json:"tracking" gorm:"foreignKey:PackageID"`
	ContainerID             int64      `json:"container_id"`
	ContainerCode           string     `json:"container_code"`
	ContainerLabelUrl       string     `json:"container_label_url"`
	ContainerTrackingNumber string     `json:"container_tracking_number"`
	ShipmentID              int64      `json:"shipment_id"`
	CheckinWarehouseAt      *time.Time `json:"checkin_warehouse_at"`
	Status                  int64      `json:"status"`
}

type QueueRefundPackage struct {
	PackageID   int64  `json:"package_id"`
	UserID      int64  `json:"user_id"`
	Description string `json:"description"`
}

type PackageReturn struct {
	dbgorm.Model
	PackageID int64              `json:"package_id"`
	Reason    string             `json:"reason"`
	Content   string             `json:"content"`
	Images    dbgorm.SliceString `json:"images"`
}

type PackageRefund struct {
	dbgorm.Model
	PackageID int64    `json:"package_id"`
	Amount    float64  `json:"amount"`
	Status    int      `json:"status"`
	Package   *Package `json:"package" gorm:"save_associations:false"`
}

type PackageRefundDTO struct {
	dbgorm.Model
	PackageID   int64   `json:"package_id"`
	Amount      float64 `json:"amount"`
	Status      int     `json:"status"`
	OrderNumber string  `json:"order_number"`
}

type QueuePackageCancelCarrier struct {
	TrackingNumber string `json:"tracking_number"`
}

type PackageExport struct {
	dbgorm.Model
	BillCode          int64   `json:"bill_code"`
	BillCreate        string  `json:"bill_create"`
	OrderNumber       string  `json:"order_number"`
	PackageCode       string  `json:"package_code"`
	PkgUpdatedAt      string  `json:"pkg_updated_at"`
	FullName          string  `json:"full_name"`
	Username          string  `json:"username"`
	Class             int64   `json:"class"`
	CreatedAt         string  `json:"created_at"`
	TrackingNumber    string  `json:"tracking_number"`
	ServiceName       string  `json:"service_name"`
	Address           string  `json:"address"`
	City              string  `json:"city"`
	StateCode         string  `json:"state_code"`
	Zipcode           string  `json:"zipcode"`
	CountryCode       string  `json:"country_code"`
	Detail            string  `json:"detail"`
	ShippingFee       float64 `json:"shipping_fee"`
	ExtraFee          float64 `json:"extra_fee"`
	TotalFee          float64 `json:"total_fee"`
	ExtraFeeType      string  `json:"extra_fee_type"`
	ExtraFeeCreatedAt string  `json:"extra_fee_created_at"`
	Description       string  `json:"description"`
	Weight            float64 `json:"weight"`
	Length            float64 `json:"length"`
	Width             float64 `json:"width"`
	Height            float64 `json:"height"`
	WeightReal        float64 `json:"weight_real"`
	LengthReal        float64 `json:"length_real"`
	WidthReal         float64 `json:"width_real"`
	HeightReal        float64 `json:"height_real"`
	CarrierName       string  `json:"carrier_name"`
	CarrierService    string  `json:"carrier_service"`
	Warehouse         string  `json:"warehouse"`
	ShipmentCost      float64 `json:"shipment_cost"`
	HandlingFee       float64 `json:"handling_fee"`
	Status            int     `json:"status"`
	DeliveredAt       string  `json:"delivered_at"`
	Rate              float64 `json:"rate"`
}

type Coupon struct {
	dbgorm.Model
	Code      string       `json:"code"`
	Point     int          `json:"point"`
	Type      int          `json:"type"`
	MinApply  float64      `json:"min_apply"`
	MaxApply  float64      `json:"max_apply"`
	StartDate time.Time    `json:"start_date"`
	EndDate   time.Time    `json:"end_date"`
	Value     float64      `json:"value"`
	UserID    int64        `json:"user_id"`
	Status    int          `json:"status"`
	IsShow    bool         `json:"is_show"`
	Users     []CouponUser `json:"users" gorm:"foreignKey:CouponID"`
}

type CouponUser struct {
	dbgorm.Model
	CustomerID int64   `json:"customer_id"`
	Quantity   int     `json:"quantity"`
	Used       int     `json:"used"`
	CouponID   int64   `json:"coupon_id"`
	Coupon     *Coupon `json:"coupon" gorm:"foreignKey:CouponID"`
}
