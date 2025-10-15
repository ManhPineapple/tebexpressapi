package sqlmanager

import (
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/dto"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/dbgorm"
	"time"

	"github.com/spf13/cast"

	"gorm.io/gorm"
)

type ContainerManager struct {
	db *gorm.DB
}

type ContainerQueryOptions struct {
	Status          int
	Search          string
	Code            string
	Codes           []string
	PackageCode     string
	Barcode         string
	LoadPackage     bool
	LimitPackage    int
	OffsetPackage   int
	ID              int64
	IDs             []int64
	Limit           int
	Offset          int
	IgnorePerload   bool
	HubID           int64
	NotInShipment   bool
	WarehouseID     int64
	Type            int
	OrderByUpdated  bool
	HasHistory      bool
	FbaType         int
	ShipmentID      int64
	IsWarningWeight bool
}

type ContainerBoxQueryOptions struct {
	ID  int64
	IDS []int64
}

func NewContainerManager(db *gorm.DB) *ContainerManager {
	return &ContainerManager{
		db: db,
	}
}

func (m ContainerManager) BuildContainerQuery(options ContainerQueryOptions) *gorm.DB {
	db := m.db

	if options.ID > 0 {
		db = db.Where("containers.id = ?", options.ID)
	}

	if len(options.IDs) > 0 {
		db = db.Where("containers.id IN (?)", options.IDs)
	}

	if options.Status > 0 {
		if options.Status == constant.ContainerDelivered {
			db = db.Where("containers.status IN (?)", []int64{constant.ContainerDelivered, constant.ContainerImportHub, constant.ContainerExportHub})
		} else {
			db = db.Where("containers.status = ?", options.Status)
		}
	}

	if len(options.Search) > 0 {

		db = db.Joins(`
			LEFT JOIN container_items ON containers.id = container_items.container_id
			AND (container_items.status = ? or container_items.status is null)
		`, constant.ContainerItemActive).
			Joins(`LEFT JOIN packages ON container_items.package_id = packages.id`).
			Joins(`LEFT JOIN package_codes ON package_codes.id = packages.package_code_id`).
			Where("containers.id = ? OR containers.code = ? OR containers.tracking_number = ? OR package_codes.code = ?",
				cast.ToInt(options.Search), options.Search, options.Search, options.Search)

		db = db.Group("containers.id")
	}

	if options.HubID > 0 {
		db = db.Joins("LEFT JOIN warehouses wh1 ON wh1.id = containers.hub_id")
		// db = db.Where("wh1.status = ?", constant.WareHouseStatusActive)
		db = db.Where(`containers.hub_id = ?`, options.HubID)
	}

	if options.WarehouseID > 0 {
		db = db.Joins("LEFT JOIN warehouses wh2 ON wh2.id = containers.warehouse_id")
		// db = db.Where("wh2.status = ?", constant.WareHouseStatusActive)
		db = db.Where(`containers.warehouse_id = ?`, options.WarehouseID)
	}
	if len(options.Barcode) > 0 {
		db = db.Where("containers.code = ?", options.Barcode)
	}

	if len(options.Code) > 0 {
		db = db.Where("containers.code = ? OR containers.tracking_number = ?", options.Code, options.Code)
	}

	if len(options.Codes) > 0 {
		db = db.Where("containers.code IN (?)", options.Codes)
	}

	if options.NotInShipment {
		db = db.Where("containers.shipment_id IS NULL")
	}

	if options.Type > 0 {
		db = db.Where("containers.type = ?", options.Type)
	}

	if options.FbaType > 0 {
		db = db.Where("containers.fba_type = ?", options.FbaType)
	}

	if options.FbaType < 0 {
		db = db.Where("containers.fba_type = ? OR containers.fba_type IS NULL", 0)
	}

	if options.ShipmentID > 0 {
		db = db.Where("containers.shipment_id = ?", options.ShipmentID)
	}

	if options.IsWarningWeight {
		db = db.Where("containers.actual_weight > 0 AND containers.weight > 0 AND containers.actual_weight <> containers.weight")
	}

	if options.Limit > 0 {
		db = db.Limit(options.Limit)
	}

	if options.Offset > 0 {
		db = db.Offset(options.Offset)
	}

	return db
}

