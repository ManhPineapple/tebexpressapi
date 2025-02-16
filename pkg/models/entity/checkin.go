package entity

import "tebexpressapi/pkg/utils/dbgorm"

type CheckinRequest struct {
	dbgorm.Model
	Status         int64            `json:"status"`
	CloseUserID    int64            `json:"close_user_id" gorm:"default:NULL"`
	CheckinPackage []CheckinPackage `json:"checkin_package" gorm:"save_associations:false;foreignKey:CheckinID"`
}

type CheckinPackage struct {
	CheckinID int64   `json:"checkin_id" gorm:"primary_key;auto_increment:false"`
	PackageID int64   `json:"package_id" gorm:"primary_key;auto_increment:false"`
	Status    int     `json:"status"`
	Package   Package `json:"package" gorm:"save_associations:false;foreignKey:PackageID"`
}
