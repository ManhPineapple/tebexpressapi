package sqlmanager

import (
	"gorm.io/gorm"
)

type AnalyticsManager struct {
	db *gorm.DB
}

type PackageAnalyticsOptions struct {
	UserID        int64
	InUserID      []int64
	StartDate     string
	EndDate       string
	Status        int
	InStatus      []int
	ExcludeStatus []int
}

func NewAnalyticsManager(db *gorm.DB) *AnalyticsManager {
	return &AnalyticsManager{db: db}
}

func (m *AnalyticsManager) PackageAnalyticsStatus(opts PackageAnalyticsOptions, v interface{}) error {
	db := m.db.Table("packages")

	if opts.UserID > 0 {
		db = db.Where("packages.user_id=?", opts.UserID)
	}

	if len(opts.InUserID) > 0 {
		db = db.Where("packages.user_id IN (?)", opts.InUserID)
	}

	if opts.Status > 0 {
		db = db.Where("packages.status=?", opts.Status)
	}

	if len(opts.InStatus) > 0 {
		db = db.Where("packages.status IN (?)", opts.InStatus)
	}

	if len(opts.ExcludeStatus) > 0 {
		db = db.Where("packages.status NOT IN (?)", opts.ExcludeStatus)
	}

	if len(opts.StartDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(packages.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') >= DATE(?)", opts.StartDate)
	}

	if len(opts.EndDate) > 0 {
		db = db.Where("DATE_FORMAT(convert_tz(packages.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') <= DATE(?)", opts.EndDate)
	}

	db = db.Select("status, DATE_FORMAT(convert_tz(packages.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d') AS date_time, COUNT(1) AS count")
	db = db.Group("status, DATE_FORMAT(convert_tz(packages.created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d')")
	db = db.Scan(v)

	return db.Error
}

func (m *AnalyticsManager) GetLogs(v interface{}, userID int64, limit int) error {
	db := m.db.Table("package_audit_logs")
	db = db.Select(`
		package_audit_logs.*,
		packages.order_number,
		package_codes.code,
		users.full_name as updated_user_name,
		users.role as updated_user_role,
		extra_fees.amount as extra_fee,
		extra_fees.status as extra_fee_status
	`)

	db = db.Joins("LEFT JOIN packages ON packages.id = package_audit_logs.package_id")
	db = db.Joins("LEFT JOIN package_codes ON package_codes.id=packages.package_code_id")
	db = db.Joins("LEFT JOIN users ON users.id = package_audit_logs.updated_user_id")
	db = db.Joins("LEFT JOIN extra_fees ON extra_fees.id = package_audit_logs.extra_fee_id")
	db = db.Order("package_audit_logs.id DESC")
	db = db.Where("packages.user_id=?", userID).Limit(limit)
	db = db.Scan(v)

	return db.Error
}