func (m ContainerManager) BuildContainerBoxQuery(options ContainerBoxQueryOptions) *gorm.DB {
	db := m.db
	if options.ID > 0 {
		db = db.Where("id=?", options.ID)
	}

	if len(options.IDS) > 0 {
		db = db.Where("id IN (?)", options.IDS)
	}

	return db
}

func (m ContainerManager) GetContainer(options ContainerQueryOptions) (*entity.Container, error) {
	db := m.BuildContainerQuery(options)
	var container *entity.Container
	db = db.First(&container)
	return container, db.Error
}

func (m ContainerManager) GetContainerLogs(options ContainerQueryOptions) ([]entity.ContainerDeliverLog, error) {
	db := m.db.Where("container_id = ?", options.ID)
	var logs []entity.ContainerDeliverLog
	db = db.Find(&logs)
	return logs, db.Error
}

func (m ContainerManager) GetContainerItems(containerID int64) ([]*entity.ContainerItem, error) {
	db := m.db
	var containerItems = make([]*entity.ContainerItem, 0)
	db = db.Model(&entity.ContainerItem{}).
		Where("container_id = ? and status = ?", containerID, constant.ContainerItemActive).Find(&containerItems)
	return containerItems, db.Error
}

func (m ContainerManager) GetAllItemFailContainer(containerID int64) ([]*entity.ContainerItem, error) {
	db := m.db
	var containerItems = make([]*entity.ContainerItem, 0)
	db = db.Order("id DESC")
	db = db.Model(&entity.ContainerItem{}).
		Where("container_id = ? and status IN (?)", containerID, []int64{constant.ContainerItemFail}).Find(&containerItems)
	return containerItems, db.Error
}

func (m ContainerManager) GetAllItemContainer(containerID int64) ([]*entity.ContainerItem, error) {
	db := m.db
	var containerItems = make([]*entity.ContainerItem, 0)
	db = db.Order("id DESC")
	db = db.Model(&entity.ContainerItem{}).
		Where("container_id = ? and status IN (?)", containerID, []int64{constant.ContainerItemActive, constant.ContainerItemFail}).Find(&containerItems)
	return containerItems, db.Error
}

func (m ContainerManager) CountContainerItems(containerID int64) (int64, error) {
	db := m.db
	var count int64
	db = db.Model(&entity.ContainerItem{}).
		Where("container_id = ? and status IN (?)", containerID, []int64{constant.ContainerItemActive}).Count(&count)
	return count, db.Error
}

func (m ContainerManager) CountContainerAllItems(containerID int64) (int64, error) {
	db := m.db
	var count int64
	db = db.Model(&entity.ContainerItem{}).
		Where("container_id = ? and status IN (?)", containerID, []int64{constant.ContainerItemActive, constant.ContainerItemFail}).Select("COUNT(DISTINCT  package_id,status) as count").Count(&count)
	return count, db.Error
}

func (m ContainerManager) CountContainerFailItems(containerID int64) (int64, error) {
	db := m.db
	var count int64
	db = db.Model(&entity.ContainerItem{}).
		Where("container_id = ? and status IN (?)", containerID, []int64{constant.ContainerItemFail}).Select("COUNT(DISTINCT  package_id,status) as count").Count(&count)
	return count, db.Error
}

func (m ContainerManager) GetContainerByItem(packageID int64) (*entity.ContainerItem, error) {
	//var containerItem *entity.ContainerItem
	containerItem := &entity.ContainerItem{}
	db := m.db.Model(&entity.ContainerItem{}).
		Where("package_id = ? and status = ?", packageID, constant.ContainerItemActive).First(containerItem)
	return containerItem, db.Error
}

func (m *ContainerManager) CountPackageInContainer(containerID int64) (int64, error) {
	db := m.db
	db = db.Where("container_id = ?", containerID)
	var count int64
	db = db.Model(&entity.Package{}).Count(&count)
	return count, db.Error
}

