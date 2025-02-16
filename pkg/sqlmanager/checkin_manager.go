package sqlmanager

import (
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/utils/dbgorm"
	"time"

	"gorm.io/gorm"
)

type CheckinManager struct {
	db *gorm.DB
}

func NewCheckinManager(db *gorm.DB) *CheckinManager {
	return &CheckinManager{
		db: db,
	}
}

func (m CheckinManager) FetchCheckinRequest(warehouseID int) (*entity.CheckinRequest, error) {
	checkin := &entity.CheckinRequest{}
	db := m.db

	if warehouseID > 0 {
		db = db.Preload("CheckinPackage").Preload("CheckinPackage.Package", func(db *gorm.DB) *gorm.DB {
			db = db.Joins("JOIN warehouses ON packages.warehouse_id = warehouses.id")
			db = db.Where("warehouses.status = ?", constant.WareHouseStatusActive)
			db = db.Where("packages.warehouse_id = ?", warehouseID)
			return db
		})
	} else {
		db = db.Preload("CheckinPackage").Preload("CheckinPackage.Package")
	}
	db = db.Where("status = ?", constant.CheckinStatusAwaiting).Preload("CheckinPackage.Package.User").Preload("CheckinPackage.Package.PackageCode").Preload("CheckinPackage.Package.Tracking")

	err := db.First(checkin).Error
	if err == gorm.ErrRecordNotFound {
		checkin = &entity.CheckinRequest{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Status: constant.CheckinStatusAwaiting,
		}

		if err := m.db.Create(checkin).Error; err != nil {
			return nil, err
		}

		return checkin, nil
	}

	return checkin, err
}

func (m *CheckinManager) CloseCheckin(ID int64, UserID int64) (int64, error) {
	db := m.db.Model(&entity.CheckinRequest{}).Where("id = ?", ID)

	db = db.Updates(&entity.CheckinRequest{Status: constant.CheckinStatusClosed, CloseUserID: UserID})

	return db.RowsAffected, db.Error
}

func (m CheckinManager) GetCheckinByID(id int64) (*entity.CheckinRequest, error) {
	checkin := &entity.CheckinRequest{}
	db := m.db.Where("id=?", id).First(checkin)
	return checkin, db.Error
}
