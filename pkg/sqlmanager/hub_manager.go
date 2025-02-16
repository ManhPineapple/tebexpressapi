package sqlmanager

import (
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/utils/dbgorm"
	"time"

	"gorm.io/gorm"
)

type HubManager struct {
	DB *gorm.DB
}

type HubQueryFilters struct {
	UserID    int64
	HubID     int64
	Keywork   string
	Status    int64
	Limit     int
	Offset    int
	StartDate string
	EndDate   string
}

func NewHubManager(db *gorm.DB) *HubManager {
	return &HubManager{DB: db}
}

type HubQueryOption struct {
	HubID  int64
	Limit  int
	Offset int
}

func (m HubManager) BuildPackageQuery(opts HubQueryOption) *gorm.DB {
	db := m.DB

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	return db
}

func (m *HubManager) FindItems(q HubQueryFilters, v interface{}) error {
	db1 := m.DB.Table("packages")
	db2 := m.DB.Table("containers")
	db3 := m.DB.Table("shipments")

	db1 = db1.Select(`
		packages.id,
		package_codes.code,
		trackings.tracking_number,
		packages.status,
		'package' AS type,
		package_returns.reason as description,
		packages.created_at,
		packages.updated_at,
		package_returns.id AS package_return_id,
		packages.hub_exported_at,
		packages.hub_imported_at,
		packages.label as label_url,
		(SELECT  manifests.manifest_number FROM manifests WHERE manifests.package_id = packages.id order by created_at limit 1) as manifest_number,
		NULL AS count_container,
		packages.reship_at as reship_at,
		packages.returned_at as returned_at
	`)

	db2 = db2.Select(`
		containers.id,
		containers.code,
		containers.tracking_number,
		containers.status,
		'container' AS type,
		NULL as description,
		containers.created_at,
		containers.updated_at,
		NULL AS package_return_id,
		containers.hub_exported_at,
		containers.hub_imported_at,
		(SELECT  manifests.manifest_url FROM manifests WHERE manifests.container_id = containers.id order by created_at limit 1) as label_url,
		(SELECT  manifests.manifest_number FROM manifests WHERE manifests.container_id = containers.id order by created_at limit 1) as manifest_number,
		NULL AS count_container,
		NULL AS reship_at,
		NULL AS returned_at
	`)

	db3 = db3.Select(`
		shipments.id,
		'' AS code,
		manifests.manifest_number AS tracking_number,
		shipments.status,
		'shipment' AS type,
		shipments.created_at,
		shipments.updated_at,
		NULL AS package_return_id,
		NULL AS hub_exported_at, 
		NULL AS hub_imported_at, 
		manifests.manifest_url AS label_url,
		NULL AS manifest_number,
		(SELECT COUNT(*) FROM containers WHERE containers.shipment_id = shipments.id) as count_container,
		NULL AS reship_at,
		NULL AS returned_at
`)

	db1 = db1.Joins("LEFT JOIN package_codes ON packages.package_code_id=package_codes.id")
	db1 = db1.Joins("LEFT JOIN trackings ON packages.id=trackings.package_id AND trackings.status=?", constant.TrackingStatusSuccess)
	db1 = db1.Joins("LEFT JOIN package_returns ON packages.id=package_returns.package_id")
	db3 = db3.Joins("LEFT JOIN manifests ON shipments.id=manifests.shipment_id")

	if q.HubID > 0 {
		db1 = db1.Where("packages.hub_id=?", q.HubID)
		db2 = db2.Where("containers.hub_id=?", q.HubID)
		db3 = db3.Where("shipments.hub_id=?", q.HubID)
	}

	if q.Keywork != "" {
		db1 = db1.Where("packages.id=? OR package_codes.code=? OR trackings.tracking_number=?", q.Keywork, q.Keywork, q.Keywork)
		db2 = db2.Where("containers.id=? OR containers.code=? OR containers.tracking_number=?", q.Keywork, q.Keywork, q.Keywork)

		if q.Status == constant.HubItemFilterStatusPending {
			db1 = db1.Joins("LEFT JOIN container_items ON packages.id=container_items.package_id AND container_items.status=?", constant.ContainerItemActive)
			db1 = db1.Joins("LEFT JOIN containers ON containers.id=container_items.container_id")
			db1 = db1.Select("containers.shipment_id")

			db2 = db2.Select("containers.shipment_id")

			db3 = db3.Where("shipments.id=? OR manifests.manifest_number=? OR shipments.id IN (?) OR shipments.id IN (?)", q.Keywork, q.Keywork, db1, db2)
		}

	}

	if len(q.StartDate) > 0 {
		if q.Status == constant.HubItemFilterStatusExport {
			db1 = db1.Where("DATE_FORMAT(convert_tz(packages.hub_exported_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)", q.StartDate)
			db2 = db2.Where("DATE_FORMAT(convert_tz(hub_exported_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)", q.StartDate)
		} else if q.Status != constant.HubItemFilterStatusPending {
			db1 = db1.Where("DATE_FORMAT(convert_tz(packages.hub_imported_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)", q.StartDate)
			db2 = db2.Where("DATE_FORMAT(convert_tz(hub_imported_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)", q.StartDate)
		}
	}

	if len(q.EndDate) > 0 {
		if q.Status == constant.HubItemFilterStatusExport {
			db1 = db1.Where("DATE_FORMAT(convert_tz(packages.hub_exported_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)", q.EndDate)
			db2 = db2.Where("DATE_FORMAT(convert_tz(hub_exported_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)", q.EndDate)
		} else if q.Status != constant.HubItemFilterStatusPending {
			db1 = db1.Where("DATE_FORMAT(convert_tz(packages.hub_imported_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)", q.EndDate)
			db2 = db2.Where("DATE_FORMAT(convert_tz(hub_imported_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)", q.EndDate)
		}
	}

	db := m.DB

	if q.Status == constant.HubItemFilterStatusReturn {
		db = db1
		db = db.Where("alert=?", constant.PackageAlertTypeHubReturn)
		db = db.Order("returned_at DESC")
	} else if q.Status == constant.HubItemFilterStatusReship {
		db = db1.Select(`
		packages.id,
		package_codes.code,
		trackings.tracking_number,
		packages.status,
		'package' AS type,
		packages.created_at,
		packages.updated_at,
		package_returns.id AS package_return_id,
		packages.hub_exported_at,
		packages.hub_imported_at,
		packages.label AS label_url,
		NULL AS manifest_number,
		NULL AS count_container,
		packages.reship_at as reship_at,
		packages.returned_at as returned_at
	`)
		db = db.Where("packages.status = ?", constant.PackageStatusReship)
		db = db.Order("reship_at DESC")
	} else if q.Status == constant.HubItemFilterStatusPending {
		db = db3
		db3 = db3.Where("shipments.status=?", constant.ShipmentIntransit)
		db = db.Order("shipments.created_at DESC ")

	} else {
		if q.Status == constant.HubItemFilterStatusIn {
			db1 = db1.Where("packages.status=?", constant.PackageStatusImportHub)
			db2 = db2.Where("status=?", constant.ContainerImportHub)
			db = db.Order("hub_imported_at DESC ")
		}

		if q.Status == constant.HubItemFilterStatusExport {
			db1 = db1.Select(`
				packages.id,
				package_codes.code,
				trackings.tracking_number,
				packages.status,
				'package' AS type,
				package_returns.reason as description,
				packages.created_at,
				packages.updated_at,
				package_returns.id AS package_return_id,
				packages.hub_exported_at,
				packages.hub_imported_at,
				(SELECT  manifests.manifest_url FROM manifests WHERE manifests.package_id = packages.id order by created_at limit 1) as label_url,
				(SELECT  manifests.manifest_number FROM manifests WHERE manifests.package_id = packages.id order by created_at limit 1) as manifest_number,
				NULL AS count_container,
				packages.reship_at as reship_at,
				packages.returned_at as returned_at
			`)
			db1 = db1.Where("packages.status=?", constant.PackageStatusExportHub)
			db2 = db2.Where("status=?", constant.ContainerExportHub)
			db = db.Order("hub_exported_at DESC")
		}

		db = db.Table("((?) UNION (?)) AS items", db1, db2)
	}

	if q.Limit > 0 {
		db = db.Limit(q.Limit)
	}

	if q.Offset > 0 {
		db = db.Offset(q.Offset)
	}

	return db.Scan(v).Error
}

