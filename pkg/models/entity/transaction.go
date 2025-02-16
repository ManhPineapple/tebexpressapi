package entity

import (
	"tebexpressapi/pkg/utils/dbgorm"
)

type Transaction struct {
	dbgorm.Model
	UserID      int64   `json:"user_id"`
	AdminID     int64   `json:"admin_id" gorm:"default:NULL"`
	BillID      *int64  `json:"bill_id" gorm:"default:NULL"`
	Bill        *Bill   `json:"bill"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	Type        int64   `json:"type"`
	Status      int64   `json:"status" gorm:"type:int(11);not null;index:status_idx"`
	User        *User   `json:"user"`
	Admin       *User   `json:"admin" gorm:"save_associations:false;foreignKey:AdminID"`
}

type TransactionLog struct {
	dbgorm.Model
	UserID        int64   `json:"user_id"`
	BillID        *int64  `json:"bill_id" gorm:"default:NULL"`
	AdminID       int64   `json:"admin_id" gorm:"default:NULL"`
	Amount        float64 `json:"amount"`
	TransactionID int64   `json:"transaction_id"`
	Type          int64   `json:"type"`
	Status        int64   `json:"status" gorm:"type:int(11);not null;index:status_idx"`
	Description   string  `json:"description"`
	User          *User   `json:"user"`
}
