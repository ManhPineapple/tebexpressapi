package dto

import (
	"tebexpressapi/pkg/models/entity"
	"time"
)

type User struct {
	ID       int64  `json:"id"`
	UserName string `json:"username"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	Status   int64  `json:"status"`
}

func UserTransform(user *entity.User) {
	// Hide password on response
	user.Password = "********"
}

type UserInfoDto struct {
	UserID          int64      `json:"id"`
	DebtMaxAmount   float64    `json:"debt_max_amount"`
	DebtMaxDay      int        `json:"debt_max_day"`
	DebtTime        *time.Time `json:"debt_time"`
	AppraiserID     int64      `json:"appraiser_id"`
	CancelMaxAmount float64    `json:"cancel_max_amount"`
	RefundDay       int        `json:"refund_day"`
}

type UserResponseDto struct {
	ID               int64        `json:"id"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
	UserName         string       `json:"username"`
	Email            string       `json:"email"`
	PhoneNumber      string       `json:"phone_number"`
	FullName         string       `json:"full_name"`
	Birthday         string       `json:"birthday"`
	Role             string       `json:"role"`
	Status           int64        `json:"status"`
	Class            int          `json:"class"`
	Balance          float64      `json:"balance"`
	WarehouseID      int64        `json:"warehouse_id"`
	SlackID          string       `json:"slack_id"`
	UserInfo         *UserInfoDto `json:"user_info,omitempty"`
	SupportID        []int64      `json:"support_id"`
	CustomerID       []int64      `json:"customer_id"`
	Package          int64        `json:"package"`
	AppraiserName    string       `json:"appraiser_name"`
	AppraiserID      int64        `json:"appraiser_id"`
	TaxCode          string       `json:"tax_code"`
	Volume           string       `json:"volume"`
	ItemType         string       `json:"item_type"`
	WarehouseAddress string       `json:"warehouse_address"`
	HoldingMoney     float64      `json:"holding_money"`
	RefID            *int64       `json:"ref_id,omitempty"`
	ReferringName    *string      `json:"referring_name,omitempty"`
	IsCustomerRefer  bool         `json:"is_customer_refer"`
}
