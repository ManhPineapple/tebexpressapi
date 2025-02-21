package sqlmanager

import (
	"errors"
	"fmt"
	"log"
	"math"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/dbgorm"
	"time"

	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BillManager struct {
	db *gorm.DB
}

type BillQueryOption struct {
	ID                 int64
	UserID             int64
	Search             string
	SearchBy           string
	SearchDate         string
	Status             int
	Limit              int
	Offset             int
	BillID             int64
	BillCode           string
	StartDate          string
	PackageFee         string
	ExtraFee           string
	EndDate            string
	SupportID          int64
	SubDay             int64
	HidePreloadUser    bool
	HidePreloadPackage bool
	IgnoreUsers        []int64
	PartnerID          int64
	ServiceCode		   string
}
type BillFeeQueryOption struct {
	UserID   int64
	BillCode string
	Type     int
	Limit    int
	Offset   int
}

type BillPackageQueryOption struct {
	Select          string
	BillID          int64
	UserID          int64
	Limit           int
	Offset          int
	BillCode        string
	PerloadExtraFee bool
	PerloadTracking bool
}

type CreateBillOption struct {
	UserID      int64
	Packages    []entity.Package
	ShippingFee float64
	BillID      int64
	IsPrePaid   bool
	Point       int
}

type UpdateExtrafee struct {
	UserID       int64
	Extrafee     *entity.ExtraFee
	ExtraFeeType *entity.ExtraFeeType
	BillID       int64
	UserBillID   int64
}

func NewBillManager(db *gorm.DB) *BillManager {
	return &BillManager{
		db: db,
	}
}

func (m *BillManager) BuildQueryBill(opts BillQueryOption) *gorm.DB {
	db := m.db

	if opts.UserID > 0 {
		db = db.Where("bills.user_id = ?", opts.UserID)
	}

	if len(opts.IgnoreUsers) > 0 {
		db = db.Where("bills.user_id NOT IN (?)", opts.IgnoreUsers)
	}

	if opts.PartnerID > 0 {
		db = db.Joins("JOIN users partneruser ON partneruser.id = bills.user_id")
		db = db.Where("partneruser.partner_id = ?", opts.PartnerID)
	}

	if opts.SupportID > 0 {
		db = db.Joins("JOIN users aliasuser ON aliasuser.id = bills.user_id")
		db = db.Joins("JOIN user_permissions  ON aliasuser.id=user_permissions.customer_id")
		db = db.Where("user_permissions.support_id = ?", opts.SupportID)
	}

	if opts.SearchDate != "" {
		db = db.Where("DATE_FORMAT(convert_tz(created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') = DATE(?)", opts.SearchDate)
	}

	if len(opts.StartDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(bills.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)", opts.StartDate)
	}

	if len(opts.EndDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(bills.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)", opts.EndDate)
	}

	if opts.Search != "" {
		switch opts.SearchBy {
		case "bill_code":
			db = db.Where("code = ?", opts.Search)
		case "package":
			var pkg = entity.Package{}
			m.db.Model(&entity.Package{}).Where("id=?", opts.Search).Limit(1).Select("id, bill_id").First(&pkg)

			var billIDs []int64
			if pkg.ID > 0 {
				m.db.Model(&entity.ExtraFee{}).Where("package_id=?", pkg.ID).Select("bill_id").Pluck("bill_id", &billIDs)
				billIDs = append(billIDs, pkg.BillID)
				db = db.Where("bills.id IN (?)", billIDs)
			} else {
				db = db.Where("1=2")
			}
		case "code":
			var pkg = entity.Package{}
			m.db.Model(&entity.Package{}).Joins("JOIN package_codes on packages.package_code_id = package_codes.id").
				Where("package_codes.code=?", opts.Search).Limit(1).Select("packages.id, packages.bill_id").First(&pkg)

			var billIDs []int64
			if pkg.ID > 0 {
				m.db.Model(&entity.ExtraFee{}).Where("package_id=?", pkg.ID).Select("bill_id").Pluck("bill_id", &billIDs)
				billIDs = append(billIDs, pkg.BillID)
				db = db.Where("bills.id IN (?)", billIDs)
			} else {
				db = db.Where("1=2")
			}
		case "customer_id":
			db = db.Where("user_id=?", opts.Search)
		case "customer":
			subquery := m.db.Model(&entity.User{}).Where("email=? OR phone_number=?", opts.Search, opts.Search).Select("id")
			db = db.Where("user_id=(?)", subquery)
		case "tracking":
			var pkg = entity.Package{}
			m.db.Model(&entity.Package{}).
				Joins("LEFT JOIN trackings ON trackings.package_id = packages.id").
				Joins("LEFT JOIN package_codes on packages.package_code_id = package_codes.id").
				Where("(trackings.tracking_number = ? AND trackings.status = ?) OR package_codes.code=?", opts.Search, constant.TrackingStatusSuccess, opts.Search).
				Limit(1).Select("packages.id, packages.bill_id").First(&pkg)

			var billIDs []int64
			if pkg.ID > 0 {
				m.db.Model(&entity.ExtraFee{}).Where("package_id=?", pkg.ID).Select("bill_id").Pluck("bill_id", &billIDs)
				billIDs = append(billIDs, pkg.BillID)
				db = db.Where("bills.id IN (?)", billIDs)
			} else {
				db = db.Where("1=2")
			}
			break
		case "customer_full_name":
			db = db.Joins("JOIN users ON users.id = bills.user_id").
				Where("users.full_name LIKE (?)", fmt.Sprintf("%%%s%%", opts.Search))
			break
		case "order_number":
			var pkg = entity.Package{}
			m.db.Model(&entity.Package{}).
				Where("order_number=?", opts.Search).Limit(1).Select("packages.id, packages.bill_id").First(&pkg)

			var billIDs []int64
			if pkg.ID > 0 {
				m.db.Model(&entity.ExtraFee{}).Where("package_id=?", pkg.ID).Select("bill_id").Pluck("bill_id", &billIDs)
				billIDs = append(billIDs, pkg.BillID)
				db = db.Where("bills.id IN (?)", billIDs)
			} else {
				db = db.Where("1=2")
			}
		default:
			var pkg = entity.Package{}
			m.db.Model(&entity.Package{}).
				Joins("LEFT JOIN package_codes on packages.package_code_id = package_codes.id").
				Joins("LEFT JOIN trackings ON packages.id = trackings.package_id").
				Where("package_codes.code = ? OR trackings.tracking_number = ?", opts.Search, opts.Search).Limit(1).Select("packages.id, packages.bill_id").First(&pkg)

			var billIDs []int64
			if pkg.ID > 0 {
				m.db.Model(&entity.ExtraFee{}).Where("package_id=?", pkg.ID).Select("bill_id").Pluck("bill_id", &billIDs)
				billIDs = append(billIDs, pkg.BillID)
				db = db.Where("id IN (?)", billIDs)
			} else {
				db = db.Where("code = ?", opts.Search)
			}

		}
	}

	if opts.Status > 0 {
		db = db.Where("status = ?", opts.Status)
	}

	if (opts.BillID) > 0 {
		db = db.Where("id = ?", opts.BillID)
		db = db.Preload("User")
	}

	if opts.BillCode != "" {
		db = db.Where("code = ?", opts.BillCode)
		db = db.Preload("User")
	}

	if opts.SubDay > 0 {
		db = db.Where("DATE(created_at) = CURDATE() - INTERVAL ? DAY  ", opts.SubDay)
	}

	return db
}

func (m *BillManager) GetBill(opts BillQueryOption) (entity.Bill, error) {
	db := m.BuildQueryBill(opts)
	bill := entity.Bill{}

	db = db.Preload("Package", func(db *gorm.DB) *gorm.DB {
		db = db.Preload("PackageCode")
		db = db.Preload("Tracking", func(db *gorm.DB) *gorm.DB {
			db = db.Where("status !=?", constant.TrackingStatusCanceled)
			return db
		})

		db = db.Preload("ExtraFee")
		db = db.Limit(opts.Limit)
		db = db.Offset(opts.Offset)
		db = db.Order("id DESC")

		return db
	})

	db = db.First(&bill)
	return bill, db.Error
}

func (m *BillManager) GetBillByCode(code string, userID int64) (*entity.Bill, error) {
	bill := &entity.Bill{}

	db := m.db.Where("code=?", code)

	if userID > 0 {
		db = db.Where("user_id=?", userID)
	}

	db = db.First(bill)
	return bill, db.Error
}

func (m *BillManager) BuildExtraFeeQuery(opts BillFeeQueryOption) *gorm.DB {
	db := m.db

	statusRefundList := []int64{constant.ExtraFeeTypeRefund, constant.ExtraFeeTypeDiscount}
	sdb := m.db.Model(&entity.ExtraFeeType{}).Where("status=?", constant.ExtraFeeStatusEnable).Select("id")
	if opts.Type == constant.ExtraFeeTypeRefund {
		sdb = sdb.Where("id IN (?)", statusRefundList)
	} else if opts.Type == constant.ExtraFeeTypeAffiliate {
		sdb = sdb.Where("id = ?", opts.Type)
	} else {
		statusNotIn := append(statusRefundList, constant.ExtraFeeTypeAffiliate)
		sdb = sdb.Where("id NOT IN (?)", statusNotIn)
	}

	db = db.Where("extra_fees.extra_fee_type_id IN (?)", sdb)
	db = db.Joins("JOIN bills ON bills.id = extra_fees.bill_id").Where("bills.user_id = ? AND bills.code = ?", opts.UserID, opts.BillCode)
	db = db.Where("extra_fees.status IN (?)", []int64{constant.ExtraFeeStatusEnable, constant.ExtraFeeStatusUnPaid})

	return db
}

func (m *BillManager) UpdateStatusExtraFee(ExtraID int64, PackageID int64) error {
	db := m.db.Model(&entity.ExtraFee{}).Where("id=?", ExtraID).
		Updates(map[string]interface{}{"status": constant.ExtraFeeStatusDisable, "extra_fee_type_id": constant.ExtraFeeTypeRefund, "description": fmt.Sprintf("Hoàn tiền cho đơn hàng %v ", PackageID)})
	return db.Error
}

func (m *BillManager) GetExtraFeebyID(ExtraID int64) (*entity.ExtraFee, error) {
	extrafee := &entity.ExtraFee{}
	db := m.db.Where("id=?", ExtraID).First(extrafee)
	return extrafee, db.Error
}

func (m *BillManager) BuildExtraFeeAdminQuery(opts BillFeeQueryOption) *gorm.DB {
	db := m.db
	db = db.Where("bill_id IN (?) AND extra_fee_type_id IN (?)", m.db.Model(&entity.Bill{}).Where("code = ?", opts.BillCode).Select("id"),
		m.db.Model(&entity.ExtraFeeType{}).Where(" status = ?", constant.ExtraFeeStatusEnable).Select("id"))
	db = db.Preload("ExtraFeeType")
	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	db = db.Where("extra_fees.status IN (?)", []int64{constant.ExtraFeeStatusEnable, constant.ExtraFeeStatusUnPaid})
	db = db.Order("extra_fees.id DESC")
	return db
}

func (m *BillManager) GetBillExtraFee(opts BillFeeQueryOption) ([]entity.ExtraFee, error) {
	db := m.BuildExtraFeeQuery(opts)

	db = db.Preload("Package", func(db *gorm.DB) *gorm.DB {
		db = db.Select("id, package_code_id")
		return db
	})

	db = db.Preload("Package.PackageCode", func(db *gorm.DB) *gorm.DB {
		db = db.Select("id, code")
		return db
	})

	db = db.Preload("ExtraFeeType", func(db *gorm.DB) *gorm.DB {
		db = db.Select("id, name")
		return db
	})

	db = db.Preload("Coupon", func(db *gorm.DB) *gorm.DB {
		db = db.Select("id, code")
		return db
	})

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	db = db.Order("extra_fees.id DESC")

	extraFees := []entity.ExtraFee{}
	db = db.Find(&extraFees)
	return extraFees, db.Error
}

func (m *BillManager) GetBillAdminExtraFee(opts BillFeeQueryOption) ([]entity.ExtraFee, error) {
	db := m.BuildExtraFeeAdminQuery(opts)
	db = db.Preload("Package", func(db *gorm.DB) *gorm.DB {
		db = db.Preload("PackageCode")
		return db
	})
	extraFees := []entity.ExtraFee{}
	db = db.Find(&extraFees)
	return extraFees, db.Error
}

func (m *BillManager) CountBillPackages(opts BillQueryOption) (int64, error) {
	db := m.db.Model(&entity.Package{}).Where("bill_id = ?", opts.ID)
	var count int64
	db = db.Count(&count)
	return count, db.Error
}

func (m *BillManager) CountBillExtraFee(opts BillFeeQueryOption) (int64, error) {
	db := m.BuildExtraFeeQuery(opts)
	var count int64
	db = db.Model(&entity.ExtraFee{}).Count(&count)
	return count, db.Error
}

func (m *BillManager) CountBillAdminExtraFee(opts BillFeeQueryOption) (int64, error) {
	db := m.BuildExtraFeeAdminQuery(opts)
	var count int64
	db = db.Model(&entity.ExtraFee{}).Count(&count)
	return count, db.Error
}

func (m *BillManager) UpdateExtraFeeBill(opts UpdateExtrafee) (int64, error) {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			log.Println("panic :", r)
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return 0, err
	}
	txtDes := fmt.Sprintf("Hoàn tiền cho đơn hàng %v ", opts.Extrafee.PackageID)
	if opts.Extrafee.CustomerShipmentID != nil {
		txtDes = fmt.Sprintf("Hoàn tiền cho lô FBA #%v ", utils.Int64Value(opts.Extrafee.CustomerShipmentID))
	}
	var mapExtraFee = map[string]interface{}{
		"status":            constant.ExtraFeeStatusDisable,
		"extra_fee_type_id": constant.ExtraFeeTypeRefund,
		"description":       txtDes,
		"updated_at":        time.Now(),
	}

	if err := tx.Model(&entity.ExtraFee{}).Where("id=?", opts.Extrafee.ID).
		Updates(mapExtraFee).Error; err != nil {
		fmt.Errorf("update extra Fee error %v", err)
		tx.Rollback()
		return 0, err
	}

	billID := opts.BillID
	var index int64
	if opts.Extrafee.ExtraFeeTypeID == constant.ExtraFeeTypeRefund || opts.Extrafee.ExtraFeeTypeID == constant.ExtraFeeTypeAffiliate {
		index = constant.TransactionLogTypePay
	} else {
		index = constant.TransactionLogTypeRefund
	}
	transaction := &entity.Transaction{
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID:  opts.UserBillID,
		AdminID: opts.UserID,
		Type:    index,
		Status:  constant.TransactionStatusSuccess,
		Amount:  opts.Extrafee.Amount,
		BillID:  &billID,
	}

	if err := tx.Create(&transaction).Error; err != nil {
		fmt.Errorf("Create transaction error %v", err)
		tx.Rollback()
		return 0, err
	}

	transactionLog := &entity.TransactionLog{
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID:        opts.UserBillID,
		AdminID:       opts.UserID,
		TransactionID: transaction.ID,
		Amount:        transaction.Amount,
		Type:          transaction.Type,
		Status:        transaction.Status,
		BillID:        transaction.BillID,
	}

	if err := tx.Model(entity.TransactionLog{}).Create(transactionLog).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Exec("update bills SET extra_fee = extra_fee - ? where bills.id = ? ", transaction.Amount, opts.BillID).Error; err != nil {
		fmt.Errorf("Update extrafee  error %v", err)
		tx.Rollback()
		return 0, err
	}

	if err := tx.Exec("update users SET balance = balance + ? where users.id = ? ", transaction.Amount, opts.UserBillID).Error; err != nil {
		fmt.Errorf("Update user balance error %v", err)
		tx.Rollback()
		return 0, err
	}

	auditLogType := constant.MapTypeAuditLogByExtraFeeTypeID[opts.Extrafee.ExtraFeeTypeID]

	if auditLogType > 0 && opts.Extrafee.PackageID != nil {
		logs := &entity.PackageAuditLog{
			PackageID:     utils.Int64Value(opts.Extrafee.PackageID),
			Type:          auditLogType,
			UpdatedUserID: opts.UserID,
			Value:         fmt.Sprintf("Hủy %v", opts.ExtraFeeType.Name),
			Fee:           -transaction.Amount,
		}

		if err := tx.Model(&entity.PackageAuditLog{}).Create(&logs).Error; err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	return billID, tx.Commit().Error
}

func (m *BillManager) GetBillByID(BillID int64) (*entity.Bill, error) {
	bill := &entity.Bill{}
	db := m.db.Where("id=?", BillID).First(bill)
	return bill, db.Error
}

func (m *BillManager) CreateBillWithLabelPromotion(opts CreateBillOption, user *entity.User, refundCoupon *entity.ExtraFee, failPkgLength int) (int64, error) {
	trackings := []*entity.Tracking{}
	for _, pkg := range opts.Packages {
		if pkg.Tracking != nil && pkg.Tracking.ID < 1 {
			trackings = append(trackings, pkg.Tracking)
		}
	}

	if len(trackings) > 0 {
		if err := m.db.Create(trackings).Error; err != nil {
			fmt.Errorf("create tracking package: %v", err)
			return 0, err
		}
	}

	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return 0, err
	}

	bill := &entity.Bill{}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=?", bill.ID).Find(bill).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	user1 := &entity.User{}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=?", user.ID).Find(user1).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	userinfo := &entity.UserInfo{}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id=?", user.ID).Find(userinfo).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	billID := opts.BillID

	extras := opts.UserID / 1000
	prefix := int64(constant.BillCodePrefix)
	codeUID := opts.UserID

	if extras > 0 {
		prefix += extras
		codeUID = codeUID - extras*1000
	}

	if billID < 1 {
		t := time.Now().Add(time.Hour * 7)

		bill := &entity.Bill{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Code:   fmt.Sprintf("%d%03d%d%02d%02d", prefix, codeUID, t.Year(), t.Month(), t.Day()),
			UserID: opts.UserID,
			Status: constant.BillingStatusAwaitingPayment,
		}

		err := tx.Create(&bill).Error
		if err != nil {
			fmt.Errorf("Create bill error %v", err)
			tx.Rollback()
			return 0, err
		}

		billID = bill.ID
	}
	cIDs := make([]int64, 0)
	for _, pkg := range opts.Packages {
		if pkg.Tracking != nil && pkg.Tracking.ID > 0 {
			if err := tx.Save(pkg.Tracking).Error; err != nil {
				fmt.Errorf("create tracking package: %v", err)
				return 0, err
			}
		}

		mapPkg := make(map[string]interface{})
		mapPkg["updated_at"] = time.Now()
		mapPkg["status"] = constant.PackageStatusPendingPickup
		mapPkg["bill_id"] = billID
		mapPkg["label"] = pkg.Label
		mapPkg["label_promotion"] = true

		err := tx.Model(&entity.Package{}).Where("id = ?", pkg.ID).UpdateColumns(mapPkg).Error
		if err != nil {
			fmt.Errorf("Update package error %v", err)
			tx.Rollback()
			return 0, err
		}
		cIDs = append(cIDs, pkg.PackageCode.ID)
		deliverLog := &entity.PackageDeliverLog{
			PackageID: pkg.ID,
			Status:    constant.DeliverLogTebexpressInTransit,
			Type:      constant.PackageDeliverLogTypePendingPickup,
		}

		if err := tx.Create(&deliverLog).Error; err != nil {
			tx.Rollback()
			return 0, err
		}

		if err := tx.Model(&entity.ExtraFee{}).Where("package_id = ?", pkg.ID).Update("bill_id", billID).Error; err != nil {
			tx.Rollback()
			return 0, err
		}

		for _, extraFee := range pkg.ExtraFee {
			if extraFee.ID < 1 {
				if extraFee.ExtraFeeTypeID == constant.ExtraFeeTypePeak {
					if err := tx.Table("extra_fees").
						Where("package_id = ?", pkg.ID).
						Where("status = ?", constant.ExtraFeeStatusEnable).
						Where("extra_fee_type_id=?", constant.ExtraFeeTypePeak).
						UpdateColumn("status", constant.ExtraFeeStatusDisable).Error; err != nil {

						tx.Rollback()
						return 0, err
					}
				}

				extraFee.BillID = utils.Int64(billID)
				extraFee.PackageID = utils.Int64(pkg.ID)
				if err := tx.Create(&extraFee).Error; err != nil {
					tx.Rollback()
					return 0, err
				}
			}
		}
	}

	if err := tx.Model(&entity.PackageCode{}).Where("id IN (?)", cIDs).Update("status", constant.PackageCodeEnable).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	//use coupon
	var err error
	if refundCoupon != nil && failPkgLength == 0 {
		refundCoupon.BillID = &opts.BillID
		tx, err = m.ApplyCoupon(tx, user, refundCoupon)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	var shippingFeeOfBill *float64
	if err := tx.Model(&entity.Package{}).
		Where("bill_id = ?", opts.BillID).
		Select("SUM(shipping_fee) as total").
		Row().Scan(&shippingFeeOfBill); err != nil {
		tx.Rollback()
		return 0, err
	}

	var extraFeeOfBill *float64
	if err := tx.Model(&entity.ExtraFee{}).
		Where("bill_id = ? AND status != ?", opts.BillID, constant.ExtraFeeStatusDisable).
		Select("SUM(amount) as total").
		Row().Scan(&extraFeeOfBill); err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Exec("UPDATE bills SET extra_fee = ?, shipping_fee = ? WHERE id=?",
		utils.Float64Value(extraFeeOfBill), utils.Float64Value(shippingFeeOfBill), billID).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Exec("UPDATE users SET balance=balance-? WHERE id=?", opts.ShippingFee, opts.UserID).Error; err != nil {
		fmt.Errorf("Update user balance error %v", err)
		tx.Rollback()
		return 0, err
	}

	info := &entity.UserInfo{}
	if err := tx.First(info).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			tx.Rollback()
			return 0, err
		}

		now := time.Now()
		info = &entity.UserInfo{UserID: opts.UserID, UpdatedAt: &now, CancelMaxAmount: constant.DefaultCancelMaxAMount}
		if err := tx.Create(info).Error; err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	sqlString := `UPDATE user_infos SET debt_time = ? WHERE user_id = ? AND debt_time IS NULL AND (SELECT balance FROM users WHERE id = ? limit 1) < 0`
	if err := tx.Exec(sqlString, time.Now(), opts.UserID, opts.UserID).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	transaction := &entity.Transaction{
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID: opts.UserID,
		Type:   constant.TransactionLogTypePay,
		Status: constant.TransactionStatusSuccess,
		Amount: opts.ShippingFee,
		BillID: &billID,
	}

	err = tx.Create(&transaction).Error
	if err != nil {
		fmt.Errorf("Create transaction error %v", err)
		tx.Rollback()
		return 0, err
	}

	transactionLog := &entity.TransactionLog{
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID:        opts.UserID,
		TransactionID: transaction.ID,
		Amount:        transaction.Amount,
		Type:          transaction.Type,
		Status:        transaction.Status,
		BillID:        transaction.BillID,
	}

	if err = tx.Model(entity.TransactionLog{}).Create(transactionLog).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	return billID, tx.Commit().Error
}

