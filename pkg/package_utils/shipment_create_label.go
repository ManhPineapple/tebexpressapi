package packageutils

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
	"sync"
	"tebexpressapi/pkg/alert"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/createlabel"
	"tebexpressapi/pkg/label"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/order"
	"tebexpressapi/pkg/providers"
	"tebexpressapi/pkg/providers/ibblue"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/dbgorm"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CreateLabelHandler struct {
	Logger  *zap.SugaredLogger
	LocalS3 storage.S3
	Redis   *redis.Client
	Alert   alert.Alert

	CreateLabel *createlabel.CreateLabel

	SettingManager   *sqlmanager.SettingManager
	PackageManager   *sqlmanager.PackageManager
	BillManager      *sqlmanager.BillManager
	UserManager      *sqlmanager.UserManager
	WareHouseManager *sqlmanager.WareHouseManager
	ServiceManager   *sqlmanager.ServiceManager
}

func NewCreateLabelHandler(l *zap.SugaredLogger, r *redis.Client, s3 storage.S3, stm *sqlmanager.SettingManager, pm *sqlmanager.PackageManager,
	bm *sqlmanager.BillManager, um *sqlmanager.UserManager, wh *sqlmanager.WareHouseManager, sm *sqlmanager.ServiceManager,
	createLabel *createlabel.CreateLabel, alert alert.Alert) *CreateLabelHandler {
	return &CreateLabelHandler{
		Logger:  l,
		LocalS3: s3,
		Redis:   r,
		Alert:   alert,

		CreateLabel: createLabel,

		SettingManager:   stm,
		PackageManager:   pm,
		BillManager:      bm,
		UserManager:      um,
		WareHouseManager: wh,
		ServiceManager:   sm,
	}
}

// Handle --
func (h *CreateLabelHandler) Handle(c context.Context, ids []int64, promotionLabel bool, isChinaPackage bool, shipmentId int64) error {
	if isChinaPackage {
		err := h.HandleChinaPkgs(c, ids)
		if err != nil {
			return err
		}
		return nil
	}

	if !promotionLabel {
		// create label no promotion
		err := h.HanldeNonPromotionLabelPkgs(c, ids, shipmentId)
		if err != nil {
			return err
		}

		return nil
	}

	//create label promotion
	err := h.HanldePromotionLabelPkgs(c, ids)
	if err != nil {
		h.Logger.Errorf("Handle promotion label packages error: %v", err)
		return err
	}

	return nil
}

