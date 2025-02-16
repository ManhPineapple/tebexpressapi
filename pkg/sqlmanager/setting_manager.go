package sqlmanager

import (
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"

	"gorm.io/gorm"
)

// OrderManager --
type SettingManager struct {
	DB *gorm.DB
}
type SettingQueryOption struct {
	ID     int64
	UserID int64
	Key    string
	Value  string
}

// NewOrderManager --
func NewSettingManager(db *gorm.DB) *SettingManager {
	return &SettingManager{DB: db}
}

func (m *SettingManager) BuildQueryOptions(options SettingQueryOption) *gorm.DB {
	db := m.DB
	if options.UserID > 0 {
		db = db.Where("user_id = ?", options.UserID)
	}
	if len(options.Key) > 0 {
		db = db.Where("`key` = ?", options.Key)
	}
	if options.ID > 0 {
		db = db.Where("id = ?", options.ID)
	}
	db = db.Where("status = ?", constant.SettingEnableStatus)
	return db
}

func (m *SettingManager) GetSetting(options SettingQueryOption) (*entity.Setting, error) {
	db := m.BuildQueryOptions(options)
	var setting *entity.Setting
	db = db.First(&setting)

	return setting, db.Error
}
func (m *SettingManager) GetSettings(options SettingQueryOption) ([]*entity.Setting, error) {
	db := m.BuildQueryOptions(options)
	settings := []*entity.Setting{}
	db = db.Find(&settings)

	return settings, db.Error
}

func (m *SettingManager) SaveSetting(Setting *entity.Setting) error {
	db := m.DB
	db = db.Save(Setting)
	return db.Error
}

func (m *SettingManager) CheckExistSetting(options SettingQueryOption) (bool, error) {
	var exist bool
	db := m.DB.Raw("SELECT EXISTS(SELECT 1 FROM settings WHERE `key` = ? AND `value` = ? AND `status` = ? AND`user_id` <> ? )", options.Key, options.Value, constant.SettingEnableStatus, options.UserID).Row()
	err := db.Scan(&exist)
	if err != nil {
		return false, err
	}
	return exist, nil
}
