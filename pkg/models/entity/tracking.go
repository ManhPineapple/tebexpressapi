package entity

import "tebexpressapi/pkg/utils/dbgorm"

type Tracking struct {
	dbgorm.Model
	PackageID      int64   `json:"package_id"`
	ShipmentID     string  `json:"shipment_id"`
	TrackingNumber string  `json:"tracking_number"`
	LabelURL       string  `json:"label_url"`
	CarrierID      int64   `json:"carrier_id"`
	Status         int     `json:"status"`
	Weight         float64 `json:"weight"`
	Length         float64 `json:"length"`
	Width          float64 `json:"width"`
	Height         float64 `json:"height"`
	ShipmentCost   float64 `json:"shipment_cost"`
	UserID         int64   `json:"user_id"`
	HandlingFee    float64 `json:"handling_fee"`
	CarrierService string  `json:"carrier_service"`
	HubID          *int64  `json:"hub_id"`
	Zone           int     `json:"zone"`
	Version        string  `json:"version"`
	IsManifested   int     `json:"is_manifested"`

	User      *User      `json:"user,omitempty" gorm:"save_associations:false;foreignkey:UserID"`
	Package   *Package   `json:"package,omitempty" gorm:"save_associations:false;foreignkey:PackageID"`
	Carrier   *Carrier   `json:"carrier,omitempty" gorm:"save_associations:false;foreignkey:CarrierID"`
	Warehouse *Warehouse `json:"warehouse,omitempty" gorm:"save_associations:false;foreignkey:HubID"`
}
