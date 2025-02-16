package entity

import "tebexpressapi/pkg/utils/dbgorm"

type State struct {
	dbgorm.Model
	Name    string `json:"name"`
	Code    string `json:"code"`
	Country string `json:"country"`
	Status  int    `json:"status"`
}
