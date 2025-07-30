package tiktokuploadlabelscheduler

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"tebexpressapi/pkg/order"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"
	"time"

	"go.uber.org/zap"
)

const (
	limit          = 1000
	numberRoutines = 100
)

type ShipmentTrackingHandler struct {
	Logger  *zap.SugaredLogger
	LocalS3 storage.S3

	PackageManager *sqlmanager.PackageManager
}

type Page struct {
	sync.Mutex
	index int
}

func NewHandler(l *zap.SugaredLogger, s3 storage.S3, pm *sqlmanager.PackageManager) *ShipmentTrackingHandler {
	return &ShipmentTrackingHandler{
		Logger:         l,
		LocalS3:        s3,
		PackageManager: pm,
	}
}

func (h *ShipmentTrackingHandler) Process() {
	err := h.uploadTiktokLabel()
	if err != nil {
		h.Logger.Errorf("Upload tiktok label error: %v", err)
	}
}

func (h *ShipmentTrackingHandler) uploadTiktokLabel() error {
	twelveHoursAgo := time.Now().Add(-12 * time.Hour).Format("2006-01-02 15:04:05")

	tiktokPkgs, err := h.PackageManager.GetPackages(sqlmanager.PackageQueryOption{
		HasTiktokLabel: true,
		StartDate:      twelveHoursAgo,
	})
	if err != nil {
		return err
	}

	for _, pkg := range tiktokPkgs {
		if strings.HasPrefix(pkg.Label, "https") {
			filePath, err := order.StoreLabelS3(
				h.LocalS3,
				*pkg.CustomTiktokBarcode,
				"pdf",
				fmt.Sprintf("tiktok_%s_%03d", pkg.OrderNumber, rand.Intn(1000)),
			)
			if err != nil {
				pkg.Label = *pkg.CustomTiktokBarcode
			} else {
				pkg.Label = filePath
			}

			if saveErr := h.PackageManager.UpdatePackage(&pkg, pkg.ID); saveErr != nil {
				return saveErr
			}
		}
	}

	return nil
}
