package entity

import "tebexpressapi/pkg/utils/dbgorm"

type Warehouse struct {
	dbgorm.Model
	Name           string  `json:"name"`
	Status         int64   `json:"status"`
	Type           int64   `json:"type"`
	Address        string  `json:"address"`
	City           string  `json:"city"`
	State          string  `json:"state"`
	Zipcode        string  `json:"zipcode"`
	Country        string  `json:"country"`
	Phone          string  `json:"phone"`
	Company        string  `json:"company"`
	HandlingFee    float64 `json:"handling_fee"`
	TimeOpen       string  `json:"time_open"`
	TimeActive     string  `json:"time_active"`
	LinkAddress    string  `json:"link_address"`
	ManifestActive int64   `json:"manifest_active"`
}

type PackageWarehouseCost struct {
	dbgorm.Model
	PackageID int64   `json:"package_id"`
	HubID     int64   `json:"hub_id"`
	CarrierID int64   `json:"carrier_id"`
	Cost      float64 `json:"cost"`
	OrgCost   float64 `json:"org_cost"`
	//Status      int64      `json:"status"`
	Zone      int        `json:"zone"`
	Warehouse *Warehouse `json:"warehouse" gorm:"foreignKey:HubID"`
}

type QueuePushCreateLabel struct {
	PackageID       int64  `json:"package_id"`
	Carrier         string `json:"carrier"`
	Warehouse       int64  `json:"warehouse"`
	PostmarkDate    int64  `json:"postmark_date"`
	UserID          int64  `json:"user_id"`
	IsPackageExceed bool   `json:"is_package_exceed"`
}

type QueuePushUpdateLabel struct {
	PackageID int64 `json:"package_id"`
	UserID    int64 `json:"user_id"`
}
