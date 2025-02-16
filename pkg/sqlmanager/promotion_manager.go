package sqlmanager

import (
	"fmt"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"time"

	"gorm.io/gorm"
)

type PromotionManager struct {
	db *gorm.DB
}

func NewPromotionManager(db *gorm.DB) *PromotionManager {
	return &PromotionManager{
		db: db,
	}
}

type PromotionQueryOption struct {
	ID     int64
	Search string
	Limit  int
	Offset int
	UserID int64
	Status int64
	Type   int64
}

type SettingPointQueryOption struct {
	ID     int64
	Limit  int
	Class  int
	Offset int
	Order  string
	Status int64
}

func (m *PromotionManager) buildPromotionQuery(opts PromotionQueryOption) *gorm.DB {
	db := m.db

	if opts.ID > 0 {
		db = db.Where("id=?", opts.ID)
	}

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	if opts.Search != "" {
		db = db.Where("name LIKE (?)", fmt.Sprintf("%%%s%%", opts.Search))
	}

	if opts.UserID > 0 {
		db = db.Where("user_id = ?", opts.UserID)
	}

	if opts.Status > 0 {
		db = db.Where("status=?", opts.Status)
	}

	if opts.Type > 0 {
		db = db.Where("type=?", opts.Type)
	}

	db = db.Where("status > ?", 0)
	return db
}

func (m *PromotionManager) buildSettingPointQuery(opts SettingPointQueryOption) *gorm.DB {
	db := m.db

	if opts.ID > 0 {
		db = db.Where("id=?", opts.ID)
	}

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	if opts.Status > 0 {
		db = db.Where("status=?", opts.Status)
	}

	if opts.Class > 0 {
		db = db.Where("class = ?", opts.Class)
	}

	if opts.Order != "" {
		db = db.Order(opts.Order)
	}

	return db
}

