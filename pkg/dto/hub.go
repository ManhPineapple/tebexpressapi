package dto

import "time"

type HubItem struct {
	ID              int64      `json:"id"`
	Code            string     `json:"code"`
	TrackingNumber  string     `json:"tracking_number"`
	Status          int        `json:"status"`
	Type            string     `json:"type"`
	CreatedAt       string     `json:"created_at"`
	UpdatedAt       string     `json:"updated_at"`
	LabelUrl        string     `json:"label_url"`
	ManifestNumber  string     `json:"manifest_number"`
	Description     string     `json:"description"`
	PackageReturnID int64      `json:"package_return_id,omitempty"`
	HubImportedAt   *time.Time `json:"hub_imported_at,omitempty"`
	HubExportedAt   *time.Time `json:"hub_exported_at,omitempty"`
	CountContainer  int64      `json:"count_container"`
	ReshipAt        *time.Time `json:"reship_at,omitempty"`
	ReturnedAt      *time.Time `json:"returned_at,omitempty"`
}
