package sqlmanager

import (
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/dto"
	"tebexpressapi/pkg/models/entity"

	"gorm.io/gorm/clause"

	"gorm.io/gorm"
)

type WareHouseManager struct {
	db *gorm.DB
}

func NewWareHouseManager(db *gorm.DB) *WareHouseManager {
	return &WareHouseManager{
		db: db,
	}
}

type OptionQueryWareHouse struct {
	Status          int64
	ID              int64
	IDs             []int64
	Limit           int
	Offset          int
	Code            string
	IgnoreUsers     []int64
	WarehouseID     int64
	IsWarehouseRole bool
}

func (m *WareHouseManager) buildQueryOption(opts OptionQueryWareHouse) *gorm.DB {
	db := m.db

	db = db.Joins("LEFT JOIN package_codes ON package_codes.id = packages.package_code_id")

	if opts.ID > 0 {
		db = db.Where("packages.id = ?", opts.ID)
	}

	if len(opts.IDs) > 0 {
		db = db.Where("packages.id IN (?)", opts.IDs)
	}

	if len(opts.IgnoreUsers) > 0 {
		db = db.Where("packages.user_id NOT IN (?)", opts.IgnoreUsers)
	}

	if opts.Status > 0 {
		db = db.Where("packages.status = ?", opts.Status)
	}

	if opts.Code != "" {
		var ids []int64
		m.db.Table("trackings").Where("tracking_number=? AND status != ?", opts.Code, constant.TrackingStatusCanceled).Pluck("package_id", &ids)

		if len(ids) > 0 {
			db = db.Where("package_codes.code = ? or packages.id IN (?)", opts.Code, ids)
		} else {
			db = db.Where("package_codes.code = ?", opts.Code)
		}
	}

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	if opts.WarehouseID > 0 {
		if opts.IsWarehouseRole {
			subquery := m.db.Model(&entity.Warehouse{}).Where("type = ?", constant.WareHouseTypeInternational).Where("warehouses.status = ?", constant.WareHouseStatusActive).Select("id")
			db = db.Where("packages.warehouse_id IN (?) or packages.warehouse_id = ?", subquery, opts.WarehouseID)
		} else {
			db = db.Where("packages.warehouse_id = ?", opts.WarehouseID)
		}

	}

	return db
}

func (m *WareHouseManager) GetPackagesInWareHouse(opts OptionQueryWareHouse) ([]*entity.PackageInWarehouse, error) {
	var listPackages []*entity.PackageInWarehouse

	db := m.buildQueryOption(opts).
		Table("packages").
		Select(`
					packages.id,
					package_codes.code,
					order_number,
					state_code,
					country_code,
					zipcode,
					city,
					packages.status,
					packages.checkin_warehouse_at,
					container_items.container_id as container_id,
					containers.code as container_code,
					containers.tracking_number as container_tracking_number,
					containers.label_url as container_label_url,
					containers.shipment_id as shipment_id,
					services.code service_code, services.name service_name`).
		Joins("LEFT JOIN services ON packages.service_id = services.id").
		Joins("LEFT JOIN container_items ON container_items.package_id = packages.id AND (container_items.status = ? or container_items.status is null)", constant.ContainerItemActive).
		Joins("LEFT JOIN containers ON container_items.container_id = containers.id").
		Preload("Tracking", func(db *gorm.DB) *gorm.DB {
			db = db.Where("trackings.status != ?", constant.TrackingStatusCanceled)
			return db
		}).
		Where("packages.status IN (?)", constant.WarehouseStatus).
		Order("packages.id DESC").
		Find(&listPackages)

	return listPackages, db.Error
}

func (m *WareHouseManager) CountPackagesInWareHouse(opts OptionQueryWareHouse) (int64, error) {
	var countPackage int64

	db := m.buildQueryOption(opts).
		Table("packages").
		Joins("LEFT JOIN services ON packages.service_id = services.id").
		Joins("LEFT JOIN container_items ON container_items.package_id = packages.id AND (container_items.status = ? or container_items.status is null)", constant.ContainerItemActive).
		Joins("LEFT JOIN containers ON container_items.container_id = containers.id").
		Where("packages.status IN (?)", constant.WarehouseStatus).
		Count(&countPackage)

	return countPackage, db.Error
}

func (m *WareHouseManager) CountAllStatusPackages(opts OptionQueryWareHouse) ([]dto.CountStatusPackage, error) {
	opts.Status = 0
	db := m.buildQueryOption(opts)
	db = db.Model(&entity.Package{}).Select("packages.status, COUNT(packages.id) as count").
		Where("packages.status IN (?) ", constant.WarehouseStatus)
	db = db.Group("packages.status")
	countStatus := []dto.CountStatusPackage{}
	db = db.Scan(&countStatus)
	return countStatus, db.Error
}

type OptionWareHouse struct {
	Status       int64
	ID           int64
	IDs          []int64
	Limit        int
	Offset       int
	Type         int64
	IsHasDisable bool
	Country      string
	State        string
	IgnoreState  string
}

