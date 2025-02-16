package packagerefundscheduler

import (
	"tebexpressapi/pkg/sqlmanager"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	limit                  = 200
	numberRoutines         = 100
	dayExpireIntransit     = 30
	dayExpireProcessing    = 365
	dayCancelReturnExpire  = 14
	dayRefundExpirePending = 1
)

type PackageStatusHandler struct {
	Logger *zap.SugaredLogger

	BillManager    *sqlmanager.BillManager
	PackageManager *sqlmanager.PackageManager
	UserManager    *sqlmanager.UserManager
}

func NewHandler(l *zap.SugaredLogger, bm *sqlmanager.BillManager, pm *sqlmanager.PackageManager, um *sqlmanager.UserManager) *PackageStatusHandler {
	return &PackageStatusHandler{
		Logger: l,

		BillManager:    bm,
		PackageManager: pm,
		UserManager:    um,
	}
}

func (h *PackageStatusHandler) Process() {
	h.Logger.Info("process refund")
	day := dayRefundExpirePending
	if viper.GetInt("package_refund.day_expire_pending") > 0 {
		day = viper.GetInt("package_refund.day_expire_pending")
	}

	items, err := h.PackageManager.GetPackagesRefundExpiredPending(day)
	if err != nil {
		h.Logger.Errorf("get package, %v", err)
		return
	}

	h.Logger.Info("items: ", len(items))
	for _, v := range items {
		h.Logger.Info("PackageID: ", v.Package.ID)
		if v.Package == nil {
			continue
		}

		userInfo, err := h.UserManager.GetUserInfoByUserID(v.Package.UserID)
		if err != nil {
			h.Logger.Errorf("get user info, %v", err)
			continue
		}

		if float64(time.Since(v.CreatedAt).Hours()) < float64(24*userInfo.RefundDay) {
			continue
		}

		billID, err := h.BillManager.GetOrCreateNowBillID(v.Package.UserID)
		if err != nil {
			h.Logger.Errorf("Get bill id error %v", err)
			return
		}

		err = h.BillManager.PackageRefund(v.Package.UserID, v.ID, v.PackageID, billID, v.Amount)
		if err != nil {
			h.Logger.Errorf("refund package, %v", err)
			return
		}
	}
}
