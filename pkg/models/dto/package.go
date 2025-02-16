package dto

import (
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/dbgorm"
	"time"
)

type PackageCustomer struct {
	ID              int64                          `json:"id,omitempty"`
	Label           string                         `json:"label,omitempty"`
	CreatedAt       time.Time                      `json:"created_at"`
	UpdatedAt       time.Time                      `json:"updated_at"`
	Code            string                         `json:"code" gorm:"default:NULL"`
	OrderNumber     string                         `json:"order_number"`
	Recipient       string                         `json:"recipient"`
	Company         string                         `json:"company"`
	PhoneNumber     string                         `json:"phone_number"`
	Address1        string                         `json:"address_1" gorm:"column:address_1"`
	Address2        string                         `json:"address_2" gorm:"column:address_2"`
	City            string                         `json:"city"`
	StateCode       string                         `json:"state_code"`
	Zipcode         string                         `json:"zipcode"`
	CountryCode     string                         `json:"country_code"`
	Detail          string                         `json:"detail"`
	Weight          float64                        `json:"weight"`
	Width           float64                        `json:"width"`
	Length          float64                        `json:"length"`
	Height          float64                        `json:"height"`
	Status          string                         `json:"status"`
	UserID          int64                          `json:"user_id"`
	UserFullName    string                         `json:"user_full_name"`
	UserEmail       string                         `json:"user_email"`
	UserPhoneNumber string                         `json:"user_phone_number"`
	ShippingFee     float64                        `json:"shipping_fee,omitempty"`
	BillCode        string                         `json:"bill_code"`
	ServiceCode     string                         `json:"service_code"`
	TotalCost       float64                        `json:"total_cost,omitempty"`
	IncludeBattery  bool                           `json:"include_battery"`
	IsPackageExceed bool                           `json:"-"`
	OrderID         int64                          `json:"order_id"`
	ExtraFees       []entity.ExtraFeeCustom        `json:"extra_fees" gorm:"foreignKey:PackageID;"`
	Tracking        entity.TrackingCustom          `json:"tracking" gorm:"foreignKey:PackageID;references:ID"`
	PackageProducts []entity.PackageProductsCustom `json:"package_products" gorm:"foreignKey:PackageID"`
}

type PackageContainerDTO struct {
	entity.Package
	ItemStatus  int64  `json:"item_status"`
	Description string `json:"description"`
}

type OverPretransitPackageDTO struct {
	entity.Package
	DayLeft string `json:"day_left"`
}

func TransformPackageCustomer(in entity.PackageCustomer) PackageCustomer {
	var total float64 = 0
	for i, v := range in.ExtraFees {
		if v.Description == "" {
			in.ExtraFees[i].Description = in.ExtraFees[i].ExtraFeeType
		}

		total += v.Amount
	}

	if in.Status == constant.PackageStatusArchived || in.PCodeStatus == constant.PackageCodeTemp {
		in.Code = ""
	}

	if in.Status == constant.PackageStatusCreated {
		in.Tracking = entity.TrackingCustom{}
	}

	return PackageCustomer{
		ID:              in.ID,
		Label:           in.Label,
		CreatedAt:       in.CreatedAt,
		UpdatedAt:       in.UpdatedAt,
		Code:            in.Code,
		OrderNumber:     in.OrderNumber,
		Recipient:       in.Recipient,
		Company:         in.Company,
		PhoneNumber:     in.PhoneNumber,
		Address1:        in.Address1,
		Address2:        in.Address2,
		City:            in.City,
		StateCode:       in.StateCode,
		Zipcode:         in.Zipcode,
		CountryCode:     in.CountryCode,
		Detail:          in.Detail,
		Weight:          in.Weight,
		Width:           in.Width,
		Length:          in.Length,
		Height:          in.Height,
		Status:          constant.MapTextStatusCustomerPackage[in.Status],
		UserID:          in.UserID,
		UserFullName:    in.UserFullName,
		UserEmail:       in.UserEmail,
		UserPhoneNumber: in.UserPhoneNumber,
		ShippingFee:     in.ShippingFee,
		BillCode:        in.BillCode,
		ServiceCode:     in.ServiceCode,
		ExtraFees:       in.ExtraFees,
		Tracking:        in.Tracking,
		PackageProducts: in.PackageProducts,
		IsPackageExceed: in.IsPackageExceed,
		TotalCost:       utils.Ceil(in.ShippingFee+total, 2),
		IncludeBattery:  in.IncludeBattery,
		OrderID:         utils.Int64Value(in.OrderID),
	}
}

type InfoPackageReturnDTO struct {
	PackageID       int64      `json:"package_id"`
	Alert           int        `json:"alert"`
	OrderNumber     string     `json:"order_number"`
	Label           string     `json:"label"`
	TrackingNumber  string     `json:"tracking_number"`
	PackageReturnID *int64     `json:"package_return_id"`
	HubImportedAt   *time.Time `json:"hub_imported_at"`
	Description     string     `json:"description"`
	FullName        string     `json:"full_name"`
	PackageCode     string     `json:"package_code"`
	ShippingFee     float64    `json:"shipping_fee"`
	ReshipExtraFee  float64    `json:"reship_extra_fee"`
	ReturnedAt      *time.Time `json:"returned_at"`
	RequestReship   bool       `json:"request_reship"`
}

func TransformPackagesCustomerArray(inArr []entity.PackageCustomer) []PackageCustomer {
	packages := make([]PackageCustomer, 0)

	for _, v := range inArr {
		pkg := TransformPackageCustomer(v)
		packages = append(packages, pkg)
	}

	return packages
}

type PackageProduct struct {
	dbgorm.Model
	PackageID     int64   `json:"package_id"`
	ProductID     int64   `json:"product_id"`
	Status        int     `json:"status"`
	Quantity      int     `json:"quantity"`
	UserID        int64   `json:"user_id"`
	Name          string  `json:"name"`
	SKU           string  `json:"sku"`
	Weight        float64 `json:"weight"`
	Width         float64 `json:"width"`
	Length        float64 `json:"length"`
	Height        float64 `json:"height"`
	Detail        string  `json:"detail"`
	Material      string  `json:"material"`
	ProductStatus int     `json:"product_status"`
}
