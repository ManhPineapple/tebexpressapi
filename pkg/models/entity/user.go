package entity

import (
	"html/template"
	"tebexpressapi/pkg/utils/dbgorm"
	"time"
)

type User struct {
	dbgorm.Model
	UserName      string     `json:"username" gorm:"column:username"`
	Password      string     `json:"password"`
	Email         string     `json:"email" gorm:"default:NULL"`
	Package       int64      `json:"package"`
	PhoneNumber   string     `json:"phone_number" gorm:"default:NULL"`
	FullName      string     `json:"full_name"`
	Birthday      string     `json:"birthday"`
	Role          string     `json:"role"`
	Status        int64      `json:"status"`
	Balance       float64    `json:"balance"`
	Rewards       float64    `json:"rewards"`
	Class         int64      `json:"class"`
	WarehouseID   int64      `json:"warehouse_id" gorm:"default:NULL"`
	SlackID       string     `json:"slack_id" gorm:"default:NULL"`
	UserInfo      *UserInfo  `json:"user_info,omitempty"`
	HoldingMoney  float64    `gorm:"-" json:"holding_money"`
	ReferralCode  string     `json:"referral_code"`
	RefID         *int64     `json:"ref_id"`
	Point         int        `json:"point"`
	PartnerID     int64      `json:"partner_id"`
	Warehouse     *Warehouse `json:"warehouse" gorm:"save_associations:false;foreignKey:warehouse_id"`
	ReferringUser *User      `json:"referring_user,omitempty" gorm:"save_associations:false;foreignKey:ref_id"`
}

type UserToken struct {
	dbgorm.Model
	UserID int64  `json:"user_id"`
	Status int64  `json:"status"`
	Token  string `json:"token"`
}

type PointLog struct {
	dbgorm.Model
	Point       int    `json:"point"`
	Status      int    `json:"status"`
	Description string `json:"description"`
	UserID      int64  `json:"user_id"`
}

type DataMailConsumer struct {
	ID              int64
	Email           string
	UserName        string
	FullName        string
	Code            string
	EmailType       int64
	Row             int64
	Download        string
	Error           []string
	OrdersSuccess   int
	OrdersFails     int
	FileName        string
	ShopName        string
	OverdueDate     string
	Status          int64
	Title           string
	Link            string
	RateLink        string
	Amount          string
	TypeTransaction string
	TransactionID   string
	Body            template.HTML
}

type UserInfo struct {
	ID               int64      `json:"id" gorm:"primary_key;size:20;AUTO_INCREMENT;NOT NULL"`
	UserID           int64      `json:"user_id" gorm:"NOT NULL"`
	DebtMaxAmount    float64    `json:"debt_max_amount"`
	DebtMaxDay       int        `json:"debt_max_day"`
	RefundDay        int        `json:"refund_day"`
	DebtTime         *time.Time `json:"debt_time"`
	AppraiserID      int64      `json:"appraiser_id"`
	TaxCode          string     `json:"tax_code"`
	Volume           string     `json:"volume"`
	ItemType         string     `json:"item_type"`
	WarehouseAddress string     `json:"warehouse_address"`
	UpdatedAt        *time.Time `json:"updated_at"`
	CancelMaxAmount  float64    `json:"cancel_max_amount"`
}

type UserPermission struct {
	SupportID  int64 `json:"support_id" gorm:"primary_key;auto_increment:false"`
	CustomerID int64 `json:"customer_id" gorm:"primary_key;auto_increment:false"`
}
