package packages

import (
	"fmt"
	"math"
	"net/http"
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
		userId := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		userClass := cast.ToInt64(c.Request.Header.Get("X-User-Class"))
		id := cast.ToInt64(c.Param("id"))
		if id < 1 {
			c.JSON(http.StatusNotFound, httputil.ErrorResponse{
				Error: constant.APIResponseMessageNotFound,
			})

			return
		}

		user, err := h.UserManager.GetUserByID(userId)
		if err != nil {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: constant.APIResponseMessageParseRequestBody,
			})
		}

		pkg, err := h.PackageManager.GetPackageByPackageID(id)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, httputil.ErrorResponse{
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

		// thêm service inus và us48 và actus and au and eu k support qua api do darius k có api update
		if pkg.Service.Code == constant.ServiceFBACode || pkg.Service.Code == constant.ServiceINUSCode || pkg.Service.Code == constant.ServiceUS48Code || pkg.Service.Code == constant.ServiceACTUSCode || pkg.Service.Code == constant.ServiceAUCode || pkg.Service.Code == constant.ServiceEUCode || pkg.Service.Code == constant.ServiceAUFCode {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{fmt.Sprintf("The service %s is not support", pkg.Service.Name)},
			})

			return
		}

		rKey := fmt.Sprintf("%s_%d", constant.RedisKeyPackageCheckExists, pkg.ID)
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

		if pkg.UserID != userId {
			c.JSON(http.StatusNotFound, httputil.ErrorResponse{
				Error: constant.APIResponseMessageNotFound,
			})

			return
		}

		if pkg.Status != constant.PackageStatusCreated {
			c.JSON(http.StatusForbidden, httputil.ErrorResponse{
				Error: constant.APIResponseMessagePermissionDenied,
			})

			return
		}

		validator := order.MakeValidator(h.StateManager).SetLang("EN")
		form, err := validator.Decode(c.Request, false)
		if err != nil {
			h.Logger.Error("validator fail: ", err)
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: constant.APIResponseMessageValidateInput,
			})

			return
		}

		serviceID := pkg.ServiceID
		var service *entity.Service

		form.Service = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Service)
		if form.Service == "" {
			form.Service = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.ServiceCode)
		}
		if form.Service != "" {
			service, err = h.ServiceManager.GetServiceByCode(form.Service)
			if err != nil {
				h.Logger.Errorf("get state: %v", err)
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.APIResponseMessageValidateInput,
					Messages: []string{"The service id is invalid"},
				})

				return
			}

			serviceID = service.ID
		} else {
			service, err = h.ServiceManager.GetServiceByID(serviceID)
			if err != nil {
				h.Logger.Errorf("get state: %v", err)
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.APIResponseMessageValidateInput,
					Messages: []string{"The service id is invalid"},
				})

				return
			}
		}

		if service.Country != form.Country {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{fmt.Sprintf("The service %s not support country %s", service.Name, form.Country)},
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

		if service.Code == constant.ServiceFBACode {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{fmt.Sprintf("The service %s is not support", service.Name)},
			})

			return
		}

		form.ServiceCode = service.Code
		if form != nil {
			validator.Validate(form)
			validator.ValidateVolumes(form)
			validator.ValidateFbaServicePackage(form)
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
		var hasupdateprice bool

		if form.Recipient != pkg.Recipient {
			mapchange["recipient"] = form.Recipient
			logs = append(logs, entity.PackageAuditLog{
				OldValue: pkg.Recipient,
				Value:    form.Recipient,
				Type:     constant.PackageUpdateTypeRecipient,
			})
		}

		if form.Phone != pkg.PhoneNumber {
			mapchange["phone_number"] = form.Phone
			logs = append(logs, entity.PackageAuditLog{
				OldValue: pkg.PhoneNumber,
				Value:    form.Phone,
				Type:     constant.PackageUpdateTypePhoneNumber,
			})
		}

		if form.Address1 != pkg.Address1 {
			mapchange["address_1"] = form.Address1
			logs = append(logs, entity.PackageAuditLog{
				OldValue: pkg.Address1,
				Value:    form.Address1,
				Type:     constant.PackageUpdateTypeAddress,
			})
		}

		if form.Address2 != pkg.Address2 {
			mapchange["address_2"] = form.Address2
			logs = append(logs, entity.PackageAuditLog{
				OldValue: pkg.Address2,
				Value:    form.Address2,
				Type:     constant.PackageUpdateTypeAddress2,
			})
		}

		if form.Company != pkg.Company {
			mapchange["company"] = form.Company
		}

		if form.City != pkg.City {
			mapchange["city"] = form.City
			logs = append(logs, entity.PackageAuditLog{
				OldValue: pkg.City,
				Value:    form.City,
				Type:     constant.PackageUpdateTypeCity,
			})
		}

		if pkg.StateCode != form.State {
			mapchange["state_code"] = form.State
			logs = append(logs, entity.PackageAuditLog{
				OldValue: pkg.StateCode,
				Value:    form.State,
				Type:     constant.PackageUpdateTypeStateCode,
			})
		}

		if form.Zipcode != pkg.Zipcode {
			mapchange["zipcode"] = form.Zipcode
			logs = append(logs, entity.PackageAuditLog{
				OldValue: pkg.Zipcode,
				Value:    form.Zipcode,
				Type:     constant.PackageUpdateTypeZipcode,
			})
		}

		if form.Country != pkg.CountryCode {
			hasupdateprice = true
			mapchange["country_code"] = form.Country
			logs = append(logs, entity.PackageAuditLog{
				OldValue: pkg.CountryCode,
				Value:    form.Country,
				Type:     constant.PackageUpdateTypeCountryCode,
			})
		}

		if form.Weight != pkg.Weight {
			hasupdateprice = true
			mapchange["weight"] = form.Weight
			logs = append(logs, entity.PackageAuditLog{
				OldValue: fmt.Sprintf("%.2f", pkg.Weight),
				Value:    fmt.Sprintf("%.2f", form.Weight),
				Type:     constant.PackageUpdateTypeWeight,
			})
		}

		if serviceID != pkg.ServiceID {
			hasupdateprice = true
			mapchange["service_id"] = serviceID

			oldname := ""
			if pkg.Service != nil {
				oldname = pkg.Service.Name
			}

			logs = append(logs, entity.PackageAuditLog{
				OldValue: oldname,
				Value:    service.Name,
				Type:     constant.PackageUpdateTypeService,
			})
		}

		if form.OrderNumber != pkg.OrderNumber {
			mapchange["order_number"] = form.OrderNumber
			logs = append(logs, entity.PackageAuditLog{
				OldValue: pkg.OrderNumber,
				Value:    form.OrderNumber,
				Type:     constant.PackageUpdateTypeOrderNumber,
			})
		}

		if form.Detail != pkg.Detail {
			mapchange["detail"] = form.Detail
			logs = append(logs, entity.PackageAuditLog{
				OldValue: pkg.Detail,
				Value:    form.Detail,
				Type:     constant.PackageUpdateTypeDetail,
			})
		}

		if form.Length != pkg.Length || form.Width != pkg.Width || form.Height != pkg.Height {
			hasupdateprice = true

			cv := fmt.Sprintf("%vx%vx%v", pkg.Length, pkg.Width, pkg.Height)
			uv := fmt.Sprintf("%vx%vx%v", form.Length, form.Width, form.Height)

			mapchange["length"] = form.Length
			mapchange["width"] = form.Width
			mapchange["height"] = form.Height

			logs = append(logs, entity.PackageAuditLog{
				OldValue: cv,
				Value:    uv,
				Type:     constant.PackageUpdateTypeVolume,
			})

		}

		if form.IncludeBattery != pkg.IncludeBattery {
			mapchange["include_battery"] = form.IncludeBattery
		}

		var price float64 = 0
		var priceOutSize float64 = 0
		var isPackageExceed bool
		var isErrorEsPrice bool
		if hasupdateprice || pkg.IsPackageExceed {
			price, priceOutSize, err = h.CalculatePrice.Price3(c, userId, serviceID, userClass, form.Weight, form.Length, form.Height, form.Width, pkg.CountryCode)
			if err == calculate.ErrorNotService {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.APIResponseMessageValidateInput,
					Messages: []string{"The service code is invalid"},
				})

				return
			}

			if service.Code == constant.ServiceLABELCode {
				carrier := providers.NewCarrier(service.DomesticCarrier.Code, userId)
				if carrier == nil {
					c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
						Error:    "Bad request",
						Messages: []string{"The service code is invalid"},
					})

					return
				}

				h.Logger.Info("LABEL CODE: ", price)
				cost, err := h.EstimateCost(c, carrier, pkg)
				if err == nil {
					price = cost + price/100*cost
				}
				h.Logger.Info("LABEL CODE: ", price, cost, err)
			}

			if form.Country == "AU" || service.Code == constant.ServiceFBACode || service.Code == constant.ServiceINUSCode || service.Code == constant.ServiceUS48Code {
				if err == calculate.ErrorMaxWeight {
					msg := "The weight allowance exceeds limit"
					if price > 0 {
						msg = fmt.Sprintf("The weight should not exceed %v grams", math.Ceil(price)-1)
					}

					c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
						Error:    constant.APIResponseMessageValidateInput,
						Messages: []string{msg},
					})

					return
				}

				if err == calculate.ErrorMaxVolume {
					msg := "The volume allowance exceeds limit"
					if price > 0 {
						msg = fmt.Sprintf("The dimensions is invalid (LxHxW/5 <= %v)", math.Ceil(price)-1)
					}

					c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
						Error:    constant.APIResponseMessageValidateInput,
						Messages: []string{msg},
					})

					return
				}
			}

			if err == calculate.ErrorMaxWeight || err == calculate.ErrorMaxVolume {
				isPackageExceed = true

				cPkg := &entity.Package{}
				if err := utils.DeepCopy(pkg, cPkg); err != nil {
					h.Logger.Errorf("parse body: %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

					return
				}

				cPkg.Weight = form.Weight
				cPkg.Length = form.Length
				cPkg.Width = form.Width
				cPkg.Height = form.Height
				cPkg.Address1 = form.Address1
				cPkg.Address2 = form.Address2
				cPkg.City = form.City
				cPkg.StateCode = form.State
				cPkg.Zipcode = form.Zipcode
				cPkg.CountryCode = form.Country
				cPkg.PhoneNumber = form.Phone
				cPkg.Recipient = form.Recipient
				cPkg.IsPackageExceed = true

				cost, err := h.EstimateCost(c, nil, cPkg)
				if err != nil {
					isErrorEsPrice = true
				}

				if cost == 0 {
					isErrorEsPrice = true
				}

				if !isErrorEsPrice {
					pw, _ := calculate.CalcPriceWeight(form.Weight, form.Length, form.Height, form.Width, serviceID)
					price, err = h.CalculatePrice.CalculateExceedPackagePrice(pw, cost)
					if err != nil {
						isErrorEsPrice = true
					}
				} else {
					price = 0
					priceOutSize = 0
				}
			}

			if err != nil && err != calculate.ErrorMaxWeight && err != calculate.ErrorMaxVolume {
				h.Logger.Errorf("parse body: %v", err)
				c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
					Error: constant.APIResponseMessageServerInternalError,
				})

				return
			}
		}

		if (price > 0 && price != pkg.ShippingFee) || isErrorEsPrice {
			mapchange["shipping_fee"] = price
			mapchange["is_package_exceed"] = isPackageExceed
		}
		pkgCode := ""
		if pkg.PackageCode != nil {
			pkgCode = pkg.PackageCode.Code
		}
		form.ID = pkg.ID
		form.Code = pkgCode

		if form.OrderNumber == "" {
			form.OrderNumber = pkg.OrderNumber
		}

		if len(mapchange) == 0 || (isErrorEsPrice && pkg.ShippingFee == 0 && len(mapchange) == 0) {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: "The order is not change",
			})

			return
		}

		pkg.Weight = form.Weight
		pkg.Length = form.Length
		pkg.Width = form.Width
		pkg.Height = form.Height
		pkg.ActualWeight = form.Weight
		pkg.ActualLength = form.Length
		pkg.ActualWidth = form.Width
		pkg.ActualHeight = form.Height

		var fees []entity.ExtraFee
		var cPrice float64 = price
		if !isErrorEsPrice {
			if !pkg.IsPackageExceed && !hasupdateprice {
				cPrice = pkg.ShippingFee
			}
			fees, err = h.CalculatePrice.PromotionExtras(pkg, pkg.ExtraFee, cPrice)
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
				PackageID:      &pkg.ID,
				Amount:         fee,
				ExtraFeeTypeID: constant.ExtraFeeTypeBattery,
				Status:         constant.ExtraFeeStatusEnable,
			})
		}

		extraFeeService := h.CalculatePrice.GetServiceExtraFeee(form.Width, form.Height, form.Length, *service)
		if extraFeeService > 0 {
			fees = append(fees, entity.ExtraFee{
				Amount:         extraFeeService,
				PackageID:      utils.Int64(pkg.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeService,
				Status:         constant.ExtraFeeStatusEnable,
			})
		}

		h.Logger.Info("mapchange: ", pkg.Status)
		err = h.PackageManager.SaveUpdatePackage2(pkg.ID, userId, mapchange, logs, priceOutSize, fees, pkg.Status)
		if err != nil {
			errDetail := strings.Split(cast.ToString(err), ":")
			if errDetail[1] == " Incorrect string value" {
				c.JSON(http.StatusInternalServerError, "Invalid character")

				return
			}

			h.Logger.Errorf("update change package: %v", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.MessageServerInternalError,
			})

			return
		}

		extraFees, err := h.PackageManager.GetExtraFeeByPkgID(pkg.ID)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Redis.Del(c, rKey)
			h.Logger.Errorf("Get Package Deliver Logs %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

			return
		}

		form.CreatedAt = pkg.CreatedAt
		form.UpdatedAt = pkg.UpdatedAt

		if price > 0 || isErrorEsPrice {
			form.ShippingFee = price
		} else {
			form.ShippingFee = pkg.ShippingFee
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

		if pkg.Status == constant.PackageStatusCreated {
			if !isErrorEsPrice {
				amount := calculate.PeakFee(form.Weight)
				if amount > 0 {
					peakFee, err := h.BillManager.GetExtraFeeTypeByID(constant.ExtraFeeTypePeak)
					if err != nil && err != gorm.ErrRecordNotFound {
						h.Logger.Errorf("get extra peak fee: %v", err)
						c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

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

		form.Status = constant.MapTextStatusCustomerPackage[pkg.Status]
		form.Service = ""
		form.Recipient = ""
		form.FullName = pkg.Recipient
		c.JSON(http.StatusOK, UpdateResponse{Package: form})
	}
}