func (m *BillManager) ApplyCoupon(tx *gorm.DB, user *entity.User, extraFee *entity.ExtraFee) (*gorm.DB, error) {

	var couponUser *entity.CouponUser
	if err := tx.Model(entity.CouponUser{}).Where("customer_id = ? AND coupon_id = ?", user.ID, extraFee.CouponID).Preload("Coupon").First(&couponUser).Error; err != nil {
		fmt.Errorf("Error find info coupon %v", err)
		return tx, err
	}

	if couponUser.Used >= couponUser.Quantity || couponUser.Coupon.EndDate.Before(time.Now()) {
		log.Println("Coupon expired")
		return tx, nil
	}

	if err := tx.Model(entity.ExtraFee{}).Create(extraFee).Error; err != nil {
		fmt.Errorf("Create extra error %v", err)
		return tx, err
	}

	transaction := &entity.Transaction{
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID: user.ID,
		Type:   constant.TransactionLogTypeRefund,
		Status: constant.TransactionStatusSuccess,
		Amount: extraFee.Amount,
		BillID: extraFee.BillID,
	}
	if err := tx.Create(&transaction).Error; err != nil {
		fmt.Errorf("Create transaction error %v", err)
		return tx, err
	}

	transactionLog := &entity.TransactionLog{
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID:        user.ID,
		TransactionID: transaction.ID,
		Amount:        transaction.Amount,
		Type:          transaction.Type,
		Status:        transaction.Status,
		BillID:        extraFee.BillID,
	}

	if err := tx.Model(entity.TransactionLog{}).Create(transactionLog).Error; err != nil {
		fmt.Errorf("Create transaction error %v", err)
		return tx, err
	}

	if err := tx.Exec("update users SET balance = balance + ? where users.id = ? ", math.Abs(transaction.Amount), user.ID).Error; err != nil {
		fmt.Errorf("Update user balance error %v", err)
		return tx, err
	}

	sqlString := `UPDATE user_infos SET debt_time = ? WHERE user_id = ? AND debt_time IS NULL AND (SELECT balance FROM users WHERE id = ? limit 1) < 0`
	if err := tx.Exec(sqlString, time.Now(), user.ID, user.ID).Error; err != nil {
		fmt.Errorf("Update user info error %v", err)
		return tx, err
	}

	if err := tx.Exec("update coupon_users SET used = used + 1 where coupon_users.id = ? ", couponUser.ID).Error; err != nil {
		fmt.Errorf("Update coupon error %v", err)
		return tx, err
	}

	return tx, nil
}