func (m *HubManager) CountFindItems(q HubQueryFilters) (int64, error) {
	db1 := m.DB.Table("packages")
	db2 := m.DB.Table("containers")
	db3 := m.DB.Table("shipments")

	db1 = db1.Select(`packages.id`)
	db2 = db2.Select(`id`)
	db3 = db3.Select(`shipments.id`)

	db1 = db1.Joins("LEFT JOIN package_codes ON packages.package_code_id=package_codes.id")
	db1 = db1.Joins("LEFT JOIN trackings ON packages.id=trackings.package_id AND trackings.status=?", constant.TrackingStatusSuccess)

	db3 = db3.Joins("LEFT JOIN manifests ON shipments.id=manifests.shipment_id")

	if q.HubID > 0 {
		db1 = db1.Where("packages.hub_id=?", q.HubID)
		db2 = db2.Where("containers.hub_id=?", q.HubID)
		db3 = db3.Where("shipments.hub_id=?", q.HubID)
	}

	if q.Keywork != "" {
		db1 = db1.Where("packages.id=? OR package_codes.code=? OR trackings.tracking_number=?", q.Keywork, q.Keywork, q.Keywork)
		db2 = db2.Where("containers.id=? OR containers.code=? OR containers.tracking_number=?", q.Keywork, q.Keywork, q.Keywork)

		if q.Status == constant.HubItemFilterStatusPending {
			db1 = db1.Joins("LEFT JOIN container_items ON packages.id=container_items.package_id AND container_items.status=?", constant.ContainerItemActive)
			db1 = db1.Joins("LEFT JOIN containers ON containers.id=container_items.container_id")
			db1 = db1.Select("containers.shipment_id")

			db2 = db2.Select("containers.shipment_id")

			db3 = db3.Where("shipments.id=? OR manifests.manifest_number=? OR shipments.id IN (?) OR shipments.id IN (?)", q.Keywork, q.Keywork, db1, db2)
		}
	}

	if len(q.StartDate) > 0 {
		if q.Status == constant.HubItemFilterStatusExport {
			db1 = db1.Where("DATE_FORMAT(convert_tz(packages.hub_exported_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)", q.StartDate)
			db2 = db2.Where("DATE_FORMAT(convert_tz(hub_exported_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)", q.StartDate)
		} else if q.Status != constant.HubItemFilterStatusPending {
			db1 = db1.Where("DATE_FORMAT(convert_tz(packages.hub_imported_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)", q.StartDate)
			db2 = db2.Where("DATE_FORMAT(convert_tz(hub_imported_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)", q.StartDate)
		}
	}

	if len(q.EndDate) > 0 {
		if q.Status == constant.HubItemFilterStatusExport {
			db1 = db1.Where("DATE_FORMAT(convert_tz(packages.hub_exported_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)", q.EndDate)
			db2 = db2.Where("DATE_FORMAT(convert_tz(hub_exported_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)", q.EndDate)
		} else if q.Status != constant.HubItemFilterStatusPending {
			db1 = db1.Where("DATE_FORMAT(convert_tz(packages.hub_imported_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)", q.EndDate)
			db2 = db2.Where("DATE_FORMAT(convert_tz(hub_imported_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)", q.EndDate)
		}
	}

	db := m.DB

	if q.Status == constant.HubItemFilterStatusReturn {
		db1 = db1.Where("alert=?", constant.PackageAlertTypeHubReturn)
		db = db1
	} else if q.Status == constant.HubItemFilterStatusReship {
		db = db1
		db = db.Where("packages.status = ?", constant.PackageStatusReship)
	} else if q.Status == constant.HubItemFilterStatusPending {
		db3 = db3.Where("shipments.status=?", constant.ShipmentIntransit)
		db = db3
	} else {
		if q.Status == constant.HubItemFilterStatusIn {
			db1 = db1.Where("packages.status=?", constant.PackageStatusImportHub)
			db2 = db2.Where("status=?", constant.ContainerImportHub)
		}

		if q.Status == constant.HubItemFilterStatusExport {
			db1 = db1.Where("packages.status=?", constant.PackageStatusExportHub)
			db2 = db2.Where("status=?", constant.ContainerExportHub)
		}

		db = db.Table("((?) UNION (?)) AS items", db1, db2)
	}

	var count int64
	db = db.Count(&count)

	return count, db.Error
}

