package sqlmanager

import (
	"fmt"
	"log"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/dbgorm"
	"time"

	"github.com/spf13/viper"

	"gorm.io/gorm"
)

type TrackingManager struct {
	db *gorm.DB
}

func NewTrackingManager(db *gorm.DB) *TrackingManager {
	return &TrackingManager{db: db}
}

type TrackingOption struct {
	PackageID       int64
	PackageIDs      []int64
	TrackingNumber  string
	TrackingNumbers []string
	Status          int
	Limit           int
	Offset          int
	CarrierID       int64
	CarrierIDs      []int64
	TimePoint       *time.Time
}

func (m *TrackingManager) BuildStateQuery(opts TrackingOption) *gorm.DB {
	db := m.db

	if opts.PackageID > 0 {
		db = db.Where("package_id=?", opts.PackageID)
	}

	if len(opts.PackageIDs) > 0 {
		db = db.Where("package_id IN (?)", opts.PackageIDs)
	}

	if opts.TrackingNumber != "" {
		db = db.Where("tracking_number=?", opts.TrackingNumber)
	}

	if len(opts.TrackingNumbers) > 0 {
		db = db.Where("tracking_number IN (?)", opts.TrackingNumbers)
	}

	if opts.Status > 0 {
		db = db.Where("status=?", opts.Status)
	}

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	if opts.CarrierID > 0 {
		db = db.Where("carrier_id=?", opts.CarrierID)
	}

	if len(opts.CarrierIDs) > 0 {
		db = db.Where("carrier_id IN (?)", opts.CarrierIDs)
	}

	if opts.TimePoint != nil {
		db = db.Where("created_at < ? AND created_at >=  ? - interval 1315 MINUTE", opts.TimePoint, opts.TimePoint)
		db = db.Where("is_manifested = ?", constant.TrackingNotManifested)
	}

	db = db.Order("id DESC")
	return db
}

func (m *TrackingManager) GetTrackings(opts TrackingOption) ([]entity.Tracking, error) {
	db := m.BuildStateQuery(opts)

	var tks []entity.Tracking
	db = db.Preload("Package")
	db = db.Preload("Carrier")
	db = db.Find(&tks)
	return tks, db.Error
}

func (m *TrackingManager) GetTrackingsIntransit(opts TrackingOption) ([]entity.Tracking, error) {
	db := m.BuildStateQuery(opts)
	db = db.Joins("INNER JOIN packages ON packages.id = trackings.package_id")
	db = db.Where("packages.status >= ? AND packages.status <= ?", constant.PackageStatusWareHouseLabeled, constant.PackageStatusInTransit)
	db = db.Where("trackings.status = ?", constant.TrackingStatusSuccess)
	db = db.Where("packages.created_at > ?", "2025-01-01")

	var tks []entity.Tracking
	db = db.Preload("Package").Preload("Package.PackageCode").Preload("Package.Warehouse")
	db = db.Preload("Carrier")
	db = db.Find(&tks)
	return tks, db.Error
}

func (m *TrackingManager) GetTrackingsCheck17Track(opts TrackingOption) ([]entity.Tracking, error) {
	db := m.BuildStateQuery(opts)
	db = db.Joins("INNER JOIN packages ON packages.id = trackings.package_id")
	db = db.Joins("JOIN container_items ON container_items.package_id = packages.id")
	db = db.Joins("JOIN containers ON containers.id = container_items.container_id")

	db = db.Where("packages.status IN  (?) AND packages.user_id NOT IN (?)", []int64{constant.PackageStatusInTransit, constant.PackageStatusImportHub, constant.PackageStatusExportHub}, viper.GetIntSlice("17track.blacklist"))
	db = db.Where("trackings.status = ?", constant.TrackingStatusSuccess)
	db = db.Where("container_items.status = ?", constant.ContainerItemActive)
	db = db.Where("containers.status IN  (?)", []int64{constant.ContainerDelivered, constant.ContainerExportHub, constant.ContainerImportHub})

	var tks []entity.Tracking
	db = db.Preload("Package")
	db = db.Preload("Carrier")
	db = db.Find(&tks)
	return tks, db.Error
}

