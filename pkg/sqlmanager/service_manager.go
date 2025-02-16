package sqlmanager

import (
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"time"

	"gorm.io/gorm"
)

type ServiceManager struct {
	db *gorm.DB
}

type ServiceQueryOption struct {
	ID     int64
	IDS    []int64
	Status int
	Search string
	Limit  int
	Offset int

	IsHasPrice bool
	PriceClass int64
	PartnerID  int64
}

func NewServiceManager(db *gorm.DB) *ServiceManager {
	return &ServiceManager{
		db: db,
	}
}

func (m *ServiceManager) buildServiceQuery(opts ServiceQueryOption) *gorm.DB {
	db := m.db

	if opts.ID > 0 {
		db = db.Where("id=?", opts.ID)
	}

	if len(opts.IDS) > 0 {
		db = db.Where("id IN (?)", opts.IDS)
	}

	if opts.Status > 0 {
		db = db.Where("status=?", opts.Status)
	}

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Search != "" {
		db = db.Where("name like ? or code like ?", opts.Search, opts.Search)
	}

	return db
}

func (m *ServiceManager) GetServices(opts ServiceQueryOption) ([]*entity.Service, error) {
	db := m.buildServiceQuery(opts)
	var services []*entity.Service
	db = db.Preload("DomesticCarrier")

	if opts.PartnerID > 0 {
		db = db.Where("partner_id=?", opts.PartnerID)
	}

	if opts.IsHasPrice {
		db = db.Preload("Prices", func(db *gorm.DB) *gorm.DB {
			if opts.PriceClass > 0 {
				db = db.Where("user_class=?", opts.PriceClass)
			}

			return db
		})
	}

	db = db.Find(&services)
	return services, db.Error
}

func (m *ServiceManager) GetServiceImportPackage(mapStrings []string) ([]entity.Service, error) {
	db := m.db
	db = db.Where("name IN (?) or code IN (?)", mapStrings, mapStrings)
	db = db.Where("status=?", constant.StatusActive)
	services := []entity.Service{}
	db = db.Preload("DomesticCarrier").Find(&services)
	return services, db.Error
}

func (m *ServiceManager) GetServiceByKeyword(q string) (*entity.Service, error) {
	service := &entity.Service{}
	db := m.db.Where("name=? or code=?", q, q).
		Where("status=?", constant.StatusActive).
		Preload("DomesticCarrier").
		First(service)
	return service, db.Error
}

func (m *ServiceManager) GetServiceByID(id int64) (*entity.Service, error) {
	service := &entity.Service{}
	db := m.db.Where("id=?", id).
		Where("status=?", constant.StatusActive).
		Preload("DomesticCarrier").Preload("WWCarrier").First(service)
	return service, db.Error
}

func (m *ServiceManager) GetServiceByCode(q string) (*entity.Service, error) {
	service := &entity.Service{}
	db := m.db.Where(" (code=? OR name = ?) AND status=?", q, q, constant.StatusActive).Preload("DomesticCarrier").First(service)
	return service, db.Error
}

func (m *ServiceManager) GetCarrierByID(id int64) (*entity.Carrier, error) {
	carrier := &entity.Carrier{}
	db := m.db.Where("id = ? AND status=?", id, constant.StatusActive).First(carrier)
	return carrier, db.Error
}

func (m *ServiceManager) UpdatePrices(prices map[int64]map[string]interface{}) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return nil
	}

	for id, price := range prices {
		if err := tx.Table("prices").Where("id=?", id).Updates(price).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m *ServiceManager) CreateExchangeRateLog(log *entity.ExchangeRateLog) error {
	log.CreatedAt = time.Now()
	log.UpdatedAt = log.CreatedAt
	return m.db.Create(log).Error
}

func (m *ServiceManager) GetCarrierByCode(code string) (*entity.Carrier, error) {
	carrier := &entity.Carrier{}
	db := m.db.Where("code = ? AND status=?", code, constant.StatusActive).First(carrier)
	return carrier, db.Error
}

func (m *ServiceManager) CreateCheckPriceLogs(log *entity.CheckPriceLog) error {
	log.CreatedAt = time.Now()
	log.UpdatedAt = log.CreatedAt
	return m.db.Create(log).Error
}

type CheckPriceLogOption struct {
	Limit  int
	Offset int
}

func (m *ServiceManager) FetchCheckPriceLogs(opts CheckPriceLogOption) ([]entity.CheckPriceLog, error) {
	db := m.db
	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	db = db.Order("created_at DESC")

	var logs = []entity.CheckPriceLog{}
	db = db.Find(&logs)

	return logs, db.Error
}

func (m *ServiceManager) FetchCountCheckPriceLogs() (int64, error) {
	var count int64 = 0
	db := m.db.Model(&entity.CheckPriceLog{}).Count(&count)
	return count, db.Error
}

func (m *ServiceManager) GetCarriersByCodes(codes []string) ([]entity.Carrier, error) {
	carriers := []entity.Carrier{}
	db := m.db.Where("code IN (?) AND status=?", codes, constant.StatusActive).Find(&carriers)
	return carriers, db.Error
}
