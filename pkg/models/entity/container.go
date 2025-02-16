package entity

import (
	"tebexpressapi/pkg/utils/dbgorm"
	"time"
)

type Container struct {
	dbgorm.Model
	Code             string                `json:"code"`
	TrackingNumber   string                `json:"tracking_number"`
	LabelUrl         string                `json:"label_url"`
	CarrierID        int64                 `json:"carrier_id"`
	Width            float64               `json:"width"`
	Height           float64               `json:"height"`
	Length           float64               `json:"length"`
	MaxWeight        float64               `json:"max_weight"`
	Weight           float64               `json:"weight"`
	ActualWeight     float64               `json:"actual_weight"`
	Status           int                   `json:"status"`
	Barcode          string                `json:"barcode"`
	ShipmentID       *int64                `json:"shipment_id"`
	HubID            int64                 `json:"hub_id"`
	CloseAt          *time.Time            `json:"close_at"`
	HubExportedAt    *time.Time            `json:"hub_exported_at"`
	HubImportedAt    *time.Time            `json:"hub_imported_at"`
	WarehouseID      int64                 `json:"warehouse_id"`
	Type             int                   `json:"type"`
	IsFba            int                   `json:"is_fba"`
	Warehouse        *Warehouse            `json:"warehouse" gorm:"foreignKey:WarehouseID"`
	Shipment         *Shipment             `json:"shipment" gorm:"save_associations:false"`
	ContainerItems   []ContainerItem       `json:"container_items"`
	Packages         []Package             `gorm:"-" json:"packages"`
	ContainerHistory []ContainerDeliverLog `json:"container_history"`
}

type ContainerBox struct {
	dbgorm.Model
	ID        int64   `json:"id" gorm:"primary_key;size:20;AUTO_INCREMENT;NOT NULL"`
	Width     float64 `json:"width"`
	Height    float64 `json:"height"`
	Length    float64 `json:"length"`
	MaxWeight float64 `json:"max_weight"`
}

type ContainerItem struct {
	dbgorm.Model
	Status      int64     `json:"status"`
	ContainerID int64     `json:"container_id"`
	PackageID   int64     `json:"package_id"`
	Description string    `json:"description"`
	Container   Container `json:"container" gorm:"foreignKey:ContainerID;references:ID"`
	Package     Package   `json:"package" gorm:"foreignKey:PackageID;references:ID"`
}

type ContainerDeliverLog struct {
	dbgorm.Model
	ContainerID int64     `json:"container_id"`
	Location    string    `json:"location"`
	ShipTime    time.Time `json:"ship_time"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	Code        string    `json:"code"`
}
