package entity

import "tebexpressapi/pkg/utils/dbgorm"

type Order struct {
	dbgorm.Model
	UserID int64 `json:"user_id"`
	Status int   `json:"status"`
	Count  int64 `json:"count"`
}
