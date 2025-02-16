package dbgorm

import "time"

type Model struct {
	ID        int64     `gorm:"primary_key;size:20;AUTO_INCREMENT;NOT NULL" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
