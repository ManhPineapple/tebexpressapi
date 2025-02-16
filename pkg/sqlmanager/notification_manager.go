package sqlmanager

import (
	"fmt"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/dto"
	"tebexpressapi/pkg/models/entity"
	"time"

	"gorm.io/gorm"
)

type NotifyManager struct {
	DB *gorm.DB
}

type NotifyQueryParams struct {
	ID             int64
	Search         string
	Limit          int
	Offset         int
	StartDate      string
	EndDate        string
	UserID         int64
	Status         int
	Type           int
	Readed         int
	Unread         int
	NotifyCustomer bool
}

type NotifyEmailQueryParams struct {
	ID        int64
	Search    string
	Limit     int
	Offset    int
	StartDate string
	EndDate   string
	UserID    int64
}

func NewNotifyManager(db *gorm.DB) *NotifyManager {
	return &NotifyManager{DB: db}
}

func (m *NotifyManager) buildQueryNotify(params NotifyQueryParams) *gorm.DB {
	db := m.DB

	if params.ID > 0 {
		db = db.Where("id=?", params.ID)
	}

	if len(params.StartDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= ?", params.StartDate)
	}

	if len(params.EndDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= ?", params.EndDate)
	}

	if params.UserID > 0 {
		db = db.Where("user_id = ?", params.UserID)
	}

	if params.Search != "" {
		if params.NotifyCustomer {
			db = db.Joins("JOIN users ON users.id = notifications.sender_id")
			db = db.Where("notifications.title LIKE ? OR users.full_name LIKE ?", fmt.Sprintf("%%%s%%", params.Search), fmt.Sprintf("%%%s%%", params.Search))
		} else {
			db = db.Where("title LIKE ?", fmt.Sprintf("%%%s%%", params.Search))
		}
	}

	if params.Type > 0 {
		db = db.Where("type = ?", params.Type)
	}

	if params.Limit > 0 {
		db = db.Limit(params.Limit)
	}

	if params.Offset > 0 {
		db = db.Offset(params.Offset)
	}
	if params.Status > 0 {
		db = db.Where("status = ?", params.Status)
	}
	if params.Readed > 0 {
		db = db.Where("readed = ?", constant.NotifyStatusReaded)
	}
	if params.Unread > 0 {
		db = db.Where("readed = ?", constant.NotifyStatusUnread)
	}

	if params.NotifyCustomer {
		db = db.Where("creator_id > 0")
	}

	return db
}

func (m *NotifyManager) First(params NotifyQueryParams) (*entity.Notification, error) {
	noti := &entity.Notification{}

	db := m.buildQueryNotify(params)
	db = db.First(noti)
	return noti, db.Error
}

func (m *NotifyManager) Find(params NotifyQueryParams) ([]entity.Notification, error) {
	var notifies []entity.Notification

	db := m.buildQueryNotify(params)
	db = db.Order("id DESC")
	db = db.Find(&notifies)
	return notifies, db.Error
}

func (m *NotifyManager) CountNotifyCustomer(opts NotifyQueryParams) (int64, error) {
	db := m.buildQueryNotify(opts)

	var count int64

	db = db.Model(&entity.Notification{}).Select("COUNT(notifications.id) as count").Count(&count)
	return count, db.Error
}

func (m *NotifyManager) Count(opts NotifyQueryParams) (int64, error) {
	db := m.buildQueryNotify(opts)

	var count int64

	db = db.Model(&entity.Notification{}).Select("COUNT(id) as count").Count(&count)
	return count, db.Error
}

func (m *NotifyManager) CountNotifyByStatus(params NotifyQueryParams) ([]dto.CountStatusNotify, error) {
	result := []dto.CountStatusNotify{}
	db := m.buildQueryNotify(params)

	db = db.Table("notifications").Select("COUNT(notifications.id) AS count,notifications.type")
	db = db.Group("notifications.type").Scan(&result)

	return result, db.Error
}

func (m *NotifyManager) Create(noti *entity.Notification) error {
	now := time.Now()
	noti.CreatedAt = &now
	return m.DB.Create(noti).Error
}

func (m *NotifyManager) Save(noti *entity.Notification) error {
	now := time.Now()
	noti.UpdatedAt = &now
	return m.DB.Save(noti).Error
}

func (m *NotifyManager) Read(id int64, userID int64) (int64, error) {
	db := m.DB.Model(&entity.Notification{}).Where("id = ?", id)
	db = db.Where("user_id = ?", userID)
	db = db.Update("readed", constant.NotifyStatusReaded)
	return db.RowsAffected, db.Error
}

func (m *NotifyManager) Reads(userID int64) (int64, error) {
	db := m.DB.Model(&entity.Notification{})
	db = db.Where("user_id = ?", userID)
	db = db.Update("readed", constant.NotifyStatusReaded)
	return db.RowsAffected, db.Error
}

func (m *NotifyManager) buildQueryNotifyEmail(params NotifyEmailQueryParams) *gorm.DB {
	db := m.DB

	if params.ID > 0 {
		db = db.Where("id = ?", params.ID)
	}

	if len(params.StartDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= ?", params.StartDate)
	}

	if len(params.EndDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= ?", params.EndDate)
	}

	if params.UserID > 0 {
		db = db.Where("user_id = ?", params.UserID)
	}

	if params.Search != "" {
		db = db.Joins("LEFT JOIN users ON users.id = notify_emails.sender_id")
		db = db.Where("notify_emails.title LIKE ? OR users.full_name LIKE ?", fmt.Sprintf("%%%s%%", params.Search), fmt.Sprintf("%%%s%%", params.Search))
	}

	if params.Limit > 0 {
		db = db.Limit(params.Limit)
	}

	if params.Offset > 0 {
		db = db.Offset(params.Offset)
	}
	db = db.Order("id DESC")
	return db
}

func (m *NotifyManager) GetNotifyEmails(options NotifyEmailQueryParams) ([]entity.NotifyEmail, error) {
	db := m.buildQueryNotifyEmail(options)
	var notifies []entity.NotifyEmail
	db = db.Preload("Sender").Preload("Creator").Find(&notifies)
	return notifies, db.Error
}

func (m *NotifyManager) GetNotifyEmail(options NotifyEmailQueryParams) (*entity.NotifyEmail, error) {
	db := m.buildQueryNotifyEmail(options)
	var notify *entity.NotifyEmail
	db = db.First(&notify)
	return notify, db.Error
}

func (m *NotifyManager) CountNotifyEmail(options NotifyEmailQueryParams) (int64, error) {
	db := m.buildQueryNotifyEmail(options)
	var count int64
	db = db.Model(&entity.NotifyEmail{}).Count(&count)
	return count, db.Error
}

func (m *NotifyManager) SaveNotifyEmail(noti *entity.NotifyEmail) error {
	return m.DB.Save(noti).Error
}

func (m *NotifyManager) GetListNotifyCustomer(params NotifyQueryParams) ([]entity.Notification, error) {
	var notifies []entity.Notification

	db := m.buildQueryNotify(params)
	db = db.Preload("Sender").Preload("Creator")
	db = db.Order("id DESC")
	db = db.Find(&notifies)
	return notifies, db.Error
}
