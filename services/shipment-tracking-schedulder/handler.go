package shipmenttrackingschedulder

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/providers"
	"tebexpressapi/pkg/sqlmanager"
	"time"

	"tebexpressapi/pkg/constant"

	"github.com/spf13/cast"
	"go.uber.org/zap"
)

const (
	limit          = 1000
	numberRoutines = 100
)

type ShipmentTrackingHandler struct {
	Logger *zap.SugaredLogger

	Page Page

	ContainerManager *sqlmanager.ContainerManager
	ShipmentManager  *sqlmanager.ShipmentManager
	TrackingManager  *sqlmanager.TrackingManager
	PackageManager   *sqlmanager.PackageManager
	UserManager      *sqlmanager.UserManager
	BillManager      *sqlmanager.BillManager
}

type Page struct {
	sync.Mutex
	index int
}

func NewHandler(l *zap.SugaredLogger, cm *sqlmanager.ContainerManager, sm *sqlmanager.ShipmentManager, tm *sqlmanager.TrackingManager, pm *sqlmanager.PackageManager, um *sqlmanager.UserManager, bm *sqlmanager.BillManager) *ShipmentTrackingHandler {
	return &ShipmentTrackingHandler{
		Logger: l,

		ContainerManager: cm,
		ShipmentManager:  sm,
		TrackingManager:  tm,
		PackageManager:   pm,
		UserManager:      um,
		BillManager:      bm,
	}
}

func (h *ShipmentTrackingHandler) Process() {
	err2 := h.trackingUSPS()
	if err2 != nil {
		h.Logger.Errorf("Tracking usps error: %v", err2)
	}
}

func (h *ShipmentTrackingHandler) trackingUSPS() error {
	h.Logger.Info("\n -------------------------- Start cron job tracking usps ------------------------ \n")
	count, err := h.TrackingManager.CountTrackingsIntransit(sqlmanager.TrackingOption{})
	if err != nil {
		return err
	}

	h.Logger.Info("count: ", count)
	if count < 1 {
		return nil
	}

	var core = cast.ToInt(numberRoutines)
	loop := cast.ToInt(math.Ceil(float64(count) / float64(limit)))
	if core > loop {
		core = loop
	}

	var wg sync.WaitGroup
	wg.Add(loop)

	h.Page.index = 0
	for j := 0; j < core; j++ {
		go h.checkTrackingInfo(&wg, loop)
	}

	wg.Wait()
	return nil
}

