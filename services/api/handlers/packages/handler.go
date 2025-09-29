package packages

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"net/http"
	"sync"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/createlabel"
	pkg_dto "tebexpressapi/pkg/dto"
	"tebexpressapi/pkg/models/dto"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/order"
	packageutils "tebexpressapi/pkg/package_utils"
	"tebexpressapi/pkg/providers"
	"tebexpressapi/pkg/rabbitmq"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"
	"tebexpressapi/pkg/utils"
	"time"

	"github.com/disintegration/imaging"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type PackageHandler struct {
	Logger    *zap.SugaredLogger
	Redis     *redis.Client
	StorageS3 storage.S3

	CalculatePrice *calculate.CalculatePrice
	CreateLabel    *createlabel.CreateLabel
	Producer       *rabbitmq.Producer

	ShipmentEstimateCost       *packageutils.EstimateCost
	ShipmentRefund             *packageutils.PackageRefund
	ShipmentRefundCarrier      *packageutils.ShipmentCancelCarrier
	ShipmentCreateLabelHandler *packageutils.CreateLabelHandler

	UserManager      *sqlmanager.UserManager
	ServiceManager   *sqlmanager.ServiceManager
	PackageManager   *sqlmanager.PackageManager
	BillManager      *sqlmanager.BillManager
	StateManager     *sqlmanager.StateManager
	ProductManager   *sqlmanager.ProductManager
	WareHouseManager *sqlmanager.WareHouseManager
	SettingManager   *sqlmanager.SettingManager
	PromotionManager *sqlmanager.PromotionManager
	TrackingManager  *sqlmanager.TrackingManager
}

type CheckAddressForm struct {
	Company     string `json:"company"`
	Address1    string `json:"address_1"`
	Address2    string `json:"address_2"`
	City        string `json:"city"`
	StateCode   string `json:"state_code"`
	ZipCode     string `json:"zipcode"`
	CountryCode string `json:"country_code"`
}

type CheckAddressResponse struct {
	Error   string `json:"error"`
	Message string `json:"messages,omitempty"`
}

type ListPackageResponse struct {
	Packages []dto.PackageCustomer `json:"packages"`
}

type CreateResponse struct {
	Package *order.PackageResource `json:"package"`
}

type UpdateResponse struct {
	Package *order.PackageResource `json:"package"`
}

type OrderDetailCustomer struct {
	ID                int64                   `json:"id,omitempty"`
	Code              string                  `json:"code"`
	Sku               string                  `json:"order_number"`
	Recipient         string                  `json:"recipient"`
	Company           string                  `json:"company"`
	PhoneNumber       string                  `json:"phone_number"`
	Address1          string                  `json:"address_1"`
	Address2          string                  `json:"address_2"`
	City              string                  `json:"city"`
	StateCode         string                  `json:"state_code"`
	Zipcode           string                  `json:"zipcode"`
	CountryCode       string                  `json:"country_code"`
	Detail            string                  `json:"detail"`
	Weight            float64                 `json:"weight"`
	Width             float64                 `json:"width"`
	Length            float64                 `json:"length"`
	Height            float64                 `json:"height"`
	Status            string                  `json:"status"`
	UserID            int64                   `json:"user_id"`
	UserFullName      string                  `json:"user_full_name"`
	UserEmail         string                  `json:"user_email"`
	UserPhoneNumber   string                  `json:"user_phone_number"`
	ShippingFee       float64                 `json:"shipping_fee,omitempty"`
	BillCode          string                  `json:"bill_code"`
	ServiceCode       string                  `json:"service_code"`
	CustomCNBarcode   string                  `json:"custom_cn_barcode"`
	TotalCost         float64                 `json:"total_cost,omitempty"`
	OrderID           int64                   `json:"order_id"`
	IncludeBattery    bool                    `json:"include_battery"`
	CreatedAt         time.Time               `json:"created_at"`
	UpdatedAt         time.Time               `json:"updated_at"`
	ExtraFees         []entity.ExtraFeeCustom `json:"extra_fees"`
	Tracking          entity.TrackingCustom   `json:"tracking"`
	PackageName       string                  `json:"package_name"`
	PackageQuantity   int64                   `json:"package_quantity"`
	TotalProductPrice float64                 `json:"product_price"`
}

