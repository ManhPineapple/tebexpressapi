package ocrtiktoklabelscheduler

import (
	"context"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type OcrLabelHandler struct {
	ctx    context.Context
	logger *zap.SugaredLogger
	redis  *redis.Client

	packageManager   *sqlmanager.PackageManager
	trackingManager  *sqlmanager.TrackingManager
	warehouseManager *sqlmanager.WareHouseManager
}

func NewHandler(
	ctx context.Context,
	logger *zap.SugaredLogger,
	redis *redis.Client,
	packageManager *sqlmanager.PackageManager,
	trackingManager *sqlmanager.TrackingManager,
	warehouseManager *sqlmanager.WareHouseManager,
) *OcrLabelHandler {
	return &OcrLabelHandler{
		ctx:              ctx,
		logger:           logger,
		redis:            redis,
		packageManager:   packageManager,
		trackingManager:  trackingManager,
		warehouseManager: warehouseManager,
	}
}

func (h *OcrLabelHandler) Process() {
	start := time.Now().AddDate(0, 0, -14).Format(time.RFC3339)
	tiktokPkgs, err := h.packageManager.GetPackages(sqlmanager.PackageQueryOption{
		HasTiktokLabel:  true,
		NeedToOcr:       true,
		IgnoreStatusArr: []int64{constant.PackageStatusCancelled, constant.PackageStatusArchived},
		StartDate:       start,
	})
	if err != nil {
		h.logger.Errorf("Failed to get packages for OCR: %v", err)
		return
	}

	for _, pkg := range tiktokPkgs {
		if pkg.CustomTiktokBarcode == nil {
			continue
		}

		trackingNumber, mapRecipientChange, err := utils.GetNslogOcrOutput(*pkg.CustomTiktokBarcode)
		if err != nil {
			h.logger.Errorf("Failed to OCR for package %s: %v", pkg.ID, err)

			err := h.packageManager.SaveUpdatePackage2(
				pkg.ID,
				pkg.UserID,
				map[string]interface{}{"recipient": "N/A"},
				[]entity.PackageAuditLog{},
				0,
				[]entity.ExtraFee{},
				pkg.Status,
			)
			if err != nil {
				h.logger.Errorf("Failed to mark OCR-failed package %s as N/A: %v", pkg.ID, err)
			}

			continue
		}

		err = h.packageManager.SaveUpdatePackage2(
			pkg.ID,
			pkg.UserID,
			mapRecipientChange,
			[]entity.PackageAuditLog{},
			0,
			[]entity.ExtraFee{},
			pkg.Status,
		)
		if err != nil {
			h.logger.Errorf("Failed to update package %s: %v", pkg.ID, err)
			continue
		}

		texasWarehouse, err := h.warehouseManager.GetWareHouse(sqlmanager.OptionWareHouse{
			Status: 1,
			State:  "TX",
		})
		if err != nil {
			h.logger.Errorf("Error get Warehouse: %v", err)
			continue
		}

		trackings := []entity.Tracking{{
			PackageID:      pkg.ID,
			TrackingNumber: trackingNumber,
			LabelURL:       pkg.Label,
			CarrierID:      1, //hard-coded
			Status:         constant.TrackingStatusSuccess,
			Weight:         pkg.Weight,
			Width:          pkg.Width,
			Length:         pkg.Length,
			Height:         pkg.Height,
			ShipmentCost:   pkg.ShippingFee,
			HubID:          &texasWarehouse.ID,
			UserID:         pkg.UserID,
			CarrierService: "FirstClass",
		}}

		err = h.trackingManager.CreateTrackingTiktok(trackings)
		if err != nil {
			h.logger.Errorf("Error create tiktok tracking: %v", err)
			continue
		}

		h.logger.Infof("OCR success: pkg %s → tracking %s", pkg.ID, trackingNumber)
	}
}
