package packageutils

import (
	"context"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"

	"go.uber.org/zap"
)

type PackageRefund struct {
	Logger *zap.SugaredLogger

	PackageManager *sqlmanager.PackageManager
	BillManager    *sqlmanager.BillManager
}

func NewPackageRefund(l *zap.SugaredLogger, pm *sqlmanager.PackageManager, bm *sqlmanager.BillManager) *PackageRefund {
	return &PackageRefund{
		Logger: l,

		PackageManager: pm,
		BillManager:    bm,
	}
}

// Handle --
func (h *PackageRefund) Handle(c context.Context, packageId, userId int64, desc string) error {
	pkg, err := h.PackageManager.GetPackage(sqlmanager.PackageQueryOption{ID: packageId})
	if err != nil {
		h.Logger.Errorf("get package, %v", err)
		return err
	}

	// pkg có service code inus và us48 và au và eu sẽ không có refund
	if pkg.Service.Code == constant.ServiceINUSCode || pkg.Service.Code == constant.ServiceUS48Code || pkg.Service.Code == constant.ServiceAUCode || pkg.Service.Code == constant.ServiceEUCode {
		h.Logger.Infof("package service %v not supported refund: %v", pkg.ID, pkg.Service.Code)
		return nil
	}

	billID, err := h.BillManager.GetOrCreateNowBillID(pkg.UserID)
	if err != nil {
		h.Logger.Errorf("Get bill id error %v", err)
		return err
	}

	extrafee, err := h.PackageManager.GetTotalExtrafeeToRefund(packageId)
	if err != nil {
		h.Logger.Errorf("get package extra fee total, %v", err)
		return err
	}

	refunds, err := h.PackageManager.GetListPackagesRefundByPackageID(packageId, constant.PackageRefundPending)
	if err != nil {
		h.Logger.Errorf("get package refunds, %v", err)
		return err
	}

	var amountRefunds float64 = 0
	for _, v := range refunds {
		amountRefunds += v.Amount
	}

	amount := pkg.ShippingFee + extrafee - amountRefunds
	if amount <= 0 {
		return nil
	}

	extraFee := &entity.ExtraFee{
		PackageID:      utils.Int64(pkg.ID),
		BillID:         &billID,
		Amount:         -amount,
		Description:    desc,
		ExtraFeeTypeID: constant.ExtraFeeTypeRefund,
		Status:         constant.ExtraFeeStatusEnable,
	}

	err = h.BillManager.CreateExtraFee(extraFee, pkg.UserID, userId)
	if err != nil {
		h.Logger.Errorf("refund package, %v", err)
		return err
	}

	return nil
}