func (h *ShipmentTrackingHandler) checkTrackingInfo(wg *sync.WaitGroup, loop int) {
	defer wg.Done()
	h.Page.Lock()
	defer h.Page.Unlock()

	h.Page.index++

	offset := (h.Page.index - 1) * limit
	curPage := h.Page.index

	trackings, err := h.TrackingManager.GetTrackingsIntransit(sqlmanager.TrackingOption{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		h.Logger.Errorf("fetch shipping containers: %v", err)
		return
	}
	var carrier providers.Carrier
	for _, tracking := range trackings {
		if tracking.Package == nil || tracking.Carrier == nil {
			continue
		}

		if tracking.Carrier.Code != providers.CarrierTypeKiloship {
			continue
		}

		h.Logger.Info("tracking: ", tracking.ID, tracking.Carrier.Code)
		carrier = providers.NewCarrier(tracking.Carrier.Code, tracking.Package.UserID)
		if carrier == nil {
			continue
		}

		trackResult, err := carrier.TrackInfo(tracking.TrackingNumber)
		if err != nil {
			h.Logger.Errorf("get detail track error %v, tracking %v:", err, tracking.TrackingNumber)
			continue
		}

		if len(trackResult) < 1 {
			h.Logger.Warn("Tracking Information Not Found")
			continue
		}
		var isReturnPackage bool = false
		var logs []entity.PackageDeliverLog
		pkg := tracking.Package

		trackingTime := pkg.TrackingTime
		if trackingTime == nil {
			trackingTime = &pkg.CreatedAt
			pkg.TrackingTime = &pkg.CreatedAt
		}
		count := len(trackResult)
		if tracking.Carrier.Code == providers.CarrierTypeIBBlue {

			for i := count - 1; i >= 0; i-- {
				result := trackResult[i]
				if !result.Datetime.IsZero() {
					result.Datetime = result.Datetime.Add(8 * time.Hour)
				}
				if result.Datetime.After(*pkg.TrackingTime) && !result.Datetime.IsZero() {
					logStatus, logType := h.ConvertDataResponse(tracking.Carrier.Code, result.Status)
					location := ""
					if result.State != "" {
						location = result.State
					}
					if result.City != "" {
						location = fmt.Sprintf("%s,%s", location, result.City)
					}

					if result.Country != "" {
						location = fmt.Sprintf("%s,%s", location, result.Country)
					}

					txtStatus := strings.TrimSpace(result.Status)
					if strings.ToLower(txtStatus) == "received data" {
						location = fmt.Sprintf("%s, %s", tracking.Package.Warehouse.City, tracking.Package.Warehouse.Country)
					}

					log := entity.PackageDeliverLog{
						PackageID:   tracking.PackageID,
						Location:    location,
						Description: txtStatus,
						Status:      logStatus,
						Type:        logType,
					}
					log.CreatedAt = result.Datetime
					log.UpdatedAt = result.Datetime
					logs = append(logs, log)
					if result.Datetime.After(*trackingTime) {
						trackingTime = &result.Datetime
					}
					if logStatus == constant.DeliverLogTebexpressDelivered {
						pkg.Status = constant.PackageStatusDelivered
						pkg.DeliveredAt = &result.Datetime
					}
					if strings.Contains(result.Status, "Return") {
						pkg.Alert = constant.PackageAlertTypeHubReturn
						isReturnPackage = true
					}
				}
			}
		} else if tracking.Carrier.Code == providers.CarrierTypeAuspost {
			for _, result := range trackResult {
				if result.Datetime.After(*pkg.TrackingTime) {
					logStatus := constant.DeliverLogTebexpressInTransit
					logType := constant.PackageDeliverLogTypeInTransit
					location := result.Location
					if pkg.CountryCode == "AU" && location == "" {
						location = "AU"
					}

					description := result.Description
					if strings.ToLower(description) == "received data" {
						location = fmt.Sprintf("%s, %s", tracking.Package.Warehouse.City, tracking.Package.Warehouse.Country)
					}

					if strings.Contains(description, "Delivered") {
						logStatus = constant.DeliverLogTebexpressDelivered
						logType = constant.PackageDeliverLogTypeDelivered

						pkg.Status = constant.PackageStatusDelivered
						pkg.DeliveredAt = &result.Datetime
					}

					if strings.Contains(description, "Returned") && pkg.ReturnedAt == nil {
						pkg.Alert = constant.PackageAlertTypeHubReturn
						pkg.AlertAt = &result.Datetime
						pkg.ReturnedAt = &result.Datetime
						isReturnPackage = true
					}

					log := entity.PackageDeliverLog{
						PackageID:   tracking.PackageID,
						Location:    location,
						Description: description,
						Status:      logStatus,
						Type:        logType,
					}

					log.CreatedAt = result.Datetime
					log.UpdatedAt = result.Datetime
					logs = append(logs, log)

					if result.Datetime.After(*trackingTime) {
						trackingTime = &result.Datetime
					}
				}
			}
		} else {
			for _, result := range trackResult {
				if result.Datetime.After(*pkg.TrackingTime) {
					logStatus, logType := h.ConvertDataResponse(tracking.Carrier.Code, result.Status)

					location := ""
					if result.Location != "" {
						location = result.Location
					} else {
						location = fmt.Sprintf("%s, %s, %s", result.State, result.City, result.Country)
					}
					txtStatus := strings.TrimSpace(result.Description)
					if strings.ToLower(txtStatus) == "received data" {
						location = fmt.Sprintf("%s, %s", tracking.Package.Warehouse.City, tracking.Package.Warehouse.Country)
					}

					if txtStatus == "" {
						txtStatus = "In-Transit"
					}

					log := entity.PackageDeliverLog{
						PackageID:   tracking.PackageID,
						Location:    location,
						Description: txtStatus,
						Status:      logStatus,
						Type:        logType,
					}
					log.CreatedAt = result.Datetime
					log.UpdatedAt = result.Datetime
					logs = append(logs, log)
					if result.Datetime.After(*trackingTime) {
						trackingTime = &result.Datetime
					}
					if logStatus == constant.DeliverLogTebexpressDelivered {
						pkg.Status = constant.PackageStatusDelivered
						pkg.DeliveredAt = &result.Datetime
					}

					if strings.ToUpper(result.Status) == "DELIVERED" {
						pkg.Status = constant.PackageStatusDelivered
						pkg.DeliveredAt = &result.Datetime
					}
				}
			}
		}
		pkg.TrackingTime = trackingTime
		if len(logs) < 1 {
			continue
		}

		var billID int64
		if isReturnPackage && pkg.CountryCode == "AU" {
			billID, err = h.BillManager.GetOrCreateNowBillID(pkg.UserID)
			if err != nil {
				h.Logger.Errorf("Get bill id error %v", err)
				continue
			}
		}

		err = h.PackageManager.SaveDeliverLogPackage(logs, pkg, isReturnPackage, billID)
		if err != nil {
			h.Logger.Errorf("get detail track ups: %v", err)
			continue
		}
	}

	fmt.Printf("Page %v: Done\n", curPage)
	if h.Page.index >= loop {
		return
	}

}

func (h *ShipmentTrackingHandler) ConvertDataResponse(carrier_code, status string) (string, int) {
	switch carrier_code {
	case providers.CarrierTypeIBBlue:
		if strings.Contains(status, "Delivered") {
			return constant.DeliverLogTebexpressDelivered, constant.PackageDeliverLogTypeDelivered
		}

		return constant.DeliverLogTebexpressInTransit, constant.PackageDeliverLogTypeInTransit
	case providers.CarrierTypeStamps:
		if strings.Contains(status, "DELIVERED") {
			return constant.DeliverLogTebexpressDelivered, constant.PackageDeliverLogTypeDelivered
		}
		return constant.DeliverLogTebexpressInTransit, constant.PackageDeliverLogTypeInTransit
	case providers.CarrierTypeKiloship:
		if strings.Contains(status, "DELIVERED") {
			return constant.DeliverLogTebexpressDelivered, constant.PackageDeliverLogTypeDelivered
		}
		return constant.DeliverLogTebexpressInTransit, constant.PackageDeliverLogTypeInTransit
	default:
		return constant.DeliverLogTebexpressInTransit, constant.PackageDeliverLogTypeInTransit
	}
}