type PackageDetailResponse struct {
	Package *OrderDetailCustomer `json:"package"`
}

type DeliverPackageResponse struct {
	Success         bool   `json:"success"`
	PackageCode     string `json:"package_code"`
	BillCode        string `json:"bill_code"`
	Base64Label     string `json:"base64_label"`
	LastMileCarrier string `json:"last_mile_carrier"`
	TrackingNumber  string `json:"tracking_number"`
}

type CancelResponse struct {
	Sucess bool `json:"sucess"`
}

type CancelForm struct {
	IDs []int64 `json:"ids"`
}

type TrackResponse []pkg_dto.TrackDeDeliverDTO

var MapStatusDescription = map[int]string{
	constant.PackageStatusCreated:              "Đơn hàng được tạo mới",
	constant.PackageStatusPendingPickup:        "Đơn hàng đang chờ lấy",
	constant.PackageStatusPicked:               "Đơn hàng được xác nhận đã giao cho nhân viên kho",
	constant.PackageStatusWareHouseLabeled:     "Đơn hàng đang được xử lý",
	constant.PackageStatusWareHouseInContainer: "Đơn hàng đang được xử lý",
	constant.PackageStatusWareHouseInShipment:  "Đơn hàng đang được xử lý",
	constant.PackageStatusWareHouseExport:      "Đơn hàng đã xuất kho",
	constant.PackageStatusDelivered:            "Đơn hàng đã được giao thành công",
	constant.PackageStatusInTransit:            "Đơn hàng đang giao",
	constant.PackageStatusCancelled:            "Đơn hàng đã bị hủy",
	constant.PackageStatusReturned:             "Đơn hàng đã bị hoàn trả",
}

func (h *PackageHandler) EstimateCost(c context.Context, carrier providers.Carrier, pkg *entity.Package) (float64, error) {
	if carrier == nil {
		if pkg.CountryCode == "AU" {
			carrier = providers.NewCarrier(pkg.Service.DomesticCarrier.Code, pkg.UserID)
		} else {
			carrierCode, err := h.CreateLabel.GetCarrierCode(c, *pkg, pkg.UserID, "")
			if err != nil {
				return 0, err
			}

			if carrierCode != "" {
				carrier = providers.NewCarrier(carrierCode, pkg.UserID)
			} else {
				carrier = providers.NewCarrier(pkg.Service.DomesticCarrier.Code, pkg.UserID)
			}
		}
	}

	isWl := utils.IsStateWhiteList(pkg.UserID)
	igState := ""
	if isWl {
		igState = "CA"
	}

	wareHouses, err := h.WareHouseManager.GetWareHouses(sqlmanager.OptionWareHouse{
		Type:        constant.WareHouseTypeInternational,
		IgnoreState: igState,
		Status:      constant.WareHouseStatusActive,
	})

	if err != nil {
		h.Logger.Errorf("Get estimate warehouses error, %v", err)
		return 0, err
	}

	clone := &entity.Package{}
	if err := utils.DeepCopy(pkg, clone); err != nil {
		h.Logger.Errorf("copy deep, %v", err)
		return 0, err
	}

	var pkgWhs []entity.PackageWarehouseCost
	var wg sync.WaitGroup
	var m sync.Mutex
	apiCost := make(map[int64]float64, 0)
	for _, wareHouse := range wareHouses {
		wg.Add(1)

		go func(wareHouse entity.Warehouse) {
			defer wg.Done()

			if wareHouse.Country != pkg.CountryCode {
				return
			}

			cost := entity.PackageWarehouseCost{
				HubID:     wareHouse.ID,
				Warehouse: &wareHouse,
				OrgCost:   0,
			}

			res, msg, err := order.EstimateCost(carrier, clone, wareHouse)
			if msg != "" {
				h.Logger.Errorf("Estimate cost org err: %v", msg)
				return
			}

			if err != nil {
				h.Logger.Errorf("Estimate cost org err: %v", err)
				return
			}

			cost.OrgCost = res.TotalCost + wareHouse.HandlingFee
			cost.Zone = res.Zone

			if wareHouse.Status == constant.WareHouseStatusActive {
				m.Lock()
				pkgWhs = append(pkgWhs, cost)
				apiCost[wareHouse.ID] = res.TotalCost
				m.Unlock()
			}

			err = h.WareHouseManager.CreateEstimateCost(&cost)
			if err != nil {
				h.Logger.Errorf("Create estimate cost err: %v", err)
			}
		}(wareHouse)

	}

	wg.Wait()

	if len(pkgWhs) == 0 {
		return 0, errors.New("can't find lowest cost warehouse")
	}

	minW := pkgWhs[0]

	for _, wh := range pkgWhs {
		if wh.OrgCost < minW.OrgCost {
			minW = wh
		}
	}
	return apiCost[minW.HubID], nil
}

