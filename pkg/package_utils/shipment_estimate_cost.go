package packageutils

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/createlabel"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/order"
	"tebexpressapi/pkg/providers"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"
	"time"

	"go.uber.org/zap"
)

// EstimateCostHandler --
type EstimateCost struct {
	Logger *zap.SugaredLogger

	PackageManager   *sqlmanager.PackageManager
	WareHouseManager *sqlmanager.WareHouseManager
	ServiceManager   *sqlmanager.ServiceManager

	CreateLabel *createlabel.CreateLabel
}

type Dimensions struct {
	Width  float64 `json:"width"`
	Length float64 `json:"length"`
	Height float64 `json:"height"`
}

func NewEstimateCost(l *zap.SugaredLogger, pm *sqlmanager.PackageManager, whm *sqlmanager.WareHouseManager,
	sm *sqlmanager.ServiceManager, createLabel *createlabel.CreateLabel) *EstimateCost {
	return &EstimateCost{
		Logger: l,

		PackageManager:   pm,
		WareHouseManager: whm,
		ServiceManager:   sm,
		CreateLabel:      createLabel,
	}
}

// Handle --
func (h *EstimateCost) Handle(c context.Context, ids []int64, postmarkDate int64) error {
	packages, err := h.PackageManager.GetPackages(sqlmanager.PackageQueryOption{
		IDs:     ids,
		Preload: []string{"Service"},
	})
	if err != nil {
		h.Logger.Errorf("Get packages consumer error, %v", err)
		return err
	}

	wareHouses, err := h.WareHouseManager.GetWareHouses(sqlmanager.OptionWareHouse{
		Type:   constant.WareHouseTypeInternational,
		Status: constant.WareHouseStatusActive, //estimaste ca hub inactive
	})

	if err != nil {
		h.Logger.Errorf("Get warehouses error, %v", err)
		return err
	}

	for _, packageItem := range packages {

		var msgErr []string
		warehouses := 0

		err = h.WareHouseManager.DeactivateOldCost(packageItem)
		if err != nil {
			h.Logger.Errorf("Estimate cost err: %v", err)
			return err
		}

		var carrierCode string
		var carrier providers.Carrier = nil
		dbcarrier := &packageItem.Service.DomesticCarrier

		h.Logger.Info("aaa: ", packageItem.Service.DomesticCarrier.Code, packageItem.UserID)
		if packageItem.CountryCode == "AU" {
			carrier = providers.NewCarrier(packageItem.Service.DomesticCarrier.Code, packageItem.UserID)
		} else {
			carrierCode, err = h.CreateLabel.GetCarrierCode(c, packageItem, packageItem.UserID, "")
			if err != nil {
				h.Logger.Errorf("get carrier code: %v", err)
				return err
			}

			if carrierCode != "" {
				carrier = providers.NewCarrier(carrierCode, packageItem.UserID)
			} else {
				carrier = providers.NewCarrier(packageItem.Service.DomesticCarrier.Code, packageItem.UserID)
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

		var wg sync.WaitGroup
		for _, wareHouse := range wareHouses {
			isWl := utils.IsStateWhiteList(packageItem.UserID)
			if isWl && wareHouse.State == "CA" {
				continue
			}

			wg.Add(1)

			go func(wareHouse entity.Warehouse) {
				defer wg.Done()

				if wareHouse.Country != packageItem.CountryCode {
					return
				}

				if packageItem.Address1 == "" && packageItem.Address2 != "" {
					packageItem.Address1 = packageItem.Address2
				}

				resultOrg, s, err := order.EstimateCost(carrier, &packageItem, wareHouse)
				if s != "" {
					msgErr = append(msgErr, s)
					h.Logger.Errorf("Estimate cost org err: %v", s)
					return
				}
				if err != nil {
					h.Logger.Errorf("Estimate cost org err: %v", err)
					return
				}

				cost := &entity.PackageWarehouseCost{
					PackageID: packageItem.ID,
					HubID:     wareHouse.ID,
					Cost:      wareHouse.HandlingFee + resultOrg.TotalCost,
					OrgCost:   wareHouse.HandlingFee + resultOrg.TotalCost,
					Zone:      resultOrg.Zone,
				}

				if !packageItem.IsPackageExceed && packageItem.CountryCode == "US" {
					weight, length, height, width, err := h.CreateLabel.Fake(c, packageItem.Weight, packageItem.Length, packageItem.Height, packageItem.Width)
					if err != nil {
						h.Logger.Errorf("fake volume: %v", err)
						return
					}

					packageItem.ActualWeight = weight
					packageItem.ActualLength = length
					packageItem.ActualHeight = height
					packageItem.ActualWidth = width

					//estimate cost fake
					result, s, err := order.EstimateCost(carrier, &packageItem, wareHouse)
					if s != "" {
						msgErr = append(msgErr, s)
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

				warehouses += 1

			}(wareHouse)

		}

		// wait 10 second for call api estimate price
		if utils.WaitTimeout(&wg, 10*time.Second) {
			fmt.Println("Timed out waiting for estimate price")
		} else {
			fmt.Println("Estimate price done")
		}

		if warehouses < 1 {
			if len(msgErr) > 0 {
				h.Logger.Error("Estimate cost warehouse error. PackageID %d", packageItem.ID)
			} else {
				h.Logger.Error(fmt.Sprintf("Estimate cost warehouse error. PackageID %d", packageItem.ID))
			}
		}

	}

	return nil
}
