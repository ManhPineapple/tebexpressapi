package entity

import "tebexpressapi/pkg/utils/dbgorm"

type Setting struct {
	dbgorm.Model
	UserID int64  `json:"user_id"`
	Key    string `json:"key"`
	Value  string `json:"value"`
	Status int64  `json:"status"`
}
