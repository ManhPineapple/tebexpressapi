package dto

import "time"

type OrderPackage struct {
	ID             int64     `json:"id"`
	OrderNumber    string    `json:"order_number"`
	Code           string    `json:"code"`
	TrackingNumber string    `json:"tracking_number"`
	OrderID        *int64    `json:"order_id"`
	CreatedAt      time.Time `json:"created_at"`
	Status         int       `json:"status"`
	StatusText     string    `json:"status_text"`
	ServiceCode    string    `json:"service_code"`
}

type CountOrderPackage struct {
	Status int `json:"status"`
	Total  int `json:"total"`
}

type CountStatusOrder struct {
	Status int   `json:"status"`
	Count  int64 `json:"count"`
}
