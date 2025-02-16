package entity

import (
	"tebexpressapi/pkg/utils/dbgorm"
	"time"
)

type Notification struct {
	ID     int64 `json:"id" gorm:"primary_key;size:20;AUTO_INCREMENT;NOT NULL"`
	UserID int64 `json:"user_id" gorm:"index:user_idx"`

	UserIDs   dbgorm.JSON `json:"user_ids"`
	SenderID  int64       `json:"sender_id"`
	CreatorID int64       `json:"creator_id"`
	IsSendAll int         `json:"is_send_all"`
	Title     string      `json:"title"`
	Body      string      `json:"body"`
	Image     string      `json:"image"`
	Badge     string      `json:"badge" gorm:"-"`
	Link      string      `json:"link"`
	Status    int         `json:"status"`
	Readed    int         `json:"readed"`
	ParentID  int64       `json:"parent_id"`
	Type      int         `json:"type"`
	SentAt    *time.Time  `json:"sent_at"`
	CreatedAt *time.Time  `json:"created_at"`
	UpdatedAt *time.Time  `json:"updated_at"`
	Sender    *User       `json:"sender" gorm:"foreignKey:SenderID;"`
	Creator   *User       `json:"creator" gorm:"foreignKey:CreatorID;"`
}

type NotifyEmail struct {
	dbgorm.Model
	UserID      int64       `json:"user_id" gorm:"index:user_idx"`
	SenderID    int64       `json:"sender_id" gorm:"index:sender_idx"`
	ReceiverIDs dbgorm.JSON `json:"receiver_ids" gorm:"index:receiver_idx"`
	Title       string      `json:"title"`
	Body        string      `json:"body"`
	Status      int         `json:"status"`
	SentAt      *time.Time  `json:"sent_at"`
	IsSendAll   int         `json:"is_send_all"`
	Sender      *User       `json:"sender" gorm:"foreignKey:SenderID;"`
	Creator     *User       `json:"creator" gorm:"foreignKey:UserID;"`
}

type FirebaseToken struct {
	Date   time.Time `json:"date"`
	Device string    `json:"device"`
	Token  string    `json:"token"`
}
