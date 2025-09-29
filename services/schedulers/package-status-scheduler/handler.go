package packagestatusscheduler

import (
	"context"
	"fmt"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/order"
	packageutils "tebexpressapi/pkg/package_utils"
	"tebexpressapi/pkg/providers"
	"tebexpressapi/pkg/sqlmanager"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	limit                  = 200
	numberRoutines         = 100
	dayExpireIntransit     = 30
	dayExpireProcessing    = 365
	pendingPickUpMaxActive = 7
	dayCancelReturnExpire  = 14
)

type PackageStatusHandler struct {
	Ctx    context.Context
	Logger *zap.SugaredLogger

	ShipmentRefund *packageutils.PackageRefund

	PackageManager  *sqlmanager.PackageManager
	TrackingManager *sqlmanager.TrackingManager
	OrderManager    *sqlmanager.OrderManager
}

func NewHandler(l *zap.SugaredLogger, tm *sqlmanager.TrackingManager, pm *sqlmanager.PackageManager, om *sqlmanager.OrderManager, bm *sqlmanager.BillManager) *PackageStatusHandler {
	return &PackageStatusHandler{
		Ctx:    context.Background(),
		Logger: l,

		ShipmentRefund: packageutils.NewPackageRefund(l, pm, bm),

		TrackingManager: tm,
		PackageManager:  pm,
		OrderManager:    om,
	}
}

func (h *PackageStatusHandler) Process() {
	// nowHour := time.Now().Format("15")

	// timeDeactive := viper.GetString("package_status.deactive_package_code")
	// fmt.Println("=======timeDeactive", timeDeactive)
	// fmt.Println("=======nowHour", nowHour)
	// if nowHour == timeDeactive {
	h.deactivePackageCode()
	// }

	// timeUndelivered := viper.GetString("package_status.un_delivered")
	// if nowHour == timeUndelivered {
	h.Undelivered()
	// }

	// timeCancelReturnExpire := viper.GetString("package_status.cancel_return_expire")
	// if nowHour == timeCancelReturnExpire {
	h.CancelReturnExpirePackages()
	// }

	// timeUpdateOrderStatus := viper.GetString("package_status.update_order_status")
	// if nowHour == timeUpdateOrderStatus {
	h.UpdateOrderStatus()
	// }
}

// Deactivate mã package code (mã LB) khi đơn ở trạng thái pending pickup (kho chờ nhận hang)
// quá x ngày, hoàn lại tiền ngay cho khách khi đơn chưa dc tạo mã label bên carrier
func (h *PackageStatusHandler) deactivePackageCode() {
	// Undelivered  đơn ở trạng thái in-transit quá ngày
	day := viper.GetInt64("package_status.pending_pickup_max_active")
	if day < 1 {
		day = dayExpireIntransit
	}

	ids, err := h.PackageManager.DeactivatePackageCodes(day)
	if err != nil {
		h.Logger.Errorf("Query update deactive package code error: %v", err)
		return
	}

	for _, id := range ids {
		pkg, err := h.PackageManager.GetPackageByPackageID(id)
		if err != nil {
			h.Logger.Errorf("get package: %v", err)
			continue
		}

		if !pkg.LabelPromotion {
			h.Refund(pkg)
		}

		h.CancelTracking(pkg)
	}
}

// Chuyển trạng thái đơn hàng ở in transit quá x ngày sang trạng thái Undelivered
// chuyển trạng thái đơn hàng ở processing quá x ngày sang cancel
func (h *PackageStatusHandler) Undelivered() {
	// Undelivered  đơn ở trạng thái in-transit quá ngày
	day := viper.GetInt("package_status.day_expire_intransit")
	if day < 1 {
		day = dayExpireIntransit
	}

	if err := h.PackageManager.UndeliveredPackageIntransitExpire(day); err != nil {
		h.Logger.Errorf("update status: %v", err)
		return
	}

	// Cancel  đơn ở trạng thái processing quá ngày
	dayProcessing := viper.GetInt("package.day_expire_processing")
	if dayProcessing < 1 {
		dayProcessing = dayExpireProcessing
	}

	h.cancelPackageProcessExpire(dayProcessing)
}

func (h *PackageStatusHandler) cancelPackageProcessExpire(day int) {
	ids, err := h.PackageManager.CancelPackageProcessingExpire(day, 200)
	if err != nil {
		h.Logger.Errorf("update status: %v", err)
		return
	}

	if len(ids) < 1 {
		return
	}

	for _, id := range ids {
		pkg, err := h.PackageManager.GetPackageByPackageID(id)
		if err != nil {
			h.Logger.Errorf("get package: %v", err)
			continue
		}

		h.CancelTracking(pkg)
	}

	if len(ids) >= 200 {
		h.cancelPackageProcessExpire(day)
	}
}

