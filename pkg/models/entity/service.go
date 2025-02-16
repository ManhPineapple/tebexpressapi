package entity

import (
	"tebexpressapi/pkg/utils/dbgorm"
)

type Service struct {
	dbgorm.Model
	Name                   string  `json:"name"`
	Code                   string  `json:"code"`
	DomesticCarrierID      int64   `json:"domestic_carrier_id,omitempty"`
	DomesticCarrierService string  `json:"domestic_carrier_service,omitempty"`
	WWCarrierID            int64   `json:"ww_carrier_id,omitempty"`
	WWCarrierService       string  `json:"ww_carrier_service,omitempty"`
	Country                string  `json:"country"`
	DomesticCarrier        Carrier `json:"domestic_carrier" gorm:"foreignKey:DomesticCarrierID"`
	WWCarrier              Carrier `json:"ww_carrier" gorm:"foreignKey:WWCarrierID"`
	ExtraFee_1             float64 `json:"extra_fee_1"`
	ExtraFee_2             float64 `json:"extra_fee_2"`
	Prices                 []Price `json:"prices,omitempty"`
	PartnerID              int64   `json:"partner_id,omitempty"`
}

type CheckPriceLog struct {
	dbgorm.Model
	Data dbgorm.JSON `json:"data"`
}

type ServiceCustomer struct {
	Name string `json:"name"`
	Code string `json:"code"`
}
