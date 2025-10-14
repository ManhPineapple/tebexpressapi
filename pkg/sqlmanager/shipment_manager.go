package sqlmanager

import (
	"fmt"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/dto"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/utils/dbgorm"
	"time"

	"github.com/spf13/cast"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ShipmentManager struct {
	db *gorm.DB
}
type ShipmentQueryOptions struct {
	ID                     int64
	Search                 string
	ContainerCode          string
	IDSContainer           []int64
	HubID                  int64
	LimitContainer         int
	OffsetContainer        int
	LoadContainer          bool
	LoadPackage            bool
	LoadWarehouseContainer bool
	Limit                  int
	Offset                 int
	Status                 int
	LoadManifest           bool
	WarehouseID            int64
	IsFba                  int
	PackageID              int64
}

func NewShipmentManager(db *gorm.DB) *ShipmentManager {
	return &ShipmentManager{
		db: db,
	}
}
func (m *ShipmentManager) BuildShipmentQuery(opts ShipmentQueryOptions) *gorm.DB {
	db := m.db

	if opts.ID > 0 {
		db = db.Where("shipments.id = ?", opts.ID)
	}
	if opts.Status > 0 {
		db = db.Where("shipments.status = ?", opts.Status)
	}
	if len(opts.IDSContainer) > 0 {
		db = db.Joins("LEFT JOIN containers c ON shipments.id = c.shipment_id")
		db = db.Where("c.id IN ?", opts.IDSContainer)
		db = db.Group("shipments.id")

	}
	if len(opts.Search) > 0 {
		if cast.ToInt(opts.Search) > 0 {
			db = db.Where("shipments.id = ?", opts.Search)
		} else {
			db = db.Where("shipments.id IN (?)", m.db.Model(&entity.Container{}).Select("shipment_id").Where("code = ?", opts.Search))
		}
	}

	if opts.HubID > 0 {
		db = db.Joins("LEFT JOIN warehouses wh1 ON wh1.id = shipments.hub_id")
		// db = db.Where("wh1.status = ?", constant.WareHouseStatusActive)
		db = db.Where("shipments.hub_id = ?", opts.HubID)
	}

	if opts.WarehouseID > 0 {
		db = db.Joins("LEFT JOIN warehouses wh2 ON wh2.id = shipments.warehouse_id")
		db = db.Where("shipments.warehouse_id = ?", opts.WarehouseID)
	}

	if opts.IsFba > 0 {
		db = db.Where("fba_type > ?", 0)
	}

	if opts.IsFba < 0 {
		db = db.Where("fba_type = ? OR fba_type IS NULL", 0)
	}

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	return db
}

func (m *ShipmentManager) GetShipment(opts ShipmentQueryOptions) (*entity.Shipment, error) {
	db := m.BuildShipmentQuery(opts)
	shipment := &entity.Shipment{}
	if opts.LoadContainer {
		db = db.Preload("Containers", func(db *gorm.DB) *gorm.DB {
			if opts.WarehouseID > 0 {
				db = db.Where("warehouse_id = ?", opts.WarehouseID)
			}
			if len(opts.ContainerCode) > 0 {
				db = db.Where("code = ?", opts.ContainerCode)
			}
			if opts.LimitContainer > 0 {
				db = db.Limit(opts.LimitContainer)
			}
			if opts.OffsetContainer > 0 {
				db = db.Offset(opts.OffsetContainer)
			}
			db = db.Preload("ContainerItems", func(db *gorm.DB) *gorm.DB {
				db = db.Joins("JOIN packages ON packages.id = container_items.package_id")
				db = db.Where("container_items.status = ?", constant.ContainerItemActive)
				if opts.PackageID > 0 {
					db = db.Where("container_items.package_id = ?", opts.PackageID)
				}
				return db
			})
			if opts.LoadPackage {
				db = db.Preload("ContainerItems.Package")
				db = db.Preload("ContainerItems.Package.Warehouse")
			}
			db = db.Order("id DESC")
			return db
		})
		if opts.LoadWarehouseContainer {
			db = db.Preload("Containers.Warehouse")
		}
	}

	if opts.LoadManifest {
		db = db.Preload("Manifest")
	}
	db = db.First(shipment)
	return shipment, db.Error
}

func (m ShipmentManager) GetShipments(options ShipmentQueryOptions) ([]entity.Shipment, error) {
	db := m.BuildShipmentQuery(options)
	var shipments []entity.Shipment
	db = db.Preload("Warehouse", func(db *gorm.DB) *gorm.DB {
		db = db.Where("warehouses.status = ?", constant.WareHouseStatusActive)
		return db
	})
	db = db.Preload("Containers")
	db = db.Order("id DESC").Find(&shipments)
	return shipments, db.Error
}
func (m ShipmentManager) SaveShipment(shipment *entity.Shipment) error {
	db := m.db.Save(shipment)
	return db.Error
}
func (m ShipmentManager) CountShipments(options ShipmentQueryOptions) (int64, error) {
	db := m.BuildShipmentQuery(options)
	var count int64
	db = db.Model(&entity.Shipment{}).Count(&count)
	return count, db.Error
}

func (m ShipmentManager) CountAllStatusShipment(options ShipmentQueryOptions) ([]dto.CountStatusShipment, error) {
	options.Status = 0
	db := m.BuildShipmentQuery(options)
	db = db.Model(&entity.Shipment{}).Select("shipments.status, COUNT(shipments.id) as count")
	db = db.Group("status")
	var countStatus []dto.CountStatusShipment
	db = db.Scan(&countStatus)
	return countStatus, db.Error
}

func (m *ShipmentManager) GetContainerInShipment(opts ShipmentQueryOptions, result interface{}) error {
	db := m.db
	db = db.Joins("LEFT JOIN container_items ON containers.id = container_items.container_id AND container_items.status = ?", constant.ContainerItemActive)
	db = db.Group("containers.id")
	db = db.Where("shipment_id = ?", opts.ID)
	db = db.Select("COUNT(container_items.id) as count_items,containers.id,containers.code,containers.width,containers.height,containers.length,containers.weight,containers.tracking_number,containers.created_at,containers.type,containers.label_url")
	db = db.Limit(opts.LimitContainer)
	db = db.Offset(opts.OffsetContainer)
	db = db.Model(&entity.Container{}).Find(result)
	return db.Error
}

func (m *ShipmentManager) CountContainerShipment(opts ShipmentQueryOptions) (int64, error) {
	db := m.db
	db = db.Where("shipment_id = ?", opts.ID)
	if len(opts.ContainerCode) > 0 {
		db = db.Where("code = ?", opts.ContainerCode)
	}
	var count int64
	db = db.Model(&entity.Container{}).Count(&count)
	return count, db.Error
}
func (m ShipmentManager) CloseShipment(shipment *entity.Shipment, labelApiContainers []entity.Container, userID int64) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Omit(clause.Associations).Save(shipment).Error; err != nil {
		tx.Rollback()
		return err
	}

	if len(labelApiContainers) > 0 {
		if err := tx.Omit(clause.Associations).Save(labelApiContainers).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	var packageIDs []int64
	var logsWareHouseExport []entity.PackageDeliverLog

	trackings := []entity.Tracking{}
	itemsAPI := make(map[int64]bool)
	if shipment.FbaType > 0 {
		for _, container := range labelApiContainers {
			for _, containerItem := range container.ContainerItems {
				itemsAPI[containerItem.PackageID] = true
				trackings = append(trackings, entity.Tracking{
					PackageID:      containerItem.PackageID,
					TrackingNumber: container.TrackingNumber,
					Status:         constant.TrackingStatusSuccess,
					HubID:          nil,
					Model: dbgorm.Model{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				})
			}
		}
	}

	for _, container := range shipment.Containers {
		for _, containerItem := range container.ContainerItems {
			packageIDs = append(packageIDs, containerItem.PackageID)
			logsWareHouseExport = append(logsWareHouseExport, entity.PackageDeliverLog{
				PackageID: containerItem.PackageID,
				Status:    constant.DeliverLogTebexpressInTransit,
				Type:      constant.PackageDeliverLogTypeWareHouseExport,
				Location:  fmt.Sprintf("%s, %s", container.Warehouse.City, container.Warehouse.Country),
				UserID:    &userID,
			})

			if !itemsAPI[containerItem.PackageID] {
				trackings = append(trackings, entity.Tracking{
					PackageID:      containerItem.PackageID,
					TrackingNumber: container.TrackingNumber,
					Status:         constant.TrackingStatusSuccess,
					HubID:          nil,
					Model: dbgorm.Model{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				})
			}
		}
	}

	if shipment.FbaType > 0 && len(trackings) > 0 {
		if err := tx.Create(&trackings).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Model(&entity.Package{}).Where("id IN (?)", packageIDs).
		Updates(map[string]interface{}{
			"updated_at": time.Now(),
			"status":     constant.PackageStatusWareHouseExport,
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if len(logsWareHouseExport) > 1000 {
		chunk := 1000
		count := 1
		i := 0
	SplitData:
		data := make([]entity.PackageDeliverLog, 0)
		for i < len(logsWareHouseExport) {
			data = append(data, logsWareHouseExport[i])
			i++
			if i > chunk*count {
				count++
				break
			}
		}

		if err := tx.Save(&data).Error; err != nil {
			tx.Rollback()
			return err
		}
		if i < len(logsWareHouseExport) {
			goto SplitData
		}
	} else {
		if err := tx.Save(&logsWareHouseExport).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m ShipmentManager) IntransitShipment(pkgs []entity.Package, shipment *entity.Shipment, userID int64) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Omit(clause.Associations).Save(shipment).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Omit(clause.Associations).Save(shipment.Containers).Error; err != nil {
		tx.Rollback()
		return err
	}
	var ids []int64
	for _, pkg := range pkgs {
		ids = append(ids, pkg.ID)
	}
	if err := tx.Model(&entity.Package{}).
		Where("id IN (?)", ids).
		Updates(map[string]interface{}{
			"status":     constant.PackageStatusInTransit,
			"updated_at": time.Now(),
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	var logsInTransit []entity.PackageDeliverLog
	var idx int = 0
	for _, pkg := range pkgs {
		logsInTransit = append(logsInTransit, entity.PackageDeliverLog{
			PackageID: pkg.ID,
			Status:    constant.DeliverLogTebexpressInTransit,
			Type:      constant.PackageDeliverLogTypeInTransit,
			Location:  fmt.Sprintf("%s, %s", pkg.Warehouse.City, pkg.Warehouse.Country),
			UserID:    &userID,
		})

		idx++

		if idx < 200 {
			continue
		}

		if err := tx.Save(&logsInTransit).Error; err != nil {
			tx.Rollback()
			return err
		}

		idx = 0
		logsInTransit = []entity.PackageDeliverLog{}
	}

	if len(logsInTransit) > 0 {
		if err := tx.Save(&logsInTransit).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m *ShipmentManager) GetShipmentExport(result interface{}, id int64) error {
	db := m.db
	sqlString := `SELECT
	packages.id,packages.order_number,packages.detail,packages.shipping_fee,packages.recipient,packages.address_1,packages.status, package_codes.code,trackings.tracking_number,trackings.height,trackings.width,trackings.length,trackings.weight,containers.code as container_code,users.full_name
FROM 
	containers
LEFT JOIN container_items ON containers.id = container_items.container_id
LEFT JOIN packages ON packages.id = container_items.package_id
LEFT JOIN users ON packages.user_id = users.id
LEFT JOIN package_codes ON packages.package_code_id = package_codes.id
LEFT JOIN trackings ON trackings.package_id = packages.id

WHERE 
containers.shipment_id = ? AND container_items.status = ? AND trackings.status != ?
`
	db = db.Raw(sqlString, id, constant.ContainerItemActive, constant.TrackingStatusCanceled).Scan(result)
	return db.Error
}

func (m ShipmentManager) GetShipmentFirstPackage(shipmentID int64) (*entity.Package, error) {
	pkg := &entity.Package{}

	db := m.db.Where("containers.shipment_id = ? AND container_items.status = ?", shipmentID, constant.ContainerItemActive)
	db = db.Where("packages.status NOT IN (?)", []int{constant.PackageStatusCancelled})
	db = db.Joins("INNER JOIN container_items ON container_items.package_id = packages.id")
	db = db.Joins("INNER JOIN containers ON container_items.container_id = containers.id")
	db = db.Select(`
		packages.id,
		packages.order_number,
		packages.recipient,
		packages.company,
		packages.phone_number,
		packages.address_1,
		packages.address_2,
		packages.city,
		packages.state_code,
		packages.zipcode,
		packages.country_code,
		packages.status
	`)

	db = db.First(pkg)
	return pkg, db.Error
}
