package entity

import "tebexpressapi/pkg/utils/dbgorm"

type Bill struct {
	dbgorm.Model
	Code        string  `json:"code"`
	ShippingFee float64 `json:"shipping_fee"`
	ExtraFee    float64 `json:"extra_fee"`
	Status      int64   `json:"status"`
	UserID      int64   `json:"user_id"`

	Package []Package `json:"package" gorm:"save_associations:false;foreignKey:BillID"`
	User    *User     `json:"user,omitempty" gorm:"save_associations:false;foreignKey:UserID"`
}

// type BillPackage struct {
// 	dbgorm.Model
// 	BillID      int64   `json:"bill_id"`
// 	PackageID   int64   `json:"package_id"`
// 	ShippingFee float64 `json:"shipping_fee"`
// 	ExtraFee    float64 `json:"extra_fee"`
// }
