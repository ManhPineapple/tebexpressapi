package packageutils

import (
	"tebexpressapi/pkg/order"
	"tebexpressapi/pkg/providers"
	"tebexpressapi/pkg/sqlmanager"

	"go.uber.org/zap"
)

type ShipmentCancelCarrier struct {
	Logger *zap.SugaredLogger

	PackageManager  *sqlmanager.PackageManager
	TrackingManager *sqlmanager.TrackingManager
}

func NewShipmentCancelCarrier(l *zap.SugaredLogger, pm *sqlmanager.PackageManager, tm *sqlmanager.TrackingManager) *ShipmentCancelCarrier {
	return &ShipmentCancelCarrier{
		Logger:          l,
		PackageManager:  pm,
		TrackingManager: tm,
	}
}

// Handle --
func (h *ShipmentCancelCarrier) Handle(trackingNumber string) error {
	tracking, err := h.TrackingManager.GetTracking(sqlmanager.TrackingOption{TrackingNumber: trackingNumber})
	if err != nil {
		h.Logger.Errorf("get package: %v", err)
		return err
	}

	pkg, err := h.PackageManager.GetPackageByPackageID(tracking.PackageID)
	if err != nil {
		h.Logger.Errorf("get package: %v", err)
		return err
	}

	if tracking.CarrierID < 0 {
		return nil
	}

	carrier, err := h.TrackingManager.GetCarrierByID(tracking.CarrierID)
	if err != nil {
		h.Logger.Errorf("Get carrier %v", err)
		return err
	}

	provider := providers.NewCarrier(carrier.Code, pkg.UserID)
	pkg.Tracking = tracking
	if _, err := order.CancelLabel(provider, pkg); err != nil {
		h.Logger.Errorf("cancel label %v", err)
		return err
	}

	return nil
}