func (m *BillManager) CreateBill(opts CreateBillOption, user *entity.User, refundCoupon *entity.ExtraFee) (int64, error) {
	if opts.BillID == 0 {
		billID, err := m.GetOrCreateNowBillID(opts.UserID)
		if err != nil {
			fmt.Errorf("Create bill error %v", err)
			return 0, err
		}

		opts.BillID = billID
	}

	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			fmt.Errorf("recover: %v", r)
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	bill := &entity.Bill{}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=?", bill.ID).Find(bill).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	user1 := &entity.User{}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=?", user.ID).Find(user1).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	userinfo := &entity.UserInfo{}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id=?", user.ID).Find(userinfo).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	env := viper.GetString("env")

	extraFees := []entity.ExtraFee{}
	deliverLogs := []entity.PackageDeliverLog{}
	var disablePackageExtraFeeTypePeaks []int64
	var packageIDs []int64

	for _, pkg := range opts.Packages {
		packageIDs = append(packageIDs, pkg.ID)

		var cID int64
		// check temp package code
		if pkg.PackageCode != nil && pkg.PackageCode.Status == constant.PackageCodeTemp {
			pkg.PackageCode.Status = constant.PackageCodeEnable

			if err := tx.Save(pkg.PackageCode).Error; err != nil {
				tx.Rollback()
				return 0, err
			}

			cID = pkg.PackageCode.ID
		} else {

			// gen package code
			PackageIDGenCode := pkg.ID
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
					return 0, err
				}

				PackageIDGenCode = result.PackageIDGenerate
			}

			NumGen := fmt.Sprintf("%05d%02d%08d", opts.UserID, pkg.ServiceID, PackageIDGenCode)

			prefixCode := entity.PartnerPrefixCodeMap[pkg.PartnerID]
			Code := fmt.Sprintf("%v%05d%02d%08d%d", prefixCode, opts.UserID, pkg.ServiceID, PackageIDGenCode, m.GenerateLastDigitCode(NumGen))

			if env == "development" {
				Code = fmt.Sprintf("%s%s", Code, "DEV")
			}

			var count int64
			if err := tx.Model(&entity.PackageCode{}).Where("code=? AND (user_id = ? OR status=?)", Code, opts.UserID, constant.PackageCodeEnable).Count(&count).Error; err != nil {
				tx.Rollback()
				return 0, err
			}

			if count > 0 {
				tx.Rollback()
				return 0, errors.New("create package code duplicate or is active")
			}

			packageCode := &entity.PackageCode{
				UserID:            opts.UserID,
				Code:              Code,
				Status:            constant.PackageCodeEnable,
				ServiceID:         pkg.ServiceID,
				PackageIDGenerate: PackageIDGenCode,
			}

			if err := tx.Create(packageCode).Error; err != nil {
				tx.Rollback()
				return 0, err
			}

			if packageCode.ID < 1 {
				return 0, errors.New("create package code failed")
			}

			cID = packageCode.ID
		}

		mapPkg := make(map[string]interface{})
		mapPkg["updated_at"] = time.Now()
		mapPkg["status"] = constant.PackageStatusPendingPickup
		mapPkg["bill_id"] = opts.BillID
		mapPkg["package_code_id"] = cID

		err := tx.Model(&entity.Package{}).Where("id=?", pkg.ID).UpdateColumns(mapPkg).Error
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		deliverLog := entity.PackageDeliverLog{
			PackageID: pkg.ID,
			Status:    constant.DeliverLogTebexpressInTransit,
			Type:      constant.PackageDeliverLogTypePendingPickup,
		}
		deliverLogs = append(deliverLogs, deliverLog)

		for _, extraFee := range pkg.ExtraFee {
			if extraFee.ID < 1 {
				if extraFee.ExtraFeeTypeID == constant.ExtraFeeTypePeak {
					disablePackageExtraFeeTypePeaks = append(disablePackageExtraFeeTypePeaks, pkg.ID)
				}

				extraFee.BillID = utils.Int64(opts.BillID)
				extraFee.PackageID = utils.Int64(pkg.ID)
				extraFees = append(extraFees, extraFee)
			}
		}
	}

	if err := tx.Model(&entity.ExtraFee{}).Where("package_id IN (?)", packageIDs).Update("bill_id", opts.BillID).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if len(deliverLogs) > 0 {
		if err := tx.Create(&deliverLogs).Error; err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	if len(disablePackageExtraFeeTypePeaks) > 0 {
		if err := tx.Table("extra_fees").
			Where("package_id IN (?) AND status=? AND extra_fee_type_id=?", disablePackageExtraFeeTypePeaks, constant.ExtraFeeStatusEnable, constant.ExtraFeeTypePeak).
			UpdateColumn("status", constant.ExtraFeeStatusDisable).Error; err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	if len(extraFees) > 0 {
		if err := tx.Create(&extraFees).Error; err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	var err error
	if refundCoupon != nil {
		refundCoupon.BillID = &opts.BillID
		tx, err = m.ApplyCoupon(tx, user, refundCoupon)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	var shippingFeeOfBill *float64
	if err := tx.Model(&entity.Package{}).
		Where("bill_id = ?", opts.BillID).
		Select("SUM(shipping_fee) as total").
		Row().Scan(&shippingFeeOfBill); err != nil {
		tx.Rollback()
		return 0, err
	}

	var extraFeeOfBill *float64
	if err := tx.Model(&entity.ExtraFee{}).
		Where("bill_id = ? AND status != ?", opts.BillID, constant.ExtraFeeStatusDisable).
		Select("SUM(amount) as total").
		Row().Scan(&extraFeeOfBill); err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Exec("UPDATE bills SET extra_fee = ?, shipping_fee = ? WHERE id=?",
		utils.Float64Value(extraFeeOfBill), utils.Float64Value(shippingFeeOfBill), opts.BillID).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Exec("UPDATE users SET balance=balance-? WHERE id=?", opts.ShippingFee, opts.UserID).Error; err != nil {
		fmt.Errorf("Update user balance error %v", err)
		tx.Rollback()
		return 0, err
	}

	info := &entity.UserInfo{}
	if err := tx.First(info).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			tx.Rollback()
			return 0, err
		}

		now := time.Now()
		info = &entity.UserInfo{UserID: opts.UserID, UpdatedAt: &now, CancelMaxAmount: constant.DefaultCancelMaxAMount}
		if err := tx.Create(info).Error; err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	sqlString := `UPDATE user_infos SET debt_time = ? WHERE user_id = ? AND debt_time IS NULL AND (SELECT balance FROM users WHERE id = ? limit 1) < 0`
	if err := tx.Exec(sqlString, time.Now(), opts.UserID, opts.UserID).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	transaction := &entity.Transaction{
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID: opts.UserID,
		Type:   constant.TransactionLogTypePay,
		Status: constant.TransactionStatusSuccess,
		Amount: opts.ShippingFee,
		BillID: &opts.BillID,
	}

	err = tx.Create(&transaction).Error
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	transactionLog := &entity.TransactionLog{
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID:        opts.UserID,
		TransactionID: transaction.ID,
		Amount:        transaction.Amount,
		Type:          transaction.Type,
		Status:        transaction.Status,
		BillID:        transaction.BillID,
	}

	if err = tx.Model(entity.TransactionLog{}).Create(transactionLog).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	return opts.BillID, tx.Commit().Error
}

func (m BillManager) GenerateLastDigitCode(numGen string) int64 {
	return utils.GenerateLastDigitCode(numGen)
}

func (m *BillManager) GetTotalBillFee(BillID int64) (float64, error) {
	sql := `SELECT SUM(total) as total FROM (
			SELECT SUM(shipping_fee) as total FROM packages WHERE packages.bill_id = ? 
			UNION 
			SELECT SUM(amount) as total FROM extra_fees WHERE bill_id = ?
			AND status IN  (?)) a`
	var result = struct {
		Total float64 `json:"total"`
	}{}
	db := m.db.Raw(sql, BillID, BillID, []int64{constant.ExtraFeeStatusEnable, constant.ExtraFeeStatusUnPaid}).Scan(&result)
	return result.Total, db.Error
}
func (m *BillManager) GetTotalBillFeeUnPaid(BillID int64) (float64, error) {
	sql := `SELECT SUM(amount) as total FROM extra_fees WHERE package_id IN (SELECT id FROM packages WHERE packages.bill_id = ? )
			AND status = ?`
	var result = struct {
		Total float64 `json:"total"`
	}{}
	db := m.db.Raw(sql, BillID, constant.ExtraFeeStatusUnPaid).Scan(&result)
	return result.Total, db.Error
}

func (m *BillManager) Count(opts BillQueryOption) (int64, error) {
	db := m.BuildQueryBill(opts)

	var count int64
	db = db.Model(&entity.Bill{}).Count(&count)
	return count, db.Error
}
func (m *BillManager) GetExtraFeeTypeByID(id int64) (*entity.ExtraFeeType, error) {
	db := m.db
	db = db.Where("id = ? AND status = ?", id, constant.ExtraFeeStatusEnable)
	extraFeeType := &entity.ExtraFeeType{}
	db = db.First(extraFeeType)
	return extraFeeType, db.Error
}
func (m *BillManager) Fetch(opts BillQueryOption) ([]entity.Bill, error) {
	var bills []entity.Bill
	db := m.BuildQueryBill(opts).Order("id DESC")

	if !opts.HidePreloadUser {
		db = db.Preload("User").Preload("User.UserInfo")
	}

	if !opts.HidePreloadPackage {
		db = db.Preload("Package", func(db *gorm.DB) *gorm.DB {
			db = db.Order("id DESC")
			return db
		})
	}

	if opts.ServiceCode != "" {
		db = db.Joins("JOIN packages ON packages.bill_id = bills.id").
			Joins("JOIN services ON services.id = packages.service_id").
			Where("services.code = ?", opts.ServiceCode)
	}

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}
	db = db.Find(&bills)

	return bills, db.Error
}

func (m *BillManager) GetOrCreateNowBill(userID int64) (*entity.Bill, error) {
	date := time.Now().Add(7 * time.Hour).Format("2006-01-02")
	db := m.db.Where("user_id = ?", userID)
	db = db.Where("DATE_FORMAT(convert_tz(created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d %H:%i:%s') BETWEEN ?  AND ?", fmt.Sprintf("%s 00:00:00", date), fmt.Sprintf("%s 23:59:59", date))

	bill := &entity.Bill{}
	err := db.First(bill).Error

	if err == gorm.ErrRecordNotFound {
		extras := userID / 1000
		prefix := int64(constant.BillCodePrefix)
		codeUID := userID

		if extras > 0 {
			prefix += extras
			codeUID = codeUID - extras*1000
		}

		t := time.Now().Add(time.Hour * 7)

		bill = &entity.Bill{
			Model: dbgorm.Model{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Code:   fmt.Sprintf("%d%03d%d%02d%02d", prefix, codeUID, t.Year(), t.Month(), t.Day()),
			UserID: userID,
			Status: constant.BillingStatusAwaitingPayment,
		}

		if err := m.db.Create(bill).Error; err != nil {
			return nil, err
		}

		return bill, nil
	}

	return bill, err
}

func (m *BillManager) GetOrCreateNowBillID(userID int64) (int64, error) {
	bill, err := m.GetOrCreateNowBill(userID)

	if err != nil {
		return 0, err
	}

	return bill.ID, err
}

func (m *BillManager) CreateExtraFee(extraFee *entity.ExtraFee, userID, adminID int64) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			return
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Save(extraFee).Error; err != nil {
		tx.Rollback()
		return err
	}

	var operatorBalance, operatorBill string
	status := constant.TransactionStatusSuccess
	transactionType := constant.TransactionLogTypeRefund
	if extraFee.Amount > 0 {
		transactionType = constant.TransactionLogTypePay
		operatorBill = `+`
		operatorBalance = `-`
	} else {
		operatorBill = `-`
		operatorBalance = `+`
	}

	sqlString := `UPDATE bills SET extra_fee = extra_fee ` + operatorBill + ` ?, updated_at = ? WHERE id = ?`

	if err := tx.Exec(sqlString, math.Abs(extraFee.Amount), time.Now(), extraFee.BillID).Error; err != nil {
		tx.Rollback()
		return err
	}

	sqlString = `UPDATE users SET balance = balance ` + operatorBalance + ` ?, updated_at = ? WHERE id = ?`
	if err := tx.Exec(sqlString, math.Abs(extraFee.Amount), time.Now(), userID).Error; err != nil {
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
		info = &entity.UserInfo{UserID: userID, UpdatedAt: &now, CancelMaxAmount: constant.DefaultCancelMaxAMount}
		if err := tx.Create(info).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if extraFee.Amount > 0 {
		sqlString = `UPDATE user_infos SET debt_time = ? WHERE user_id = ? AND debt_time IS NULL AND (SELECT balance FROM users WHERE id = ? limit 1) < 0`
		if err := tx.Exec(sqlString, time.Now(), userID, userID).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if extraFee.Amount < 0 {
		sqlString = `UPDATE user_infos SET debt_time = NULL WHERE user_id = ? AND debt_time IS NOT NULL AND (SELECT balance FROM users WHERE id = ? limit 1) >= 0`
		if err := tx.Exec(sqlString, userID, userID).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	auditLogType := constant.MapTypeAuditLogByExtraFeeTypeID[extraFee.ExtraFeeTypeID]

	if auditLogType > 0 {

		logs := &entity.PackageAuditLog{
			PackageID:     utils.Int64Value(extraFee.PackageID),
			Type:          auditLogType,
			UpdatedUserID: adminID,
			Fee:           extraFee.Amount,
			Value:         extraFee.Description,
			Description:   extraFee.Description,
		}

		if err := tx.Model(&entity.PackageAuditLog{}).Create(&logs).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	transaction := &entity.Transaction{
		UserID: userID,
		Type:   cast.ToInt64(transactionType),
		Status: cast.ToInt64(status),
		Amount: math.Abs(extraFee.Amount),
		BillID: extraFee.BillID,
	}

	err := tx.Create(&transaction).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	transactionLog := &entity.TransactionLog{
		UserID:        userID,
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
func (m *BillManager) GetAllExtraFeeTypes() ([]entity.ExtraFeeType, error) {
	db := m.db
	db = db.Where("status = ?", constant.ExtraFeeStatusEnable)
	db = db.Where("is_show = ?", constant.ExtraFeeShow)
	types := []entity.ExtraFeeType{}
	db = db.Find(&types)
	return types, db.Error
}

func (m *BillManager) GetBillExport(result interface{}, billID int64) error {
	db := m.db
	sqlString := `
		SELECT * FROM
		(
		SELECT 
		bills.code as bill_code,
		packages.status,
		packages.order_number,
		package_codes.code AS package_code,
		users.full_name,
		DATE_FORMAT(convert_tz(packages.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d %H:%i:%s') AS created_at,
		trackings.tracking_number,
		services.name AS service_name,
		(CASE
			WHEN packages.address_1 IS NULL THEN packages.address_2
			ELSE packages.address_1
		END) AS address,
		packages.city,
		packages.state_code,
		packages.zipcode,
		packages.country_code,
		packages.detail,
		packages.shipping_fee,
		0 AS extra_fee,
		packages.customer_shipment_id AS customer_shipment_id,
		packages.shipping_fee + COALESCE(e.extra_fee_total, 0) AS total_fee,
		'' AS extra_fee_type,
		'' AS extra_fee_created_at,
		'' AS description,
		packages.weight,
		packages.length,
		packages.width,
		packages.height,
		trackings.weight AS weight_real,
		trackings.length AS length_real,
		trackings.width AS width_real,
		trackings.height AS height_real,
		carriers.name AS carrier_name,
		trackings.carrier_service,
		warehouses.name AS warehouse,
		trackings.shipment_cost,
		trackings.handling_fee,
		users.username
	FROM
		packages
			LEFT JOIN
		users ON users.id = packages.user_id
			LEFT JOIN
		package_codes ON package_codes.id = packages.package_code_id  AND package_codes.status = ?
			LEFT JOIN
		bills ON bills.id = packages.bill_id
			LEFT JOIN
		trackings ON trackings.package_id = packages.id AND trackings.status = ?
			LEFT JOIN
		services ON services.id = packages.service_id
			LEFT JOIN
		extra_fees ON extra_fees.package_id = packages.id AND extra_fees.status = ? AND extra_fees.bill_id = packages.bill_id
			LEFT JOIN
		extra_fee_types ON extra_fee_types.id = extra_fees.extra_fee_type_id
			LEFT JOIN
		(SELECT 
			package_id,
				SUM(COALESCE(extra_fees.amount, 0)) AS extra_fee_total
		FROM
			extra_fees
		WHERE
			extra_fees.status = ?
		GROUP BY package_id) e ON e.package_id = packages.id
			LEFT JOIN
		carriers ON carriers.id = trackings.carrier_id
			LEFT JOIN
		warehouses ON warehouses.id = trackings.hub_id
	WHERE
		packages.bill_id = ?
	`

	sqlString += `
		UNION SELECT 
		bills.code as bill_code,
		packages.status,
		packages.order_number,
		package_codes.code AS package_code,
		users.full_name,
		DATE_FORMAT(CONVERT_TZ(packages.created_at,
						@@SESSION .time_zone,
						'+07:00'),
				'%Y-%m-%d %H:%i:%s') AS created_at,
		trackings.tracking_number,
		services.name AS service_name,
		(CASE
			WHEN packages.address_1 IS NULL THEN packages.address_2
			ELSE packages.address_1
		END) AS address,
		packages.city,
		packages.state_code,
		packages.zipcode,
		packages.country_code,
		packages.detail,
		0 AS shipping_fee,
		extra_fees.amount AS extra_fee,
		extra_fees.customer_shipment_id,
		packages.shipping_fee + COALESCE(e.extra_fee_total, 0) AS total_fee,
		extra_fee_types.name AS extra_fee_type,
		DATE_FORMAT(CONVERT_TZ(extra_fees.created_at,
						@@SESSION .time_zone,
						'+07:00'),
				'%Y-%m-%d %H:%i:%s') AS extra_fee_created_at,
		extra_fees.description,
		packages.weight,
		packages.length,
		packages.width,
		packages.height,
		trackings.weight AS weight_real,
		trackings.length AS length_real,
		trackings.width AS width_real,
		trackings.height AS height_real,
		carriers.name AS carrier_name,
		trackings.carrier_service,
		warehouses.name AS warehouse,
		trackings.shipment_cost,
		trackings.handling_fee,
		users.username
	FROM
		extra_fees
			LEFT JOIN
		bills ON extra_fees.bill_id = bills.id
			LEFT JOIN
		packages ON extra_fees.package_id = packages.id
			LEFT JOIN
		users ON users.id = packages.user_id
			LEFT JOIN
		package_codes ON package_codes.id = packages.package_code_id
			AND package_codes.status = 1
			LEFT JOIN
		trackings ON trackings.package_id = packages.id AND trackings.status = ?
			LEFT JOIN
		services ON services.id = packages.service_id
			LEFT JOIN
		extra_fee_types ON extra_fee_types.id = extra_fees.extra_fee_type_id
			LEFT JOIN
		(SELECT 
			package_id,
				SUM(COALESCE(extra_fees.amount, 0)) AS extra_fee_total
		FROM
			extra_fees
		WHERE
			extra_fees.status = ?
		GROUP BY package_id) e ON e.package_id = packages.id
			LEFT JOIN
		carriers ON carriers.id = trackings.carrier_id
			LEFT JOIN
		warehouses ON warehouses.id = trackings.hub_id
	WHERE
	extra_fees.bill_id = ? AND extra_fees.status = ? 
	`

	sqlString += `) A ORDER BY package_code , created_at`

	db = db.Raw(sqlString, constant.PackageCodeStatusActive, constant.TrackingStatusSuccess, constant.ExtraFeeStatusEnable, constant.ExtraFeeStatusEnable, billID, constant.TransactionStatusSuccess, constant.ExtraFeeStatusEnable, billID, constant.ExtraFeeStatusEnable).Scan(result)
	return db.Error
}

func (m *BillManager) buildQueryBillPackages(opts BillPackageQueryOption) *gorm.DB {
	db := m.db

	if opts.UserID > 0 {
		db = db.Where("user_id", opts.UserID)
	}

	if opts.BillID > 0 {
		db = db.Where("bill_id=?", opts.BillID)
	} else {
		sdb := m.db.Model(&entity.Bill{}).Select("id").Where("code=?", opts.BillCode).Limit(1)
		db = db.Where("bill_id=(?)", sdb)
	}

	return db
}

func (m *BillManager) GetBillPackages(opts BillPackageQueryOption) ([]entity.Package, error) {
	db := m.buildQueryBillPackages(opts)

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	if opts.PerloadExtraFee {
		db = db.Preload("ExtraFee")
	}

	if opts.PerloadTracking {
		db = db.Preload("Tracking", func(db *gorm.DB) *gorm.DB {
			db = db.Where("status!=?", constant.TrackingStatusCanceled)
			return db
		})
	}

	if opts.Select != "" {
		db = db.Select(opts.Select)
	}

	db = db.Preload("PackageCode")
	db = db.Order("id DESC")

	var packages []entity.Package
	db = db.Find(&packages)

	return packages, db.Error
}

func (m *BillManager) CountBillItemPackages(opts BillPackageQueryOption) (int64, error) {
	db := m.buildQueryBillPackages(opts)

	var count int64
	db = db.Model(&entity.Package{}).Count(&count)
	return count, db.Error
}

func (m *BillManager) PackageRefund(customerID, refundID, packageID, billID int64, amount float64) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			return
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	bill := &entity.Bill{}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=?", billID).Find(bill).Error; err != nil {
		tx.Rollback()
		return err
	}

	user1 := &entity.User{}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=?", customerID).Find(user1).Error; err != nil {
		tx.Rollback()
		return err
	}

	userinfo := &entity.UserInfo{}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id=?", customerID).Find(userinfo).Error; err != nil {
		tx.Rollback()
		return err
	}

	extraFeeType := &entity.ExtraFeeType{}
	if err := tx.Where("id=?", constant.ExtraFeeTypeRefund).Find(extraFeeType).Error; err != nil {
		return err
	}

	extraFee := &entity.ExtraFee{
		PackageID:      utils.Int64(packageID),
		BillID:         utils.Int64(billID),
		Amount:         -amount,
		ExtraFeeTypeID: constant.ExtraFeeTypeRefund,
		Description:    extraFeeType.Name,
		Status:         constant.ExtraFeeStatusEnable,
	}

	if err := tx.Save(extraFee).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&entity.PackageRefund{}).
		Where("id=?", refundID).
		Updates(map[string]interface{}{
			"status":     constant.PackageRefundCompleted,
			"updated_at": time.Now(),
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	sql := `UPDATE bills SET extra_fee = extra_fee - ?, updated_at = ? WHERE id = ?`
	if err := tx.Exec(sql, amount, time.Now(), billID).Error; err != nil {
		tx.Rollback()
		return err
	}

	sql = `UPDATE users SET balance = balance + ?, updated_at = ? WHERE id = ?`
	if err := tx.Exec(sql, amount, time.Now(), customerID).Error; err != nil {
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
		info = &entity.UserInfo{UserID: customerID, UpdatedAt: &now, CancelMaxAmount: constant.DefaultCancelMaxAMount}
		if err := tx.Create(info).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	sql = `UPDATE user_infos SET debt_time = NULL WHERE user_id = ? AND debt_time IS NOT NULL AND (SELECT balance FROM users WHERE id = ? limit 1) >= 0`
	if err := tx.Exec(sql, customerID, customerID).Error; err != nil {
		tx.Rollback()
		return err
	}

	transaction := &entity.Transaction{
		UserID: customerID,
		Type:   constant.TransactionLogTypeRefund,
		Status: constant.TransactionStatusSuccess,
		Amount: amount,
		BillID: extraFee.BillID,
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return err
	}

	transactionLog := &entity.TransactionLog{
		UserID:        customerID,
		TransactionID: transaction.ID,
		Amount:        transaction.Amount,
		Type:          transaction.Type,
		Status:        transaction.Status,
		BillID:        transaction.BillID,
	}

	if err := tx.Model(entity.TransactionLog{}).Create(transactionLog).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m *BillManager) PackageRefundCancel(refundID int64) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			return
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Model(&entity.PackageRefund{}).
		Where("id=?", refundID).
		Updates(map[string]interface{}{
			"status":     constant.PackageRefundCanceled,
			"updated_at": time.Now(),
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m BillManager) GetTotalBillByUser(userIDs []int64, searchDate string, result interface{}) error {
	db := m.db.Model(entity.Bill{})
	db = db.Where("bills.user_id IN (?)", userIDs)
	if searchDate != "" {
		db = db.Where("DATE_FORMAT(convert_tz(bills.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') = DATE(?)", searchDate)
	}
	db = db.Select("sum(bills.shipping_fee + bills.extra_fee) as revenue,user_id")
	db = db.Group("user_id")
	db = db.Scan(result)
	return db.Error
}
func (m BillManager) AddPoint(opts CreateBillOption) error {
	tx := m.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			fmt.Errorf("panic %v", r)
			tx.Rollback()
			return
		}
	}()
	if err := tx.Exec("UPDATE users SET point = point  + ? WHERE id=?", opts.Point, opts.UserID).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(entity.PointLog{}).Create(&entity.PointLog{
		Point:       opts.Point,
		UserID:      opts.UserID,
		Description: fmt.Sprintf("Cộng điểm từ ngày %s tới ngày %s", time.Now().Add(-constant.InrangPlusPointDays*time.Hour).Format("02/01/2006"), time.Now().Format("02/01/2006")),
		Status:      constant.StatusActive,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
