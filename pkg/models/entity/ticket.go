package entity

import (
	"tebexpressapi/pkg/utils/dbgorm"
)

type Ticket struct {
	dbgorm.Model
	Title   string `json:"title"`
	UserID  int64  `json:"user_id"`
	AdminID int64  `json:"admin_id" gorm:"default:NULL"`

	Content      string             `json:"content" gorm:"type:text"`
	Category     int64              `json:"category"`
	ObjectID     int64              `json:"object_id" gorm:"index:object"`
	Attachment   dbgorm.SliceString `json:"attachment" gorm:"type:text"`
	Status       int64              `json:"status" gorm:"type:int(11);not null;index:status_idx"`
	StatusRep    int64              `json:"status_rep" gorm:"type:int(11)"`
	Type         int                `json:"type"`
	Amount       float64            `json:"amount" gorm:"default:NULL"`
	AccountantID *int64             `json:"accountant_id" gorm:"default:NULL"`
	IsRated      bool               `json:"is_rated"`
	Note         string             `json:"note,omitempty"`

	Messages    []TicketMessage `json:"messages"`
	LastMessage *TicketMessage  `json:"last_message" gorm:"save_associations:false;foreignkey:TicketID;association_foreignkey:ID"`
	Package     *Package        `json:"package" gorm:"save_associations:false;foreignkey:ObjectID"`
	User        *User           `json:"user" gorm:"save_associations:false;foreignkey:UserID"`
	Admin       *User           `json:"admin" gorm:"save_associations:false;foreignkey:AdminID"`
	Accountant  *User           `json:"accountant" gorm:"save_associations:false;foreignkey:AccountantID;default:NULL"`

	Supports []TicketSupport `json:"supports,omitempty" gorm:"save_associations:false;foreignkey:TicketID"`
}

type TicketMessage struct {
	dbgorm.Model
	UserID     int64              `json:"user_id"`
	TicketID   int64              `json:"ticket_id" gorm:"index:ticket"`
	Content    string             `json:"content" gorm:"type:text"`
	Attachment dbgorm.SliceString `json:"attachment" gorm:"type:text"`
	Status     int64              `json:"status" gorm:"index:status"`

	User   *User   `json:"user" gorm:"save_associations:false;foreignkey:UserID"`
	Ticket *Ticket `json:"ticket" gorm:"save_associations:false;foreignkey:TicketID"`
}

type TicketSupport struct {
	ID       int64  `json:"id"`
	TicketID int64  `json:"ticket_id"`
	UserName string `json:"username"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

type Feedback struct {
	dbgorm.Model
	TicketID  int64  `json:"ticket_id"`
	SupportID int64  `json:"support_id"`
	Rating    int    `json:"rating"`
	Response  string `json:"response"`
}
