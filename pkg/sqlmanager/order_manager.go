package sqlmanager

import (
	"log"
	"tebexpressapi/pkg/models/dto"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/utils"
	"time"

	"gorm.io/gorm"
)

type OrderManager struct {
	db *gorm.DB
}

type OrderQueryOption struct {
	ID        int64
	UserID    int64
	Search    string
	Status    int
	ArrStatus []int
	Limit     int
	Offset    int
	StartDate string
	EndDate   string
}

func NewOrderManager(db *gorm.DB) *OrderManager {
	return &OrderManager{db: db}
}

func (m *OrderManager) buildQuery(opts OrderQueryOption) *gorm.DB {
	db := m.db

	if opts.ID > 0 {
		db = db.Where("orders.id=?", opts.ID)
	}

	if opts.Status > 0 {
		db = db.Where("orders.status=?", opts.Status)
	}

	if len(opts.ArrStatus) > 0 {
		db = db.Where("orders.status IN (?)", opts.ArrStatus)
	}

	if opts.UserID > 0 {
		db = db.Where("orders.user_id=?", opts.UserID)
	}

	if len(opts.StartDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(orders.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)", opts.StartDate)
	}

	if len(opts.EndDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(orders.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)", opts.EndDate)
	}

	if opts.Search != "" {
		sub1 := m.db.Table("package_codes").Select("id").Where("code=?", opts.Search).Limit(1)
		sub2 := m.db.Table("trackings").Select("package_id").Where("tracking_number=?", opts.Search).Limit(1)
		sub := m.db.Table("packages").Select("order_id").Where("package_code_id=(?) OR id=(?)", sub1, sub2).Limit(1)
		db = db.Where("orders.id=(?) OR orders.id=?", sub, opts.Search)
	}

	return db
}

func (m *OrderManager) GetList(opts OrderQueryOption) ([]entity.Order, error) {
	db := m.buildQuery(opts)
	db = db.Order("orders.id DESC")

	db = db.Joins("LEFT JOIN packages ON packages.order_id=orders.id")
	db = db.Select(`
		orders.id,
		orders.user_id,
		orders.status,
		orders.created_at,
		orders.updated_at,
		count(packages.id) AS count
	`).Group("orders.id")

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	orders := []entity.Order{}
	db = db.Find(&orders)

	return orders, db.Error
}

func (m *OrderManager) GetCount(opts OrderQueryOption) (int64, error) {
	db := m.buildQuery(opts)

	var count int64
	db = db.Model(&entity.Order{}).Count(&count)
	return count, db.Error
}

func (m *OrderManager) StaticsStatus(opts OrderQueryOption) ([]dto.CountStatusOrder, error) {
	db := m.buildQuery(opts)

	result := []dto.CountStatusOrder{}
	db = db.Model(&entity.Order{}).Group("status").Select("status, count(*) as count").Scan(&result)
	return result, db.Error
}

func (m *OrderManager) Create(order *entity.Order, packageIDs []int64) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	order.CreatedAt = time.Now()
	order.UpdatedAt = order.CreatedAt

	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		return err
	}

	mapChange := map[string]interface{}{
		"order_id":   utils.Int64(order.ID),
		"updated_at": time.Now(),
	}

	if err := tx.Model(&entity.Package{}).Where("id IN (?)", packageIDs).Updates(mapChange).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m *OrderManager) GetByID(id int64) (*entity.Order, error) {
	order := &entity.Order{}
	db := m.db.Where("id=?", id).First(order)
	return order, db.Error
}

func (m *OrderManager) Update(order *entity.Order) error {
	order.UpdatedAt = time.Now()
	return m.db.Save(order).Error
}

type OrderPackageQueryOption struct {
	OrderID      int64
	UserID       int64
	PackageIDs   []int64
	PackageCodes []string
	Status       int
	ArrStatus    []int
	HasOrder     bool
	Limit        int
	Offset       int
	Search       string
}

func (m *OrderManager) buildPackageQuery(opts OrderPackageQueryOption) *gorm.DB {
	db := m.db

	if opts.OrderID > 0 {
		db = db.Where("packages.order_id=?", opts.OrderID)
	}

	if opts.UserID > 0 {
		db = db.Where("packages.user_id=?", opts.UserID)
	}

	if len(opts.PackageIDs) > 0 {
		db = db.Where("packages.id IN (?)", opts.PackageIDs)
	}

	if opts.Status > 0 {
		db = db.Where("packages.status=?", opts.Status)
	}

	if len(opts.ArrStatus) > 0 {
		db = db.Where("packages.status IN (?)", opts.ArrStatus)
	}

	if opts.HasOrder {
		db = db.Where("packages.order_id > 0 AND packages.order_id IS NOT NULL")
	}

	return db
}

func (m *OrderManager) GetListPackageItems(opts OrderPackageQueryOption) ([]dto.OrderPackage, error) {
	db := m.buildPackageQuery(opts)

	db = db.Select(`
		packages.id id,
		packages.order_number order_number,
		package_codes.code code,
		trackings.tracking_number tracking_number,
		packages.order_id order_id,
		packages.status status,
		packages.created_at created_at,
		services.code service_code
	`)

	db = db.Joins("INNER JOIN package_codes ON package_codes.id=packages.package_code_id")
	db = db.Joins("LEFT JOIN trackings ON trackings.package_id=packages.id")
	db = db.Joins("LEFT JOIN services ON services.id = packages.service_id")

	if len(opts.PackageCodes) > 0 {
		db = db.Where("package_codes.code IN (?)", opts.PackageCodes)
	}

	if opts.Search != "" {
		db = db.Where("package_codes.code=? OR trackings.tracking_number=? OR packages.id=?", opts.Search, opts.Search, opts.Search)
	}

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	packages := []dto.OrderPackage{}
	db = db.Order("packages.id DESC")

	db = db.Model(&entity.Package{}).Scan(&packages)
	return packages, db.Error
}

func (m *OrderManager) GetCountPackageItems(opts OrderPackageQueryOption) (int64, error) {
	db := m.buildPackageQuery(opts)

	if len(opts.PackageCodes) > 0 || opts.Search != "" {
		db = db.Joins("INNER JOIN package_codes ON package_codes.id=packages.package_code_id")
	}

	if len(opts.PackageCodes) > 0 {
		db = db.Where("package_codes.code IN (?)", opts.PackageCodes)
	}

	if opts.Search != "" {
		db = db.Joins("LEFT JOIN trackings ON trackings.package_id=packages.id")
		db = db.Where("package_codes.code=? OR trackings.tracking_number=? OR packages.id=?", opts.Search, opts.Search, opts.Search)
	}

	var count int64
	db = db.Model(&entity.Package{}).Count(&count)
	return count, db.Error
}

func (m *OrderManager) GetCountStatusPackageItems(opts OrderPackageQueryOption) ([]dto.CountOrderPackage, error) {
	db := m.buildPackageQuery(opts)

	result := []dto.CountOrderPackage{}
	db = db.Model(&entity.Package{}).Select("status, COUNT(id) as total").Group("status").Scan(&result)

	return result, db.Error
}

func (m *OrderManager) Delete(orderID int64) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	mapChange := map[string]interface{}{"order_id": nil, "updated_at": time.Now()}
	if err := tx.Model(&entity.Package{}).Where("order_id=?", orderID).Updates(mapChange).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Delete(&entity.Order{}, orderID).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
