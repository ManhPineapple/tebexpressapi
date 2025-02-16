package entity

import "tebexpressapi/pkg/utils/dbgorm"

type Price struct {
	dbgorm.Model
	ServiceID int64   `json:"service_id"`
	Weight    float64 `json:"weight"`
	Price     float64 `json:"price"`
	UserClass int64   `json:"user_class"`
}

type ExchangeRateLog struct {
	dbgorm.Model
	UserID int64   `json:"user_id"`
	Rate   float64 `json:"rate"`
}
