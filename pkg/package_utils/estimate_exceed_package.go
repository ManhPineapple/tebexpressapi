package packageutils

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/createlabel"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/order"
	"tebexpressapi/pkg/providers"
	"tebexpressapi/pkg/providers/ibblue"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"

	"go.uber.org/zap"
)

type EstimateExceedPackage struct {
	Logger *zap.SugaredLogger

	PackageManager   *sqlmanager.PackageManager
	WareHouseManager *sqlmanager.WareHouseManager
	ServiceManager   *sqlmanager.ServiceManager
	Calculate        *calculate.CalculatePrice
	IBBlue           *ibblue.IBBlue
	CreateLabel      *createlabel.CreateLabel
}

func NewEstimateExceedPackage(l *zap.SugaredLogger, pm *sqlmanager.PackageManager, whm *sqlmanager.WareHouseManager, sm *sqlmanager.ServiceManager, calc *calculate.CalculatePrice, cl *createlabel.CreateLabel) *EstimateExceedPackage {
	return &EstimateExceedPackage{
		Logger: l,

		PackageManager:   pm,
		WareHouseManager: whm,
		ServiceManager:   sm,
		Calculate:        calc,
		IBBlue:           ibblue.NewIBBlue(nil),
		CreateLabel:      cl,
	}
}

func (h *EstimateExceedPackage) Handle(c context.Context, pkg *entity.Package, user *entity.User) error {
	if pkg.Status != constant.PackageStatusCreated || !pkg.IsPackageExceed {
		h.Logger.Errorf("Package  %v is not valid, %v", pkg.ID)
		return errors.New(fmt.Sprintf("Package  %v is not valid, %v", pkg.ID))
	}

	_, extraFee, _ := h.Calculate.Price3(c, user.ID, pkg.ServiceID, user.Class, pkg.Weight, pkg.Length, pkg.Height, pkg.Width, pkg.CountryCode)

	var carrierCode string
	var carrier providers.Carrier = nil
	var err error
	dbcarrier := &pkg.Service.DomesticCarrier

	if pkg.CountryCode == "AU" {
		carrier = providers.NewCarrier(pkg.Service.DomesticCarrier.Code, pkg.UserID)
	} else {
		carrierCode, err = h.CreateLabel.GetCarrierCode(c, *pkg, pkg.UserID, "")
		if err != nil {
			h.Logger.Errorf("get carrier code: %v", err)
			return err
		}

		if carrierCode != "" {
			carrier = providers.NewCarrier(carrierCode, pkg.UserID)
		} else {
			carrier = providers.NewCarrier(pkg.Service.DomesticCarrier.Code, pkg.UserID)
		}
	}

	if carrier == nil {
		return errors.New("carrier invalid")
	}

	if carrierCode != dbcarrier.Code && carrierCode != "" {
		dbcarrier, err = h.ServiceManager.GetCarrierByCode(carrierCode)
		if err != nil {
			h.Logger.Errorf("Estimate cost err: %v", err)
			return err
		}
	}

	cost, err := h.EstimateCostPkg(pkg, carrier, dbcarrier)
	if err != nil {
		h.Logger.Errorf("Error get estimate package %v", err)
		return err
	}

	if cost == 0 {
		h.Logger.Errorf("Estimate giá đơn hàng thất bại")
		return errors.New("cost invalid")
	}

	var exFee []entity.ExtraFee
	exFee = append(exFee, entity.ExtraFee{
		Amount:         extraFee,
		ExtraFeeTypeID: constant.ExtraFeeTypeOutSize,
	})
	pkg.ExtraFee = exFee
	pw, _ := calculate.CalcPriceWeight(pkg.Weight, pkg.Length, pkg.Height, pkg.Width, pkg.ServiceID)
	shippingFee, err := h.Calculate.CalculateExceedPackagePrice(pw, cost)

	if err != nil {
		h.Logger.Errorf("error exceed fee error %v", err)
		return err
	}

	pkg.ShippingFee = shippingFee

	fees, err := h.Calculate.PromotionExtras(pkg, pkg.ExtraFee, shippingFee)
	if err != nil {
		h.Logger.Errorf("error get discount %v", err)
		return err
	}

	pkg.ExtraFee = append(pkg.ExtraFee, fees...)
	err = h.PackageManager.SaveExceedPackage(pkg)
	if err != nil {
		h.Logger.Errorf("Update save exceed package error %v", err)
		return err
	}

	return nil
}

func (h *EstimateExceedPackage) EstimateCostPkg(pkg *entity.Package, carrier providers.Carrier, dbcarrier *entity.Carrier) (float64, error) {
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

	err = h.WareHouseManager.DeactivateOldCost(*pkg)
	if err != nil {
		h.Logger.Errorf("Estimate cost err: %v", err)
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
				PackageID: pkg.ID,
				HubID:     wareHouse.ID,
				Warehouse: &wareHouse,
				OrgCost:   0,
			}

			res, msg, err := order.EstimateCost(carrier, pkg, wareHouse)
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
			cost.Cost = cost.OrgCost

			if dbcarrier != nil {
				cost.CarrierID = dbcarrier.ID
			}

			err = h.WareHouseManager.CreateEstimateCost(&cost)
			if err != nil {
				h.Logger.Errorf("Create estimate cost err: %v", err)
			}

			if wareHouse.Status == constant.WareHouseStatusActive {
				m.Lock()
				pkgWhs = append(pkgWhs, cost)
				apiCost[wareHouse.ID] = res.TotalCost
				m.Unlock()
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
