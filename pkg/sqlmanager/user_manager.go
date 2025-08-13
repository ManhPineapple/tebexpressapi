package sqlmanager

import (
	"encoding/base64"
	"fmt"
	"strings"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"time"

	"github.com/google/uuid"

	"gorm.io/gorm"
)

type UserManager struct {
	db *gorm.DB
}
type FetchUserRequest struct {
	Id          int64
	UserName    string
	Email       string
	PhoneNumber string
	SlackID     string
}

type CreateUserForm struct {
	ID          int64   `json:"id"`
	Email       string  `json:"email"`
	PhoneNumber string  `json:"phone_number"`
	FullName    string  `json:"full_name"`
	Role        string  `json:"role"`
	Password    string  `json:"password"`
	SlackID     string  `json:"slack_id"`
	WarehouseID int64   `json:"warehouse_id"`
	CustomerID  []int64 `json:"customer_id"`
}

func NewUserManager(db *gorm.DB) *UserManager {
	return &UserManager{
		db: db,
	}
}

type UserQueryOption struct {
	ID               int64
	IDS              []int64
	Search           string
	Limit            int
	Offset           int
	UserID           int64
	Role             string
	InRole           []string
	Status           string
	ArrStatus        []int
	Code             string
	SupportQuery     int64
	IsHasInfo        bool
	AppraiserID      int64
	QueryThisWeek    bool
	IntStatus        int
	ReferralCode     string
	WarehouseID      int64
	RefID            int64
	CheckAddPointDay int64
	IsRefCustomer    bool
	IgnoreIDs        []int64
	PriceType        []int64
	IsPrePay         bool
	IsPostpaid       bool
	IsHasReferring   bool
	PartnerID        int64
}

func (m *UserManager) buildUserQuery(opts UserQueryOption) *gorm.DB {
	db := m.db

	if opts.ID > 0 {
		db = db.Where("users.id=?", opts.ID)
	}

	if len(opts.IDS) > 0 {
		db = db.Where("users.id IN (?)", opts.IDS)
	}

	if len(opts.IgnoreIDs) > 0 {
		db = db.Where("users.id NOT IN (?)", opts.IgnoreIDs)
	}
	if len(opts.PriceType) > 0 {
		db = db.Where("users.class  IN (?)", opts.PriceType)
	}

	if opts.PartnerID > 0 {
		db = db.Where("users.partner_id=?", opts.PartnerID)
	}

	if opts.IsPostpaid && !opts.IsPrePay {
		db = db.Joins(" LEFT JOIN user_infos  ON users.id = user_infos.user_id")
		db = db.Where("user_infos.debt_max_amount > 0")
	}
	if opts.IsPrePay && !opts.IsPostpaid {
		db = db.Joins("LEFT JOIN user_infos  ON users.id = user_infos.user_id")
		db = db.Where("user_infos.debt_max_amount = 0 OR user_infos.debt_max_amount IS NULL")
	}

	if opts.SupportQuery > 0 && opts.AppraiserID < 1 {
		db = db.Joins("JOIN user_permissions  ON users.id = user_permissions.customer_id")
		db = db.Where("user_permissions.support_id = ?", opts.SupportQuery)
	}

	if opts.AppraiserID > 1 {
		db = db.Joins("JOIN user_infos  ON users.id = user_infos.user_id")
		db = db.Where("user_infos.appraiser_id = ?", opts.AppraiserID)
	}

	if opts.Role != "" && len(opts.InRole) < 1 {
		db = db.Where("users.role=?", opts.Role)
	}

	if len(opts.InRole) > 0 {
		db = db.Where("users.role IN (?)", opts.InRole)
	}

	if opts.Status != "" {
		db = db.Where("users.status=?", opts.Status)
	}

	if len(opts.ArrStatus) > 0 {
		db = db.Where("users.status IN (?)", opts.ArrStatus)

	}

	if opts.Code != "" {
		db = db.Joins("JOIN packages on packages.user_id = users.id")
		db = db.Joins("JOIN package_codes ON package_codes.id=packages.package_code_id AND package_codes.code=?", opts.Code)
	}

	if opts.Search != "" {
		userID := opts.Search
		if strings.Contains(strings.ToUpper(opts.Search), "U") {
			userID = strings.Split(strings.ToUpper(opts.Search), "U")[1]
		}

		if userID == "" {
			userID = opts.Search
		}

		db = db.Where("users.username like ? OR users.email like ? OR users.full_name like ? OR users.id like ?",
			fmt.Sprintf("%%%s%%", opts.Search),
			fmt.Sprintf("%%%s%%", opts.Search),
			fmt.Sprintf("%%%s%%", opts.Search),
			fmt.Sprintf("%%%s%%", userID),
		)
	}

	if opts.CheckAddPointDay > 0 {
		db = db.Where("id IN (?)", m.db.Model(&entity.Package{}).
			Joins("INNER JOIN package_deliver_logs ON package_deliver_logs.package_id = packages.id").
			Select("DISTINCT(packages.user_id)").
			Where("package_deliver_logs.status = ? AND package_deliver_logs.type = ? AND DATE(convert_tz(package_deliver_logs.created_at, @@session.time_zone,'+07:00')) >= (NOW() - INTERVAL ? DAY) AND DATE(convert_tz(package_deliver_logs.created_at,@@session.time_zone,'+07:00')) < CURDATE()", constant.DeliverLogTebexpressInTransit, constant.PackageDeliverLogTypeInTransit, opts.CheckAddPointDay))
	}

	if opts.QueryThisWeek {
		db = db.Where("YEARWEEK(users.created_at) = YEARWEEK(NOW())")
	}

	if opts.ReferralCode != "" {
		db = db.Where("users.referral_code = ?", opts.ReferralCode)
	}

	if opts.RefID > 0 {
		db = db.Where("users.ref_id = ?", opts.RefID)
	}

	if opts.IsRefCustomer {
		db = db.Where("users.ref_id > 0")
	}

	if opts.IsHasReferring {
		db = db.Preload("ReferringUser")
	}

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	db = db.Order("users.id DESC")

	return db
}