func (m ContainerManager) GetContainers(options ContainerQueryOptions) ([]entity.Container, error) {
	db := m.BuildContainerQuery(options)
	var containers []entity.Container

	if !options.IgnorePerload {
		db = db.Preload("ContainerItems", func(db *gorm.DB) *gorm.DB {
			db = db.Joins("JOIN packages ON packages.id = container_items.package_id")
			db = db.Where("container_items.status = ?", constant.ContainerItemActive)
			return db
		}).Preload("ContainerItems.Package")

		db = db.Preload("Warehouse", func(db *gorm.DB) *gorm.DB {
			db = db.Where("warehouses.status = ?", constant.WareHouseStatusActive)
			return db
		})
	} else if options.HasHistory {
		db = db.Preload("ContainerHistory", func(db *gorm.DB) *gorm.DB {
			return db.Order("ship_time DESC")
		})
	}

	if options.OrderByUpdated {
		db = db.Order("containers.updated_at DESC").Find(&containers)
	} else {
		db = db.Order("containers.id DESC").Find(&containers)
	}

	return containers, db.Error
}

func (m ContainerManager) CountContainers(options ContainerQueryOptions) (int64, error) {
	db := m.BuildContainerQuery(options)
	var count int64
	db = db.Model(&entity.Container{}).Select("COUNT(DISTINCT containers.id) as count").Count(&count)
	return count, db.Error
}

func (m ContainerManager) CountAllStatusContainers(options ContainerQueryOptions) ([]dto.CountStatusContainer, error) {
	options.Status = 0
	db := m.BuildContainerQuery(options)
	db = db.Model(&entity.Container{}).Select("containers.status, COUNT(DISTINCT containers.id) as count")
	db = db.Group("containers.status")
	var countStatus []dto.CountStatusContainer
	db = db.Scan(&countStatus)
	return countStatus, db.Error
}

func (m ContainerManager) SaveContainer(container *entity.Container) error {
	db := m.db.Save(container)
	return db.Error
}

func (m ContainerManager) UpdateContainer(container *entity.Container) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Save(container).Error; err != nil {
		tx.Rollback()
		return err
	}

	if container.FbaType > 0 && container.Type == constant.ContainerTypeManual {
		subQ := tx.Model(&entity.ContainerItem{}).Where("container_id = ? AND status = ?", container.ID, constant.ContainerItemActive).Select("package_id")
		err := tx.Model(&entity.Tracking{}).Where("status = ? AND package_id IN (?)", constant.TrackingStatusSuccess, subQ).UpdateColumns(map[string]interface{}{
			"updated_at":      time.Now(),
			"tracking_number": container.TrackingNumber,
		}).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m ContainerManager) SaveContainerImportHUb(container *entity.Container) error {
	var mapUpdateContainer = map[string]interface{}{
		"hub_imported_at": time.Now(),
		"updated_at":      time.Now(),
		"status":          constant.ContainerImportHub,
	}
	db := m.db.Model(&entity.Container{}).Where("id = ?", container.ID).UpdateColumns(mapUpdateContainer)
	return db.Error
}

func (m ContainerManager) SaveContainerWithTransaction(tx *gorm.DB, container *entity.Container) error {
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		return err
	}
	err := tx.Save(container).Error
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (m ContainerManager) CreateContainer(container *entity.Container) (*gorm.DB, error) {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return nil, err
	}

	err := tx.Save(container).Error
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	return tx, nil
}