func (h *PackageHandler) EstimateCostWithPromotionGuest(c context.Context, pkg *entity.Package, carrier providers.Carrier, dbcarrier *entity.Carrier) (*entity.Warehouse, int, error) {
	pkgWhs, err := h.WareHouseManager.GetEstimateCosts(pkg.ID, pkg.CountryCode)
	if err != nil {
		return nil, 0, err
	}
	var msg []string

	if len(pkgWhs) == 0 {
		wareHouses, err := h.WareHouseManager.GetWareHouses(sqlmanager.OptionWareHouse{
			Type:   constant.WareHouseTypeInternational,
			Status: constant.WareHouseStatusActive,
		})

		if err != nil {
			h.Logger.Errorf("Get estimate warehouses error, %v", err)
			return nil, 0, err
		}

		err = h.WareHouseManager.DeactivateOldCost(*pkg)
		if err != nil {
			h.Logger.Errorf("Deactivate cost err: %v", err)
			return nil, 0, err
		}

		var wg sync.WaitGroup
		var m sync.Mutex
		for _, wareHouse := range wareHouses {

			wg.Add(1)
			h.Logger.Info("wareHouse: ", wareHouse.Name, wareHouse.Country, pkg.CountryCode)
			go func(wareHouse entity.Warehouse) {
				defer wg.Done()

				if !utils.ContainsString(constant.EUCountries, pkg.CountryCode) && wareHouse.Country != pkg.CountryCode {
					return
				}

				if pkg.IsPackageExceed {
					cost := entity.PackageWarehouseCost{
						PackageID: pkg.ID,
						HubID:     wareHouse.ID,
						Warehouse: &wareHouse,
						OrgCost:   0,
					}

					resultOrg, s, err := order.EstimateCost(carrier, pkg, wareHouse)
					if s != "" {
						h.Logger.Errorf("Estimate cost org err: %v", s)
					} else if err != nil {
						h.Logger.Errorf("Estimate cost org err: %v", err)
					} else {
						cost.OrgCost = resultOrg.TotalCost + wareHouse.HandlingFee
						cost.Zone = resultOrg.Zone
						cost.Cost = cost.OrgCost
						if dbcarrier != nil {
							cost.CarrierID = dbcarrier.ID
						}

						if wareHouse.Status == constant.WareHouseStatusActive {
							m.Lock()
							pkgWhs = append(pkgWhs, cost)
							m.Unlock()
						}

						err = h.WareHouseManager.CreateEstimateCost(&cost)
						if err != nil {
							h.Logger.Errorf("Create estimate cost err: %v", err)
						}
					}
				} else {
					cost := &entity.PackageWarehouseCost{
						PackageID: pkg.ID,
						HubID:     wareHouse.ID,
						Warehouse: &wareHouse,
						Cost:      0,
						OrgCost:   0,
					}

					resultOrg, s, err := order.EstimateCost(carrier, pkg, wareHouse)
					if s != "" {
						h.Logger.Errorf("Estimate cost org err: %v", s, err)
						if pkg.CountryCode == "AU" {
							return
						}
					} else if err != nil {
						h.Logger.Errorf("Estimate cost org err: %v", err)
						if pkg.CountryCode == "AU" {
							return
						}
					} else {
						cost.OrgCost = resultOrg.TotalCost + wareHouse.HandlingFee
						cost.Zone = resultOrg.Zone
						cost.Cost = cost.OrgCost
					}

					if pkg.CountryCode == "US" {
						clone := &entity.Package{}
						if err := utils.DeepCopy(pkg, clone); err != nil {
							h.Logger.Errorf("clone deeo package, %v", err)
							return
						}

						weight, length, height, width, err := h.CreateLabel.Fake(c, clone.Weight, clone.Length, clone.Height, clone.Width)
						if err != nil {
							h.Logger.Errorf("fake volume: %v", err)
							return
						}

						clone.ActualWeight = weight
						clone.ActualLength = length
						clone.ActualHeight = height
						clone.ActualWidth = width

						result, s, err := order.EstimateCost(carrier, clone, wareHouse)
						if s != "" {
							msg = append(msg, s)
							h.Logger.Errorf("Estimate cost err: %v", s)
							return
						}
						if err != nil {
							h.Logger.Errorf("Estimate cost err: %v", err)
							return
						}

						cost.Cost = result.TotalCost + wareHouse.HandlingFee
						cost.Zone = result.Zone
					}

					if dbcarrier != nil {
						cost.CarrierID = dbcarrier.ID
					}

					err = h.WareHouseManager.CreateEstimateCost(cost)
					if err != nil {
						h.Logger.Errorf("Create estimate cost err: %v", err)
						return
					}

					if wareHouse.Status == constant.WareHouseStatusActive {
						m.Lock()
						pkgWhs = append(pkgWhs, *cost)
						m.Unlock()
					}
				}

			}(wareHouse)

		}

		wg.Wait()
	}

	if len(pkgWhs) == 0 {
		message := "can't estimate cost"
		if len(msg) > 0 {
			message = msg[0]
		}

		return nil, 0, errors.New(message)
	}

	minW := pkgWhs[0]

	for _, wh := range pkgWhs {
		if wh.Cost < minW.Cost {
			minW = wh
		}
	}

	return minW.Warehouse, minW.Zone, nil
}