func (m *WareHouseManager) buildWareHouseOption(opts OptionWareHouse) *gorm.DB {
	db := m.db

	if opts.ID > 0 {
		db = db.Where("warehouses.id = ?", opts.ID)
	}

	if len(opts.IDs) > 0 {
		db = db.Where("warehouses.id IN (?)", opts.IDs)
	}

	if opts.Status > 0 {
		db = db.Where("warehouses.status = ?", opts.Status)
	} else if !opts.IsHasDisable {
		db = db.Where("warehouses.status > 0")
	}

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	if opts.Type > 0 {
		db = db.Where("warehouses.type = ?", opts.Type)
	}

	if opts.Country != "" {
		db = db.Where("warehouses.country = ?", opts.Country)
	}

	if opts.State != "" {
		db = db.Where("warehouses.state = ?", opts.State)
	}

	if opts.IgnoreState != "" {
		db = db.Where("warehouses.state <> ?", opts.IgnoreState)
	}

	return db
}

func (m *WareHouseManager) GetWareHouses(opts OptionWareHouse) ([]entity.Warehouse, error) {
	var wareHouses = make([]entity.Warehouse, 0)
	db := m.buildWareHouseOption(opts)
	db = db.Model(&entity.Warehouse{}).Find(&wareHouses)
	return wareHouses, db.Error
}

func (m *WareHouseManager) GetWareHouse(opts OptionWareHouse) (*entity.Warehouse, error) {
	var wareHouse = &entity.Warehouse{}
	db := m.buildWareHouseOption(opts)
	db = db.Model(&entity.Warehouse{}).First(&wareHouse)
	return wareHouse, db.Error
}

func (m *WareHouseManager) CreateEstimateCost(cost *entity.PackageWarehouseCost) error {
	db := m.db.Model(&entity.PackageWarehouseCost{}).Omit(clause.Associations).Create(&cost)
	return db.Error
}

func (m *WareHouseManager) UpdateEstimateCost(cost *entity.PackageWarehouseCost, change map[string]interface{}) error {
	db := m.db.Model(&entity.PackageWarehouseCost{}).Where("package_id=? AND hub_id = ?", cost.PackageID, cost.HubID).UpdateColumns(change)
	return db.Error
}

func (m *WareHouseManager) DeactivateOldCost(pkg entity.Package) error {

	db := m.db.Table("package_warehouse_costs").
		Where("package_id = ?", pkg.ID).
		Delete(entity.PackageWarehouseCost{})

	return db.Error
}

func (m *WareHouseManager) GetEstimateCostByWareHouse(packageID int64) ([]entity.PackageWarehouseCost, error) {
	var wareHouses = make([]entity.PackageWarehouseCost, 0)
	db := m.db
	db = db.Model(&entity.PackageWarehouseCost{}).
		Where("package_id = ? AND hub_id IN (?)", packageID, m.db.Model(&entity.Warehouse{}).Select("id").Where("status = ?", constant.WareHouseStatusActive)).
		Preload("Warehouse", func(db *gorm.DB) *gorm.DB {
			return db.Where("warehouses.status = ?", constant.WareHouseStatusActive)
		}).
		Find(&wareHouses)
	return wareHouses, db.Error
}

func (m *WareHouseManager) GetWareHouseMostPrice(pkgID int64) (*entity.Warehouse, error) {
	var warehouse = &entity.Warehouse{}

	db := m.db.Select("warehouses.*")
	db = db.Joins("INNER JOIN package_warehouse_costs ON package_warehouse_costs.hub_id=warehouses.id")
	db = db.Where("package_warehouse_costs.package_id=? AND warehouses.status=? AND warehouses.type=?", pkgID, constant.WareHouseStatusActive, 1)
	db = db.Order("package_warehouse_costs.cost ASC").Limit(1)

	db = db.First(warehouse)
	return warehouse, db.Error
}

func (m *WareHouseManager) GetEstimateCosts(packageID int64, country string) ([]entity.PackageWarehouseCost, error) {
	var costs []entity.PackageWarehouseCost

	db := m.db.Select("package_warehouse_costs.*")
	db = db.Joins("INNER JOIN warehouses ON warehouses.id=package_warehouse_costs.hub_id")
	db = db.Where("package_id = ?", packageID)
	db = db.Where("warehouses.status = ?", constant.WareHouseStatusActive)

	if country != "" {
		db = db.Where("warehouses.country = ?", country)
	}

	db = db.Preload("Warehouse")
	db = db.Find(&costs)
	return costs, db.Error
}

func (m *WareHouseManager) GetMostEstimateCost(packageID int64) (*entity.PackageWarehouseCost, error) {
	most := &entity.PackageWarehouseCost{}

	db := m.db.Model(&entity.PackageWarehouseCost{})
	db = db.Where("package_id = ? AND hub_id IN (?)", packageID, m.db.Model(&entity.Warehouse{}).Select("id").Where("status = ?", constant.WareHouseStatusActive)).
		Preload("Warehouse", func(db *gorm.DB) *gorm.DB {
			return db.Where("warehouses.status = ?", constant.WareHouseStatusActive)
		})

	db = db.Order("package_warehouse_costs.cost ASC").Limit(1)
	db = db.First(most)
	return most, db.Error
}