func (m ContainerManager) SaveFailPackageContainer(container *entity.Container, packageSave *entity.Package, description string) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Exec("UPDATE containers SET updated_at=now() WHERE id=?", container.ID).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&entity.ContainerItem{}).
		Where("container_id = ? AND package_id = ? AND status = ?", container.ID, packageSave.ID, constant.ContainerItemFail).
		UpdateColumns(map[string]interface{}{
			"status":     constant.ContainerItemInactive,
			"updated_at": time.Now(),
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	var containerItem = entity.ContainerItem{
		Status:      constant.ContainerItemFail,
		ContainerID: container.ID,
		PackageID:   packageSave.ID,
		Description: description,
	}

	if err := tx.Create(&containerItem).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m ContainerManager) SaveContainerAndPackage(container *entity.Container, packageSave *entity.Package, dlogs []entity.PackageDeliverLog) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	mapChangeContainer := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if packageSave.Service.Code == constant.ServiceFBACode || packageSave.Service.Code == constant.ServiceFastFBACode {
		mapChangeContainer["fba_type"] = cast.ToInt(true)
	}

	if err := tx.Model(&entity.Container{}).Where("id = ?", container.ID).
		UpdateColumns(mapChangeContainer).Error; err != nil {
		tx.Rollback()
		return err
	}

	var mapUpdatePackage = map[string]interface{}{
		"status":     constant.PackageStatusWareHouseInContainer,
		"hub_id":     utils.Int64(container.HubID),
		"updated_at": time.Now(),
	}

	if packageSave.WarehouseID < 1 {
		mapUpdatePackage["warehouse_id"] = container.WarehouseID
	}

	if err := tx.Model(&entity.Package{}).Where("id = ?", packageSave.ID).
		UpdateColumns(mapUpdatePackage).Error; err != nil {
		tx.Rollback()
		return err
	}

	var containerItem = entity.ContainerItem{
		Status:      constant.ContainerItemActive,
		ContainerID: container.ID,
		PackageID:   packageSave.ID,
	}

	if err := tx.Create(&containerItem).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&entity.ContainerItem{}).
		Where("package_id = ? AND status = ?", packageSave.ID, constant.ContainerItemFail).
		UpdateColumns(map[string]interface{}{
			"status":     constant.ContainerItemInactive,
			"updated_at": time.Now(),
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if len(dlogs) > 0 {
		if err := tx.Create(&dlogs).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m ContainerManager) AppendContainerShipment(container *entity.Container, shipment *entity.Shipment) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}
	if err := tx.Save(container).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Save(shipment).Error; err != nil {
		tx.Rollback()
		return err
	}
	err := tx.Model(&entity.Package{}).Where("id IN (?)",
		tx.Model(&entity.ContainerItem{}).
			Where("container_id = ? AND status = ?", container.ID, constant.ContainerItemActive).
			Select("package_id")).
		Updates(map[string]interface{}{
			"updated_at": time.Now(),
			"status":     constant.PackageStatusWareHouseInShipment,
		}).Error

	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (m ContainerManager) AppendListContainerShipment(containers []entity.Container, shipment *entity.Shipment) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Save(shipment).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, container := range containers {
		if err := tx.Save(&container).Error; err != nil {
			tx.Rollback()
			return err
		}

		err := tx.Model(&entity.Package{}).Where("id IN (?)",
			tx.Model(&entity.ContainerItem{}).
				Where("container_id = ? AND status = ?", container.ID, constant.ContainerItemActive).
				Select("package_id")).
			Updates(map[string]interface{}{
				"updated_at": time.Now(),
				"status":     constant.PackageStatusWareHouseInShipment,
			}).Error

		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m ContainerManager) RemoveContainerShipment(container *entity.Container, shipment *entity.Shipment) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}
	if err := tx.Save(container).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Save(shipment).Error; err != nil {
		tx.Rollback()
		return err
	}
	err := tx.Model(&entity.Package{}).Where("id IN (?)",
		tx.Model(&entity.ContainerItem{}).
			Where("container_id = ?", container.ID).
			Select("package_id")).
		Updates(map[string]interface{}{
			"updated_at": time.Now(),
			"status":     constant.PackageStatusWareHouseInContainer,
		}).Error

	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}
func (m ContainerManager) GetContainerBoxes(options ContainerBoxQueryOptions) ([]entity.ContainerBox, error) {
	db := m.BuildContainerBoxQuery(options)
	var boxes []entity.ContainerBox
	db = db.Order("id DESC").Find(&boxes)
	return boxes, db.Error
}

func (m ContainerManager) GetContainerBox(options ContainerBoxQueryOptions) (entity.ContainerBox, error) {
	db := m.BuildContainerBoxQuery(options)
	var box entity.ContainerBox
	db = db.First(&box)
	return box, db.Error
}

