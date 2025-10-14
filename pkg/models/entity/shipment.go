package entity

import (
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/utils/dbgorm"
	"time"
)

type Shipment struct {
	dbgorm.Model
	Price       float64          `json:"price"`
	Status      int              `json:"status"`
	UserID      int64            `json:"user_id"`
	Quantity    int              `json:"quantity"`
	HubID       *int64           `json:"hub_id"`
	WarehouseID int64            `json:"warehouse_id"`
	CloseAt     *time.Time       `json:"close_at"`
	FbaType     constant.FbaType `json:"fba_type"`
	Recipient   string           `json:"recipient"`
	Address     string           `json:"address"`
	City        string           `json:"city"`
	State       string           `json:"state"`
	Country     string           `json:"country"`
	Zipcode     string           `json:"zipcode"`
	Containers  []Container      `json:"containers" gorm:"jointable_foreignkey:ShipmentID"`
	Manifest    []Manifest       `json:"manifest" gorm:"jointable_foreignkey:ShipmentID"`
	Warehouse   *Warehouse       `json:"warehouse" gorm:"foreignKey:WarehouseID"`
}

type Manifest struct {
	dbgorm.Model
	ShipmentID     *int64 `json:"shipment_id"`
	PackageID      *int64 `json:"package_id"`
	ManifestNumber string `json:"manifest_number"`
	ManifestURL    string `json:"manifest_url"`
	ContainerID    *int64 `json:"container_id"`
}
