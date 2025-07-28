package packages

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/order"
	"tebexpressapi/pkg/providers"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/string_util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

func (h *PackageHandler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		packageId := cast.ToInt64(c.Param("id"))

		if packageId < 1 {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: constant.APIResponseMessageNotFound,
			})
			return
		}

		if userID <= 0 {
			c.JSON(http.StatusForbidden, httputil.ErrorResponse{
				Error: constant.APIResponseMessagePermissionDenied,
			})
			return
		}

		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: constant.APIResponseMessageParseRequestBody,
			})
		}

		currentPackage, err := h.PackageManager.GetPackageByPackageID(packageId)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: constant.APIResponseMessageNotFound,
			})
			return
		}
		if err != nil {
			h.Logger.Errorf("get package: %v", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.APIResponseMessageServerInternalError,
			})

			return
		}

		rKey := fmt.Sprintf("%s_%d", constant.RedisKeyPackageCheckExists, packageId)
		val, err := h.Redis.SetNX(c, rKey, "value", constant.RedisKeyPackageCheckExistsExp).Result()
		if err != nil {
			h.Logger.Errorf("SetNX Redis : %s", err)
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: constant.APIResponseMessageServerInternalError,
			})
		}

		if !val {
			h.Logger.Errorf("Update package error : %s", err)
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: "Sorry, the server is busy. Please try again later",
			})
			return
		}

		defer h.Redis.Del(c, rKey)

		if currentPackage.UserID != userID {
			c.JSON(http.StatusForbidden, httputil.ErrorResponse{
				Error: constant.APIResponseMessagePermissionDenied,
			})
			return
		}

		if currentPackage.Status != constant.PackageStatusCreated && currentPackage.Status != constant.PackageStatusCNPurchased {
			c.JSON(http.StatusForbidden, httputil.ErrorResponse{
				Error: constant.APIResponseMessagePermissionDenied,
			})
			return
		}

		if currentPackage.Service.Code == constant.ServiceFBACode {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: fmt.Sprintf("Service %s is not supported.", currentPackage.Service.Name),
			})
			return
		}

		validator := order.MakeValidator(h.StateManager).SetLang("EN")
		form, err := validator.Decode(c.Request, true)
		if err != nil {
			h.Logger.Errorf("validator fail %v", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.APIResponseMessageValidateInput,
			})
			return
		}

		form.Service = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Service)
		if form.Service == "" {
			form.Service = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.ServiceCode)
		}
		if form.Service == "" {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: "Service can't be empty",
			})
			return
		}
		service, err := h.ServiceManager.GetServiceByCode(form.Service)
		if err != nil {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: "Service invalid",
			})
			return
		}
		form.ServiceCode = service.Code
		if service.Country != form.Country {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{fmt.Sprintf("The service %s not support country %s", service.Name, form.Country)},
			})
			return
		}

		if service.Code == constant.ServiceFBACode {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{fmt.Sprintf("The service %s is not support", service.Name)},
			})
			return
		}

		if form != nil {
			if form.ServiceCode == constant.ServiceCNCode {
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
				Error: constant.APIResponseMessageParseRequestBody,
			})
			return
		}

		form.Weight = utils.Ceil(form.Weight, 2)
		form.Length = utils.Ceil(form.Length, 2)
		form.Width = utils.Ceil(form.Width, 2)
		form.Height = utils.Ceil(form.Height, 2)

		var logs []entity.PackageAuditLog
		mapchange := make(map[string]interface{})

		var hasUpdatePrice bool

		if form.Recipient != currentPackage.Recipient {
			mapchange["recipient"] = form.Recipient
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.Recipient,
				Value:    form.Recipient,
				Type:     constant.PackageUpdateTypeRecipient,
			})
		}

		if form.Phone != currentPackage.PhoneNumber {
			mapchange["phone_number"] = form.Phone
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.PhoneNumber,
				Value:    form.Phone,
				Type:     constant.PackageUpdateTypePhoneNumber,
			})
		}

		if form.Address1 != currentPackage.Address1 {
			mapchange["address_1"] = form.Address1
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.Address1,
				Value:    form.Address1,
				Type:     constant.PackageUpdateTypeAddress,
			})
		}

		if form.Address2 != currentPackage.Address2 {
			mapchange["address_2"] = form.Address2
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.Address2,
				Value:    form.Address2,
				Type:     constant.PackageUpdateTypeAddress2,
			})
		}

		if form.City != currentPackage.City {
			mapchange["city"] = form.City
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.City,
				Value:    form.City,
				Type:     constant.PackageUpdateTypeCity,
			})
		}

		if form.State != currentPackage.StateCode {
			mapchange["state_code"] = form.State
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.StateCode,
				Value:    form.State,
				Type:     constant.PackageUpdateTypeStateCode,
			})
		}

		if form.Zipcode != currentPackage.Zipcode {
			mapchange["zipcode"] = form.Zipcode
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.Zipcode,
				Value:    form.Zipcode,
				Type:     constant.PackageUpdateTypeZipcode,
			})
		}

		if form.Country != currentPackage.CountryCode {
			hasUpdatePrice = true
			mapchange["country_code"] = form.Country
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.CountryCode,
				Value:    form.Country,
				Type:     constant.PackageUpdateTypeCountryCode,
			})
		}

		if fmt.Sprintf("%.2f", form.Weight) != fmt.Sprintf("%.2f", currentPackage.Weight) {
			hasUpdatePrice = true
			mapchange["weight"] = form.Weight
			logs = append(logs, entity.PackageAuditLog{
				OldValue: fmt.Sprintf("%.2f", currentPackage.Weight),
				Value:    fmt.Sprintf("%.2f", form.Weight),
				Type:     constant.PackageUpdateTypeWeight,
			})
		}

		if service.ID != currentPackage.ServiceID {
			hasUpdatePrice = true
			mapchange["service_id"] = service.ID

			var newLog = entity.PackageAuditLog{
				OldValue: "",
				Value:    service.Name,
				Type:     constant.PackageUpdateTypeService,
			}

			if currentPackage.Service != nil {
				newLog.OldValue = currentPackage.Service.Name
			}
			logs = append(logs, newLog)
		}

		if currentPackage.Service.Code == constant.ServiceCNCode {
			// Update when pending
			if form.CNProductLink != currentPackage.CNProductLink {
				mapchange["cn_product_link"] = form.CNProductLink
				logs = append(logs, entity.PackageAuditLog{
					OldValue: currentPackage.CNProductLink,
					Value:    form.CNProductLink,
					Type:     constant.PackageUpdateTypeCNLabel,
				})
			}

			if form.CNProductPrice != currentPackage.CNProductPrice {
				mapchange["cn_product_price"] = form.CNProductPrice
				newLog := entity.PackageAuditLog{
					OldValue: strconv.FormatFloat(currentPackage.CNProductPrice, 'f', -1, 64),
					Value:    strconv.FormatFloat(form.CNProductPrice, 'f', -1, 64),
					Type:     constant.PackageUpdateExtraFeeCNProduct,
				}
				logs = append(logs, newLog)

				err := h.PackageManager.UpdateExtraFee(currentPackage.ID, 0, userID, []entity.PackageAuditLog{newLog})
				if err != nil {
					h.Logger.Errorf("Update cn extra fee err: %v", err)
					c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
						Error: constant.APIResponseMessageServerInternalError,
					})
					return
				}
			}

			// Update when purchased
			if form.CNShippingFee != currentPackage.CNShippingFee {
				mapchange["cn_shipping_fee"] = form.CNShippingFee
				newLog := entity.PackageAuditLog{
					OldValue: strconv.FormatFloat(currentPackage.CNShippingFee, 'f', -1, 64),
					Value:    strconv.FormatFloat(form.CNShippingFee, 'f', -1, 64),
					Type:     constant.PackageUpdateExtraFeeCNShipping,
				}
				logs = append(logs, newLog)

				err := h.PackageManager.UpdateExtraFee(currentPackage.ID, 0, userID, []entity.PackageAuditLog{newLog})
				if err != nil {
					h.Logger.Errorf("Update cn extra fee err: %v", err)
					c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
						Error: constant.APIResponseMessageServerInternalError,
					})
					return
				}
			}

			if form.CustomCNBarcode != currentPackage.CustomCNBarcode {
				mapchange["custom_cn_barcode"] = form.CustomCNBarcode
				var oldValue string
				if currentPackage.CustomCNBarcode != nil {
					oldValue = *currentPackage.CustomCNBarcode
				}

				var newValue string
				if form.CustomCNBarcode != nil {
					newValue = *form.CustomCNBarcode
				}

				logs = append(logs, entity.PackageAuditLog{
					OldValue: oldValue,
					Value:    newValue,
					Type:     constant.PackageUpdateTypeCNLabel,
				})
			}

			if form.ImageUpload != "" {
				mapchange["cn_invoice_image"] = form.ImageUpload
			}

			if form.CNNote != currentPackage.CNNote {
				mapchange["cn_note"] = form.CNNote
				logs = append(logs, entity.PackageAuditLog{
					OldValue: currentPackage.CNNote,
					Value:    form.CNNote,
					Type:     constant.PackageUpdateTypeNote,
				})
			}

			if form.CNProductImage != currentPackage.CNProductImage {
				mapchange["cn_product_image"] = form.CNProductImage
				logs = append(logs, entity.PackageAuditLog{
					OldValue: currentPackage.CNProductImage,
					Value:    form.CNProductImage,
					Type:     constant.PackageUpdateTypeProduct,
				})
			}
		}

		if form.OrderNumber == "" {
			form.OrderNumber = currentPackage.OrderNumber
		}

		if form.OrderNumber != currentPackage.OrderNumber {
			mapchange["order_number"] = form.OrderNumber
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.OrderNumber,
				Value:    form.OrderNumber,
				Type:     constant.PackageUpdateTypeOrderNumber,
			})
		}

		if form.Detail != currentPackage.Detail {
			mapchange["detail"] = form.Detail
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.Detail,
				Value:    form.Detail,
				Type:     constant.PackageUpdateTypeDetail,
			})
		}

		if form.Length != currentPackage.Length || form.Width != currentPackage.Width || form.Height != currentPackage.Height {
			hasUpdatePrice = true

			currentVolume := cast.ToString(currentPackage.Length) + "x" + cast.ToString(currentPackage.Width) + "x" + cast.ToString(currentPackage.Height)
			updateVolume := cast.ToString(form.Length) + "x" + cast.ToString(form.Width) + "x" + cast.ToString(form.Height)

			mapchange["length"] = form.Length
			mapchange["width"] = form.Width
			mapchange["height"] = form.Height
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentVolume,
				Value:    updateVolume,
				Type:     constant.PackageUpdateTypeVolume,
			})
		}

		mapchange["include_battery"] = form.IncludeBattery

		if form.PackageName != currentPackage.PackageName {
			mapchange["package_name"] = form.PackageName
		}

		if form.PackageQuantity != currentPackage.PackageQuantity {
			mapchange["package_quantity"] = form.PackageQuantity
		}

		if form.TotalProductPrice != currentPackage.TotalProductPrice {
			mapchange["total_product_price"] = form.TotalProductPrice
		}

		var price float64 = 0
		var priceOutSize float64 = 0
		var isPackageExceed bool
		var isErrorEsPrice bool

		serviceIDToCalculatePrice := utils.GetServiceIDToCalculatePrice(currentPackage.CustomTiktokBarcode, service)
		if hasUpdatePrice || currentPackage.IsPackageExceed {
			price, priceOutSize, err = h.CalculatePrice.Price3(c, userID, serviceIDToCalculatePrice, user.Class, form.Weight, form.Length, form.Height, form.Width, form.Country)
			if err == calculate.ErrorNotService {
				if service.Code != constant.ServiceCNCode {
					c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
						Error: "Service invalid",
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
						Error:    "Bad request",
						Messages: []string{"The service code is invalid"},
					})

					return
				}

				h.Logger.Info("LABEL CODE: ", price)
				cost, err := h.EstimateCost(c, carrier, currentPackage)
				if err == nil {
					price = cost + price/100*cost
				}
				h.Logger.Info("LABEL CODE: ", price, cost, err)
			}

			// if form.Country == "AU" || service.Code == constant.ServiceFBACode {
			if err == calculate.ErrorMaxWeight {
				msg := "The allowed weight exceeds the limit"
				if price > 0 {
					msg = fmt.Sprintf("Weight must not exceed %v grams", math.Ceil(price)-1)
				}

				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    "Bad request",
					Messages: []string{msg},
				})
				return
			}

			if err == calculate.ErrorMaxVolume {
				msg := "The dimensions exceed the allowable limit"
				if price > 0 {
					msg = fmt.Sprintf("Invalid dimensions (LxHxW/5 <= %v)", math.Ceil(price)-1)
				}

				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    "Bad request",
					Messages: []string{msg},
				})
				return
			}

			if err != nil && err != calculate.ErrorMaxVolume && err != calculate.ErrorMaxWeight {
				h.Logger.Errorf("parse body: %v", err)
				c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
					Error: constant.APIResponseMessageServerInternalError,
				})
				return
			}
		}

		if (price != currentPackage.ShippingFee) || isErrorEsPrice {
			if (service.Code == constant.ServiceCNCode && hasUpdatePrice) || price > 0 {
				mapchange["shipping_fee"] = price
				mapchange["is_package_exceed"] = isPackageExceed
			}
		}

		if len(mapchange) == 0 {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: "The order is not change",
			})
			return
		}

		currentPackage.Weight = form.Weight
		currentPackage.Length = form.Length
		currentPackage.Width = form.Width
		currentPackage.Height = form.Height
		currentPackage.ActualWeight = form.Weight
		currentPackage.ActualLength = form.Length
		currentPackage.ActualWidth = form.Width
		currentPackage.ActualHeight = form.Height

		var fees []entity.ExtraFee
		var cPrice float64 = price
		if !isErrorEsPrice {
			if !currentPackage.IsPackageExceed && !hasUpdatePrice {
				cPrice = currentPackage.ShippingFee
			}
			fees, err = h.CalculatePrice.PromotionExtras(currentPackage, currentPackage.ExtraFee, cPrice)
			if err != nil {
				c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
					Error: constant.APIResponseMessageServerInternalError,
				})
				return
			}
		}

		if form.IncludeBattery {
			fee := h.CalculatePrice.GetExtraFeeBaterry()
			fees = append(fees, entity.ExtraFee{
				PackageID:      &currentPackage.ID,
				Amount:         fee,
				ExtraFeeTypeID: constant.ExtraFeeTypeBattery,
				Status:         constant.ExtraFeeStatusEnable,
			})
		}

		extraFeeService := h.CalculatePrice.GetServiceExtraFeee(form.Width, form.Height, form.Length, *service)
		if extraFeeService > 0 {
			fees = append(fees, entity.ExtraFee{
				Amount:         extraFeeService,
				PackageID:      utils.Int64(currentPackage.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeService,
				Status:         constant.ExtraFeeStatusEnable,
			})
		}

		err = h.PackageManager.SaveUpdatePackage2(currentPackage.ID, userID, mapchange, logs, priceOutSize, fees, currentPackage.Status)
		if err != nil {
			errDetail := strings.Split(cast.ToString(err), ":")
			if errDetail[1] == " Incorrect string value" {
				c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
					Error: "Kí tự không hợp lệ",
				})
				return
			}
			h.Logger.Errorf("update change package: %v", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.MessageServerInternalError,
			})
			return
		}

		extraFees, err := h.PackageManager.GetExtraFeeByPkgID(currentPackage.ID)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Redis.Del(c, rKey)
			h.Logger.Errorf("Get Package Deliver Logs %v", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.MessageServerInternalError,
			})
			return
		}

		form.CreatedAt = currentPackage.CreatedAt
		form.UpdatedAt = currentPackage.UpdatedAt

		if price > 0 || isErrorEsPrice {
			form.ShippingFee = price
		} else {
			form.ShippingFee = currentPackage.ShippingFee
		}

		form.TotalCost = form.ShippingFee
		form.ExtraFees = []order.ExtraFee{}

		for _, v := range extraFees {
			name := ""
			if v.ExtraFeeType != nil {
				name = v.ExtraFeeType.Name
			}

			description := v.Description
			if v.ExtraFeeType != nil && description == "" {
				description = v.ExtraFeeType.Name
			}

			form.ExtraFees = append(form.ExtraFees, order.ExtraFee{
				ExtraFeeType: name,
				Amount:       v.Amount,
				Description:  description,
			})

			form.TotalCost += v.Amount
		}

		if currentPackage.Status == constant.PackageStatusCreated {
			if !isErrorEsPrice {
				amount := calculate.PeakFee(form.Weight)
				if amount > 0 {
					peakFee, err := h.BillManager.GetExtraFeeTypeByID(constant.ExtraFeeTypePeak)
					if err != nil && err != gorm.ErrRecordNotFound {
						h.Logger.Errorf("get extra peak fee: %v", err)
						c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
							Error: constant.MessageServerInternalError,
						})
						return
					}

					form.ExtraFees = append(form.ExtraFees, order.ExtraFee{
						ExtraFeeType: peakFee.Name,
						Amount:       amount,
						Description:  peakFee.Name,
					})

					form.TotalCost += amount
				}
			}
		}

		form.TotalCost = utils.ToFixed(form.TotalCost, 2)

		form.Status = constant.MapTextStatusCustomerPackage[currentPackage.Status]
		form.Service = ""
		form.Recipient = ""
		form.FullName = currentPackage.Recipient
		c.JSON(http.StatusOK, UpdateResponse{Package: form})
	}
}
