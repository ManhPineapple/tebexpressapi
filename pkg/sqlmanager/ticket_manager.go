package sqlmanager

import (
	"fmt"
	"log"
	"math"
	"strings"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/dbgorm"

	"time"

	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type TicketManager struct {
	DB *gorm.DB
}

type TicketQueryParams struct {
	ID          int64
	PackageID   int64
	SupportID   int64
	Search      string
	Category    string
	Status      int64
	InStatus    []string
	Sort        string
	Limit       int
	Offset      int
	StartDate   string
	EndDate     string
	Role        string
	UserID      int64
	IgnoreUsers []int64
	HasSupport  bool
	HasUser     bool
	SearchBy    string
	Type        int
	IsRefund    bool
	HasPackage  bool
}

type TicketMessageQueryParams struct {
	UserID   int64
	TicketID int64
	Keyword  string
	Status   int
	Sort     string
	Limit    int
	Offset   int
	LastID   int64
	FirstID  int64
}

type CountTicketsByStatus struct {
	Count  int64 `json:"count"`
	Status int64 `json:"status" `
}

func NewTicketManager(db *gorm.DB) *TicketManager {
	return &TicketManager{DB: db}
}

func (m *TicketManager) buildQueryTickets(params TicketQueryParams) *gorm.DB {
	db := m.DB

	if params.ID > 0 {
		db = db.Where("tickets.id=?", params.ID)
	}

	if params.PackageID > 0 {
		db = db.Where("tickets.object_id=?", params.PackageID)
	}

	if params.Category != "" {
		db = db.Where("tickets.category=?", params.Category)
	}

	if params.Status > 0 {
		db = db.Where("tickets.status=?", params.Status)
	} else if params.Role == constant.UserRoleCustomer && params.Status <= 0 {
		db = db.Where("tickets.status IN (?)", []int64{constant.TicketStatusPending, constant.TicketStatusProcessed, constant.TicketStatusApplying})
	}

	if len(params.InStatus) > 0 {
		db = db.Where("tickets.status IN (?)", params.InStatus)
	}

	if len(params.StartDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(tickets.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= ?", params.StartDate)
	}

	if len(params.EndDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(tickets.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= ?", params.EndDate)
	}

	if params.SupportID > 0 {
		db = db.Joins("JOIN users aliasuser ON aliasuser.id = tickets.user_id")
		db = db.Joins("JOIN user_permissions  ON aliasuser.id=user_permissions.customer_id")
		db = db.Where("user_permissions.support_id = ?", params.SupportID)
	}
	if params.UserID > 0 {
		db = db.Where("tickets.user_id = ?", params.UserID)
	}

	if len(params.IgnoreUsers) > 0 {
		db = db.Where("tickets.user_id NOT IN (?)", params.IgnoreUsers)
	}

	if params.Search != "" {
		switch params.SearchBy {
		case "code":
			db = db.Joins("LEFT JOIN packages ON packages.id = tickets.object_id")
			db = db.Joins("JOIN package_codes on package_codes.id = packages.package_code_id")
			db = db.Joins("LEFT JOIN trackings ON trackings.package_id = packages.id AND trackings.status = ?", constant.TransactionStatusSuccess)
			db = db.Where("package_codes.code = ? OR trackings.tracking_number = ?", params.Search, params.Search)
			break
		case "recipient":
			roles := []string{constant.UserRoleAdmin, constant.UserRoleSupport, constant.UserRoleSupportLeader, constant.UserRoleAccountant, constant.UserRoleSale}
			subquery := m.DB.Model(&entity.TicketMessage{}).Select("ticket_id").Joins("LEFT JOIN users ON users.id = ticket_messages.user_id").Where("users.full_name LIKE ? AND users.role IN (?)", fmt.Sprintf("%%%s%%", params.Search), roles).Group("ticket_id")
			db = db.Where("tickets.id IN (?)", subquery)

			break
		case "customer_code":
			if strings.Contains(params.Search, "U") {
				params.Search = strings.ReplaceAll(params.Search, "U", "")
			}
			db = db.Joins("LEFT JOIN users ON users.id = tickets.user_id")
			db = db.Where("users.id = ?", params.Search)
			break
		case "id":
			db = db.Where("tickets.id=?", params.Search)
			break

		default:
			db = db.Joins("LEFT JOIN packages ON packages.id = tickets.object_id")
			db = db.Joins("JOIN package_codes on package_codes.id = packages.package_code_id")
			db = db.Joins("LEFT JOIN trackings ON trackings.package_id = packages.id AND trackings.status = ?", constant.TransactionStatusSuccess)
			db = db.Where("package_codes.code = ? OR trackings.tracking_number = ?", params.Search, params.Search)
			break
		}
	}

	if params.Type > 0 {
		db = db.Where("tickets.type=?", params.Type)
	}

	if params.IsRefund {
		db = db.Where("tickets.type = ? OR (tickets.type = ? AND amount < 0)", constant.TicketTypeRefund, constant.TicketTypeReship)
	}

	if params.Limit > 0 {
		db = db.Limit(params.Limit)
	}

	if params.Offset > 0 {
		db = db.Offset(params.Offset)
	}

	return db
}

func (m *TicketManager) GetTickets(params TicketQueryParams, tickets interface{}) error {
	db := m.buildQueryTickets(params)
	db = db.Order(params.Sort)
	db = db.Preload("Package").Preload("Admin").Preload("Package.PackageCode")

	if params.HasSupport {
		db = db.Preload("Supports", func(db *gorm.DB) *gorm.DB {
			roles := []string{constant.UserRoleAdmin, constant.UserRoleSupport, constant.UserRoleSupportLeader, constant.UserRoleAccountant, constant.UserRoleSale}
			db = db.Table("ticket_messages")
			db = db.Joins("INNER JOIN users ON ticket_messages.user_id=users.id")
			db = db.Where("users.role IN (?)", roles)
			db = db.Select("users.id, ticket_messages.ticket_id, users.username, users.full_name, users.role")
			db = db.Group("users.id, ticket_messages.ticket_id, users.username, users.full_name, users.role")
			return db
		})
	}

	if params.HasUser {
		db = db.Preload("User")
		db = db.Preload("Accountant")
	}

	return db.Find(tickets).Error
}

func (m *TicketManager) CountTickets(params TicketQueryParams) (int64, error) {
	var count int64

	db := m.buildQueryTickets(params)
	db = db.Model(entity.Ticket{}).Select("COUNT(tickets.id) as count").Count(&count)

	return count, db.Error

}

func (m *TicketManager) GetTicketsByPackageID(params TicketQueryParams, tickets interface{}) error {
	db := m.buildQueryTickets(params)
	if params.PackageID > 0 {
		db = db.Where("object_id=?", params.PackageID)
	}
	if len(params.Sort) > 0 {
		db = db.Order(params.Sort)
	}
	db = db.Preload("Package")
	db = db.Joins("LEFT JOIN packages ON packages.id = tickets.object_id")
	return db.Find(tickets).Error
}

func (m *TicketManager) CountTicketsByPackageID(params TicketQueryParams) (int64, error) {
	var count int64

	db := m.buildQueryTickets(params)
	if params.PackageID > 0 {
		db = db.Where("object_id=?", params.PackageID)
	}
	db = db.Preload("Package")
	db = db.Model(entity.Ticket{}).Count(&count)

	return count, db.Error
}

// CountAllTicketsByStatus is statictis count ticket by status
func (m *TicketManager) CountTicketsByStatus(params TicketQueryParams) ([]CountTicketsByStatus, error) {
	result := []CountTicketsByStatus{}
	db := m.buildQueryTickets(params)

	db = db.Table("tickets").Select("COUNT(tickets.id) AS count,tickets.status")
	db = db.Group("tickets.status").Scan(&result)

	return result, db.Error
}

func (m *TicketManager) GetTicketByID(id int64) (*entity.Ticket, error) {
	ticket := &entity.Ticket{}
	db := m.DB.Where("id=?", id)
	db = db.Preload("User").Preload("Package").Preload("Package.PackageCode")
	db = db.First(ticket)
	return ticket, db.Error
}

func (m *TicketManager) GetTicket(opts TicketQueryParams) (*entity.Ticket, error) {
	db := m.buildQueryTickets(opts)
	ticket := &entity.Ticket{}
	db = db.Preload("User")

	if opts.HasPackage {
		db = db.Preload("Package").Preload("Package.PackageCode")
		db = db.Preload("Package").Preload("Package.Service")
		db = db.Preload("Package.ExtraFee", func(db *gorm.DB) *gorm.DB {
			return db.Where("status=?", constant.ExtraFeeStatusEnable)
		})
	}

	db = db.Find(ticket)
	return ticket, db.Error
}

func (m *TicketManager) NewTicket(ticket *entity.Ticket) error {
	ticket.CreatedAt = time.Now()
	ticket.UpdatedAt = time.Now()
	db := m.DB.Model(&entity.Ticket{}).Create(ticket)
	return db.Error
}

func (m *TicketManager) UpdateTicket(ticket *entity.Ticket) error {
	ticket.UpdatedAt = time.Now()

	db := m.DB

	ticketMap := map[string]interface{}{
		"UpdatedAt":  time.Now(),
		"ObjectID":   ticket.ObjectID,
		"Category":   ticket.Category,
		"Title":      ticket.Title,
		"Content":    ticket.Content,
		"Attachment": ticket.Attachment,
		"IsRated":    ticket.IsRated,
	}

	if err := db.Model(&entity.Ticket{}).Where("id = ?", ticket.ID).
		UpdateColumns(ticketMap).Error; err != nil {
		return err
	}

	return db.Error
}

func (m *TicketManager) UpdateStatusTicket(ticketID int64, change map[string]interface{}) error {
	db := m.DB.Model(entity.Ticket{}).Where("id=?", ticketID).Updates(change)
	return db.Error
}

func (m *TicketManager) UpdateTicketLastTime(ticketID int64, t time.Time) error {
	db := m.DB.Model(entity.Ticket{}).Where("id=?", ticketID).Update("updated_at", t)
	return db.Error
}

func (m *TicketManager) UpdateReplyTicket(ticketID int64, isAdmin bool, isCustomer bool) error {
	db := m.DB.Model(entity.Ticket{}).
		Where("id=?", ticketID).
		Updates(map[string]interface{}{
			"is_admin_rep":    isAdmin,
			"is_customer_rep": isCustomer,
			"updated_at":      time.Now(),
		})
	return db.Error
}

func (m *TicketManager) buildTicketMessageQuery(params TicketMessageQueryParams) *gorm.DB {
	db := m.DB
	if params.UserID > 0 {
		db = db.Where("user_id=?", params.UserID)
	}

	if params.TicketID > 0 {
		db = db.Where("ticket_id=?", params.TicketID)
	}

	if params.Status > 0 {
		db = db.Where("status=?", params.Status)
	}

	if params.Keyword != "" {
		db = db.Where("content LIKE ?",
			fmt.Sprintf("%%%s%%", params.Keyword),
		)
	}

	return db
}

func (m *TicketManager) GetTicketMessages(params TicketMessageQueryParams) ([]entity.TicketMessage, error) {
	db := m.buildTicketMessageQuery(params)

	if params.Sort != "" {
		db = db.Order(params.Sort)
	}

	if params.Limit > 0 {
		db = db.Limit(params.Limit)
	}

	if params.Offset > 0 {
		db = db.Offset(params.Offset)
	}

	if params.LastID > 0 {
		db = db.Where("id < ?", params.LastID)
	}

	if params.FirstID > 0 {
		db = db.Where("id > ?", params.FirstID)
	}

	db = db.Preload("User")

	tickets := []entity.TicketMessage{}
	db = db.Find(&tickets)
	return tickets, db.Error
}

func (m *TicketManager) CountTicketMessages(params TicketMessageQueryParams) (int64, error) {
	db := m.buildTicketMessageQuery(params)
	tickets := []entity.TicketMessage{}
	var count int64
	db = db.Model(&tickets)
	db = db.Count(&count)
	return count, db.Error
}

func (m *TicketManager) PushMessage(message *entity.TicketMessage, ticket *entity.Ticket, role string) error {
	tx := m.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	message.CreatedAt = time.Now()
	message.UpdatedAt = message.CreatedAt

	if err := tx.Create(message).Error; err != nil {
		tx.Rollback()
		return err
	}

	ticket.UpdatedAt = time.Now()
	mapUpdate := map[string]interface{}{
		"status_rep": ticket.StatusRep,
		"updated_at": ticket.UpdatedAt,
	}

	if role == constant.UserRoleAdmin {
		mapUpdate["admin_id"] = message.UserID
	}

	if err := tx.Table("tickets").Where("id=?", ticket.ID).Updates(mapUpdate).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m *TicketManager) CancelTicket(id int64) error {
	mapUpdate := map[string]interface{}{
		"status":     constant.TicketStatusProcessed,
		"status_rep": constant.TicketStatusProcessed,
	}
	db := m.DB.Model(entity.Ticket{}).Where("id=?", id).Updates(mapUpdate)
	return db.Error
}

func (m *TicketManager) GetTicketMessageByID(ID int64) (*entity.TicketMessage, error) {
	ticketMessage := &entity.TicketMessage{}
	db := m.DB.Where("id=?", ID).Preload("Ticket").First(ticketMessage)
	return ticketMessage, db.Error
}

func (m *TicketManager) UpdateTicketAndRefund(ticket *entity.Ticket) error {
	tx := m.DB.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Println(r)
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	change := map[string]interface{}{
		"status":     constant.TicketStatusPending,
		"amount":     -ticket.Amount,
		"type":       constant.TicketTypeRefund,
		"note":       ticket.Note,
		"updated_at": time.Now(),
	}

	if err := tx.Model(&entity.Ticket{}).Where("id=?", ticket.ID).Updates(change).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// 05/03/2025 this func wasnt be used, so balance_china wasnt be updated here. Update it when use
func (m *TicketManager) UpdateTicketAndReship(ticket *entity.Ticket, mapchange map[string]interface{}, amount float64, userID, billID int64, tracking *entity.Tracking, description string, logs []entity.PackageAuditLog) error {
	tx := m.DB.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Println(r)
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	ticket.Amount = amount
	ticket.Status = constant.TicketStatusProcessed
	if amount < 0 {
		ticket.Status = constant.TicketStatusPending
	}

	change := map[string]interface{}{
		"status":     ticket.Status,
		"amount":     ticket.Amount,
		"type":       constant.TicketTypeReship,
		"note":       ticket.Note,
		"updated_at": time.Now(),
	}

	if err := tx.Model(&entity.Ticket{}).Where("id=?", ticket.ID).Updates(change).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&entity.Tracking{}).Where("package_id=?", ticket.ObjectID).Update("status", constant.TrackingStatusCanceled).Error; err != nil {
		tx.Rollback()
		return err
	}

	tracking.CreatedAt = time.Now()
	tracking.UpdatedAt = tracking.CreatedAt
	if err := tx.Create(tracking).Error; err != nil {
		tx.Rollback()
		return err
	}

	packageID := ticket.ObjectID
	customerID := ticket.UserID

	mapchange["updated_at"] = time.Now()
	mapchange["alert"] = constant.PackageAlertTypeDisable
	mapchange["status"] = constant.PackageStatusReship
	mapchange["reship_at"] = time.Now()
	mapchange["request_reship"] = false
	if err := tx.Table("packages").Where("id=?", packageID).UpdateColumns(mapchange).Error; err != nil {
		tx.Rollback()
		return err
	}

	extraFee := &entity.ExtraFee{
		BillID:         &billID,
		PackageID:      utils.Int64(packageID),
		ExtraFeeTypeID: constant.ExtraFeeTypeReship,
		Amount:         amount,
		Status:         constant.ExtraFeeStatusEnable,
		Description:    description,
	}

	if err := tx.Create(extraFee).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Exec("UPDATE bills SET extra_fee=extra_fee+? WHERE id=?", amount, billID).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Exec("UPDATE users SET balance=balance-? WHERE id=?", amount, customerID).Error; err != nil {
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

	strsql := "UPDATE user_infos SET debt_time = ? WHERE user_id = ? AND debt_time IS NULL AND (SELECT balance FROM users WHERE id = ? limit 1) < 0"
	if err := tx.Exec(strsql, time.Now(), customerID, customerID).Error; err != nil {
		tx.Rollback()
		return err
	}

	transaction := &entity.Transaction{
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID:  customerID,
		AdminID: userID,
		Type:    constant.TransactionLogTypePay,
		Status:  constant.TransactionStatusSuccess,
		Amount:  -amount,
		BillID:  &billID,
	}

	if err := tx.Create(transaction).Error; err != nil {
		tx.Rollback()
		return err
	}

	transactionLog := &entity.TransactionLog{
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID:        customerID,
		AdminID:       userID,
		TransactionID: transaction.ID,
		Amount:        transaction.Amount,
		Type:          transaction.Type,
		Status:        transaction.Status,
		BillID:        transaction.BillID,
	}

	if err := tx.Create(transactionLog).Error; err != nil {
		tx.Rollback()
		return err
	}

	for i := 0; i < len(logs); i++ {
		logs[i].PackageID = packageID
		logs[i].UpdatedUserID = userID
	}

	log := &entity.PackageAuditLog{
		PackageID:     packageID,
		Type:          constant.PackageUpdateExtraFeeTypeReship,
		UpdatedUserID: userID,
		Fee:           extraFee.Amount,
		Value:         description,
		Description:   description,
	}

	logs = append(logs, *log)
	if err := tx.Create(&logs).Error; err != nil {
		tx.Rollback()
		return err
	}

	deliverLog := &entity.PackageDeliverLog{
		PackageID:   packageID,
		Status:      constant.DeliverLogTebexpressReship,
		Type:        constant.PackageDeliverLogTypeReship,
		Description: description,
		UserID:      &userID,
		Model: dbgorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	if err := tx.Create(deliverLog).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// 21/02/2025 this func wasnt be used, so balance_china wasnt be updated here. Update it when used
func (m *TicketManager) UpdateTicketAndConfirmRefund(ticket *entity.Ticket, userID, billID int64) error {
	tx := m.DB.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Println(r)
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	change := map[string]interface{}{
		"status":        constant.TicketStatusProcessed,
		"accountant_id": userID,
		"updated_at":    time.Now(),
	}

	if err := tx.Model(&entity.Ticket{}).Where("id=?", ticket.ID).Updates(change).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Exec("UPDATE bills SET extra_fee = extra_fee + ?, updated_at = ? WHERE id = ?", ticket.Amount, time.Now(), billID).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Exec("UPDATE users SET balance = balance - ?, updated_at = ? WHERE id = ?", ticket.Amount, time.Now(), ticket.UserID).Error; err != nil {
		tx.Rollback()
		return err
	}

	extraFee := &entity.ExtraFee{
		PackageID:      utils.Int64(ticket.ObjectID),
		BillID:         utils.Int64(billID),
		ExtraFeeTypeID: constant.ExtraFeeTypeRefund,
		Amount:         ticket.Amount,
		Status:         constant.ExtraFeeStatusEnable,
		Description:    fmt.Sprintf("Hoàn tiền của khiếu nại #%d", ticket.ID),
	}
	if err := tx.Save(extraFee).Error; err != nil {
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
		info = &entity.UserInfo{UserID: ticket.UserID, UpdatedAt: &now, CancelMaxAmount: constant.DefaultCancelMaxAMount}
		if err := tx.Create(info).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	str := `UPDATE user_infos SET debt_time = NULL WHERE user_id = ? AND debt_time IS NOT NULL AND (SELECT balance FROM users WHERE id = ? limit 1) >= 0`
	if err := tx.Exec(str, time.Now(), ticket.UserID, ticket.UserID).Error; err != nil {
		tx.Rollback()
		return err
	}

	auditLogType := constant.MapTypeAuditLogByExtraFeeTypeID[extraFee.ExtraFeeTypeID]
	if auditLogType > 0 {
		logs := &entity.PackageAuditLog{
			PackageID:     utils.Int64Value(extraFee.PackageID),
			Type:          auditLogType,
			UpdatedUserID: userID,
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
		UserID:      ticket.UserID,
		Type:        constant.TransactionLogTypeRefund,
		Status:      constant.TransactionStatusSuccess,
		Amount:      math.Abs(ticket.Amount),
		BillID:      utils.Int64(billID),
		Description: cast.ToString(ticket.ID),
		AdminID:     userID,
	}

	err := tx.Create(&transaction).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	transactionLog := &entity.TransactionLog{
		UserID:        ticket.UserID,
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

func (m *TicketManager) NewFeedback(fbs []entity.Feedback) error {
	db := m.DB.Save(fbs)
	return db.Error
}

func (m *TicketManager) GetSupportsTicket(ticketID, userID int64) ([]entity.User, error) {
	var supports []entity.User
	db := m.DB.Model(&entity.User{}).Select("id,full_name").Where("id IN (?)", m.DB.Model(&entity.TicketMessage{}).Select("user_id").Where("ticket_id = ? AND user_id <> ?", ticketID, userID)).Find(&supports)
	return supports, db.Error
}

func (m *TicketManager) GetListTicketsCustomer(userIDs []int64, supportID int64, result interface{}) error {
	db := m.DB
	db = db.Model(&entity.Ticket{}).Where("user_id IN (?)", userIDs)
	db = db.Joins("LEFT JOIN feedbacks ON feedbacks.ticket_id = tickets.id AND support_id = ?", supportID)
	db = db.Group("rating,user_id")
	db = db.Select("tickets.user_id as customer_id,COUNT(tickets.id) as count,feedbacks.rating")
	db = db.Scan(result)
	return db.Error
}
