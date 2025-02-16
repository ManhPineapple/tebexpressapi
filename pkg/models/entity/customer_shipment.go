package entity

import "tebexpressapi/pkg/utils/dbgorm"

type CustomerShipment struct {
	dbgorm.Model
	UserID       int64      `json:"user_id"`
	Weight       float64    `json:"weight"`
	ActualWeight float64    `json:"actual_weight" gorm:"-"`
	Price        float64    `json:"price"`
	Status       int        `json:"status"`
	User         *User      `json:"user,omitempty"`
	Packages     []Package  `json:"packages,omitempty"`
	ExtraFees    []ExtraFee `json:"extra_fees,omitempty"`
}