// Cancel tracking bên hệ thống và bên carrier
func (h *PackageStatusHandler) CancelTracking(pkg *entity.Package) {
	if pkg.Tracking == nil || pkg.Tracking.TrackingNumber == "" || pkg.Tracking.CarrierID < 1 {
		return
	}

	tk := pkg.Tracking
	tk.Status = constant.TrackingStatusCanceled
	if err := h.TrackingManager.Update(tk); err != nil {
		h.Logger.Errorf("Get carrier %v", err)
		return
	}

	carrier, err := h.TrackingManager.GetCarrierByID(pkg.Tracking.CarrierID)
	if err != nil {
		h.Logger.Errorf("Get carrier %v", err)
		return
	}

	provider := providers.NewCarrier(carrier.Code, pkg.UserID)
	if _, err := order.CancelLabel(provider, pkg); err != nil {
		h.Logger.Errorf("cancel label %v", err)
	}
}

func (h *PackageStatusHandler) Refund(pkg *entity.Package) {
	if pkg.PackageCode == nil {
		return
	}

	err := h.ShipmentRefund.Handle(h.Ctx, pkg.ID, pkg.UserID, fmt.Sprintf("Hoàn tiền cho đơn %v", pkg.PackageCode.Code))
	if err != nil {
		h.Logger.Errorf("json marshal %v", err)
		return
	}
}

// Cancel đơn hàng ở trang thái expire qua x ngày
func (h *PackageStatusHandler) CancelReturnExpirePackages() {
	day := viper.GetInt("package_status.day_cancel_return_expire")
	if day < 1 {
		day = dayCancelReturnExpire
	}

	packages, err := h.PackageManager.FetchPackagesReturnExpire(day, 0)
	if err != nil {
		h.Logger.Errorf("update status: %v", err)
		return
	}

	var userID int64 = 0
	var user *entity.User = nil
	ids := []int64{}
	codes := []string{}

	for _, pkg := range packages {
		if pkg.PackageCode == nil || pkg.User == nil {
			continue
		}

		if userID > 0 && userID != pkg.UserID {
			h.CancelReturnExpire(ids, codes, user)

			ids = []int64{}
			codes = []string{}
		}

		userID = pkg.UserID
		user = pkg.User
		ids = append(ids, pkg.ID)
		codes = append(codes, pkg.PackageCode.Code)
	}

	if len(ids) > 0 {
		h.CancelReturnExpire(ids, codes, user)
	}
}

func (h *PackageStatusHandler) CancelReturnExpire(ids []int64, codes []string, user *entity.User) {
	if err := h.PackageManager.CancelPackagesReturnExpire(ids); err != nil {
		h.Logger.Errorf("update status: %v", err)
		return
	}
}

func (h *PackageStatusHandler) UpdateOrderStatus() {
	orders, err := h.OrderManager.GetList(sqlmanager.OrderQueryOption{
		ArrStatus: []int{constant.OrderStatusInTransit, constant.OrderStatusProcess},
	})
	if err != nil {
		h.Logger.Errorf("get orders %v", err)
		return
	}

	for _, order := range orders {
		if order.Status == constant.OrderStatusProcess {
			count, err := h.OrderManager.GetCountPackageItems(sqlmanager.OrderPackageQueryOption{
				OrderID:   order.ID,
				ArrStatus: []int{constant.PackageStatusPendingPickup},
			})

			if err != nil {
				h.Logger.Errorf("get orders %v", err)
				continue
			}

			if count > 0 {
				continue
			}

			order.Status = constant.OrderStatusDelivered
			if err := h.OrderManager.Update(&order); err != nil {
				h.Logger.Errorf("get orders %v", err)
				continue
			}

			continue
		}

		count, err := h.OrderManager.GetCountPackageItems(sqlmanager.OrderPackageQueryOption{
			OrderID:   order.ID,
			ArrStatus: []int{constant.PackageStatusPendingPickup},
		})

		if err != nil {
			h.Logger.Errorf("get orders %v", err)
			continue
		}

		if count < 1 {
			order.Status = constant.OrderStatusDelivered
		} else if count == order.Count {
			order.Status = constant.OrderStatusInTransit
		} else {
			order.Status = constant.OrderStatusProcess
		}

		if err := h.OrderManager.Update(&order); err != nil {
			h.Logger.Errorf("get orders %v", err)
		}
	}
}
