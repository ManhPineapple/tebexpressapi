package entity

import "tebexpressapi/pkg/utils/dbgorm"

type FakeVolume struct {
	dbgorm.Model
	Milestone float64 `json:"milestone"`
	Weight    float64 `json:"weight"`
	Dimension float64 `json:"dimension"`
}

type FakeDimension struct {
	dbgorm.Model
	To   float64 `json:"to"`
	From float64 `json:"from"`
	Rate float64 `json:"rate"`
}
