package entity

import (
	"tebexpressapi/pkg/utils/dbgorm"
	"time"
)

type Promotion struct {
	dbgorm.Model
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Status       int     `json:"status"`
	UserIDS      []int64 `gorm:"-" json:"user_ids"`
	Type         int     `json:"type"`
	UserID       int64   `json:"user_id"`
	S3PathPrice  string  `json:"s3_path_price"`
	S3PathWeight string  `json:"s3_path_weight"`
	Price        float64 `json:"price"`

	Prices  []*PromotionPrice  `json:"prices" gorm:"->"`
	Weights []*PromotionWeight `json:"weights" gorm:"->"`
	User    *User              `json:"user,omitempty" gorm:"->"`
}

type UserPromotion struct {
	UserID      int64     `json:"user_id" gorm:"primary_key;auto_increment:false"`
	PromotionID int64     `json:"promotion_id" gorm:"primary_key;auto_increment:false"`
	Status      int       `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	User      *User      `json:"user,omitempty" gorm:"->"`
	Promotion *Promotion `json:"promotion,omitempty" gorm:"->"`
}

type PromotionPrice struct {
	dbgorm.Model
	ServiceID   int64   `json:"service_id"`
	PromotionID int64   `json:"promotion_id"`
	Weight      float64 `json:"weight"`
	Price       float64 `json:"price"`
}

type PromotionWeight struct {
	dbgorm.Model
	PromotionID      int64   `json:"promotion_id"`
	Weight           float64 `json:"weight"`
	ErrorWeightAllow float64 `json:"error_weight_allow"`
}

type SettingPoint struct {
	dbgorm.Model
	From   float64 `json:"from"`
	To     float64 `json:"to"`
	Value  float64 `json:"value"`
	Class  int     `json:"class"`
	Status int     `json:"status"`
}