func (m *HubManager) SaveReturn(v *entity.PackageReturn) error {
	v.CreatedAt = time.Now()
	v.UpdatedAt = v.CreatedAt

	return m.DB.Save(v).Error
}

func (m *HubManager) GetPackageByID(id int64) (*entity.Package, error) {
	pkg := &entity.Package{}

	db := m.DB.Where("id=?", id)
	db = db.Preload("PackageReturn")
	db = db.First(pkg)

	return pkg, db.Error
}

func (m *HubManager) ExportContainerHub(codes []int64, pkgs []entity.Package, userID int64) error {

	tx := m.DB.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Model(&entity.Container{}).Where("id IN ?", codes).
		UpdateColumns(map[string]interface{}{
			"hub_exported_at": time.Now(),
			"updated_at":      time.Now(),
			"status":          constant.ContainerExportHub,
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	subquery := m.DB.Model(&entity.ContainerItem{}).
		Joins("INNER JOIN containers on container_items.container_id = containers.id").
		Where("containers.id IN (?) and container_items.status = ?", codes, constant.ContainerItemActive).
		Select("package_id")

	if err := tx.Model(&entity.Package{}).Where("id IN (?) and alert != ?", subquery, constant.PackageAlertTypeHubReturn).
		Where("status != ?", constant.PackageStatusUndelivered).
		UpdateColumns(map[string]interface{}{
			"hub_exported_at": time.Now(),
			"updated_at":      time.Now(),
			"status":          constant.PackageStatusExportHub,
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, pkg := range pkgs {
		deliverLog := &entity.PackageDeliverLog{
			PackageID: pkg.ID,
			Status:    constant.DeliverLogTebexpressInTransit,
			Type:      constant.PackageDeliverLogTypeExportHub,
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

func (m *HubManager) GetContainerByPkgID(package_id []int64) ([]entity.Container, error) {
	pkgs := []entity.Container{}
	db := m.DB

	db = db.Model(&entity.Container{})
	subquery := m.DB.Model(&entity.ContainerItem{}).Joins("LEFT JOIN packages on container_items.package_id = packages.id").Where("packages.id IN (?) and container_items.status = ?", package_id, constant.ContainerItemActive).Select("container_items.container_id")
	db = db.Preload("ContainerItems")
	db = db.Preload("ContainerItems.Package")
	db = db.Where("containers.id IN (?)", subquery).Find(&pkgs)
	return pkgs, db.Error
}

func (m *HubManager) GetPackagesInContainer(container_id int64) ([]entity.Package, error) {
	pkgs := []entity.Package{}
	db := m.DB

	db = db.Model(&entity.Package{})
	db = db.Joins("LEFT JOIN container_items on container_items.package_id = packages.id")
	db = db.Joins("LEFT JOIN containers on container_items.container_id = containers.id")
	db = db.Where("containers.id  = ? and container_items.status = ?", container_id, constant.ContainerItemActive).Find(&pkgs)

	return pkgs, db.Error
}

func (m *HubManager) GetPackageInContainer(container_code string, code string) (entity.Package, error) {
	pkg := entity.Package{}
	db := m.DB

	db = db.Model(&entity.Package{})
	db = db.Joins("LEFT JOIN package_codes on package_codes.id = packages.package_code_id")
	db = db.Joins("LEFT JOIN trackings on trackings.package_id = packages.id AND trackings.status!=?", constant.TrackingStatusCanceled)
	db = db.Joins("LEFT JOIN container_items on container_items.package_id = packages.id")
	db = db.Joins("LEFT JOIN containers on container_items.container_id = containers.id")
	db = db.Where("containers.code = (?) and container_items.status = ? and package_codes.code = ? OR trackings.tracking_number = ?", container_code, constant.ContainerItemActive, code, code).Find(&pkg)

	return pkg, db.Error
}

func (m *HubManager) GetContainerIdManifest(container_id []int64) ([]int64, error) {
	var container_ids []int64
	db := m.DB
	db = db.Model(&entity.Manifest{}).Select("container_id")
	db = db.Where("container_id IN (?)", container_id).Find(&container_ids)

	return container_ids, db.Error
}
func (m *HubManager) GetPackageIdManifest(package_id []int64) ([]int64, error) {
	package_ids := []int64{}
	db := m.DB
	db = db.Model(&entity.Manifest{}).Select("package_id")
	db = db.Where("package_id  IN ?", package_id).Find(&package_ids)

	return package_ids, db.Error
}
func (m *HubManager) GetListPackagesInContainer(container_id []int64) ([]entity.Package, error) {

	pkgs := []entity.Package{}
	db := m.DB
	db = db.Model(&entity.Package{})
	db = db.Joins("LEFT JOIN container_items on container_items.package_id = packages.id")
	db = db.Joins("LEFT JOIN containers on container_items.container_id = containers.id")
	db = db.Where("containers.id IN ? and container_items.status = ?", container_id, constant.ContainerItemActive).Find(&pkgs)

	return pkgs, db.Error
}

func (m *HubManager) ExportPackageHub(id []int64, userID int64) error {

	tx := m.DB.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Model(&entity.Package{}).Where("id IN (?)", id).
		Where("status != ?", constant.PackageStatusUndelivered).
		UpdateColumns(map[string]interface{}{
			"hub_exported_at": time.Now(),
			"updated_at":      time.Now(),
			"status":          constant.PackageStatusExportHub,
			"alert":           constant.PackageAlertTypeDisable,
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, pkgID := range id {
		deliverLog := &entity.PackageDeliverLog{
			PackageID: pkgID,
			Status:    constant.DeliverLogTebexpressInTransit,
			Type:      constant.PackageDeliverLogTypeExportHub,
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

func (m *HubManager) ImportPackageHub(ids []int64, userID int64) error {

	tx := m.DB.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Model(&entity.Package{}).Where("id IN (?)", ids).
		Where("status != ? AND status != ?", constant.PackageStatusUndelivered, constant.PackageStatusReship).
		UpdateColumns(map[string]interface{}{
			"hub_imported_at": time.Now(),
			"updated_at":      time.Now(),
			"status":          constant.PackageStatusImportHub,
			"alert":           constant.PackageAlertTypeHubReturn,
			"returned_at":     time.Now(),
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&entity.Package{}).Where("id IN (?)", ids).
		Where("status = ?", constant.PackageStatusReship).
		UpdateColumns(map[string]interface{}{
			"hub_imported_at": time.Now(),
			"updated_at":      time.Now(),
			"status":          constant.PackageStatusImportHub,
			"alert":           constant.PackageAlertTypeDisable,
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, pkgID := range ids {
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

func (m *HubManager) GetListImportHub(tp string, code string, opts HubQueryOption) ([]string, error) {
	db := m.BuildPackageQuery(opts)

	var codes []string

	if tp == "container" {
		if len(code) > 0 {
			db = db.Where("containers.code = ? OR containers.tracking_number = ?", code, code)
		}
		db = db.Order("containers.updated_at DESC")

		if opts.HubID > 0 {
			db = db.Where("containers.hub_id = ?", opts.HubID)
		}

		db = db.Table("containers").Select("containers.code").Where("containers.status = ?", constant.ContainerImportHub).Pluck("code", &codes)

	} else {
		if len(code) > 0 {
			db = db.Joins("LEFT JOIN trackings ON trackings.package_id = packages.id AND trackings.status != ? ", constant.TrackingStatusCanceled)
			db = db.Where("((package_codes.code = ? AND package_codes.status = ? ) OR packages.order_number = ? OR trackings.tracking_number = ?) ", code, constant.PackageCodeEnable, code, code)
		}

		db = db.Order("packages.updated_at DESC")
		if opts.HubID > 0 {
			db = db.Where("packages.hub_id = ?", opts.HubID)
		}

		db = db.Table("packages").Joins("JOIN package_codes ON package_codes.id=packages.package_code_id and package_codes.user_id = packages.user_id").Select("package_codes.code").Where("(packages.status = ?  or packages.alert = ? ) and package_codes.status=?", constant.PackageStatusImportHub, constant.PackageAlertTypeHubReturn, constant.PackageCodeEnable).Pluck("code", &codes)

	}

	return codes, db.Error
}

func (m *HubManager) CountListImportHub(tp string, code string, opts HubQueryOption) (int64, error) {
	db := m.DB

	var count int64

	if tp == "container" {
		if len(code) > 0 {
			db = db.Where("containers.code = ? OR containers.tracking_number = ?", code, code)
		}

		if opts.HubID > 0 {
			db = db.Where("containers.hub_id = ?", opts.HubID)
		}

		db = db.Table("containers").Select("containers.code").Where("containers.status = ?", constant.ContainerImportHub).Count(&count)
		db = db.Order("containers.updated_at DESC")
	} else {
		if len(code) > 0 {
			db = db.Joins("LEFT JOIN trackings ON trackings.package_id = packages.id AND trackings.status != ? ", constant.TrackingStatusCanceled)
			db = db.Where("((package_codes.code = ? AND package_codes.status = ? ) OR packages.order_number = ? OR trackings.tracking_number = ?) ", code, constant.PackageCodeEnable, code, code)
		}

		db = db.Order("packages.updated_at DESC")
		if opts.HubID > 0 {
			db = db.Where("packages.hub_id = ?", opts.HubID)
		}

		db = db.Table("packages").Joins("JOIN package_codes ON package_codes.id=packages.package_code_id and package_codes.user_id = packages.user_id").Select("package_codes.code").Where("(packages.status = ?  or packages.alert = ? ) and package_codes.status=?", constant.PackageStatusImportHub, constant.PackageAlertTypeHubReturn, constant.PackageCodeEnable).Count(&count)
		db = db.Order("packages.updated_at DESC")
	}

	return count, db.Error
}
