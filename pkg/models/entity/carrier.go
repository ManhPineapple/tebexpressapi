package entity

import "tebexpressapi/pkg/utils/dbgorm"

type Carrier struct {
	dbgorm.Model
	Name            string `json:"name"`
	Code            string `json:"code"`
	Status          int    `json:"status"`
	Type            int    `json:"type"`
	LastMileCarrier string `json:"last_mile_carrier"`
	// URL    string `json:"url"`
}
