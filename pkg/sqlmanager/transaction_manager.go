package sqlmanager

import (
	"fmt"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/dbgorm"
	"time"

	"gorm.io/gorm"
)

type TransactionManager struct {
	DB *gorm.DB
}

type TransactionQueryParams struct {
	ID             int64
	Type           int64
	Limit          int
	Offset         int
	StartDate      string
	EndDate        string
	UserID         int64
	Status         int
	BillID         string
	AccountName    string
	IsPreloadUser  bool
	IsPreloadAdmin bool
	Search         string
	SearchBy       string
	SupportID      int64
	IgnoreUsers    []int64
	PartnerID      int64
}

func NewTransactionManager(db *gorm.DB) *TransactionManager {
	return &TransactionManager{DB: db}
}

func NewTransactionLogManager(db *gorm.DB) *TransactionManager {
	return &TransactionManager{DB: db}
}

func (m *TransactionManager) buildQueryTransaction(params TransactionQueryParams) *gorm.DB {
	db := m.DB

	if params.ID > 0 {
		db = db.Where("id=?", params.ID)
	}

	if params.PartnerID > 0 {
		db = db.Joins("JOIN users partneruser ON partneruser.id = transactions.user_id")
		db = db.Where("partneruser.partner_id = ?", params.PartnerID)
	}

	if params.Type > 0 {
		if params.Type == constant.TransactionLogTypeTopup {
			db = db.Where("type IN (?)", []int64{constant.TransactionLogTypeTopup, constant.TransactionLogTypePayoneer, constant.TransactionLogTypePingPong})
		} else if params.Type == constant.TransactionLogTypeRefund {
			db = db.Where("type IN (?)", []int64{constant.TransactionLogTypeRefund, constant.TransactionLogTypeAffliate})
		} else {
			db = db.Where("type=?", params.Type)
		}
	}

	if len(params.StartDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= ?", params.StartDate)
	}

	if len(params.EndDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= ?", params.EndDate)
	}

	if params.SupportID > 0 {
		db = db.Joins("JOIN users aliasuser ON aliasuser.id = transactions.user_id")
		db = db.Joins("JOIN user_permissions  ON aliasuser.id=user_permissions.customer_id")
		db = db.Where("user_permissions.support_id = ?", params.SupportID)
	}
	if params.UserID > 0 {
		db = db.Where("user_id = ?", params.UserID)
	}

	if len(params.IgnoreUsers) > 0 {
		db = db.Where("user_id NOT IN (?)", params.IgnoreUsers)
	}

	if params.Status > 0 {
		db = db.Where("transactions.status = ?", params.Status)
	}

	if len(params.BillID) > 0 {
		db = db.Where("transactions.bill_id = ?", params.BillID)
	}

	if len(params.AccountName) > 0 {
		db = db.Where("user_id IN (?)", m.DB.Model(&entity.User{}).Where("username = ? OR email = ? OR phone_number = ?", params.AccountName, params.AccountName, params.AccountName).Select("id"))
	}

	if params.Limit > 0 {
		db = db.Limit(params.Limit)
	}

	if params.Offset > 0 {
		db = db.Offset(params.Offset)
	}

	if params.Search != "" {
		switch params.SearchBy {
		case "account":
			db = db.Where("user_id IN (?)", m.DB.Model(&entity.User{}).Where("username = ? OR email = ? OR phone_number = ?", params.Search, params.Search, params.Search).Select("id"))
			break
		case "bill_id":
			db = db.Where("bill_id = ?", params.Search)
			break
		case "bill_code":
			db = db.Where("bill_id IN (?)", m.DB.Model(&entity.Bill{}).Where("code = ?", params.Search).Select("id"))
			break
		case "account_full_name":
			db = db.Where("user_id IN (?)", m.DB.Model(&entity.User{}).Where("full_name LIKE ?", fmt.Sprintf("%%%s%%", params.Search)).Select("id"))
			break
		case "customer":
			db = db.Where("user_id = (?)", m.DB.Model(&entity.User{}).Where("email = ? OR phone_number = ?", params.Search, params.Search).Select("id"))
			break
		default:
			db = db.Where("1=2")
			break
		}
	}

	db = db.Where("transactions.status != ?", constant.TransactionStatusDraft)
	db = db.Where("transactions.status != ?", constant.TransactionStatusRefund)

	if params.IsPreloadUser {
		db = db.Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id,username,full_name,email,balance,class")
		}).Preload("User.UserInfo")

	}
	if params.IsPreloadAdmin {
		db = db.Preload("Admin", func(db *gorm.DB) *gorm.DB {
			return db.Select("id,full_name")
		})
	}
	return db
}

