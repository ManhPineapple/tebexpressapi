package sqlmanager

import (
	"errors"
	"fmt"
	"log"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/dbgorm"
	"time"

	"github.com/spf13/viper"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CustomerShipmentManager struct {
	db *gorm.DB
}

func NewCustomerShipmentManager(db *gorm.DB) *CustomerShipmentManager {
	return &CustomerShipmentManager{db}
}

type CustomerShipmentOption struct {
	ID                 int64
	UserID             int64
	PackageID          int64
	UserPreload        bool
	ExtraFeePreload    bool
	Keyword            string
	Status             int
	InStatus           []int64
	StartDate          string
	EndDate            string
	Limit              int
	Offset             int
	IgnoreUsers        []int64
	PackageCodePreload bool
	SupportID          int64
}

func (m *CustomerShipmentManager) buildQueryCustomShipment(opts CustomerShipmentOption) *gorm.DB {
	db := m.db

	if opts.ID > 0 {
		db = db.Where("customer_shipments.id = ?", opts.ID)
	}

	if opts.UserID > 0 {
		db = db.Where("customer_shipments.user_id = ?", opts.UserID)
	}

	if opts.Keyword != "" {
		ids := []int64{}
		m.db.Table("packages").
			Joins("LEFT JOIN package_codes ON package_codes.id=packages.package_code_id").
			Where("packages.order_number=? OR package_codes.code=?", opts.Keyword, opts.Keyword).
			Pluck("customer_shipment_id", &ids)

		if len(ids) > 0 {
			db = db.Where("customer_shipments.id IN (?) OR customer_shipments.id=?", ids, opts.Keyword)
		} else {
			db = db.Where("customer_shipments.id=?", opts.Keyword)
		}
	}

	if len(opts.StartDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(customer_shipments.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)", opts.StartDate)
	}

	if len(opts.EndDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(customer_shipments.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)", opts.EndDate)
	}

	if len(opts.IgnoreUsers) > 0 {
		db = db.Where("customer_shipments.user_id NOT IN (?)", opts.IgnoreUsers)
	}

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	if opts.UserPreload {
		db = db.Preload("User")
	}

	if opts.SupportID > 0 {
		db = db.Joins("JOIN user_permissions ON customer_shipments.user_id=user_permissions.customer_id")
		db = db.Where("user_permissions.support_id=?", opts.SupportID)
	}

	if opts.ExtraFeePreload {
		db = db.Preload("ExtraFees", func(db *gorm.DB) *gorm.DB {
			db = db.Where("status = ?", constant.ExtraFeeStatusEnable)
			return db
		})
	}

	return db
}

func (m *CustomerShipmentManager) GetPackagesByShipment(opt CustomerShipmentOption) ([]entity.Package, error) {
	var packages = []entity.Package{}
	if opt.ID < 1 {
		return packages, nil
	}

	db := m.db.Where("customer_shipment_id=?", opt.ID)
	if opt.Limit > 0 {
		db = db.Limit(opt.Limit)
	}

	if opt.Offset > 0 {
		db = db.Offset(opt.Offset)
	}

	if opt.PackageCodePreload {
		db = db.Preload("PackageCode")
	}

	db = db.Find(&packages)
	return packages, db.Error
}

func (m *CustomerShipmentManager) Fulfill(shipment *entity.CustomerShipment, bill *entity.Bill, packages []entity.Package) error {
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

	shipment.UpdatedAt = time.Now()
	if err := tx.Save(shipment).Error; err != nil {
		tx.Rollback()
		return err
	}

	_bill := &entity.Bill{}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=?", bill.ID).Find(_bill).Error; err != nil {
		tx.Rollback()
		return err
	}

	user := &entity.User{}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=?", shipment.UserID).Find(user).Error; err != nil {
		tx.Rollback()
		return err
	}

	userinfo := &entity.UserInfo{}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id=?", shipment.UserID).Find(userinfo).Error; err != nil {
		tx.Rollback()
		return err
	}

	env := viper.GetString("env")
	var totalAmount float64
	deliverLogs := []entity.PackageDeliverLog{}

	for _, pkg := range packages {
		totalAmount += pkg.ShippingFee

		packageIDGenCode := pkg.ID

		// gen package code
		if pkg.ID > constant.MaxPackageIDGenerate {
			result := &entity.PackageCode{}
			if err := tx.Where(
				"status = ? AND package_id_generate IS NOT NULL AND (user_id <> ? OR (user_id=? AND service_id != ?))",
				constant.PackageCodeDisable,
				pkg.UserID,
				pkg.UserID,
				pkg.ServiceID,
			).First(result).Error; err != nil {
				tx.Rollback()
				return err
			}

			packageIDGenCode = result.PackageIDGenerate
		}

		numGen := fmt.Sprintf("%05d%02d%08d", shipment.UserID, pkg.ServiceID, packageIDGenCode)

		prefixCode := entity.PartnerPrefixCodeMap[pkg.PartnerID]
		code := fmt.Sprintf("%v%05d%02d%08d%d", prefixCode, shipment.UserID, pkg.ServiceID, packageIDGenCode, utils.GenerateLastDigitCode(numGen))

		if env == "development" {
			code = fmt.Sprintf("%s%s", code, "DEV")
		}

		var count int64
		if err := tx.Model(&entity.PackageCode{}).Where("code=? AND (user_id = ? OR status=?)", code, shipment.UserID, constant.PackageCodeEnable).Count(&count).Error; err != nil {
			tx.Rollback()
			return err
		}

		if count > 0 {
			tx.Rollback()
			return errors.New("create package code duplicate or is active")
		}

		packageCode := &entity.PackageCode{
			UserID:            shipment.UserID,
			Code:              code,
			Status:            constant.PackageCodeEnable,
			ServiceID:         pkg.ServiceID,
			PackageIDGenerate: packageIDGenCode,
		}

		if err := tx.Create(packageCode).Error; err != nil {
			tx.Rollback()
			return err
		}

		if packageCode.ID < 1 {
			tx.Rollback()
			return errors.New("create package code failed")
		}

		mapChangePackage := map[string]interface{}{
			"updated_at":      time.Now(),
			"status":          constant.PackageStatusPendingPickup,
			"bill_id":         bill.ID,
			"package_code_id": packageCode.ID,
		}

		if err := tx.Model(&entity.Package{}).Where("id=?", pkg.ID).UpdateColumns(mapChangePackage).Error; err != nil {
			tx.Rollback()
			return err
		}

		deliverLogs = append(deliverLogs, entity.PackageDeliverLog{
			PackageID: pkg.ID,
			Status:    constant.DeliverLogTebexpressInTransit,
			Type:      constant.PackageDeliverLogTypePendingPickup,
		})
	}

	if len(deliverLogs) > 0 {
		if err := tx.Create(&deliverLogs).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if len(shipment.ExtraFees) > 0 {
		for _, v := range shipment.ExtraFees {
			totalAmount += v.Amount
			v.BillID = utils.Int64(bill.ID)
			change := map[string]interface{}{
				"updated_at": time.Now(),
				"bill_id":    bill.ID,
			}

			if err := tx.Model(&entity.ExtraFee{}).Where("id=?", v.ID).UpdateColumns(change).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	if err := tx.Exec("UPDATE bills SET shipping_fee = shipping_fee + ? WHERE id=?", totalAmount, bill.ID).Error; err != nil {
		tx.Rollback()
		return err
	}

	var balanceType string = "balance"
	sqlString := fmt.Sprintf("UPDATE users SET %s = %s - ?, updated_at = ? WHERE id = ?", balanceType, balanceType)
	if err := tx.Exec(sqlString, totalAmount, time.Now(), shipment.UserID).Error; err != nil {
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
		info = &entity.UserInfo{UserID: shipment.UserID, UpdatedAt: &now, CancelMaxAmount: constant.DefaultCancelMaxAMount}
		if err := tx.Create(info).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	sql := `UPDATE user_infos SET debt_time = ? WHERE user_id = ? AND debt_time IS NULL AND (SELECT balance FROM users WHERE id = ? limit 1) < 0`
	if err := tx.Exec(sql, time.Now(), shipment.UserID, shipment.UserID).Error; err != nil {
		tx.Rollback()
		return err
	}

	transaction := &entity.Transaction{
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID: shipment.UserID,
		Type:   constant.TransactionLogTypePay,
		Status: constant.TransactionStatusSuccess,
		Amount: totalAmount,
		BillID: utils.Int64(bill.ID),
	}

	err := tx.Create(&transaction).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	transactionLog := &entity.TransactionLog{
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID:        shipment.UserID,
		TransactionID: transaction.ID,
		Amount:        transaction.Amount,
		Type:          transaction.Type,
		Status:        transaction.Status,
		BillID:        transaction.BillID,
	}

	if err = tx.Model(entity.TransactionLog{}).Create(transactionLog).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m *CustomerShipmentManager) CountCustomerShipments(options CustomerShipmentOption) (int64, error) {
	db := m.buildQueryCustomShipment(options)

	if options.Status > 0 || len(options.InStatus) > 0 {
		sdb := m.db.Table("packages").Select("customer_shipment_id, MIN(packages.status) AS status").Group("customer_shipment_id")
		sdb = sdb.Where("customer_shipment_id>?", 0)

		if options.Status > 0 {
			sdb = sdb.Having("status=?", options.Status)
		}

		if len(options.InStatus) > 0 {
			sdb = sdb.Having("status IN (?)", options.InStatus)
		}

		db = db.Where("customer_shipments.id IN (?)", m.db.Raw("SELECT customer_shipment_id FROM (?) AS ps", sdb))
	}

	var count int64
	db = db.Model(&entity.CustomerShipment{}).Count(&count)
	return count, db.Error
}

func (m *CustomerShipmentManager) GetCustomerShipments(options CustomerShipmentOption) ([]entity.CustomerShipment, error) {
	db := m.buildQueryCustomShipment(options)

	db = db.Select(`
		customer_shipments.id,
		customer_shipments.user_id,
		customer_shipments.weight,
		customer_shipments.price,
		customer_shipments.created_at,
		customer_shipments.updated_at,
		MIN(packages.status) AS status
	`)

	db = db.Joins("INNER JOIN packages ON customer_shipments.id = packages.customer_shipment_id").Group("customer_shipment_id")
	if options.Status > 0 {
		db = db.Having("status=?", options.Status)
	}

	if len(options.InStatus) > 0 {
		db = db.Having("status IN (?)", options.InStatus)
	}

	var shipments []entity.CustomerShipment
	db = db.Order("customer_shipments.id DESC").Find(&shipments)
	return shipments, db.Error
}

func (m *CustomerShipmentManager) GetCustomerShipment(options CustomerShipmentOption) (*entity.CustomerShipment, error) {
	db := m.buildQueryCustomShipment(options)
	var shipment = &entity.CustomerShipment{}
	db = db.First(shipment)
	return shipment, db.Error
}

type CustomerShipmentItemOptions struct {
	ShipmentID int64
	UserID     int64
	Keyword    string
	Limit      int
	Offset     int
	Status     int
}

func (m *CustomerShipmentManager) BuildCustomerShipmentItemQuery(opts CustomerShipmentItemOptions) *gorm.DB {
	db := m.db

	if opts.ShipmentID > 0 {
		db = db.Where("customer_shipment_id = ?", opts.ShipmentID)
	}

	if opts.UserID > 0 {
		db = db.Where("user_id = ?", opts.UserID)
	}

	if opts.Status > 0 {
		db = db.Where("status = ?", opts.Status)
	}

	if opts.Keyword != "" {
		ids := []int64{}
		m.db.Table("package_codes").Where("code", opts.Keyword).Pluck("id", &ids)
		if len(ids) > 0 {
			db = db.Where("package_code_id IN (?) OR id=?", ids, opts.Keyword)
		} else {
			db = db.Where("id=?", opts.Keyword)
		}
	}

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	return db
}

func (m *CustomerShipmentManager) CountCustomerShipmentItems(opts CustomerShipmentItemOptions) (int64, error) {
	db := m.BuildCustomerShipmentItemQuery(opts)
	var count int64
	db = db.Model(&entity.Package{}).Count(&count)
	return count, db.Error
}

func (m *CustomerShipmentManager) GetCustomerShipmentItems(opts CustomerShipmentItemOptions) ([]entity.Package, error) {
	db := m.BuildCustomerShipmentItemQuery(opts)
	var packages []entity.Package
	db = db.Preload("PackageCode")
	db = db.Preload("Tracking", func(db *gorm.DB) *gorm.DB {
		db = db.Where("trackings.status != ?", constant.TrackingStatusCanceled)
		return db
	})
	db = db.Preload("ExtraFee", func(db *gorm.DB) *gorm.DB {
		db = db.Where("extra_fees.status = ?", constant.ExtraFeeStatusEnable)
		return db
	})

	db = db.Preload("Service")

	db = db.Order("id DESC").Find(&packages)
	return packages, db.Error
}

func (m *CustomerShipmentManager) GetCustomerShipmentPrice(id int64) (float64, error) {
	var total float64 = 0
	if err := m.db.Model(&entity.Package{}).
		Select("IFNULL(SUM(shipping_fee), 0)").
		Where("customer_shipment_id=?", id).
		Row().Scan(&total); err != nil {
		return 0, err
	}

	var extraAmount float64 = 0
	if err := m.db.Model(&entity.ExtraFee{}).
		Select("IFNULL(SUM(extra_fees.amount) , 0)").
		Joins("INNER JOIN packages ON packages.id=extra_fees.package_id").
		Where("extra_fees.status = ?", constant.ExtraFeeStatusEnable).
		Where("packages.customer_shipment_id=?", id).
		Row().Scan(&extraAmount); err != nil {
		return 0, err
	}

	return total + extraAmount, nil
}

func (m *CustomerShipmentManager) GetCustomerShipmentsPrice(ids []int64) (map[int64]float64, error) {
	result := make(map[int64]float64)

	prices := []struct {
		ID     int64
		Amount float64
	}{}

	if err := m.db.Model(&entity.Package{}).
		Select("customer_shipment_id AS id, IFNULL(SUM(shipping_fee), 0) AS amount").
		Group("customer_shipment_id").
		Where("customer_shipment_id IN (?)", ids).Scan(&prices).Error; err != nil {
		return nil, err
	}

	for _, v := range prices {
		result[v.ID] = v.Amount
	}

	prices = []struct {
		ID     int64
		Amount float64
	}{}

	if err := m.db.Model(&entity.ExtraFee{}).
		Select("packages.customer_shipment_id AS id, IFNULL(SUM(extra_fees.amount) , 0) AS amount").
		Joins("INNER JOIN packages ON packages.id=extra_fees.package_id").
		Where("extra_fees.status = ?", constant.ExtraFeeStatusEnable).
		Group("packages.customer_shipment_id").
		Where("packages.customer_shipment_id IN (?)", ids).Scan(&prices).Error; err != nil {
		return nil, err
	}

	for _, v := range prices {
		result[v.ID] += v.Amount
	}

	prices = []struct {
		ID     int64
		Amount float64
	}{}
	if err := m.db.Model(&entity.ExtraFee{}).
		Select("extra_fees.customer_shipment_id AS id, IFNULL(SUM(extra_fees.amount) , 0) AS amount").
		Where("extra_fees.status = ?", constant.ExtraFeeStatusEnable).
		Group("extra_fees.customer_shipment_id").
		Where("extra_fees.customer_shipment_id IN (?)", ids).Scan(&prices).Error; err != nil {
		return nil, err
	}

	for _, v := range prices {
		result[v.ID] += v.Amount
	}

	return result, nil
}

func (m *CustomerShipmentManager) GetCustomerShipmentsMapStatus(ids []int64) (map[int64]int, error) {
	values := []struct {
		ID     int64
		Status int
	}{}

	if err := m.db.Model(&entity.Package{}).
		Select("customer_shipment_id AS id, min(status) AS status").
		Group("customer_shipment_id").
		Where("customer_shipment_id IN (?)", ids).Scan(&values).Error; err != nil {
		return nil, err
	}

	result := make(map[int64]int)
	for _, v := range values {
		result[v.ID] = v.Status
	}

	return result, nil
}

func (m *CustomerShipmentManager) GetCustomerShipmentStatus(id int64) (int, error) {
	var status int = 0
	if err := m.db.Model(&entity.Package{}).
		Select("MIN(status)").
		Group("customer_shipment_id").
		Where("customer_shipment_id=?", id).
		Row().Scan(&status); err != nil {
		return 0, err
	}

	return status, nil
}

func (m *CustomerShipmentManager) Cancel(shipment *entity.CustomerShipment, bill *entity.Bill, packages []entity.Package) error {
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

	shipment.UpdatedAt = time.Now()
	if err := tx.Omit(clause.Associations).Save(shipment).Error; err != nil {
		tx.Rollback()
		return err
	}

	var logs = []entity.PackageDeliverLog{}
	var extras = []entity.ExtraFee{}
	var totalRefund float64 = 0
	var cancelIDs = []int64{}
	var archivedIDs = []int64{}

	now := time.Now()
	model := dbgorm.Model{
		CreatedAt: now,
		UpdatedAt: now,
	}

	isHasRefundShipment := false
	for _, item := range packages {
		if item.Status == constant.PackageStatusCreated {
			archivedIDs = append(archivedIDs, item.ID)
			logs = append(logs, entity.PackageDeliverLog{
				Model:     model,
				PackageID: item.ID,
				Status:    constant.DeliverLogTebexpressArchived,
				Type:      constant.PackageDeliverLogTypeArchived,
				UserID:    utils.Int64(shipment.UserID),
			})
		} else {
			isHasRefundShipment = true
			cancelIDs = append(cancelIDs, item.ID)
			logs = append(logs, entity.PackageDeliverLog{
				Model:     model,
				PackageID: item.ID,
				Status:    constant.DeliverLogTebexpressCanceled,
				Type:      constant.PackageDeliverLogTypeCancelled,
				UserID:    utils.Int64(shipment.UserID),
			})

			amount := item.ShippingFee
			if len(item.ExtraFee) > 0 {
				for _, v := range item.ExtraFee {
					amount += v.Amount
				}
			}

			if amount > 0 {
				des := fmt.Sprintf("Hoàn tiền cho đơn #%v", item.ID)
				if item.PackageCode != nil {
					des = fmt.Sprintf("Hoàn tiền cho đơn %v", item.PackageCode.Code)
				}

				extras = append(extras, entity.ExtraFee{
					Model:          model,
					PackageID:      utils.Int64(item.ID),
					BillID:         utils.Int64(bill.ID),
					Amount:         -amount,
					Description:    des,
					ExtraFeeTypeID: constant.ExtraFeeTypeRefund,
					Status:         constant.ExtraFeeStatusEnable,
				})

				totalRefund += amount
			}
		}
	}

	shipmentExtraFees := shipment.ExtraFees
	if len(shipmentExtraFees) > 0 && isHasRefundShipment {
		for _, v := range shipmentExtraFees {
			des := fmt.Sprintf("Hoàn tiền cho lô FBA #%v", shipment.ID)
			extras = append(extras, entity.ExtraFee{
				Model:              model,
				CustomerShipmentID: utils.Int64(shipment.ID),
				BillID:             utils.Int64(bill.ID),
				Amount:             -v.Amount,
				Description:        des,
				ExtraFeeTypeID:     constant.ExtraFeeTypeRefund,
				Status:             constant.ExtraFeeStatusEnable,
			})
			totalRefund += v.Amount
		}
	}

	if len(archivedIDs) > 0 {
		change := map[string]interface{}{
			"updated_at": now,
			"status":     constant.PackageStatusArchived,
		}

		if err := tx.Model(&entity.Package{}).Where("id IN (?)", archivedIDs).Updates(change).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if len(cancelIDs) > 0 {
		change := map[string]interface{}{
			"updated_at": now,
			"status":     constant.PackageStatusCancelled,
		}

		if err := tx.Model(&entity.Package{}).Where("id IN (?)", cancelIDs).Updates(change).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if len(extras) > 0 {
		if err := tx.Create(&extras).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if len(logs) > 0 {
		if err := tx.Create(&logs).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if totalRefund > 0 {
		_bill := &entity.Bill{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=?", bill.ID).Find(_bill).Error; err != nil {
			tx.Rollback()
			return err
		}

		_user := &entity.User{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=?", shipment.UserID).Find(_user).Error; err != nil {
			tx.Rollback()
			return err
		}

		_userinfo := &entity.UserInfo{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id=?", shipment.UserID).Find(_userinfo).Error; err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Exec("UPDATE bills SET extra_fee = extra_fee - ? WHERE id=?", totalRefund, bill.ID).Error; err != nil {
			tx.Rollback()
			return err
		}

		var balanceType string = "balance"
		sqlString := fmt.Sprintf("UPDATE users SET %s = %s + ?, updated_at = ? WHERE id = ?", balanceType, balanceType)
		if err := tx.Exec(sqlString, totalRefund, time.Now(), shipment.UserID).Error; err != nil {
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
			info = &entity.UserInfo{UserID: shipment.UserID, UpdatedAt: &now, CancelMaxAmount: constant.DefaultCancelMaxAMount}
			if err := tx.Create(info).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		sql := `UPDATE user_infos SET debt_time = NULL WHERE user_id = ? AND debt_time IS NOT NULL AND (SELECT balance FROM users WHERE id = ? limit 1) >= 0`
		if err := tx.Exec(sql, shipment.UserID, shipment.UserID).Error; err != nil {
			tx.Rollback()
			return err
		}

		transaction := &entity.Transaction{
			Model:  model,
			UserID: shipment.UserID,
			Type:   constant.TransactionLogTypeRefund,
			Status: constant.TransactionStatusSuccess,
			Amount: totalRefund,
			BillID: utils.Int64(bill.ID),
		}

		err := tx.Create(&transaction).Error
		if err != nil {
			tx.Rollback()
			return err
		}

		transactionLog := &entity.TransactionLog{
			Model:         model,
			UserID:        shipment.UserID,
			TransactionID: transaction.ID,
			Amount:        transaction.Amount,
			Type:          transaction.Type,
			Status:        transaction.Status,
			BillID:        transaction.BillID,
		}

		if err = tx.Model(entity.TransactionLog{}).Create(transactionLog).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m *CustomerShipmentManager) GetCustomerShipmentActualWeight(id int64) (float64, error) {
	var total float64 = 0
	if err := m.db.Model(&entity.Package{}).
		Select("SUM(IF(actual_weight > weight, actual_weight, weight))").
		Where("customer_shipment_id=?", id).
		Row().Scan(&total); err != nil {
		return 0, err
	}

	return total, nil
}

func (m *CustomerShipmentManager) SaveShipmentAndPackages(shipment *entity.CustomerShipment) error {
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

	if err := tx.Omit(clause.Associations).Save(&shipment).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Omit(clause.Associations).Save(&shipment.Packages).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m *CustomerShipmentManager) GetFirstPackage(shipmentID int64) (*entity.Package, error) {
	db := m.db.Where("customer_shipment_id=?", shipmentID)
	db = db.Where("status NOT IN (?)", []int{constant.PackageStatusCancelled})
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

	pkg := &entity.Package{}
	db = db.First(pkg)
	return pkg, db.Error
}