func (m *UserManager) IsExistUniqueKey(req *FetchUserRequest) bool {
	var count int64
	userOrm := m.db.Model(&entity.User{})
	if req.UserName != "" {
		userOrm = userOrm.Or(entity.User{UserName: req.UserName})
	}
	if req.Email != "" {
		userOrm = userOrm.Or(entity.User{Email: req.Email})
	}
	if req.PhoneNumber != "" {
		userOrm = userOrm.Or(entity.User{PhoneNumber: req.PhoneNumber})
	}
	if req.SlackID != "" {
		userOrm = userOrm.Or(entity.User{SlackID: req.SlackID})
	}
	db := userOrm.Count(&count)

	if db.Error != nil {
		fmt.Errorf("Error while check exist user %v, details: %v", req, db.Error)
		return true
	}

	return count > 0
}

func (m *UserManager) CreateUser(user *entity.User, referralUserID int64) (*entity.User, error) {

	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return nil, err
	}

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	token := fmt.Sprintf("%s:%s", user.Email, uuid.New().String())
	if user.Email == "" {
		token = fmt.Sprintf("%s:%s", user.PhoneNumber, uuid.New().String())
	}
	tokenBase64 := base64.StdEncoding.EncodeToString([]byte(token))

	userToken := &entity.UserToken{
		UserID: user.ID,
		Status: constant.TokenActive,
		Token:  tokenBase64,
	}

	if err := tx.Create(&userToken).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	now := time.Now()
	info := &entity.UserInfo{
		UserID:          user.ID,
		UpdatedAt:       &now,
		RefundDay:       constant.DefaultRefundDay,
		CancelMaxAmount: constant.DefaultCancelMaxAMount,
	}

	if referralUserID > 0 {
		info.AppraiserID = referralUserID
	}

	if err := tx.Create(info).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if referralUserID > 0 {
		permission := &entity.UserPermission{
			SupportID:  referralUserID,
			CustomerID: user.ID,
		}

		if err := tx.Create(permission).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		if user.Status == constant.UserStatusActive {
			promotion := &entity.UserPromotion{
				PromotionID: constant.PromotionNewLabelID,
				UserID:      user.ID,
				Status:      constant.StatusActive,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}

			if err := tx.Create(promotion).Error; err != nil {
				tx.Rollback()
				return nil, err
			}
		}
	}

	return user, tx.Commit().Error
}

