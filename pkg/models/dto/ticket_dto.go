package dto

import (
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/utils/dbgorm"
	"time"
)

type Ticket struct {
	ID         int64              `json:"id"`
	Category   int64              `json:"category"`
	UserID     int64              `json:"user_id"`
	ObjectID   int64              `json:"object_id"`
	Title      string             `json:"title"`
	Content    string             `json:"content"`
	Attachment dbgorm.SliceString `json:"attachment"`
	Status     int64              `json:"status"`
	StatusRep  int64              `json:"status_rep"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`

	Package *TicketPackage `json:"package"`
	User    *User          `json:"user"`
}

type TicketMessage struct {
	ID         int64              `json:"id"`
	TicketID   int64              `json:"ticket_id"`
	UserID     int64              `json:"user_id"`
	FullName   string             `json:"full_name,omitempty"`
	Role       string             `json:"role,omitempty"`
	Content    string             `json:"content"`
	Attachment dbgorm.SliceString `json:"attachment"`
	Status     int64              `json:"status"`
	CreatedAt  time.Time          `json:"created_at"`
}

type TicketPackage struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Code      string    `json:"code"`
	Status    int64     `json:"status"`
}

type TicketsDTO struct {
	dbgorm.Model
	Title       string             `json:"title"`
	Content     string             `json:"content" gorm:"type:text"`
	Category    int64              `json:"category"`
	ObjectID    int64              `json:"object_id" gorm:"index:object"`
	Attachment  dbgorm.SliceString `json:"attachment" gorm:"type:text"`
	Status      int64              `json:"status" gorm:"type:int(11);not null;index:status_idx"`
	StatusRep   int64              `json:"status_rep" gorm:"type:int(11)"`
	Type        int64              `json:"type"`
	Amount      float64            `json:"amount"`
	IsRated     bool               `json:"is_rated"`
	PackageCode string             `json:"package_code"`
	User        *entity.User       `json:"user"`
}