func (m ContainerManager) CloseContainer(container *entity.Container) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Model(&entity.Container{}).Where("id = ?", container.ID).
		UpdateColumns(map[string]interface{}{
			"updated_at":      time.Now(),
			"status":          constant.ContainerClosed,
			"actual_weight":   container.ActualWeight,
			"weight":          container.Weight,
			"width":           container.Width,
			"length":          container.Length,
			"height":          container.Height,
			"tracking_number": container.TrackingNumber,
			"close_at":        time.Now(),
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&entity.Package{}).Where("container_id = ?", container.ID).
		UpdateColumns(map[string]interface{}{
			"updated_at": time.Now(),
			"status":     constant.PackageStatusWareHouseInContainer,
		}).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (m ContainerManager) ContainerImportHub(containerCode []int64, packageIDs []int64, userID int64) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Model(&entity.Container{}).Where("id IN  (?)", containerCode).
		UpdateColumns(map[string]interface{}{
			"hub_imported_at": time.Now(),
			"updated_at":      time.Now(),
			"status":          constant.ContainerImportHub,
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&entity.Package{}).Where("id IN (?)", packageIDs).
		Where("status != ?", constant.PackageStatusUndelivered).
		UpdateColumns(map[string]interface{}{
			"hub_imported_at": time.Now(),
			"updated_at":      time.Now(),
			"status":          constant.PackageStatusImportHub,
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, pkgID := range packageIDs {
		deliverLog := &entity.PackageDeliverLog{
			PackageID: pkgID,
			Status:    constant.DeliverLogTebexpressInTransit,
			Type:      constant.PackageDeliverLogTypeImportHub,
			UserID:    &userID,
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}

		if err := tx.Create(deliverLog).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func (m ContainerManager) CancelPackagesInContainer(container *entity.Container, packages []entity.Package, mapItems map[int64]*entity.ContainerItem) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	var packageIDs, allPkgIDs []int64

	for _, item := range packages {
		allPkgIDs = append(allPkgIDs, item.ID)
		if mapItems[item.ID] != nil && mapItems[item.ID].Status == constant.ContainerItemActive {
			packageIDs = append(packageIDs, item.ID)
		}
	}

	if len(packageIDs) > 0 {
		if err := tx.Model(&entity.Package{}).Where("id IN (?)", packageIDs).
			UpdateColumns(map[string]interface{}{
				"updated_at": time.Now(),
				"status":     constant.PackageStatusWareHouseLabeled,
				"hub_id":     nil,
			}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Model(&entity.Container{}).Where("id = ?", container.ID).
		UpdateColumns(map[string]interface{}{
			"updated_at": time.Now(),
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&entity.ContainerItem{}).
		Where("package_id IN (?) AND container_id IN (?) AND status IN (?)", allPkgIDs, container.ID, []int{constant.ContainerItemActive, constant.ContainerItemFail}).
		UpdateColumns(map[string]interface{}{
			"updated_at": time.Now(),
			"status":     constant.ContainerItemInactive,
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m *ContainerManager) CountDeliverLogsByID(id int64) (int64, error) {
	var count int64
	db := m.db.Model(&entity.ContainerDeliverLog{}).Where("container_id=?", id)
	db = db.Where("code NOT IN (?)", constant.UPSIgnoreCodeLogs)
	db = db.Count(&count)
	return count, db.Error
}

func (m *ContainerManager) CreateDeliverLogs(logs []entity.ContainerDeliverLog, delivered, isFBA bool) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := m.db.Create(&logs).Error; err != nil {
		tx.Rollback()
		return err
	}

	if delivered {
		mapchange := map[string]interface{}{
			"updated_at": time.Now(),
			"status":     constant.ContainerDelivered,
		}

		id := logs[0].ContainerID
		if err := tx.Model(&entity.Container{}).Where("id=?", id).UpdateColumns(mapchange).Error; err != nil {
			tx.Rollback()
			return err
		}

		if isFBA {
			sql1 := `
				UPDATE
					packages
				SET
					packages.status=?,
					packages.updated_at=NOW()
				FROM
					packages
					INNER JOIN container_items ON container_items.package_id = packages.id AND container_items.status=?
				WHERE
					container_items.container_id=?
			`

			if err := tx.Exec(sql1, constant.OrderStatusDelivered, constant.ContainerItemActive, id).Error; err != nil {
				tx.Rollback()
				return err
			}

			sql := `
				INSERT INTO package_deliver_logs (package_id, LOCATION, status, TYPE, description, created_at, updated_at)
				SELECT
					packages.id AS package_id,
					CONCAT(packages.city, ', ', packages.country_code),
					?,
					?,
					?,
					NOW(),
					NOW()
				FROM
					packages
					INNER JOIN container_items ON packages.id = container_items.package_id
						AND container_items.status=?
				WHERE
					container_items.container_id=?
			`

			if err := tx.Exec(sql, constant.DeliverLogTebexpressDelivered, constant.PackageStatusDelivered, "Delivered", constant.ContainerItemActive, id).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit().Error
}

func (m *ContainerManager) GetDeliverLogs(containerID int64) ([]entity.ContainerDeliverLog, error) {
	var logs []entity.ContainerDeliverLog
	db := m.db.Where("container_id=?", containerID).Find(&logs)

	return logs, db.Error
}

func (m *ContainerManager) UpdateDeliverLogs(id int64, change map[string]interface{}) error {
	db := m.db.Table("container_deliver_logs").Where("id=?", id).UpdateColumns(change)
	return db.Error
}

func (m *ContainerManager) UpdateShipmentDeliver(id []int64) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Model(&entity.Shipment{}).Where("id IN (?)", id).
		UpdateColumns(map[string]interface{}{
			"updated_at": time.Now(),
			"status":     constant.ShipmentDelivered,
		}).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}
func (m *ContainerManager) UpdateContainerExport(id []int64) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}
	if err := tx.Model(&entity.Container{}).Where("id IN (?)", id).
		UpdateColumns(map[string]interface{}{
			"hub_exported_at": time.Now(),
			"updated_at":      time.Now(),
			"status":          constant.ContainerExportHub,
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
func (m *ContainerManager) UpdateContainerImport(id []int64) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}
	if err := tx.Model(&entity.Container{}).Where("id IN (?)", id).
		UpdateColumns(map[string]interface{}{
			"hub_imported_at": time.Now(),
			"updated_at":      time.Now(),
			"status":          constant.ContainerImportHub,
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m *ContainerManager) CreateDeliverLog(log *entity.ContainerDeliverLog, delivered bool) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := m.db.Create(log).Error; err != nil {
		tx.Rollback()
		return err
	}

	if delivered {
		mapchange := map[string]interface{}{
			"updated_at": time.Now(),
			"status":     constant.ContainerDelivered,
		}

		if err := tx.Model(&entity.Container{}).Where("id=?", log.ContainerID).UpdateColumns(mapchange).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m ContainerManager) GetContainerFirstPackage(containerID int64) (*entity.Package, error) {
	pkg := &entity.Package{}

	db := m.db.Where("container_items.container_id = ? AND container_items.status = ?", containerID, constant.ContainerItemActive)
	db = db.Where("packages.status NOT IN (?)", []int{constant.PackageStatusCancelled})
	db = db.Joins("INNER JOIN container_items ON container_items.package_id = packages.id")
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

func (m ContainerManager) GetContainerPackages(containerIDs []int64) ([]*entity.Package, error) {
	pkg := []*entity.Package{}

	sub := m.db.Where("container_items.container_id IN (?) AND container_items.status = ?", containerIDs, constant.ContainerItemActive).
		Model(&entity.ContainerItem{}).Select("package_id")

	db := m.db.Where("packages.status NOT IN (?) AND packages.id IN (?)", []int{constant.PackageStatusCancelled}, sub)
	db = db.Preload("Tracking").
		Select(`
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
		`).Joins("LEFT JOIN trackings ON trackings.package_id = packages.id")

	db = db.Find(&pkg)
	return pkg, db.Error
}

func (m ContainerManager) GetContainerItemsByPackageIDs(packageIDs []int64) ([]entity.ContainerItem, error) {
	items := []entity.ContainerItem{}
	db := m.db.Where("package_id IN (?) AND status=?", packageIDs, constant.ContainerItemActive).Find(&items)
	return items, db.Error
}