func (m *TrackingManager) CountTrackingsIntransit(opts TrackingOption) (int64, error) {
	db := m.BuildStateQuery(opts)
	db = db.Joins("INNER JOIN packages ON packages.id = trackings.package_id")
	db = db.Where("packages.status >= ? AND packages.status <= ?", constant.PackageStatusWareHouseLabeled, constant.PackageStatusInTransit)
	db = db.Where("trackings.status = ?", constant.TrackingStatusSuccess)
	db = db.Where("packages.created_at > ?", "2025-01-01")

	var count int64
	db = db.Model(&entity.Tracking{}).Count(&count)
	return count, db.Error
}

func (m *TrackingManager) CountTrackingsCheck17Track(opts TrackingOption) (int64, error) {
	db := m.BuildStateQuery(opts)
	db = db.Joins("INNER JOIN packages ON packages.id = trackings.package_id")
	db = db.Joins("JOIN container_items ON container_items.package_id = packages.id")
	db = db.Joins("JOIN containers ON containers.id = container_items.container_id")

	db = db.Where("packages.status IN (?) AND packages.user_id NOT IN (?)", []int64{constant.PackageStatusInTransit, constant.PackageStatusImportHub, constant.PackageStatusExportHub}, viper.GetIntSlice("17track.blacklist"))
	db = db.Where("trackings.status = ?", constant.TrackingStatusSuccess)
	db = db.Where("container_items.status = ?", constant.ContainerItemActive)
	db = db.Where("containers.status IN  (?)", []int64{constant.ContainerDelivered, constant.ContainerExportHub, constant.ContainerImportHub})

	var count int64
	db = db.Model(&entity.Tracking{}).Count(&count)
	return count, db.Error
}

func (m *TrackingManager) CountTrackings(opts TrackingOption) (int64, error) {
	db := m.BuildStateQuery(opts)

	var count int64
	db = db.Model(&entity.Tracking{}).Count(&count)
	return count, db.Error
}

func (m *TrackingManager) GetTracking(opts TrackingOption) (*entity.Tracking, error) {
	db := m.BuildStateQuery(opts)

	tk := &entity.Tracking{}
	db = db.Preload("Warehouse").First(tk)
	return tk, db.Error
}

func (m *TrackingManager) GetTrackingByField(opts TrackingOption, field string, result interface{}) error {
	db := m.BuildStateQuery(opts)
	db = db.Model(&entity.Tracking{}).Select(field)
	db = db.Scan(result)
	return db.Error
}

func (m *TrackingManager) Create(tk *entity.Tracking) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	tk.CreatedAt = time.Now()
	tk.UpdatedAt = tk.CreatedAt

	if err := tx.Create(tk).Error; err != nil {
		tx.Rollback()
		return err
	}

	err := tx.Table("packages").Where("id = ?", tk.PackageID).Update("shipping_fee", tk.ShipmentCost).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m *TrackingManager) GetCarrier(code string) (*entity.Carrier, error) {
	db := m.db
	carrier := &entity.Carrier{}
	db = db.Where("code = ?", code).First(carrier)
	return carrier, db.Error
}

func (m *TrackingManager) GetCarrierIDs(codes []string) ([]int64, error) {
	db := m.db
	var result []int64
	db = db.Model(&entity.Carrier{}).Where("code IN (?)", codes).Select("id").Scan(&result)
	return result, db.Error
}

func (m *TrackingManager) GetCarrierByID(id int64) (*entity.Carrier, error) {
	db := m.db
	carrier := &entity.Carrier{}
	db = db.Where("id = ?", id).First(carrier)
	return carrier, db.Error
}

func (m *TrackingManager) CreateManifest(manifest []*entity.Manifest) error {
	db := m.db.Create(&manifest)
	return db.Error
}

func (m *TrackingManager) SaveManifestTracking(IDs []int64) error {
	db := m.db.Model(&entity.Tracking{}).Where("id IN (?)", IDs).UpdateColumns(map[string]interface{}{
		"is_manifested": constant.TrackingManifested,
		"updated_at":    time.Now(),
	})
	return db.Error
}