func (m *TransactionManager) GetTransactions(params TransactionQueryParams, transactions interface{}) error {
	db := m.buildQueryTransaction(params)
	db = db.Preload("Bill")
	db = db.Order("id DESC")

	return db.Find(transactions).Error
}

func (m *TransactionManager) GetTransactionLog(params TransactionQueryParams) (entity.TransactionLog, error) {
	db := m.buildQueryTransaction(params)
	db = db.Order("id DESC")
	transactionLog := entity.TransactionLog{}
	db = db.First(&transactionLog)
	return transactionLog, db.Error
}

func (m *TransactionManager) SaveTransaction(transaction *entity.Transaction) error {
	tx := m.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			fmt.Errorf("Panic save transaction: %v", r)
			tx.Rollback()
			return
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Save(transaction).Error; err != nil {
		tx.Rollback()
		return err
	}

	transactionLog := &entity.TransactionLog{
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID:        transaction.UserID,
		AdminID:       transaction.AdminID,
		TransactionID: transaction.ID,
		Amount:        transaction.Amount,
		Type:          transaction.Type,
		Status:        transaction.Status,
	}

	if err := tx.Model(entity.TransactionLog{}).Create(transactionLog).Error; err != nil {
		tx.Rollback()
		return err
	}
	if (transaction.Type == constant.TransactionLogTypeTopup || transaction.Type == constant.TransactionLogTypePayoneer || transaction.Type == constant.TransactionLogTypePingPong || transaction.Type == constant.TransactionLogTypeAffliate) &&
		transaction.Status == constant.TransactionStatusSuccess {
		sqlString := "UPDATE users SET balance = balance + ? WHERE id = ?"
		if err := tx.Exec(sqlString, transaction.Amount, transaction.UserID).Error; err != nil {
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
			info = &entity.UserInfo{UserID: transaction.UserID, UpdatedAt: &now, CancelMaxAmount: constant.DefaultCancelMaxAMount}
			if err := tx.Create(info).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		sqlString = "UPDATE user_infos SET debt_time = NULL WHERE user_id = ? AND (SELECT balance FROM users WHERE id = ? limit 1) >= 0"
		if err := tx.Exec(sqlString, transaction.UserID, transaction.UserID).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m *TransactionManager) CountTransactions(params TransactionQueryParams) (int64, error) {
	var count int64
	db := m.buildQueryTransaction(params)
	db = db.Model(entity.Transaction{}).Select("COUNT(transactions.id) as count").Count(&count)
	return count, db.Error
}

func (m *TransactionManager) CountTransactionLogs(params TransactionQueryParams) (int64, error) {
	var count int64

	db := m.buildQueryTransaction(params)
	db = db.Model(entity.TransactionLog{}).Select("COUNT(id) as count").Count(&count)

	return count, db.Error
}

func (m *TransactionManager) CountAllStatusTransactions(params TransactionQueryParams, countAllStatus interface{}) error {
	params.Status = 0
	db := m.buildQueryTransaction(params)
	db = db.Model(entity.Transaction{}).Select("COUNT( transactions.status) as count,transactions.status").Group("status").Scan(countAllStatus)

	return db.Error
}

func (m *TransactionManager) GetProcessMoney(userID int64) (float64, error) {
	db := m.DB
	db = db.Table("transactions")
	db = db.Where("user_id = ? AND status = ?",
		userID, constant.TransactionStatusProcess)
	db = db.Select("IFNULL(SUM(amount), 0)")
	var sum float64
	err := db.Row().Scan(&sum)
	return sum, err
}

func (m *TransactionManager) UpdateTransaction(transaction *entity.Transaction) error {
	tx := m.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			return
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	transactionMapString := map[string]interface{}{
		"UpdatedAt": time.Now(),
		"status":    transaction.Status,
	}

	if transaction.Amount > 0 {
		transactionMapString["Amount"] = transaction.Amount
	}

	if err := tx.Model(&entity.Transaction{}).Where("id = ?", transaction.ID).UpdateColumns(transactionMapString).Error; err != nil {
		tx.Rollback()
		return err
	}

	transactionLog := &entity.TransactionLog{
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID:        transaction.UserID,
		TransactionID: transaction.ID,
		Amount:        transaction.Amount,
		Type:          transaction.Type,
		Status:        transaction.Status,
	}

	if err := tx.Model(entity.TransactionLog{}).Create(transactionLog).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m *TransactionManager) CreateTopup(transaction *entity.Transaction) (*entity.Transaction, error) {
	db := m.DB.Model(&entity.Transaction{}).Create(transaction)

	return transaction, db.Error
}

func (m *TransactionManager) GetTransactionByID(transactionID int64) (*entity.Transaction, error) {
	transaction := &entity.Transaction{}
	db := m.DB.Where("id=?", transactionID).First(transaction)
	return transaction, db.Error
}

func (m *TransactionManager) GetLastTransactionByUserID(userID int64, status []int) (*entity.Transaction, error) {
	transaction := &entity.Transaction{}
	db := m.DB.Where("user_id=?", userID)

	if len(status) > 0 {
		db = db.Where("status IN (?)", status)
	}

	db = db.Order("id DESC")
	db = db.First(transaction)
	return transaction, db.Error
}

func (m *TransactionManager) GetTransactionAffiliate(userIDs []int64, searchDate string) (error, []entity.Transaction) {
	db := m.DB.Model(entity.Transaction{})
	db = db.Where("admin_id IN (?) AND type = ? AND status = ?", userIDs, constant.TransactionLogTypeAffliate, constant.TransactionStatusSuccess)
	if searchDate != "" {
		db = db.Where("DATE_FORMAT(convert_tz(transactions.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') = DATE(?)", searchDate)
	}
	var transactions []entity.Transaction
	db = db.Find(&transactions)
	return db.Error, transactions
}

func (m *TransactionManager) SaveAffiliateTransaction(transaction *entity.Transaction, extraFee entity.ExtraFee) error {
	tx := m.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			fmt.Errorf("Panic save transaction: %v", r)
			tx.Rollback()
			return
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Save(transaction).Error; err != nil {
		tx.Rollback()
		return err
	}

	transactionLog := &entity.TransactionLog{
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID:        transaction.UserID,
		AdminID:       transaction.AdminID,
		TransactionID: transaction.ID,
		Amount:        transaction.Amount,
		Type:          transaction.Type,
		Status:        transaction.Status,
	}

	if err := tx.Model(entity.TransactionLog{}).Create(transactionLog).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Create(&extraFee).Error; err != nil {
		tx.Rollback()
		return err
	}

	var shippingFeeOfBill *float64
	if err := tx.Model(&entity.Package{}).
		Where("bill_id = ?", extraFee.BillID).
		Select("SUM(shipping_fee) as total").
		Row().Scan(&shippingFeeOfBill); err != nil {
		tx.Rollback()
		return err
	}

	var extraFeeOfBill *float64
	if err := tx.Model(&entity.ExtraFee{}).
		Where("bill_id = ? AND status != ?", extraFee.BillID, constant.ExtraFeeStatusDisable).
		Select("SUM(amount) as total").
		Row().Scan(&extraFeeOfBill); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Exec("UPDATE bills SET extra_fee = ?, shipping_fee = ? WHERE id=?",
		utils.Float64Value(extraFeeOfBill), utils.Float64Value(shippingFeeOfBill), extraFee.BillID).Error; err != nil {
		tx.Rollback()
		return err
	}

	sqlString := "UPDATE users SET balance = balance + ? WHERE id = ?"
	if err := tx.Exec(sqlString, transaction.Amount, transaction.UserID).Error; err != nil {
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
		info = &entity.UserInfo{UserID: transaction.UserID, UpdatedAt: &now, CancelMaxAmount: constant.DefaultCancelMaxAMount}
		if err := tx.Create(info).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	sqlString = "UPDATE user_infos SET debt_time = NULL WHERE user_id = ? AND (SELECT balance FROM users WHERE id = ? limit 1) >= 0"
	if err := tx.Exec(sqlString, transaction.UserID, transaction.UserID).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