func (m *UserManager) GetUserByID(userID int64) (*entity.User, error) {
	user := &entity.User{}
	db := m.db.Where("id=?", userID).Preload("UserInfo").First(user)
	return user, db.Error
}

func (m *UserManager) FetchUserWithoutInfo(userID int64) (*entity.User, error) {
	user := &entity.User{}
	db := m.db.Where("id=?", userID).First(user)
	return user, db.Error
}

func (m *UserManager) UpdateUser(user *entity.User) (*entity.User, error) {

	userMapString := map[string]interface{}{
		"UpdatedAt": time.Now(),
	}

	if user.FullName != "" {
		userMapString["FullName"] = user.FullName
	}

	if user.Email != "" {
		userMapString["Email"] = user.Email
	}

	if user.Password != "" {
		userMapString["Password"] = user.Password
	}

	if user.UserName != "" {
		userMapString["UserName"] = user.UserName
	}

	if user.Birthday != "" {
		userMapString["Birthday"] = user.Birthday
	}

	db := m.db.Model(&entity.User{}).Where("id = ?", user.ID).UpdateColumns(userMapString)

	return user, db.Error
}

func (m *UserManager) GetUserByUniqueKey(req *FetchUserRequest) (*entity.User, error) {
	user := &entity.User{}
	userOrm := m.db
	if req.PhoneNumber != "" {
		userOrm = userOrm.Or(entity.User{PhoneNumber: req.PhoneNumber})
	}
	if req.Email != "" {
		userOrm = userOrm.Or(entity.User{Email: req.Email})
	}
	db := userOrm.First(user)
	return user, db.Error
}
func (m *UserManager) GetUsers(opts UserQueryOption) ([]*entity.User, error) {
	users := make([]*entity.User, 0)
	db := m.buildUserQuery(opts)

	if opts.IsHasInfo {
		db = db.Preload("UserInfo")
	}

	db = db.Find(&users)
	return users, db.Error
}

func (m *UserManager) GetUsersByOptions(opts UserQueryOption) ([]*entity.User, error) {
	users := make([]*entity.User, 0)
	db := m.buildUserQuery(opts)

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	if opts.IsHasInfo {
		db = db.Preload("UserInfo")
	}

	if opts.IsHasReferring {
		db = db.Preload("ReferringUser")
	}

	db = db.Find(&users)
	var ids []int64
	for i, _ := range users {
		ids = append(ids, users[i].ID)
	}

	var result []struct {
		ID    int64   `json:"id"`
		Total float64 `json:"total"`
	}
	err := m.GetHoldingMoneyByListUser(ids, &result)
	if err != nil {
		fmt.Errorf("Get user holding money error, %v", err)
		return nil, err
	}
	mapR := make(map[int64]float64)
	for _, i := range result {
		mapR[i.ID] = i.Total
	}
	for i, _ := range users {
		users[i].HoldingMoney = mapR[users[i].ID]
	}

	return users, db.Error
}

func (m *UserManager) CountUserByOptions(opts UserQueryOption) (int64, error) {
	db := m.buildUserQuery(opts)

	var count int64

	db = db.Model(&entity.User{}).Select("COUNT(users.id) as count").Count(&count)

	return count, db.Error
}

