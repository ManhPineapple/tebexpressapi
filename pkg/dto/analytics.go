package dto

type PackageAnalyticsStatus struct {
	Status     int    `json:"status"`
	StatusText int    `json:"status_text"`
	Count      int64  `json:"count"`
	DateTime   string `json:"date_time"`
}