func (m *TrackingManager) CreateManifestWithTx(manifest []*entity.Manifest) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if len(manifest) > 1000 {
		chunk := 1000
		count := 1
		i := 0
	SplitData:
		data := make([]*entity.Manifest, 0)
		for i < len(manifest) {
			data = append(data, manifest[i])
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
		if i < len(manifest) {
			goto SplitData
		}
	} else {
		if err := tx.Save(&manifest).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m *TrackingManager) GetContainerInShipment(shipmentID int64) ([]int64, error) {
	container_ids := []int64{}
	db := m.db
	db = db.Model(&entity.Container{}).Select("id")
	db = db.Where("shipment_id  = ?", shipmentID).Find(&container_ids)

	return container_ids, db.Error
}

func (m *TrackingManager) GetContainerPackageInShipment(shipmentID int64, pkgIds []int64) ([]int64, error) {
	container_ids := []int64{}
	db := m.db
	db = db.Model(&entity.ContainerItem{}).Select("DISTINCT container_id")
	db = db.Joins("JOIN containers ON containers.id = container_items.container_id AND container_items.status = ?", constant.ContainerItemActive)
	db = db.Where("containers.shipment_id  = ? AND container_items.package_id IN (?)", shipmentID, pkgIds).Find(&container_ids)

	return container_ids, db.Error
}

func (m *TrackingManager) UpdateTracking(billID int64, tracking *entity.Tracking, pkg *entity.Package, userID int64, totalAMount float64, extrafees []entity.ExtraFee, auditlogs []entity.PackageAuditLog) error {
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

	var transactionAmount float64 = 0
	if len(extrafees) > 0 {
		if err := tx.Create(&extrafees).Error; err != nil {
			tx.Rollback()
			return err
		}

		for _, v := range extrafees {
			transactionAmount += v.Amount
		}
	}

	if tracking.ID > 0 {
		trackingMapString := map[string]interface{}{
			"LabelURL":     tracking.LabelURL,
			"ShipmentCost": tracking.ShipmentCost,
			"Weight":       tracking.Weight,
			"Height":       tracking.Height,
			"Width":        tracking.Width,
			"Length":       tracking.Length,
			"UpdatedAt":    time.Now(),
		}

		err := tx.Model(&entity.Tracking{}).Where("tracking_number = ? AND status != ?", tracking.TrackingNumber, constant.TrackingStatusCanceled).UpdateColumns(trackingMapString).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	} else {
		err := tx.Model(&entity.Tracking{}).Where("package_id = ?", pkg.ID).Updates(map[string]interface{}{
			"status":     constant.TrackingStatusCanceled,
			"updated_at": time.Now(),
		}).Error
		if err != nil {
			tx.Rollback()
			return err
		}

		tracking.Status = constant.TrackingStatusSuccess
		if err := tx.Create(tracking).Error; err != nil {
			tx.Rollback()
			return err
		}

		if totalAMount > 0 {
			transactionAmount += totalAMount

			extra := entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				BillID:         &billID,
				PackageID:      utils.Int64(pkg.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeCancelLabel,
				Amount:         totalAMount,
				Status:         constant.ExtraFeeStatusEnable,
			}

			if pkg.PackageCode != nil {
				extra.Description = fmt.Sprintf("Phí hủy label cho đơn %s (hold %.2f)", pkg.PackageCode.Code, extra.Amount)
			}

			if err := tx.Create(&extra).Error; err != nil {
				tx.Rollback()
				return err
			}

			auditLogType := constant.MapTypeAuditLogByExtraFeeTypeID[extra.ExtraFeeTypeID]
			if auditLogType > 0 {
				logs := &entity.PackageAuditLog{
					PackageID:     utils.Int64Value(extra.PackageID),
					Type:          auditLogType,
					UpdatedUserID: userID,
					Fee:           extra.Amount,
					Value:         extra.Description,
					Description:   extra.Description,
				}

				if err := tx.Model(&entity.PackageAuditLog{}).Create(&logs).Error; err != nil {
					tx.Rollback()
					return err
				}
			}

			refund := entity.PackageRefund{
				PackageID: pkg.ID,
				Amount:    totalAMount,
				Status:    constant.PackageRefundPending,
			}

			if err := tx.Create(&refund).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	if transactionAmount > 0 {
		if err := tx.Exec("UPDATE bills SET extra_fee=extra_fee+? WHERE id=?", transactionAmount, billID).Error; err != nil {
			tx.Rollback()
			return err
		}

		var balanceType string
		if pkg.Service.Code == constant.ServiceCNCode {
			// balanceType = "balance_china" // remove china wallet
			balanceType = "balance"
		} else {
			balanceType = "balance"
		}

		sqlString := fmt.Sprintf("UPDATE users SET %s = %s - ?, updated_at = ? WHERE id = ?", balanceType, balanceType)
		if err := tx.Exec(sqlString, transactionAmount, time.Now(), pkg.UserID).Error; err != nil {
			tx.Rollback()
			return err
		}

		info := &entity.UserInfo{}
		if err := tx.First(info).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				tx.Rollback()
				return err
			}

			now := time.Now()
			info = &entity.UserInfo{UserID: pkg.UserID, UpdatedAt: &now, CancelMaxAmount: constant.DefaultCancelMaxAMount}
			if err := tx.Create(info).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		strsql := "UPDATE user_infos SET debt_time = ? WHERE user_id = ? AND debt_time IS NULL AND (SELECT balance FROM users WHERE id = ? limit 1) < 0"
		if err := tx.Exec(strsql, time.Now(), pkg.UserID, pkg.UserID).Error; err != nil {
			tx.Rollback()
			return err
		}

		transaction := entity.Transaction{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			UserID:  pkg.UserID,
			AdminID: userID,
			Type:    constant.TransactionLogTypePay,
			Status:  constant.TransactionStatusSuccess,
			Amount:  transactionAmount,
			BillID:  &billID,
		}

		if err := tx.Create(&transaction).Error; err != nil {
			tx.Rollback()
			return err
		}

		transactionLog := entity.TransactionLog{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			UserID:        pkg.UserID,
			AdminID:       userID,
			TransactionID: transaction.ID,
			Amount:        transaction.Amount,
			Type:          transaction.Type,
			Status:        transaction.Status,
			BillID:        transaction.BillID,
		}

		if err := tx.Create(&transactionLog).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	pkgMapString := map[string]interface{}{
		"Status":    constant.PackageStatusWareHouseLabeled,
		"UpdatedAt": time.Now(),
	}

	err := tx.Model(&entity.Package{}).Where("id = ?", pkg.ID).UpdateColumns(pkgMapString).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	if len(auditlogs) > 0 {
		if err := tx.Model(&entity.PackageAuditLog{}).Create(&auditlogs).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m *TrackingManager) Update(tk *entity.Tracking) error {
	tk.UpdatedAt = time.Now()
	db := m.db.Save(tk)
	return db.Error
}

func (m *TrackingManager) UpdateTrackingAdmin(billID int64, tracking *entity.Tracking, pkg *entity.Package, userID int64, totalAMount float64) error {
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

	if tracking.ID > 0 {
		trackingMapString := map[string]interface{}{
			"LabelURL":     tracking.LabelURL,
			"ShipmentCost": tracking.ShipmentCost,
			"Weight":       tracking.Weight,
			"Height":       tracking.Height,
			"Width":        tracking.Width,
			"Length":       tracking.Length,
			"UpdatedAt":    time.Now(),
		}

		err := tx.Model(&entity.Tracking{}).Where("tracking_number = ? AND status != ?", tracking.TrackingNumber, constant.TrackingStatusCanceled).UpdateColumns(trackingMapString).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	} else {
		err := tx.Model(&entity.Tracking{}).Where("package_id = ?", pkg.ID).Updates(map[string]interface{}{
			"status":     constant.TrackingStatusCanceled,
			"updated_at": time.Now(),
		}).Error
		if err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Create(tracking).Error; err != nil {
			tx.Rollback()
			return err
		}

		if totalAMount > 0 {
			extra := entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				BillID:         &billID,
				PackageID:      utils.Int64(pkg.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeCancelLabel,
				Amount:         totalAMount,
				Status:         constant.ExtraFeeStatusEnable,
			}

			if pkg.PackageCode != nil {
				extra.Description = fmt.Sprintf("Phí hủy label cho đơn %s (hold %.2f)", pkg.PackageCode.Code, extra.Amount)

			}

			if err := tx.Create(&extra).Error; err != nil {
				tx.Rollback()
				return err
			}

			if err := tx.Exec("UPDATE bills SET extra_fee=extra_fee+? WHERE id=?", totalAMount, billID).Error; err != nil {
				tx.Rollback()
				return err
			}

			var balanceType string
			if pkg.Service.Code == constant.ServiceCNCode {
				// balanceType = "balance_china" // remove china wallet
				balanceType = "balance"
			} else {
				balanceType = "balance"
			}

			sqlString := fmt.Sprintf("UPDATE users SET %s = %s - ?, updated_at = ? WHERE id = ?", balanceType, balanceType)
			if err := tx.Exec(sqlString, totalAMount, time.Now(), pkg.UserID).Error; err != nil {
				tx.Rollback()
				return err
			}

			info := &entity.UserInfo{}
			if err := tx.First(info).Error; err != nil {
				if err != gorm.ErrRecordNotFound {
					tx.Rollback()
					return err
				}

				now := time.Now()
				info = &entity.UserInfo{UserID: pkg.UserID, UpdatedAt: &now, CancelMaxAmount: constant.DefaultCancelMaxAMount}
				if err := tx.Create(info).Error; err != nil {
					tx.Rollback()
					return err
				}
			}

			strsql := "UPDATE user_infos SET debt_time = ? WHERE user_id = ? AND debt_time IS NULL AND (SELECT balance FROM users WHERE id = ? limit 1) < 0"
			if err := tx.Exec(strsql, time.Now(), pkg.UserID, pkg.UserID).Error; err != nil {
				tx.Rollback()
				return err
			}

			auditLogType := constant.MapTypeAuditLogByExtraFeeTypeID[extra.ExtraFeeTypeID]
			if auditLogType > 0 {
				logs := &entity.PackageAuditLog{
					PackageID:     utils.Int64Value(extra.PackageID),
					Type:          auditLogType,
					UpdatedUserID: userID,
					Fee:           extra.Amount,
					Value:         extra.Description,
					Description:   extra.Description,
				}

				if err := tx.Model(&entity.PackageAuditLog{}).Create(&logs).Error; err != nil {
					tx.Rollback()
					return err
				}
			}

			transaction := entity.Transaction{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				UserID:  pkg.UserID,
				AdminID: userID,
				Type:    constant.TransactionLogTypePay,
				Status:  constant.TransactionStatusSuccess,
				Amount:  totalAMount,
				BillID:  &billID,
			}

			if err := tx.Create(&transaction).Error; err != nil {
				tx.Rollback()
				return err
			}

			transactionLog := entity.TransactionLog{
				Model: dbgorm.Model{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				UserID:        pkg.UserID,
				AdminID:       userID,
				TransactionID: transaction.ID,
				Amount:        transaction.Amount,
				Type:          transaction.Type,
				Status:        transaction.Status,
				BillID:        transaction.BillID,
			}

			if err := tx.Create(&transactionLog).Error; err != nil {
				tx.Rollback()
				return err
			}

			refund := entity.PackageRefund{
				PackageID: pkg.ID,
				Amount:    totalAMount,
				Status:    constant.PackageRefundPending,
			}

			if err := tx.Create(&refund).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	if pkg.Status == constant.PackageDeliverLogTypeInWareHouse {
		pkgMapString := map[string]interface{}{
			"Status":    constant.PackageStatusWareHouseLabeled,
			"UpdatedAt": time.Now(),
		}

		err := tx.Model(&entity.Package{}).Where("id = ?", pkg.ID).UpdateColumns(pkgMapString).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m *TrackingManager) CreateTrackingIntransit(trackings []entity.Tracking) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
			tx.Rollback()
		}
	}()

	if err := tx.Save(&trackings).Error; err != nil {
		return err
	}
	ids := []int64{}
	for _, tracking := range trackings {
		ids = append(ids, tracking.PackageID)
	}
	if err := tx.Model(&entity.Package{}).Where("id IN (?)", ids).Updates(map[string]interface{}{
		"status":     constant.PackageStatusInTransit,
		"updated_at": time.Now(),
	}).Error; err != nil {
		return err
	}
	return tx.Commit().Error
}

func (m *TrackingManager) CreateTrackingLabeled(trackings []entity.Tracking) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
			tx.Rollback()
		}
	}()

	if err := tx.Save(&trackings).Error; err != nil {
		return err
	}

	for _, tracking := range trackings {
		if err := tx.Model(&entity.Package{}).Where("id = ?", tracking.PackageID).Updates(map[string]interface{}{
			"label":      tracking.LabelURL,
			"status":     constant.PackageStatusPendingPickup,
			"updated_at": time.Now(),
		}).Error; err != nil {
			return err
		}
	}

	return tx.Commit().Error
}