func (h *CreateLabelHandler) HanldePromotionLabelPkgs(c context.Context, pkgIDs []int64) error {
	pkgs, err := h.PackageManager.GetPackages(sqlmanager.PackageQueryOption{
		IDs: pkgIDs,
	})
	if err != nil {
		h.Logger.Errorf("Get packages consumer create label error, %v", err)
		return err
	}

	defer h.RemoveCacheRedis(c, pkgIDs)
	userID := pkgs[0].UserID
	user, err := h.UserManager.GetUserByID(userID)
	if err != nil {
		return err
	}

	bill, err := h.BillManager.GetOrCreateNowBill(userID)
	if err != nil {
		h.Logger.Errorf("Get bill error: %v", err)
		return err
	}

	peakFee, err := h.BillManager.GetExtraFeeTypeByID(constant.ExtraFeeTypePeak)
	if err != nil && err != gorm.ErrRecordNotFound {
		h.Logger.Errorf("get extra peak fee: %v", err)
		return err
	}

	queryOptions := sqlmanager.SettingQueryOption{
		Key:    constant.BookmarkPushSettingKey,
		UserID: userID,
	}

	setting, err := h.SettingManager.GetSetting(queryOptions)
	if err != nil && err != gorm.ErrRecordNotFound {
		h.Logger.Errorf("Error fetch setting query: %v", err)
		return err
	}
	var pushBookmark bool
	if setting != nil && setting.ID > 0 {
		pushBookmark = cast.ToBool(setting.Value)
	}

	var amount float64 = 0
	vPkgs := make([]entity.Package, 0) // valid packages

	for _, pkg := range pkgs {
		if pkg.Status != constant.PackageStatusCreated {
			if pushBookmark && !pkg.IsBookmark {
				continue
			}

			h.Logger.Errorf(fmt.Sprintf("Đơn hàng #%d trạng thái đơn không hợp lệ", pkg.ID))
			continue
		}

		if pkg.PackageCode != nil && pkg.PackageCode.Status == constant.PackageCodeDisable {
			if pushBookmark && !pkg.IsBookmark {
				continue
			}

			h.Logger.Errorf(fmt.Sprintf("Mã vận đơn %s đã bị hủy", pkg.PackageCode.Code))
			continue
		}

		if pkg.ValidateAddress != constant.PackageValidAddress {
			if pushBookmark && !pkg.IsBookmark {
				continue
			}

			h.Logger.Errorf(fmt.Sprintf("Địa chỉ đơn hàng #%d không hợp lệ", pkg.ID))
			continue
		}
		// check amount to validate
		vPkgs = append(vPkgs, pkg)
		if peakFee != nil {
			amount := calculate.PeakFee(pkg.Weight)
			if amount > 0 {
				pkg.ExtraFee = append(pkg.ExtraFee, entity.ExtraFee{
					Model: dbgorm.Model{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					BillID:         utils.Int64(bill.ID),
					PackageID:      utils.Int64(pkg.ID),
					ExtraFeeTypeID: peakFee.ID,
					Description:    peakFee.Name,
					Amount:         amount,
					Status:         constant.ExtraFeeStatusEnable,
				})
			}
		}

		var extraFee float64 = 0
		for _, fee := range pkg.ExtraFee {
			extraFee += fee.Amount
		}

		amount += pkg.ShippingFee + extraFee
	}

	if len(vPkgs) == 0 {
		h.Logger.Errorf("No package valid to process")
		return nil
	}
	amount = utils.ToFixed(amount, 2)

	//calculat coupon
	var refundCoupon *entity.ExtraFee
	calAmount := amount

	if user.Balance < calAmount && (user.UserInfo == nil || user.UserInfo.DebtMaxAmount <= 0) {
		h.Logger.Errorf("Số dư ví không đủ. Vui lòng nạp thêm")
		return nil
	}

	if user.Balance-calAmount < 0 && user.UserInfo != nil && user.UserInfo.DebtMaxAmount > 0 {
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if user.Balance < 0 && user.UserInfo.DebtTime != nil && user.UserInfo.DebtTime.AddDate(0, 0, user.UserInfo.DebtMaxDay).Before(time.Now()) {
			h.Logger.Errorf("Tài khoản của bạn đã nợ quá thời hạn cho phép. Vui lòng nạp thêm tiền để tiếp tục sử dụng dịch vụ")
			return errors.New("Tài khoản của bạn đã nợ quá thời hạn cho phép. Vui lòng nạp thêm tiền để tiếp tục sử dụng dịch vụ")
		}

		if math.Abs(user.Balance-calAmount) > user.UserInfo.DebtMaxAmount {
			h.Logger.Errorf("Tài khoản của bạn đã nợ quá giới hạn cho phép. Vui lòng nạp thêm tiền để tiếp tục sử dụng dịch vụ")
			return errors.New("Tài khoản của bạn đã nợ quá giới hạn cho phép. Vui lòng nạp thêm tiền để tiếp tục sử dụng dịch vụ")
		}
	}

	var template string = ibblue.TemplateTebexpress
	err, pCodes := h.PackageManager.CreatePackageCodes(pkgs)

	if err != nil {
		h.Logger.Errorf("Error create package code: %v", err)
		return err
	}
	amount = 0
	var sPkgs, fPkgs []entity.Package
	var auPkgs []entity.Package
	var actusPkgs []entity.Package
	var aufPkgs []entity.Package
	var ndPkgs []entity.Package
	// var inUSPkgs []entity.Package
	var wg sync.WaitGroup
	var m sync.Mutex
	for i, pkg := range vPkgs {
		pkg.PackageCode = pCodes[i]
		vPkgs[i].PackageCode = pCodes[i]

		// check don au
		// if pkg.CountryCode == "AU" {
		// 	auPkgs = append(auPkgs, pkg)
		// 	//  cong tong tien bill don au
		// 	if peakFee != nil {
		// 		amount := calculate.PeakFee(pkg.Weight)
		// 		if amount > 0 {
		// 			pkg.ExtraFee = append(pkg.ExtraFee, entity.ExtraFee{
		// 				Model: dbgorm.Model{
		// 					CreatedAt: time.Now(),
		// 					UpdatedAt: time.Now(),
		// 				},
		// 				BillID:         utils.Int64(bill.ID),
		// 				PackageID:      utils.Int64(pkg.ID),
		// 				ExtraFeeTypeID: peakFee.ID,
		// 				Description:    peakFee.Name,
		// 				Amount:         amount,
		// 				Status:         constant.ExtraFeeStatusEnable,
		// 			})
		// 		}
		// 	}
		// 	var extraFee float64 = 0
		// 	for _, fee := range pkg.ExtraFee {
		// 		extraFee += fee.Amount
		// 	}
		// 	fee := pkg.ShippingFee + extraFee
		// 	amount += fee
		// 	continue
		// }

		// check don actus
		if pkg.Service.Code == constant.ServiceACTUSCode {
			actusPkgs = append(actusPkgs, pkg)
			//  cong tong tien bill don au
			if peakFee != nil {
				amount := calculate.PeakFee(pkg.Weight)
				if amount > 0 {
					pkg.ExtraFee = append(pkg.ExtraFee, entity.ExtraFee{
						Model: dbgorm.Model{
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						BillID:         utils.Int64(bill.ID),
						PackageID:      utils.Int64(pkg.ID),
						ExtraFeeTypeID: peakFee.ID,
						Description:    peakFee.Name,
						Amount:         amount,
						Status:         constant.ExtraFeeStatusEnable,
					})
				}
			}
			var extraFee float64 = 0
			for _, fee := range pkg.ExtraFee {
				extraFee += fee.Amount
			}
			fee := pkg.ShippingFee + extraFee
			amount += fee
			continue
		}

		if pkg.Service.Code == constant.ServiceNDCode {
			ndPkgs = append(ndPkgs, pkg)
			//  cong tong tien bill don au
			if peakFee != nil {
				amount := calculate.PeakFee(pkg.Weight)
				if amount > 0 {
					pkg.ExtraFee = append(pkg.ExtraFee, entity.ExtraFee{
						Model: dbgorm.Model{
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						BillID:         utils.Int64(bill.ID),
						PackageID:      utils.Int64(pkg.ID),
						ExtraFeeTypeID: peakFee.ID,
						Description:    peakFee.Name,
						Amount:         amount,
						Status:         constant.ExtraFeeStatusEnable,
					})
				}
			}
			var extraFee float64 = 0
			for _, fee := range pkg.ExtraFee {
				extraFee += fee.Amount
			}
			fee := pkg.ShippingFee + extraFee
			amount += fee
			continue
		}

		if pkg.Service.Code == constant.ServiceAUFCode {
			aufPkgs = append(aufPkgs, pkg)
			//  cong tong tien bill don au
			if peakFee != nil {
				amount := calculate.PeakFee(pkg.Weight)
				if amount > 0 {
					pkg.ExtraFee = append(pkg.ExtraFee, entity.ExtraFee{
						Model: dbgorm.Model{
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						BillID:         utils.Int64(bill.ID),
						PackageID:      utils.Int64(pkg.ID),
						ExtraFeeTypeID: peakFee.ID,
						Description:    peakFee.Name,
						Amount:         amount,
						Status:         constant.ExtraFeeStatusEnable,
					})
				}
			}
			var extraFee float64 = 0
			for _, fee := range pkg.ExtraFee {
				extraFee += fee.Amount
			}
			fee := pkg.ShippingFee + extraFee
			amount += fee
			continue
		}

		wg.Add(1)
		go func(pkg entity.Package, template string) {
			defer wg.Done()

			if pkg.Tracking != nil && pkg.Tracking.Status != constant.TrackingStatusCanceled {
				pkg.Tracking.Status = constant.TrackingStatusSuccess
				pkg.Label = pkg.Tracking.LabelURL
			} else {
				carrier := providers.NewCarrier(pkg.Service.DomesticCarrier.Code, pkg.UserID)
				if carrier == nil {
					fPkgs = append(fPkgs, pkg)
					h.Logger.Errorf("Invalid carrier package id %v", pkg.ID)
					return
				}

				warehouse, zone, err := h.EstimateCost(c, pkg)
				if warehouse == nil || err != nil {
					h.Logger.Error("Can't find warehouse for package: ", err)
				}
				if err != nil {
					fPkgs = append(fPkgs, pkg)
					h.Logger.Errorf("Estimate pkg warehouse cost error: %v", err)
					return
				}

				tracking, msg, err := h.label(c, &pkg, carrier, warehouse, template, pkg.Service.DomesticCarrier.Code, zone)

				if err != nil {
					fPkgs = append(fPkgs, pkg)
					decodedURL, err := url.QueryUnescape(msg)
					if err != nil {
						fmt.Println("Error decoding URL:", err)
					}
					h.Alert.SendMessage(fmt.Sprintf("Error when call request create label usps for order %v: %v", decodedURL, pkg.OrderNumber))
					h.Logger.Errorf("Error when call request create label usps for order %v: %v", msg, pkg.OrderNumber)
					return
				}

				if msg != "" {
					fPkgs = append(fPkgs, pkg)

					decodedURL, err := url.QueryUnescape(msg)
					if err != nil {
						fmt.Println("Error decoding URL:", err)
					}
					h.Alert.SendMessage(fmt.Sprintf("Error when call request create label usps for order %v: %v", decodedURL, pkg.OrderNumber))
					h.Logger.Errorf("Error when call request create label usps for order %v: %v", msg, pkg.OrderNumber)
					return
				}

				ext := "png"
				if carrier.GetCode() == providers.CarrierTypeDarius {
					ext = "pdf"
				}
				path, err := order.StoreLabelS3(h.LocalS3, tracking.LabelURL, ext, tracking.TrackingNumber)
				if err != nil {
					h.Logger.Errorf("Store label error: %v", err)
				}

				tracking.LabelURL = path
				tracking.UserID = userID
				tracking.Version = viper.GetString("tracking_version")

				if tracking.CarrierID < 1 {
					tracking.CarrierID = pkg.Service.DomesticCarrierID
				}

				pkg.Label = path
				pkg.Tracking = tracking
			}

			//calculator real fee
			if peakFee != nil {
				amount := calculate.PeakFee(pkg.Weight)
				if amount > 0 {
					pkg.ExtraFee = append(pkg.ExtraFee, entity.ExtraFee{
						Model: dbgorm.Model{
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						BillID:         utils.Int64(bill.ID),
						PackageID:      utils.Int64(pkg.ID),
						ExtraFeeTypeID: peakFee.ID,
						Description:    peakFee.Name,
						Amount:         amount,
						Status:         constant.ExtraFeeStatusEnable,
					})
				}
			}
			var extraFee float64 = 0
			for _, fee := range pkg.ExtraFee {
				extraFee += fee.Amount
			}
			fee := pkg.ShippingFee + extraFee
			m.Lock()
			amount += fee
			sPkgs = append(sPkgs, pkg)
			m.Unlock()
		}(pkg, template)
	}

	wg.Wait()

	// create label au
	if len(auPkgs) > 0 {
		for _, packageItem := range auPkgs {
			// var err error
			var path string
			// _, path, err = label.CreateLabel(&packageItem, h.SettingManager, h.LocalS3)
			// if err != nil {
			// 	fPkgs = append(fPkgs, packageItem)
			// 	h.Logger.Errorf("create label au error: %v", err)
			// 	continue
			// }
			packageItem.Label = path
			sPkgs = append(sPkgs, packageItem)
		}
	}

	// create label actus
	if len(actusPkgs) > 0 {
		for _, packageItem := range actusPkgs {
			// var err error
			var path string
			// _, path, err = label.CreateLabel(&packageItem, h.SettingManager, h.LocalS3)
			// if err != nil {
			// 	fPkgs = append(fPkgs, packageItem)
			// 	h.Logger.Errorf("create label au error: %v", err)
			// 	continue
			// }
			packageItem.Label = path
			sPkgs = append(sPkgs, packageItem)
		}
	}

	// create label actus
	if len(ndPkgs) > 0 {
		for _, packageItem := range ndPkgs {
			// var err error
			var path string
			_, path, err = label.CreateLabel(&packageItem, h.SettingManager, h.LocalS3)
			if err != nil {
				fPkgs = append(fPkgs, packageItem)
				h.Logger.Errorf("create label nd error: %v", err)
				continue
			}
			packageItem.Label = path
			sPkgs = append(sPkgs, packageItem)
		}
	}

	// create label actus
	if len(aufPkgs) > 0 {
		for _, packageItem := range aufPkgs {
			// var err error
			var path string
			// _, path, err = label.CreateLabel(&packageItem, h.SettingManager, h.LocalS3)
			// if err != nil {
			// 	fPkgs = append(fPkgs, packageItem)
			// 	h.Logger.Errorf("create label au error: %v", err)
			// 	continue
			// }
			packageItem.Label = path
			sPkgs = append(sPkgs, packageItem)
		}
	}

	// // create label inus
	// if len(inUSPkgs) > 0 {
	// 	for _, packageItem := range inUSPkgs {
	// 		// var err error
	// 		var path string
	// 		// _, path, err = label.CreateLabel(&packageItem, h.SettingManager, h.LocalS3)
	// 		// if err != nil {
	// 		// 	fPkgs = append(fPkgs, packageItem)
	// 		// 	h.Logger.Errorf("create label au error: %v", err)
	// 		// 	continue
	// 		// }
	// 		packageItem.Label = path
	// 		sPkgs = append(sPkgs, packageItem)
	// 	}
	// }

	// var point int
	// process sucess package
	if len(sPkgs) > 0 {
		opt := sqlmanager.CreateBillOption{
			Packages:    sPkgs,
			BillID:      bill.ID,
			ShippingFee: amount,
			UserID:      userID,
		}

		_, err = h.BillManager.CreateBillWithLabelPromotion(opt, user, refundCoupon, len(fPkgs))
		if err != nil {
			h.Logger.Errorf("Error create bill: %v", err)
			return err
		}

	}

	//cancel fail packge
	var orderNumberFail []string
	if len(fPkgs) > 0 {
		for _, pkg := range fPkgs {
			if pushBookmark && !pkg.IsBookmark {
				continue
			}

			h.Logger.Errorf(fmt.Sprintf("Đơn hàng %v tạo tracking thất bại", pkg.OrderNumber))
			orderNumberFail = append(orderNumberFail, pkg.OrderNumber)
		}

	}
	h.Logger.Info("ids ----------------", pkgIDs)
	if len(orderNumberFail) > 0 {
		return fmt.Errorf("Đơn hàng %v tạo tracking thất bại", strings.Join(orderNumberFail, ","))
	}
	return nil
}

func (h *CreateLabelHandler) RemoveCacheRedis(c context.Context, ids []int64) {
	for _, id := range ids {
		rKey := "package_call_label"
		_ = h.Redis.SRem(c, rKey, id).Err()
	}
}

func (h *CreateLabelHandler) EstimateCost(c context.Context, pkg entity.Package) (*entity.Warehouse, int, error) {
	estimateCosts, err := h.WareHouseManager.GetEstimateCosts(pkg.ID, pkg.CountryCode)
	if err != nil {
		return nil, 0, err
	}

	isWl := utils.IsStateWhiteList(pkg.UserID)
	igState := ""
	if isWl {
		igState = "CA"
	}

	if len(estimateCosts) == 0 {
		wareHouses, err := h.WareHouseManager.GetWareHouses(sqlmanager.OptionWareHouse{
			Type:        constant.WareHouseTypeInternational,
			IgnoreState: igState,
			Status:      constant.WareHouseStatusActive,
		})

		if err != nil {
			h.Logger.Errorf("get estimate warehouses: %v", err)
			return nil, 0, err
		}

		err = h.WareHouseManager.DeactivateOldCost(pkg)
		if err != nil {
			h.Logger.Errorf("deactivate cost: %v", err)
			return nil, 0, err
		}

		var wg sync.WaitGroup
		var m sync.Mutex

		var carrierCode string
		var carrier providers.Carrier = nil
		dbcarrier := &pkg.Service.DomesticCarrier

		if pkg.CountryCode == "AU" {
			carrier = providers.NewCarrier(pkg.Service.DomesticCarrier.Code, pkg.UserID)
		} else {
			carrierCode, err = h.CreateLabel.GetCarrierCode(c, pkg, pkg.UserID, "")
			if err != nil {
				return nil, 0, err
			}

			if carrierCode != "" {
				carrier = providers.NewCarrier(carrierCode, pkg.UserID)
			} else {
				carrier = providers.NewCarrier(pkg.Service.DomesticCarrier.Code, pkg.UserID)
			}
		}

		if carrier == nil {
			return nil, 0, errors.New("carrier not found")
		}

		if carrierCode != dbcarrier.Code && carrierCode != "" {
			dbcarrier, err = h.ServiceManager.GetCarrierByCode(carrierCode)
			if err != nil {
				h.Logger.Errorf("Estimate cost err: %v", err)
				return nil, 0, err
			}
		}

		if dbcarrier == nil {
			return nil, 0, errors.New("carrier not found")
		}

		for _, wareHouse := range wareHouses {

			wg.Add(1)

			go func(wareHouse entity.Warehouse) {
				defer wg.Done()

				if pkg.CountryCode != wareHouse.Country {
					return
				}

				if pkg.IsPackageExceed {
					cost := entity.PackageWarehouseCost{
						PackageID: pkg.ID,
						HubID:     wareHouse.ID,
						Warehouse: &wareHouse,
						OrgCost:   0,
					}

					res, s, err := order.EstimateCost(carrier, &pkg, wareHouse)

					if s != "" {
						h.Logger.Errorf("estimate cost org: %v", s)
						return
					}

					if err != nil {
						h.Logger.Errorf("estimate cost org: %v", err)
						return
					}

					cost.OrgCost = res.TotalCost + wareHouse.HandlingFee
					cost.Zone = res.Zone
					cost.Cost = cost.OrgCost
					if dbcarrier != nil {
						cost.CarrierID = dbcarrier.ID
					}

					if wareHouse.Status == constant.WareHouseStatusActive {
						m.Lock()
						estimateCosts = append(estimateCosts, cost)
						m.Unlock()
					}

					err = h.WareHouseManager.CreateEstimateCost(&cost)
					if err != nil {
						h.Logger.Errorf("create estimate cost: %v", err)
					}

					return
				}
				// end hang qua co

				cost := &entity.PackageWarehouseCost{
					PackageID: pkg.ID,
					HubID:     wareHouse.ID,
					Warehouse: &wareHouse,
					Cost:      0,
					OrgCost:   0,
				}

				resultOrg, msg, err := order.EstimateCost(carrier, &pkg, wareHouse)
				if msg != "" {
					h.Logger.Errorf("estimate cost org: %v", msg)
					if pkg.CountryCode == "AU" {
						return
					}
				} else if err != nil {
					h.Logger.Errorf("estimate cost org: %v", err)
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

					result, msg, err := order.EstimateCost(carrier, clone, wareHouse)
					if msg != "" {
						h.Logger.Errorf("estimate cost: %v", msg)
						return
					}

					if err != nil {
						h.Logger.Errorf("estimate cost: %v", err)
						return
					}

					cost.Cost = wareHouse.HandlingFee + result.TotalCost
					cost.Zone = result.Zone
				}

				if dbcarrier != nil {
					cost.CarrierID = dbcarrier.ID
				}

				err = h.WareHouseManager.CreateEstimateCost(cost)
				if err != nil {
					h.Logger.Errorf("create estimate cost: %v", err)
					return
				}

				if wareHouse.Status == constant.WareHouseStatusActive {
					m.Lock()
					estimateCosts = append(estimateCosts, *cost)
					m.Unlock()
				}
			}(wareHouse)
		}

		wg.Wait()
	}

	if len(estimateCosts) == 0 {
		return nil, 0, errors.New("can't find lowest cost warehouse")
	}

	minW := estimateCosts[0]
	for _, wh := range estimateCosts {
		if wh.Cost < minW.Cost {
			minW = wh
		}
	}

	return minW.Warehouse, minW.Zone, nil
}

func (h *CreateLabelHandler) label(c context.Context, sp *entity.Package, carrier providers.Carrier, warehouse *entity.Warehouse, lalbelTemplate string, oldCarrierCode string, zone int) (*entity.Tracking, string, error) {
	h.Logger.Info("template", lalbelTemplate)
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
		HubStateCode:           warehouse.State,
		DisplayWeight:          sp.Weight,
		LabelTemplate:          lalbelTemplate,
		IsExceedPkg:            sp.IsPackageExceed,
		Zone:                   zone,
		DomesticCarrierService: sp.Service.DomesticCarrierService,

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
		return nil, "", err
	}

	if errAudit != nil {
		return nil, errAudit.Error(), nil
	}

	if res.CarrierCode != "" && res.CarrierCode != oldCarrierCode {
		dbcarrier, err := h.ServiceManager.GetCarrierByCode(res.CarrierCode)
		if err != nil {
			return nil, "", err
		}

		tracking.CarrierID = dbcarrier.ID
	}

	tracking.ShipmentID = res.ShipmentID
	tracking.TrackingNumber = res.TrackingNumber
	tracking.ShipmentCost = res.ShippingFee
	tracking.HandlingFee = warehouse.HandlingFee
	tracking.CarrierService = res.CarrierService
	tracking.LabelURL = res.LabelUrl
	tracking.Zone = res.Zone
	tracking.Weight = res.Weight
	tracking.Length = res.Length
	tracking.Width = res.Width
	tracking.Height = res.Height

	return tracking, "", nil
}

func (h *CreateLabelHandler) HanldeNonPromotionLabelPkgs(c context.Context, pkgIDs []int64, customerShipmentID int64) error {
	packages, err := h.PackageManager.GetPackages(sqlmanager.PackageQueryOption{
		IDs:     pkgIDs,
		Preload: []string{"Service"},
	})
	if err != nil {
		h.Logger.Errorf("Get packages consumer create label error, %v", err)
		return err
	}

	defer h.RemoveCacheRedis(c, pkgIDs)
	for i, packageItem := range packages {
		var err error
		var path string
		if customerShipmentID > 0 {
			_, path, err = label.CreateFBALabel(&packageItem, h.SettingManager, h.LocalS3, i+1, len(packages))
		} else {
			_, path, err = label.CreateLabel(&packageItem, h.SettingManager, h.LocalS3)
		}

		if err != nil {
			h.Logger.Errorf("create shipping package: %v", err)
			return err
		}

		if err = h.PackageManager.UpdatePackage(&entity.Package{Label: path}, packageItem.ID); err != nil {
			h.Logger.Errorf("create shipping package: %v", err)
			return err
		}

	}
	return nil
}

func (h *CreateLabelHandler) HandleChinaPkgs(c context.Context, pkgIDs []int64) error {
	pkgs, err := h.PackageManager.GetPackages(sqlmanager.PackageQueryOption{
		IDs: pkgIDs,
	})
	if err != nil {
		h.Logger.Errorf("Get packages consumer create label error, %v", err)
		return err
	}

	defer h.RemoveCacheRedis(c, pkgIDs)
	userID := pkgs[0].UserID
	user, err := h.UserManager.GetUserByID(userID)
	if err != nil {
		return err
	}

	bill, err := h.BillManager.GetOrCreateNowBill(userID)
	if err != nil {
		h.Logger.Errorf("Get bill error: %v", err)
		return err
	}

	queryOptions := sqlmanager.SettingQueryOption{
		Key:    constant.BookmarkPushSettingKey,
		UserID: userID,
	}

	setting, err := h.SettingManager.GetSetting(queryOptions)
	if err != nil && err != gorm.ErrRecordNotFound {
		h.Logger.Errorf("Error fetch setting query: %v", err)
		return err
	}
	var pushBookmark bool
	if setting != nil && setting.ID > 0 {
		pushBookmark = cast.ToBool(setting.Value)
	}

	var amountYuan float64 = 0
	var refundCoupon *entity.ExtraFee
	vPkgs := make([]entity.Package, 0)

	for _, pkg := range pkgs {
		if pkg.Status != constant.PackageStatusCNPurchased {
			if pushBookmark && !pkg.IsBookmark {
				continue
			}

			h.Logger.Errorf(fmt.Sprintf("Đơn hàng #%d trạng thái đơn không hợp lệ", pkg.ID))
			continue
		}

		if pkg.PackageCode != nil && pkg.PackageCode.Status == constant.PackageCodeDisable {
			if pushBookmark && !pkg.IsBookmark {
				continue
			}

			h.Logger.Errorf(fmt.Sprintf("Mã vận đơn %s đã bị hủy", pkg.PackageCode.Code))
			continue
		}

		if pkg.ValidateAddress != constant.PackageValidAddress {
			if pushBookmark && !pkg.IsBookmark {
				continue
			}

			h.Logger.Errorf(fmt.Sprintf("Địa chỉ đơn hàng #%d không hợp lệ", pkg.ID))
			continue
		}
		// check amount to validate
		vPkgs = append(vPkgs, pkg)

		var extraFee float64 = 0
		for _, fee := range pkg.ExtraFee {
			extraFee += fee.Amount
		}

		amountYuan += pkg.ShippingFee + extraFee
	}

	if len(vPkgs) == 0 {
		h.Logger.Errorf("No package valid to process")
		return nil
	}
	amountYuan = utils.ToFixed(amountYuan, 2)
	if user.BalanceChina < amountYuan {
		h.Logger.Errorf("Số dư ví không đủ. Vui lòng nạp thêm")
		return errors.New("Số dư ví không đủ. Vui lòng nạp thêm")
	}

	var template string = ibblue.TemplateTebexpress
	err, pCodes := h.PackageManager.CreatePackageCodes(pkgs)

	if err != nil {
		h.Logger.Errorf("Error create package code: %v", err)
		return err
	}

	amountYuan = 0
	var sPkgs, fPkgs []entity.Package
	var wg sync.WaitGroup
	var m sync.Mutex
	for i, pkg := range vPkgs {
		pkg.PackageCode = pCodes[i]
		vPkgs[i].PackageCode = pCodes[i]

		wg.Add(1)
		go func(pkg entity.Package, template string) {
			defer wg.Done()

			if pkg.Tracking != nil && pkg.Tracking.Status != constant.TrackingStatusCanceled {
				pkg.Tracking.Status = constant.TrackingStatusSuccess
				pkg.Label = pkg.Tracking.LabelURL
			} else if pkg.Weight > 0 {
				carrier := providers.NewCarrier(pkg.Service.DomesticCarrier.Code, pkg.UserID)
				if carrier == nil {
					fPkgs = append(fPkgs, pkg)
					h.Logger.Errorf("Invalid carrier package id %v", pkg.ID)
					return
				}

				warehouse, zone, err := h.EstimateCost(c, pkg)
				if warehouse == nil || err != nil {
					h.Logger.Error("Can't find warehouse for package: ", err)
				}
				if err != nil {
					fPkgs = append(fPkgs, pkg)
					h.Logger.Errorf("Estimate pkg warehouse cost error: %v", err)
					return
				}

				tracking, msg, err := h.label(c, &pkg, carrier, warehouse, template, pkg.Service.DomesticCarrier.Code, zone)

				if err != nil {
					fPkgs = append(fPkgs, pkg)
					decodedURL, err := url.QueryUnescape(msg)
					if err != nil {
						fmt.Println("Error decoding URL:", err)
					}
					h.Alert.SendMessage(fmt.Sprintf("Error when call request create label usps for order %v: %v", decodedURL, pkg.OrderNumber))
					h.Logger.Errorf("Error when call request create label usps for order %v: %v", msg, pkg.OrderNumber)
					return
				}

				if msg != "" {
					fPkgs = append(fPkgs, pkg)

					decodedURL, err := url.QueryUnescape(msg)
					if err != nil {
						fmt.Println("Error decoding URL:", err)
					}
					h.Alert.SendMessage(fmt.Sprintf("Error when call request create label usps for order %v: %v", decodedURL, pkg.OrderNumber))
					h.Logger.Errorf("Error when call request create label usps for order %v: %v", msg, pkg.OrderNumber)
					return
				}

				ext := "png"
				if carrier.GetCode() == providers.CarrierTypeDarius {
					ext = "pdf"
				}
				path, err := order.StoreLabelS3(h.LocalS3, tracking.LabelURL, ext, tracking.TrackingNumber)
				if err != nil {
					h.Logger.Errorf("Store label error: %v", err)
				}

				tracking.LabelURL = path
				tracking.UserID = userID
				tracking.Version = viper.GetString("tracking_version")

				if tracking.CarrierID < 1 {
					tracking.CarrierID = pkg.Service.DomesticCarrierID
				}

				pkg.Label = path
				pkg.Tracking = tracking
			}

			var extraFee float64 = 0
			for _, fee := range pkg.ExtraFee {
				extraFee += fee.Amount
			}
			fee := pkg.ShippingFee + extraFee
			m.Lock()
			amountYuan += fee
			sPkgs = append(sPkgs, pkg)
			m.Unlock()
		}(pkg, template)
	}

	wg.Wait()

	// process sucess package
	if len(sPkgs) > 0 {
		opt := sqlmanager.CreateBillOption{
			Packages:     sPkgs,
			BillID:       bill.ID,
			YuanCurrency: true,
			ShippingFee:  amountYuan,
			UserID:       userID,
		}

		_, err = h.BillManager.CreateBillWithLabelPromotion(opt, user, refundCoupon, len(fPkgs))
		if err != nil {
			h.Logger.Errorf("Error create bill: %v", err)
			return err
		}
	}

	//cancel fail packge
	var orderNumberFail []string
	if len(fPkgs) > 0 {
		for _, pkg := range fPkgs {
			if pushBookmark && !pkg.IsBookmark {
				continue
			}

			h.Logger.Errorf(fmt.Sprintf("Đơn hàng %v tạo tracking thất bại", pkg.OrderNumber))
			orderNumberFail = append(orderNumberFail, pkg.OrderNumber)
		}
	}
	h.Logger.Info("ids ----------------", pkgIDs)
	if len(orderNumberFail) > 0 {
		return fmt.Errorf("Đơn hàng %v tạo tracking thất bại", strings.Join(orderNumberFail, ","))
	}
	return nil
}