func (m *PromotionManager) CreatePromotion(promotion *entity.Promotion) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Create(promotion).Error; err != nil {
		tx.Rollback()
		return err
	}

	if len(promotion.Prices) > 0 {
		prices := promotion.Prices
		for i := range prices {
			prices[i].CreatedAt = time.Now()
			prices[i].UpdatedAt = prices[i].CreatedAt
			prices[i].PromotionID = promotion.ID
		}

		if err := tx.Create(&prices).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if len(promotion.Weights) > 0 {
		weights := promotion.Weights
		for i := range weights {
			weights[i].CreatedAt = time.Now()
			weights[i].UpdatedAt = weights[i].CreatedAt
			weights[i].PromotionID = promotion.ID
		}

		if err := tx.Create(&weights).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m *PromotionManager) GetPromotions(opts PromotionQueryOption) ([]*entity.Promotion, error) {
	db := m.buildPromotionQuery(opts)
	db = db.Order("id DESC").Preload("User", func(db *gorm.DB) *gorm.DB {
		db = db.Select("id, full_name, email, role")
		return db
	})
	promotions := []*entity.Promotion{}

	db = db.Find(&promotions)
	return promotions, db.Error
}

func (m *PromotionManager) CountPromotions(opts PromotionQueryOption) (int64, error) {
	var count int64

	db := m.buildPromotionQuery(opts)

	db = db.Model(&entity.Promotion{}).Count(&count)
	return count, db.Error
}

func (m *PromotionManager) GetPromotionByID(id int64) (*entity.Promotion, error) {
	promotion := &entity.Promotion{}
	db := m.db.Where("id = ?", id)
	db = db.First(promotion)
	return promotion, db.Error
}

func (m *PromotionManager) UpdatePromotion(id int64, status int) error {
	mapchange := map[string]interface{}{
		"Status":    status,
		"UpdatedAt": time.Now(),
	}
	db := m.db.Model(&entity.Promotion{}).Where("id=?", id).UpdateColumns(mapchange)
	return db.Error
}

func (m *PromotionManager) GetUserIDByPromotionID(id int64, ignoreUsers []int64) ([]int64, error) {
	var ids []int64
	db := m.db
	db = db.Table("user_promotions").Where("promotion_id=? AND status=?", id, constant.StatusActive)
	if len(ignoreUsers) > 0 {
		db = db.Where("user_id NOT IN (?)", ignoreUsers)
	}
	db = db.Pluck("user_id", &ids)
	return ids, db.Error
}

func (m *PromotionManager) AppendUserToPromotion(promotionID int64, idsDelete []int64, idsAdd []int64) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	for _, id := range idsDelete {
		mapchange := map[string]interface{}{
			"Status":    constant.StatusDeactive,
			"UpdatedAt": time.Now(),
		}
		if err := tx.Model(entity.UserPromotion{}).Where("user_id = ? AND promotion_id = ?", id, promotionID).UpdateColumns(mapchange).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	for _, id := range idsAdd {
		mapchange := map[string]interface{}{
			"Status":    constant.StatusActive,
			"UpdatedAt": time.Now(),
		}
		userPromotion := entity.UserPromotion{
			PromotionID: promotionID,
			UserID:      id,
			Status:      constant.StatusActive,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := tx.Model(entity.UserPromotion{}).Where("user_id=? and promotion_id=?", id, promotionID).First(&entity.UserPromotion{}).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := tx.Create(userPromotion).Error; err != nil {
					tx.Rollback()
					return err
				}
			} else {
				tx.Rollback()
				return err
			}
		} else {
			if err := tx.Model(entity.UserPromotion{}).Where("user_id = ? AND promotion_id = ?", id, promotionID).UpdateColumns(mapchange).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit().Error
}

func (m *PromotionManager) CheckPromotion(promotionID int64, userID int64) (bool, error) {
	var count int64

	db := m.db.Model(&entity.UserPromotion{})
	db = db.Joins("LEFT JOIN promotions ON promotions.id = user_promotions.promotion_id")
	db = db.Where("user_promotions.promotion_id = ? AND user_promotions.user_id = ? AND user_promotions.status = ? AND promotions.status = ?", promotionID, userID, constant.StatusActive, constant.StatusActive)
	db = db.Count(&count)

	return count > 0, db.Error
}

func (m *PromotionManager) Update(promotion *entity.Promotion, prices []*entity.PromotionPrice, weights []*entity.PromotionWeight) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Save(promotion).Error; err != nil {
		tx.Rollback()
		return err
	}

	if len(prices) > 0 {
		if err := tx.Where("promotion_id=?", promotion.ID).Delete(&entity.PromotionPrice{}).Error; err != nil {
			tx.Rollback()
			return err
		}

		for i := range prices {
			prices[i].CreatedAt = time.Now()
			prices[i].UpdatedAt = prices[i].CreatedAt
			prices[i].PromotionID = promotion.ID
		}

		if err := tx.Create(&prices).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if len(weights) > 0 {
		if err := tx.Where("promotion_id=?", promotion.ID).Delete(&entity.PromotionWeight{}).Error; err != nil {
			tx.Rollback()
			return err
		}

		for i := range weights {
			weights[i].CreatedAt = time.Now()
			weights[i].UpdatedAt = weights[i].CreatedAt
			weights[i].PromotionID = promotion.ID
		}

		if err := tx.Create(&weights).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m *PromotionManager) GetUsersByPromotionIDs(ids []int64, ignoreUsers []int64) ([]entity.UserPromotion, error) {
	db := m.db.Table("user_promotions").Select("user_id, promotion_id")
	db = db.Where("promotion_id IN (?) AND status=?", ids, constant.StatusActive)

	if len(ignoreUsers) > 0 {
		db = db.Where("user_id NOT IN (?)", ignoreUsers)
	}

	items := []entity.UserPromotion{}
	db = db.Find(&items)
	return items, db.Error
}

func (m *PromotionManager) GetUsersByPromotionID(id int64, ignoreUsers []int64) ([]entity.User, error) {
	db := m.db.Select("users.id, users.username, users.email, users.full_name")
	db = db.Joins("INNER JOIN user_promotions ON user_promotions.user_id=users.id AND user_promotions.promotion_id=? AND user_promotions.status=?", id, constant.StatusActive)

	if len(ignoreUsers) > 0 {
		db = db.Where("users.id NOT IN (?)", ignoreUsers)
	}

	users := []entity.User{}
	db = db.Find(&users)
	return users, db.Error
}

func (m *PromotionManager) GetMarketingPromotionsByUserIDs(ids, ignorePromotionIDs []int64) ([]entity.UserPromotion, error) {
	db := m.db.Select("user_promotions.promotion_id, user_promotions.user_id")
	db = db.Joins("INNER JOIN promotions ON user_promotions.promotion_id=promotions.id")
	db = db.Where("promotions.status=? AND user_promotions.status=?", constant.StatusActive, constant.StatusActive)
	db = db.Where("user_promotions.user_id IN (?)", ids)
	db = db.Where("promotions.type=?", constant.PromotionTypeMarketing)
	db = db.Preload("User")

	if len(ignorePromotionIDs) > 0 {
		db = db.Where("promotions.id NOT IN (?)", ignorePromotionIDs)
	}

	promotions := []entity.UserPromotion{}
	db = db.Find(&promotions)
	return promotions, db.Error
}

func (m *PromotionManager) GetCustomerPromotions(customerID int64) (*entity.Promotion, error) {
	db := m.db
	db = db.Joins("INNER JOIN user_promotions ON user_promotions.promotion_id=promotions.id")
	db = db.Where("promotions.status=? AND user_promotions.status=? AND promotions.type = ?", constant.StatusActive, constant.StatusActive, constant.PromotionTypeMarketing)
	db = db.Where("user_promotions.user_id = ?", customerID)
	promotion := &entity.Promotion{}
	db = db.First(&promotion)
	return promotion, db.Error
}

func (m *PromotionManager) SaveSettingPoint(setting *entity.SettingPoint) error {
	db := m.db
	db = db.Save(setting)
	return db.Error
}

func (m *PromotionManager) GetSettingPointByID(id int64) (*entity.SettingPoint, error) {
	setting := &entity.SettingPoint{}
	db := m.db.Where("id = ?", id)
	db = db.First(setting)
	return setting, db.Error
}

func (m *PromotionManager) GetSettingPoints(opts SettingPointQueryOption) ([]*entity.SettingPoint, error) {
	db := m.buildSettingPointQuery(opts)
	settings := []*entity.SettingPoint{}
	db = db.Find(&settings)
	return settings, db.Error
}

func (m *PromotionManager) CountSettingPoints(opts SettingPointQueryOption) (int64, error) {
	var count int64
	db := m.buildSettingPointQuery(opts)
	db = db.Model(&entity.SettingPoint{}).Count(&count)
	return count, db.Error
}

func (m *PromotionManager) SaveSettingPoints(settings []entity.SettingPoint) error {
	db := m.db
	db = db.Save(&settings)
	return db.Error
}
