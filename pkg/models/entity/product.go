package entity

import (
	"tebexpressapi/pkg/utils/dbgorm"
)

type Product struct {
	dbgorm.Model
	UserID   int64   `json:"user_id"`
	Name     string  `json:"name"`
	SKU      string  `json:"sku"`
	Weight   float64 `json:"weight"`
	Width    float64 `json:"width"`
	Length   float64 `json:"length"`
	Height   float64 `json:"height"`
	Detail   string  `json:"detail"`
	Material string  `json:"material"`
	Country  string  `json:"country"`
	Status   int     `json:"status,omitempty"`
}

type ProductDTO struct {
	dbgorm.Model
	UserID   int64   `json:"user_id"`
	Name     string  `json:"name"`
	SKU      string  `json:"sku"`
	Weight   float64 `json:"weight"`
	Width    float64 `json:"width"`
	Length   float64 `json:"length"`
	Height   float64 `json:"height"`
	Detail   string  `json:"detail"`
	Material string  `json:"material"`
	Country  string  `json:"country"`
}

type PackageProducts struct {
	dbgorm.Model
	PackageID int64    `json:"package_id"`
	ProductID int64    `json:"product_id"`
	Status    int      `json:"status"`
	Quantity  int64    `json:"quantity"`
	Product   *Product `json:"product,omitempty"`
}