func (h *PackageHandler) label(c context.Context, sp *entity.Package, carrier providers.Carrier, warehouse *entity.Warehouse, labelTemplate string, oldCarrierCode string, zone int) (*entity.Tracking, string, string, error) {
	body := providers.RequestCreateLabel{
		ID:                     sp.ID,
		OrderNumber:            sp.OrderNumber,
		Code:                   sp.PackageCode.Code,
		Company:                sp.Company,
		FirstName:              sp.Recipient,
		LastName:               sp.Recipient,
		FullName:               sp.Recipient,
		City:                   sp.City,
		Address1:               sp.Address1,
		Address2:               sp.Address2,
		State:                  sp.StateCode,
		Zipcode:                sp.Zipcode,
		Phone:                  sp.PhoneNumber,
		Country:                sp.CountryCode,
		Weight:                 sp.ActualWeight,
		Height:                 sp.ActualHeight,
		Length:                 sp.ActualLength,
		Width:                  sp.ActualWidth,
		DistanceUnit:           "in",
		ServiceCode:            sp.Service.Code[0:1],
		FullServiceCode:        sp.Service.Code,
		DomesticCarrierService: sp.Service.DomesticCarrierService,
		HubStateCode:           warehouse.State,
		DisplayWeight:          sp.Weight,
		LabelTemplate:          labelTemplate,
		IsExceedPkg:            sp.IsPackageExceed,
		Zone:                   zone,

		WarehouseCompany:  warehouse.Company,
		WarehouseCity:     warehouse.City,
		WarehouseAddress1: warehouse.Address,
		WarehouseState:    warehouse.State,
		WarehouseZipcode:  warehouse.Zipcode,
		WarehouseCountry:  warehouse.Country,
		WarehousePhone:    warehouse.Phone,
	}

	if body.Weight <= 0 {
		body.Weight = sp.Weight
	}

	if body.Length <= 0 {
		body.Length = sp.Length
	}

	if body.Width <= 0 {
		body.Width = sp.Width
	}

	if body.Height <= 0 {
		body.Height = sp.Height
	}

	tracking := &entity.Tracking{
		PackageID: sp.ID,
		Status:    constant.TrackingStatusSuccess,
		Weight:    sp.ActualWeight,
		Length:    sp.ActualLength,
		Width:     sp.ActualWidth,
		Height:    sp.ActualHeight,
		HubID:     utils.Int64(warehouse.ID),
	}

	res, errAudit, err := h.CreateLabel.Request(c, body, carrier, sp.UserID, createlabel.LabelTypeNew)
	if err != nil {
		return nil, "", "", err
	}

	if errAudit != nil {
		return nil, "", errAudit.Error(), nil
	}

	lastMileCarrier := ""
	if res.CarrierCode != "" && oldCarrierCode != res.CarrierCode {
		dbcarrier, err := h.ServiceManager.GetCarrierByCode(res.CarrierCode)
		if err != nil {
			return nil, "", "", err
		}

		tracking.CarrierID = dbcarrier.ID
		lastMileCarrier = dbcarrier.LastMileCarrier
	}

	tracking.LabelURL = res.LabelUrl

	if res.CarrierCode == providers.CarrierTypeShippo {
		base64, err := h.toLabelBase64(res.LabelUrl)
		if err != nil {
			h.Logger.Error("base64 label", err)
			tracking.LabelURL = res.LabelUrl
		} else {
			tracking.LabelURL = base64
		}
	}

	tracking.ShipmentID = res.ShipmentID
	tracking.TrackingNumber = res.TrackingNumber
	tracking.ShipmentCost = res.ShippingFee
	tracking.HandlingFee = warehouse.HandlingFee
	tracking.CarrierService = res.CarrierService
	tracking.Zone = res.Zone
	tracking.Weight = res.Weight
	tracking.Length = res.Length
	tracking.Width = res.Width
	tracking.Height = res.Height

	return tracking, lastMileCarrier, "", nil
}

func (h *PackageHandler) toLabelBase64(url string) (string, error) {
	start := time.Now()

	res, err := http.Get(url)
	if err != nil {
		h.Logger.Error("download label: %v", err)
		return "", err
	}

	defer res.Body.Close()

	im, _, err := image.Decode(res.Body)
	if err != nil {
		h.Logger.Error("decode image label: %v", err)
		return "", err
	}

	var buf bytes.Buffer
	err = imaging.Encode(&buf, im, imaging.PNG)
	if err != nil {
		h.Logger.Error("resize label: %v", err)
		return "", err
	}

	base64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	duration := time.Since(start)
	h.Logger.Info("Duration convert label png to base64: ", duration)

	return base64, nil
}

type RawScanWeightRequest struct {
	WhsID       string  `json:"whs_id"`
	MachineID   string  `json:"machine_id"`
	TicketsNum  string  `json:"ticketsNum"`
	Length      float64 `json:"length"`
	Width       float64 `json:"width"`
	Height      float64 `json:"height"`
	Volume      float64 `json:"volume"`
	Weight      float64 `json:"weight"`
	AdminUser   string  `json:"admin_username"`
	PicturePath string  `json:"picture_path"`
	Transport   *string `json:"transportway"`
}
