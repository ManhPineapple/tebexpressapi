package packages

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/order"
	"tebexpressapi/pkg/providers"
	"tebexpressapi/pkg/rabbitmq"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/string_util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func (h *PackageHandler) Create() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))

		if role != constant.UserRoleCustomer {
			h.Logger.Error("Create package: permission denied", "userID", userID, "role", role)
			c.JSON(http.StatusForbidden, httputil.ErrorResponse{
				Error:    constant.MessagePermissionDenied,
				Messages: []string{"User is not customer"},
			})
			return
		}

		if userID <= 0 {
			h.Logger.Error("Create package: userID is invalid", "userID", userID)
			c.JSON(http.StatusForbidden, httputil.ErrorResponse{
				Error:    constant.MessagePermissionDenied,
				Messages: []string{"UserID is invalid"},
			})
			return
		}

		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			c.JSON(http.StatusForbidden, httputil.ErrorResponse{
				Error:    constant.MessagePermissionDenied,
				Messages: []string{"User not found"},
			})
			return
		}

		validator := order.MakeValidator(h.StateManager).SetLang("EN")
		// Validate form
		form, err := validator.Decode(c.Request, false)
		if err != nil {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{err.Error()},
			})
			return
		}

		existedPackages, _ := h.PackageManager.GetPackages(sqlmanager.PackageQueryOption{
			OrderNumber:     form.OrderNumber,
			UserID:          userID,
			IgnoreStatusArr: []int64{constant.PackageStatusCreated, constant.PackageStatusArchived, constant.PackageStatusCancelled},
		})

		if len(existedPackages) > 0 {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{fmt.Sprintf("Mã đơn hàng %s đã tồn tại.", form.OrderNumber)},
			})
			return
		}

		form.Service = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Service)
		if form.Service == "" {
			form.Service = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.ServiceCode)
		}
		if form.Service == "" {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{"The service code is required"},
			})
			return
		}
		service, err := h.ServiceManager.GetServiceByCode(form.Service)
		if err != nil {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{"The service code is invalid"},
			})

			return
		}

		if service.Country != form.Country {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{fmt.Sprintf("Service %s doesn't support country %s", service.Name, form.Country)},
			})
			return
		}

		if user.PartnerID != 0 && service.PartnerID != user.PartnerID {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{"The service code is invalid"},
			})
			return
		}

		if service.Code == constant.ServiceFBACode || service.Code == constant.ServiceFastFBACode {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{fmt.Sprintf("Service %s doesn't support", service.Name)},
			})
			return
		}

		if service != nil {
			form.ServiceCode = service.Code
		}
		if form != nil {
			// Đơn CN có label tiktok riêng, không cần validate
			if form.CustomTiktokBarcode != "" && form.ServiceCode == constant.ServiceCNCode {
				form.OrderNumber = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.OrderNumber)
				if form.OrderNumber == "" {
					c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
						Error:    constant.APIResponseMessageValidateInput,
						Messages: []string{"Missing order number"},
					})
					return
				}

				if len(form.OrderNumber) > 200 {
					c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
						Error:    constant.APIResponseMessageValidateInput,
						Messages: []string{"Order number exceeds 200 characters"},
					})
					return
				}

				form.Detail = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Detail)
				if form.Detail == "" {
					c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
						Error:    constant.APIResponseMessageValidateInput,
						Messages: []string{"Order detail missing"},
					})
					return
				}

				if len(form.Detail) > 1000 {
					c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
						Error:    constant.APIResponseMessageValidateInput,
						Messages: []string{"Order detail exceeds 1000 characters"},
					})
					return
				}
			} else if form.ServiceCode == constant.ServiceTiktokCode || form.CustomTiktokBarcode != "" {
				validator.ValidateTiktokPkg(form)
			} else if form.ServiceCode == constant.ServiceCNCode {
				validator.ValidateChinaPackage(form)
			} else {
				validator.Validate(form)
				validator.ValidateVolumes(form)
				validator.ValidateFbaServicePackage(form)
			}
		}

		if messages := validator.Errors(); len(messages) > 0 {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: messages,
			})
			return
		}

		if err := validator.Error(); err != nil {
			h.Logger.Errorf("parse body: %v", err)
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageParseRequestBody,
				Messages: []string{err.Error()},
			})
			return
		}

		old, err := h.PackageManager.GetPackageDetailForCustomer(sqlmanager.PackageQueryOption{
			Status:      constant.PackageStatusCreated,
			UserID:      userID,
			OrderNumber: form.OrderNumber,
		})
		if err != nil && err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error:    constant.MessageServerInternalError,
				Messages: []string{"Get package error"},
			})
			return
		}

		if old.ID > 0 {
			// if package that has order number existed then fill package information to form
			form.ID = old.ID
			form.OrderNumber = old.OrderNumber
			form.FullName = old.Recipient
			form.Recipient = ""
			form.Company = old.Company
			form.Phone = old.PhoneNumber
			form.Address1 = old.Address1
			form.Address2 = old.Address2
			form.City = old.City
			form.State = old.StateCode
			form.Zipcode = old.Zipcode
			form.Country = old.CountryCode
			form.Detail = old.Detail
			form.Weight = old.Weight
			form.Width = old.Width
			form.Length = old.Length
			form.Height = old.Height
			form.Status = constant.MapTextStatusCustomerPackage[constant.PackageStatusCreated]
			form.ServiceCode = old.ServiceCode
			form.Service = ""
			form.CreatedAt = old.CreatedAt
			form.UpdatedAt = old.UpdatedAt
			form.TotalCost = old.ShippingFee
			form.ShippingFee = old.ShippingFee
			form.ExtraFees = []order.ExtraFee{}

			// fill extra fee type to form
			extraFees, err := h.BillManager.GetAllExtraFeeTypes()
			if err != nil && err != gorm.ErrRecordNotFound {
				h.Logger.Errorf("get extra fees: %v", err)
				c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
					Error:    constant.APIResponseMessageServerInternalError,
					Messages: []string{"Get extra fees error"},
				})

				return
			}

			mapExtraFeeText := make(map[int64]string)
			for _, v := range extraFees {
				mapExtraFeeText[v.ID] = v.Name
			}

			amount := calculate.PeakFee(form.Weight)
			if amount > 0 {
				form.ExtraFees = append(form.ExtraFees, order.ExtraFee{
					ExtraFeeType: mapExtraFeeText[constant.ExtraFeeTypePeak],
					Amount:       amount,
					Description:  mapExtraFeeText[constant.ExtraFeeTypePeak],
				})

				form.TotalCost += amount
			}

			for _, v := range old.ExtraFees {
				if v.Description == "" {
					v.Description = v.ExtraFeeType
				}

				form.ExtraFees = append(form.ExtraFees, order.ExtraFee{
					ExtraFeeType: v.ExtraFeeType,
					Amount:       v.Amount,
					Description:  v.Description,
				})

				form.TotalCost += v.Amount
			}

			form.TotalCost = utils.ToFixed(form.TotalCost, 2)
			c.JSON(http.StatusOK, CreateResponse{Package: form})
			return
		}

		form.Weight = math.Ceil(form.Weight*100) / 100
		form.Length = math.Ceil(form.Length*100) / 100
		form.Width = math.Ceil(form.Width*100) / 100
		form.Height = math.Ceil(form.Height*100) / 100

		hasProducts := []*entity.PackageProducts{}
		for _, packageProduct := range form.PackageProducts {
			product, err := h.ProductManager.GetProductByID(packageProduct.ProductID)
			if err != nil {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.APIResponseMessageValidateInput,
					Messages: []string{"Sản phẩm không tồn tại"},
				})
				return
			}

			if packageProduct.Quantity > product.Stock {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.APIResponseMessageValidateInput,
					Messages: []string{fmt.Sprintf("Sản phẩm %s không đủ hàng trong kho", product.SKU)},
				})
				return
			}

			hasProducts = append(hasProducts, &entity.PackageProducts{
				ProductID: packageProduct.ProductID,
				Status:    constant.PackageProductsStatusActive,
				Quantity:  packageProduct.Quantity,
				Product:   product,
			})
		}

		sp := &entity.Package{
			OrderNumber:       form.OrderNumber,
			Detail:            form.Detail,
			Recipient:         form.Recipient,
			PhoneNumber:       form.Phone,
			Address1:          form.Address1,
			Address2:          form.Address2,
			City:              form.City,
			StateCode:         form.State,
			Zipcode:           form.Zipcode,
			CountryCode:       form.Country,
			Weight:            form.Weight,
			Length:            form.Length,
			Width:             form.Width,
			Height:            form.Height,
			IncludeBattery:    form.IncludeBattery,
			ValidateAddress:   constant.PackageValidAddress,
			UserID:            userID,
			ServiceID:         service.ID,
			Status:            constant.PackageStatusCreated,
			PackageProducts:   hasProducts,
			Service:           service,
			PartnerID:         user.PartnerID,
			PackageName:       form.PackageName,
			PackageQuantity:   form.PackageQuantity,
			TotalProductPrice: form.TotalProductPrice,
		}

		if service.Code == constant.ServiceCNCode {
			sp.CNIsPurchased = form.CNIsPurchased
			if !form.CNIsPurchased {
				sp.CNProductLink = form.CNProductLink
				sp.CNProductPrice = form.CNProductPrice
			} else {
				sp.CNShippingFee = form.CNShippingFee
				sp.Status = constant.PackageStatusCNPurchased
			}

			sp.CustomCNBarcode = form.CustomCNBarcode
			sp.CNNote = form.CNNote
			sp.CNProductImage = form.CNProductImage
			if form.ImageUpload != "" {
				sp.CNInvoiceImage = form.ImageUpload
			}
		}

		sp.IsEarlyScan = form.IsEarlyScan
		if service.Code == constant.ServiceTiktokCode || form.CustomTiktokBarcode != "" {
			sp.CustomTiktokBarcode = &form.CustomTiktokBarcode
			form.CustomTiktokBarcode = utils.TransformDownloadURL(form.CustomTiktokBarcode)
			sp.Label = form.CustomTiktokBarcode
		}

		if service.Code == constant.ServiceWarehouseCode {
			sp.Label = form.ImageUpload
		}

		var isErrorEsPrice bool
		serviceIDToCalculatePrice := utils.GetServiceIDToCalculatePrice(&form.CustomTiktokBarcode, service)
		price, priceOutSize, err := h.CalculatePrice.Price3(c, userID, serviceIDToCalculatePrice, user.Class, form.Weight, form.Length, form.Height, form.Width, form.Country)
		if err == calculate.ErrorNotService {
			if service.Code != constant.ServiceCNCode {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.APIResponseMessageValidateInput,
					Messages: []string{"The service code is invalid"},
				})
				return
			} else {
				err = nil
			}
		}

		if service.Code == constant.ServiceLABELCode {
			carrier := providers.NewCarrier(service.DomesticCarrier.Code, userID)
			if carrier == nil {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.APIResponseMessageValidateInput,
					Messages: []string{"The service code is invalid"},
				})

				return
			}

			h.Logger.Info("LABEL CODE: ", price)
			cost, err := h.EstimateCost(c, carrier, sp)
			if err == nil {
				price = cost + price/100*cost
			}
			h.Logger.Info("LABEL CODE: ", price, cost, err)
		}

		if service.Code == constant.ServiceTebprintHubCode && form.CustomTiktokBarcode == "" {
			carrier := providers.NewCarrier(service.DomesticCarrier.Code, userID)
			if carrier == nil {
				c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
					Error:    constant.MessageServerInternalError,
					Messages: []string{"Get carrier error"},
				})
				return
			}

			cost, err := h.PackageEstimateCost(c, carrier, sp)
			if err == nil {
				const additionalTebprintCost = 0.5
				price = cost + additionalTebprintCost
			}
		}

		if form.Country == "AU" || service.Code == constant.ServiceFBACode || service.Code == constant.ServiceFastFBACode {
			if err == calculate.ErrorMaxWeight {
				msg := "Trọng lượng cho phép vượt quá giới hạn"
				if price > 0 {
					msg = fmt.Sprintf("Trọng lượng không được vượt quá %v grams", math.Ceil(price)-1)
				}

				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.APIResponseMessageValidateInput,
					Messages: []string{msg},
				})
				return
			}

			if err == calculate.ErrorMaxVolume {
				msg := "Kích thước vượt quá giới hạn cho phép"
				if price > 0 {
					msg = fmt.Sprintf("Kích thước không hợp lệ (LxHxW/5 <= %v)", math.Ceil(price)-1)
				}

				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.APIResponseMessageValidateInput,
					Messages: []string{msg},
				})
				return
			}
		}

		if err == calculate.ErrorMaxWeight || err == calculate.ErrorMaxVolume && service.Code != constant.ServiceFBACode && service.Code != constant.ServiceFastFBACode {
			sp.IsPackageExceed = true

			carrier := providers.NewCarrier(service.DomesticCarrier.Code, userID)
			if carrier == nil {
				c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
					Error:    constant.MessageServerInternalError,
					Messages: []string{"Get carrier error"},
				})
				return
			}

			cost, err := h.EstimateCost(c, carrier, sp)
			if err != nil {
				isErrorEsPrice = true
			}

			if cost == 0 {
				isErrorEsPrice = true
			}

			if !isErrorEsPrice {
				pw, _ := calculate.CalcPriceWeight(form.Weight, form.Length, form.Height, form.Width, service.ID)
				price, err = h.CalculatePrice.CalculateExceedPackagePrice(pw, cost)
				h.Logger.Info("CalculateExceedPackagePrice: ", price, err)
				if err != nil {
					isErrorEsPrice = true
				}
			} else {
				price = 0
				priceOutSize = 0
			}

		}

		sp.ShippingFee = price

		if err != nil && err != calculate.ErrorMaxWeight && err != calculate.ErrorMaxVolume {
			h.Logger.Errorf("parse body: %v", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageServerInternalError,
				Messages: []string{"Calculate exceed package price error"},
			})
			return
		}

		if priceOutSize > 0 {
			sp.ExtraFee = []entity.ExtraFee{{Amount: priceOutSize, ExtraFeeTypeID: constant.ExtraFeeTypeOutSize}}
		}

		if !isErrorEsPrice {
			fees, err := h.CalculatePrice.PromotionExtras(sp, sp.ExtraFee, price)
			if err != nil {
				c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
					Error:    constant.APIResponseMessageServerInternalError,
					Messages: []string{"Calculate extrafee error"},
				})
				return
			}
			if len(fees) > 0 {
				if sp.ExtraFee == nil {
					sp.ExtraFee = []entity.ExtraFee{}
				}

				sp.ExtraFee = append(sp.ExtraFee, fees...)
			}
		}

		if form.IsTradeMark {
			sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
				Amount:         1,
				ExtraFeeTypeID: constant.ExtraFeeTypeTradeMark,
			})
		}

		if sp.IncludeBattery {
			fee := h.CalculatePrice.GetExtraFeeBaterry()
			sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
				Amount:         fee,
				ExtraFeeTypeID: constant.ExtraFeeTypeBattery,
			})
		}

		isInsured, insuredFee, err := h.CalculatePrice.PromotionInsured(sp.UserID)
		if err != nil {
			h.Logger.Errorf("promotion insured: %v", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageServerInternalError,
				Messages: []string{"Calculate insured price error"},
			})
			return
		}

		if isInsured {
			sp.IsInsured = true
			if insuredFee > 0 {
				sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
					Amount:         insuredFee,
					PackageID:      utils.Int64(sp.ID),
					ExtraFeeTypeID: constant.ExtraFeeTypeInsured,
				})
			}
		}
		extraFeeService := h.CalculatePrice.GetServiceExtraFeee(form.Width, form.Height, form.Length, *service)
		if extraFeeService > 0 {
			sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
				Amount:         extraFeeService,
				PackageID:      utils.Int64(sp.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeService,
			})
		}

		if sp.Service.Code == constant.ServiceCNCode {
			if sp.Status == constant.PackageStatusCreated && sp.CNProductPrice != 0 {
				var cnPricePercentage float64
				if sp.CNProductPrice > 200 {
					cnPricePercentage = viper.GetFloat64("extra_fees.cn_high_price_percentage")
				} else {
					cnPricePercentage = viper.GetFloat64("extra_fees.cn_low_price_percentage")
				}
				minProxyPrice := viper.GetFloat64("extra_fees.cn_min_proxy_buying_fee")
				sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
					Amount:         math.Max(sp.CNProductPrice*cnPricePercentage, minProxyPrice),
					PackageID:      utils.Int64(sp.ID),
					ExtraFeeTypeID: constant.ExtraFeeTypeChinaProductPercentage,
				})

				sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
					Amount:         sp.CNProductPrice,
					PackageID:      utils.Int64(sp.ID),
					ExtraFeeTypeID: constant.ExtraFeeTypeChinaProduct,
				})

			} else if sp.Status == constant.PackageStatusCNPurchased && sp.CNShippingFee != 0 {
				sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
					Amount:         sp.CNShippingFee,
					PackageID:      utils.Int64(sp.ID),
					ExtraFeeTypeID: constant.ExtraFeeTypeChinaShipping,
				})
			}

			defaultCNShippingFeeToVN := viper.GetFloat64("extra_fees.default_cn_ship_to_vn_fee")
			sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
				Amount:         defaultCNShippingFeeToVN,
				PackageID:      utils.Int64(sp.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeCNShippingToVN,
			})
			defaultCNHandlingFee := viper.GetFloat64("extra_fees.default_cn_handling_fee") // Phí handling + active tracking
			sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
				Amount:         defaultCNHandlingFee,
				PackageID:      utils.Int64(sp.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeHandling,
			})
		}

		if sp.Service.Code == constant.ServiceWarehouseCode {
			defaultWsHandlingFee := viper.GetFloat64("extra_fees.default_ws_handling_fee")

			if form.IsTiktokWarehouse {
				price = defaultWsHandlingFee // dùng label riêng sẽ bỏ qua phí tạo tracking ibblue
				sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
					Amount:         defaultWsHandlingFee,
					PackageID:      utils.Int64(sp.ID),
					ExtraFeeTypeID: constant.ExtraFeeTypeHandling,
				})
			}
			if user.Balance+0.01 < price {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.MessageValidateInput,
					Messages: []string{"Your account does not have sufficient funds, please top up."},
				})
				return
			}
		}

		tiktokEarlyScanFee := viper.GetFloat64("extra_fees.default_tiktok_early_scan_fee")
		if sp.IsEarlyScan {
			price += tiktokEarlyScanFee

			sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
				Amount:         tiktokEarlyScanFee,
				PackageID:      utils.Int64(sp.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeEarlyScanTiktok,
			})
		}

		peakFee, _ := h.BillManager.GetExtraFeeTypeByID(constant.ExtraFeeTypePeak)
		if peakFee != nil && service.Code != constant.ServiceTiktokCode && sp.CustomTiktokBarcode == nil {
			amount := calculate.PeakFee(sp.Weight)
			if amount > 0 {
				sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
					PackageID:      utils.Int64(sp.ID),
					ExtraFeeTypeID: peakFee.ID,
					Description:    peakFee.Name,
					Amount:         amount,
					Status:         constant.ExtraFeeStatusEnable,
				})
			}
		}

		vatExtrafee, _, _ := utils.CreateOrUpdateVat(sp, sp.BillID, userID)
		sp.ExtraFee = append(sp.ExtraFee, *vatExtrafee)

		packageIDsCreated, err := h.PackageManager.CreatePackages([]*entity.Package{sp}, userID)
		if err != nil {
			errDetail := strings.Split(cast.ToString(err), ":")
			if errDetail[1] == " Incorrect string value" {
				c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
					Error:    constant.MessageValidateInput,
					Messages: []string{"Invalid character"},
				})
				return
			}

			h.Logger.Error("Error create package", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageServerInternalError,
				Messages: []string{"Create package error"},
			})
			return
		}

		if len(hasProducts) > 0 {
			for _, packageProduct := range hasProducts {
				err := h.ProductManager.AdjustStock(packageProduct.ProductID, packageProduct.PackageID, -packageProduct.Quantity)
				if err != nil {
					h.Logger.Errorf("Error when get product: %v", err)
					c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
						Error:    constant.APIResponseMessageServerInternalError,
						Messages: []string{"Update product stock error"},
					})
				}
			}
		}

		if len(packageIDsCreated) <= 0 {
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageServerInternalError,
				Messages: []string{"Create package error"},
			})
			return
		}

		options := sqlmanager.PackageQueryOption{ID: packageIDsCreated[0]}
		packageCreated, err := h.PackageManager.GetPackageDetail(options)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageNotFound,
				Messages: []string{"Get created package error"},
			})
			return
		}

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Detail %v", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageNotFound,
				Messages: []string{"Get created package error"},
			})
			return
		}

		extraFees, err := h.BillManager.GetAllExtraFeeTypes()
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("get extra fees: %v", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageNotFound,
				Messages: []string{"Get extrafeetype error"},
			})
			return
		}
		mapExtraFeeText := make(map[int64]string)
		for _, v := range extraFees {
			mapExtraFeeText[v.ID] = v.Name
		}

		form.ID = packageCreated.ID
		form.CreatedAt = packageCreated.CreatedAt
		form.UpdatedAt = packageCreated.UpdatedAt
		form.Code = ""
		form.ShippingFee = price
		form.TotalCost = price
		form.ExtraFees = []order.ExtraFee{}
		form.Base64Label = ""

		for _, v := range packageCreated.ExtraFee {
			form.ExtraFees = append(form.ExtraFees, order.ExtraFee{
				ExtraFeeType: mapExtraFeeText[v.ExtraFeeTypeID],
				Amount:       v.Amount,
				Description:  mapExtraFeeText[v.ExtraFeeTypeID],
			})

			form.TotalCost += v.Amount
		}
		if !isErrorEsPrice {
			amount := calculate.PeakFee(form.Weight)
			if amount > 0 {
				form.ExtraFees = append(form.ExtraFees, order.ExtraFee{
					ExtraFeeType: mapExtraFeeText[constant.ExtraFeeTypePeak],
					Amount:       amount,
					Description:  mapExtraFeeText[constant.ExtraFeeTypePeak],
				})

				form.TotalCost += amount
			}
		}

		if packageCreated.Service.Code == constant.ServiceTiktokCode || packageCreated.CustomTiktokBarcode != nil {
			err = h.Producer.Publish(c, "tiktok_upload_queue", rabbitmq.UploadMessage{PackageID: packageCreated.ID})
			if err != nil {
				h.Logger.Errorf("Failed to enqueue OCR message for pkg %d: %v", packageCreated.ID, err)
			}
		}

		form.TotalCost = utils.ToFixed(form.TotalCost, 2)
		form.Status = constant.MapTextStatusCustomerPackage[constant.PackageStatusCreated]
		form.Service = ""
		form.Recipient = ""
		form.FullName = packageCreated.Recipient

		c.JSON(http.StatusOK, CreateResponse{Package: form})
	}
}

func (h *PackageHandler) PackageEstimateCost(c context.Context, carrier providers.Carrier, pkg *entity.Package) (float64, error) {
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