func (m *UserManager) UpdateStatusUser(UserID int64, Status int64, support_ids []int64, idsDelete []int64, idsAdd []int64, oldStatus int64) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if Status == constant.UserStatusActive || Status == constant.UserStatusDeactive {
		if err := tx.Model(&entity.User{}).Where("id = ?", UserID).Update("status", Status).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	for _, id := range idsDelete {
		if err := tx.Model(entity.UserPermission{}).Where("support_id = ? AND customer_id = ?", id, UserID).Delete(entity.UserPermission{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	for _, id := range idsAdd {
		permission := entity.UserPermission{
			SupportID:  id,
			CustomerID: UserID,
		}
		if err := tx.Model(entity.UserPermission{}).Create(permission).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m *UserManager) UpdateRoleUser(UserID int64, Role string) (int64, error) {
	db := m.db.Model(&entity.User{}).Where("id = ?", UserID)

	db = db.Update("role", Role)

	return db.RowsAffected, db.Error
}
func (m *UserManager) GetUserTokenByUserID(userID int64) (*entity.UserToken, error) {
	userToken := &entity.UserToken{}
	db := m.db.Where("user_id=? AND status = ?", userID, constant.TokenActive).First(userToken)
	return userToken, db.Error
}

func (m *UserManager) ResetTokenByUserID(userID int64, token string) error {

	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Model(&entity.UserToken{}).
		Where("user_id = ? AND status = ?", userID, constant.TokenActive).
		Update("status", constant.TokenDeactive).Error; err != nil {
		tx.Rollback()
		return err
	}

	userToken := &entity.UserToken{
		UserID: userID,
		Status: constant.TokenActive,
		Token:  token,
	}

	if err := tx.Create(&userToken).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m *UserManager) AdminCreateUser(user *entity.User, customerIDs []int64) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	user.Status = constant.UserStatusActive
	if err := tx.Model(entity.User{}).Create(user).Error; err != nil {
		tx.Rollback()
		return err
	}

	if (user.Role == constant.UserRoleSupport || user.Role == constant.UserRoleSale) && len(customerIDs) > 0 {
		permissions := []entity.UserPermission{}
		for _, id := range customerIDs {
			if id <= 0 {
				continue
			}
			permissions = append(permissions, entity.UserPermission{
				SupportID:  user.ID,
				CustomerID: id,
			})
		}

		if err := tx.Create(&permissions).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m *UserManager) ActiveAccount(userID int64) error {

	//db := m.db.Model(entity.User{}).Where("id=?", userID).Update("status", constant.UserStatusActive)

	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if err := tx.Model(entity.User{}).Where("id=?", userID).Update("status", constant.UserStatusActive).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (m *UserManager) GetUserInfoByUserID(userID int64) (*entity.UserInfo, error) {
	ui := &entity.UserInfo{}
	db := m.db.Where("user_id=?", userID).First(ui)
	return ui, db.Error
}

func (m *UserManager) UpdateInfoUser(change map[string]interface{}, userID int64) error {
	db := m.db.Model(&entity.User{}).Where("id = ?", userID).UpdateColumns(change)
	return db.Error
}

func (m *UserManager) SaveUser(ui *entity.UserInfo, user *entity.User, class int64) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	if ui.ID > 0 {
		err := tx.Where("id=?", ui.ID).Save(ui).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	} else {
		err := tx.Create(ui).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	userMapString := map[string]interface{}{
		"UpdatedAt": time.Now(),
	}
	if user.Class != class {
		userMapString["Class"] = class
	}

	err := tx.Model(user).UpdateColumns(userMapString).Error
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (m *UserManager) AdminUpdateUser(user *entity.User, idsDelete []int64, idsAdd []int64, moveSupportID int64) error {
	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	change := map[string]interface{}{
		"UpdatedAt": time.Now(),
	}

	change["SlackID"] = user.SlackID

	if user.FullName != "" {
		change["FullName"] = user.FullName
	}

	if user.Email != "" {
		change["Email"] = user.Email
	}

	if user.PhoneNumber != "" {
		change["PhoneNumber"] = user.PhoneNumber
	}

	if user.Role != "" {
		change["Role"] = user.Role
	}

	if user.Password != "" {
		change["Password"] = user.Password
	}

	if user.WarehouseID > 0 {
		change["WarehouseID"] = user.WarehouseID
	} else {
		change["WarehouseID"] = nil

	}

	if err := tx.Model(&entity.User{}).Where("id=?", user.ID).UpdateColumns(change).Error; err != nil {
		tx.Rollback()
		return err
	}

	if len(idsDelete) > 0 {
		if err := tx.Model(entity.UserPermission{}).Where("support_id = ? AND customer_id IN (?)", user.ID, idsDelete).Delete(entity.UserPermission{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	var permissions = []entity.UserPermission{}
	for _, id := range idsAdd {
		supportID := user.ID
		if moveSupportID > 0 {
			supportID = moveSupportID
		}

		permissions = append(permissions, entity.UserPermission{
			SupportID:  supportID,
			CustomerID: id,
		})
	}

	if len(permissions) > 0 {
		if err := tx.Create(&permissions).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m *UserManager) GetCustomerIDBySupportID(id int64) ([]int64, error) {
	var ids []int64
	db := m.db
	db = db.Table("user_permissions").Where("support_id=?", id).Pluck("customer_id", &ids)

	return ids, db.Error
}

func (m *UserManager) GetSupportIDByCustomerID(id int64) ([]int64, error) {
	var ids []int64
	db := m.db
	db = db.Table("user_permissions").Where("customer_id=?", id).Pluck("support_id", &ids)

	return ids, db.Error
}

func (m *UserManager) GetWareHouseByUserSupport(userID int64) (*entity.Warehouse, error) {
	warehouse := &entity.Warehouse{}
	db := m.db.Joins("INNER JOIN users ON users.warehouse_id=warehouses.id AND users.id=?", userID)
	db = db.First(warehouse)
	return warehouse, db.Error
}

func (m *UserManager) GetSupportByUserID(id int64) ([]int64, error) {
	var ids []int64
	db := m.db
	db = db.Table("user_permissions").Where("customer_id = ?", id).Pluck("support_id", &ids)
	return ids, db.Error
}

func (m *UserManager) GetHoldingMoney(id int64) (float64, error) {
	sql := `SELECT SUM(package_refunds.amount) as total 
				FROM package_refunds
				LEFT JOIN packages ON packages.id = package_refunds.package_id
				WHERE packages.user_id = ? AND package_refunds.status = ?
		`
	var result = struct {
		Total float64 `json:"total"`
	}{}
	db := m.db.Raw(sql, id, constant.PackageRefundPending).Scan(&result)
	return result.Total, db.Error
}

func (m *UserManager) GetHoldingMoneyByListUser(ids []int64, result interface{}) error {
	sql := `SELECT packages.user_id as id ,SUM(package_refunds.amount) as total
				FROM package_refunds
				INNER JOIN packages ON packages.id = package_refunds.package_id
				WHERE packages.user_id IN (?) AND package_refunds.status = ?
				GROUP BY packages.user_id`
	db := m.db.Raw(sql, ids, constant.PackageRefundPending).Scan(result)
	return db.Error
}

func (m *UserManager) AdminAppraiUser(id int64, user *entity.User) error {
	userMapString := map[string]interface{}{
		"UpdatedAt": time.Now(),
	}
	if user.FullName != "" {
		userMapString["FullName"] = user.FullName
	}

	if user.Email != "" {
		userMapString["Email"] = user.Email
	}

	if user.PhoneNumber != "" {
		userMapString["PhoneNumber"] = user.PhoneNumber
	}

	if user.Package > 0 {
		userMapString["Package"] = user.Package
	}

	userInfoMapString := map[string]interface{}{
		"UpdatedAt": time.Now(),
	}

	userInfoMapString["AppraiserID"] = user.UserInfo.AppraiserID

	userInfoMapString["TaxCode"] = user.UserInfo.TaxCode

	userInfoMapString["Volume"] = user.UserInfo.Volume

	userInfoMapString["ItemType"] = user.UserInfo.ItemType

	userInfoMapString["WarehouseAddress"] = user.UserInfo.WarehouseAddress

	userInfoMapString["AppraiserID"] = user.UserInfo.AppraiserID

	user.UserInfo.UserID = id

	tx := m.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	err := tx.Model(&entity.User{}).Where("id=?", id).UpdateColumns(userMapString).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Table("user_infos").Where("user_id=?", id).First(&entity.UserInfo{}).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			if err := tx.Model(&entity.UserInfo{}).Save(user.UserInfo).Error; err != nil {
				tx.Rollback()
				return err
			}
		} else {
			tx.Rollback()
			return err
		}
	} else {
		if err := tx.Model(&entity.UserInfo{}).Where("user_id=?", id).UpdateColumns(userInfoMapString).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (m *UserManager) GetUsersExpired() ([]*entity.User, error) {
	var users []*entity.User
	db := m.db
	db = db.Joins("JOIN user_infos ON user_infos.user_id = users.id")
	db = db.Preload("UserInfo")
	db = db.Where("user_infos.debt_time + INTERVAL user_infos.debt_max_day day >= NOW()")

	db = db.Find(&users)
	return users, db.Error
}

func (m *UserManager) GetUsersOutOfMoney() ([]*entity.User, error) {
	var users []*entity.User

	db := m.db
	db = db.Joins("LEFT JOIN user_infos ON user_infos.user_id = users.id")
	db = db.Preload("UserInfo")
	db = db.Where(`
		(-users.balance >= user_infos.debt_max_amount AND user_infos.debt_max_amount > 0)
		OR((user_infos.debt_max_amount <= 0
			OR user_infos.debt_max_amount IS NULL)
		AND users.Balance + 0.01 <= 0
		AND users.id IN(
			SELECT
				user_id FROM transactions
			WHERE
				status in(?)))
	`, []int{constant.TransactionStatusSuccess, constant.TransactionStatusRefund})

	db = db.Where("status=?", constant.UserStatusActive)

	db = db.Find(&users)
	return users, db.Error
}

func (m *UserManager) DeactiveAccount() error {
	raw := `UPDATE users SET status = ?
			WHERE status  = ?
			AND role = ?
			AND DATEDIFF(NOW(),updated_at) >= ?
			AND id NOT IN (SELECT DISTINCT packages.user_id  from packages
			Where DATEDIFF(NOW(),packages.created_at) <= ?)`
	db := m.db.Exec(raw, constant.UserStatusDeactive, constant.UserStatusActive, constant.UserRoleCustomer, constant.CheckDeactiveDay, constant.CheckDeactiveDay)
	return db.Error
}

func (m *UserManager) CreateUserInfo(info *entity.UserInfo) error {
	now := time.Now()
	info.UpdatedAt = &now
	return m.db.Create(info).Error
}

func (m *UserManager) GetCustomersInfo(supporID int64, limit, offset int, hasRevenue bool, result interface{}) error {
	db := m.db.Model(&entity.User{})
	db = db.Where("users.id IN (?)", m.db.Model(&entity.UserPermission{}).Select("customer_id").Where("support_id = ?", supporID))
	db = db.Where("users.status = ?", constant.UserStatusActive)
	db = db.Joins("LEFT JOIN bills ON bills.user_id = users.id")
	if hasRevenue {
		db = db.Where("bills.id IS NOT NULL")
	} else {
		db = db.Where("bills.id IS NULL")
	}
	db = db.Group("users.id")
	db = db.Limit(limit)
	db = db.Offset(offset)
	db = db.Select("users.id,users.full_name,users.email,users.created_at")
	db = db.Scan(result)
	return db.Error
}

func (m *UserManager) CountCustomersInfo(supporID int64, hasRevenue bool, result interface{}) error {
	db := m.db.Model(&entity.User{})
	db = db.Where("users.id IN (?)", m.db.Model(&entity.UserPermission{}).Select("customer_id").Where("support_id = ?", supporID))
	db = db.Where("users.status = ?", constant.UserStatusActive)
	db = db.Joins("LEFT JOIN bills ON bills.user_id = users.id")
	if hasRevenue {
		db = db.Where("bills.id IS NOT NULL")
	} else {
		db = db.Where("bills.id IS NULL")
	}
	db = db.Select("COUNT(DISTINCT `users`.`id`) as count")
	db = db.Scan(result)
	return db.Error
}

func (m *UserManager) GetUserBySelectField(userID int64, field string, result interface{}) error {
	db := m.db.Model(&entity.User{}).Where("id=?", userID).Scan(result)
	db = db.Select(field)
	return db.Error
}

func (m *UserManager) FindSales(opts UserQueryOption) ([]entity.User, error) {
	db := m.db
	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Limit(opts.Offset)
	}

	if opts.Role != "" {
		db = db.Where("users.role=?", opts.Role)
	}

	if opts.IntStatus > 0 {
		db = db.Where("users.status=?", opts.IntStatus)
	}

	if opts.WarehouseID > 0 {
		db = db.Where("users.warehouse_id = ?", opts.WarehouseID)
	}

	if opts.Search != "" {
		db = db.Where("username like ? OR email like ? OR full_name like ?",
			fmt.Sprintf("%%%s%%", opts.Search),
			fmt.Sprintf("%%%s%%", opts.Search),
			fmt.Sprintf("%%%s%%", opts.Search),
		)
	}

	users := []entity.User{}
	db = db.Preload("Warehouse")
	db = db.Find(&users)
	return users, db.Error
}

func (m *UserManager) CountSales(opts UserQueryOption) (int64, error) {
	db := m.db.Model(&entity.User{})

	if opts.Role != "" {
		db = db.Where("users.role=?", opts.Role)
	}

	if opts.IntStatus > 0 {
		db = db.Where("users.status=?", opts.IntStatus)
	}

	if opts.Search != "" {
		db = db.Where("username like ? OR email like ? OR full_name like ?",
			fmt.Sprintf("%%%s%%", opts.Search),
			fmt.Sprintf("%%%s%%", opts.Search),
			fmt.Sprintf("%%%s%%", opts.Search),
		)
	}

	var count int64
	db = db.Count(&count)

	return count, db.Error
}

type CountItem struct {
	ID    int64
	Count int64
}

func (m *UserManager) MapCountCustomerOfSales(saleIDs []int64) (map[int64]int64, error) {
	db := m.db.Table("user_permissions")
	db = db.Joins("INNER JOIN users ON users.id = user_permissions.customer_id")
	db = db.Where("users.status=?", constant.UserStatusActive)

	if len(saleIDs) > 0 {
		db = db.Where("support_id IN (?)", saleIDs)
	}

	db = db.Select("user_permissions.support_id AS id, COUNT(*) AS count")
	db = db.Group("user_permissions.support_id")

	items := []CountItem{}
	if err := db.Scan(&items).Error; err != nil {
		return nil, err
	}

	var result = make(map[int64]int64)
	for _, v := range items {
		result[v.ID] = v.Count
	}

	return result, nil
}

func (m *UserManager) MapCountTicketOfSales(saleIDs []int64) (map[int64]int64, error) {
	db := m.db.Table("user_permissions")
	db = db.Joins("INNER JOIN tickets ON tickets.user_id = user_permissions.customer_id")
	db = db.Joins("INNER JOIN users ON users.id = user_permissions.customer_id")
	db = db.Where("users.status=?", constant.UserStatusActive)

	if len(saleIDs) > 0 {
		db = db.Where("support_id IN (?)", saleIDs)
	}

	db = db.Select("user_permissions.support_id AS id, COUNT(*) AS count")
	db = db.Group("user_permissions.support_id")

	items := []CountItem{}
	if err := db.Scan(&items).Error; err != nil {
		return nil, err
	}

	var result = make(map[int64]int64)
	for _, v := range items {
		result[v.ID] = v.Count
	}

	return result, nil
}

func (m *UserManager) MapRevenueOfSales(saleIDs []int64) (map[int64]float64, error) {
	db := m.db.Model(entity.Bill{})
	db = db.Joins("INNER JOIN users ON users.id = bills.user_id  AND users.status = ?", constant.UserStatusActive)
	db = db.Joins("INNER JOIN user_permissions ON users.id = user_permissions.customer_id", constant.UserStatusActive)
	db = db.Where("user_permissions.support_id IN (?)", saleIDs)
	db = db.Select("user_permissions.support_id AS id, sum(bills.shipping_fee + bills.extra_fee) AS revenue")
	db = db.Group("user_permissions.support_id")

	type item struct {
		ID      int64
		Revenue float64
	}

	items := []item{}
	if err := db.Scan(&items).Error; err != nil {
		return nil, err
	}

	var result = make(map[int64]float64)
	for _, v := range items {
		result[v.ID] = v.Revenue
	}

	return result, db.Error
}

type StaticOptions struct {
	CustomerID  int64
	CustomerIDs []int64
	DateStart   string
	DateEnd     string
}

func (m *UserManager) CustomerStatisticRevenue(v interface{}, opts *StaticOptions) error {
	db := m.db.Model(&entity.User{})
	db = db.Joins("INNER JOIN bills ON users.id = bills.user_id")
	db = db.Where("users.status=?", constant.UserStatusActive)
	db = db.Select("DATE_FORMAT(convert_tz(bills.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') AS date_time, SUM(bills.shipping_fee + bills.extra_fee) AS revenue")
	db = db.Group("date_time")

	if opts != nil {
		if opts.CustomerID > 0 {
			db = db.Where("users.id=?", opts.CustomerID)
		}

		if len(opts.CustomerIDs) > 0 {
			db = db.Where("users.id IN (?)", opts.CustomerIDs)
		}

		if opts.DateStart != "" {
			db = db.Where("DATE_FORMAT(convert_tz(bills.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= ?", opts.DateStart)
		}

		if opts.DateEnd != "" {
			db = db.Where("DATE_FORMAT(convert_tz(bills.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= ?", opts.DateEnd)
		}
	}

	if err := db.Scan(v).Error; err != nil {
		return err
	}

	return db.Error
}

func (m *UserManager) CustomerStatisticPackage(v interface{}, opts *StaticOptions) error {
	db := m.db.Model(&entity.Package{})
	db = db.Where("packages.status NOT IN (?)", []int{constant.PackageStatusCreated, constant.PackageStatusArchived})
	db = db.Select("DATE_FORMAT(convert_tz(packages.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') AS date_time, COUNT(packages.id) AS number")
	db = db.Group("date_time")

	if opts != nil {
		if opts.CustomerID > 0 {
			db = db.Where("packages.user_id=?", opts.CustomerID)
		}

		if len(opts.CustomerIDs) > 0 {
			db = db.Where("packages.user_id IN (?)", opts.CustomerIDs)
		}

		if opts.DateStart != "" {
			db = db.Where("DATE_FORMAT(convert_tz(packages.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= ?", opts.DateStart)
		}

		if opts.DateEnd != "" {
			db = db.Where("DATE_FORMAT(convert_tz(packages.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= ?", opts.DateEnd)
		}
	}

	if err := db.Scan(v).Error; err != nil {
		return err
	}

	return db.Error
}

func (m *UserManager) CustomerStatisticTicket(v interface{}, opts *StaticOptions) error {
	db := m.db.Model(&entity.Ticket{})
	db = db.Select("DATE_FORMAT(convert_tz(tickets.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') AS date_time, COUNT(tickets.id) AS number")
	db = db.Group("date_time")

	if opts != nil {
		if opts.CustomerID > 0 {
			db = db.Where("tickets.user_id=?", opts.CustomerID)
		}

		if len(opts.CustomerIDs) > 0 {
			db = db.Where("tickets.user_id IN (?)", opts.CustomerIDs)
		}

		if opts.DateStart != "" {
			db = db.Where("DATE_FORMAT(convert_tz(tickets.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= ?", opts.DateStart)
		}

		if opts.DateEnd != "" {
			db = db.Where("DATE_FORMAT(convert_tz(tickets.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= ?", opts.DateEnd)
		}
	}

	if err := db.Scan(v).Error; err != nil {
		return err
	}

	return db.Error
}

func (m *UserManager) GetUser(opts UserQueryOption) (*entity.User, error) {
	user := &entity.User{}
	db := m.buildUserQuery(opts).Preload("UserInfo").First(user)
	return user, db.Error
}

func (m *UserManager) FetchUserWarehouse(opts UserQueryOption) (*entity.Warehouse, error) {
	warehouse := &entity.Warehouse{}
	db := m.db.Where("id IN (?)", m.db.Model(&entity.User{}).Where("id = ?", opts.ID).Select("warehouse_id")).First(&warehouse)
	return warehouse, db.Error
}

func (m *UserManager) GetListUserAffiliate(opts UserQueryOption, result interface{}) error {
	db := m.db

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	if opts.RefID > 0 {
		db = db.Where("users.ref_id = ?", opts.RefID)
	}

	db = db.Model(&entity.User{}).Joins("LEFT JOIN transactions ON transactions.admin_id = users.id AND transactions.type = ?", constant.TransactionLogTypeAffliate)
	db = db.Group("users.id")
	db = db.Select("SUM(transactions.amount) as comission,users.id,users.full_name,users.created_at")
	db = db.Order("comission DESC")
	db = db.Scan(result)
	return db.Error
}
