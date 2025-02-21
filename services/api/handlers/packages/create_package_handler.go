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
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/string_util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

func (h *PackageHandler) Create() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		userClass := cast.ToInt64(c.Request.Header.Get("X-User-Class"))
		h.Logger.Info("create: ", userId, userClass)

		user, err := h.UserManager.GetUserByID(userId)
		if err != nil {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: constant.APIResponseMessageParseRequestBody,
			})
		}

		validator := order.MakeValidator(h.StateManager).SetLang("EN")
		// Validate form
		form, err := validator.Decode(c.Request, false)
		if err != nil {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
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
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{"The service code is required"},
			})

			return
		}

		service, err := h.ServiceManager.GetServiceByCode(form.Service)
		if err != nil {
			h.Logger.Errorf("get state: %v", err)
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{"The service code is invalid"},
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

		if (service.Country == constant.EUState && !utils.ContainsString(constant.EUCountries, form.Country)) || (service.Country != constant.EUState && service.Country != form.Country) {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{fmt.Sprintf("The service %s not support country %s", service.Name, form.Country)},
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
		}
		// End Validate form

		old, err := h.PackageManager.GetPackageDetailForCustomer(sqlmanager.PackageQueryOption{
			Status:      constant.PackageStatusCreated,
			UserID:      userId,
			OrderNumber: form.OrderNumber,
		})
		if err != nil && err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.MessageServerInternalError,
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
					Error: constant.APIResponseMessageServerInternalError,
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

		// new then create
		sp := &entity.Package{
			ServiceID:       service.ID,
			OrderNumber:     form.OrderNumber,
			Detail:          form.Detail,
			Recipient:       form.Recipient,
			PhoneNumber:     form.Phone,
			Company:         form.Company,
			Address1:        form.Address1,
			Address2:        form.Address2,
			City:            form.City,
			StateCode:       form.State,
			Zipcode:         form.Zipcode,
			CountryCode:     form.Country,
			Weight:          form.Weight,
			Length:          form.Length,
			Width:           form.Width,
			Height:          form.Height,
			Status:          constant.PackageStatusCreated,
			Service:         service,
			IncludeBattery:  form.IncludeBattery,
			ValidateAddress: constant.PackageValidAddress,
			PartnerID:       user.PartnerID,
		}

		var isErrorEsPrice bool
		price, priceOutSize, err := h.CalculatePrice.Price3(c, userId, service.ID, userClass, form.Weight, form.Length, form.Height, form.Width, form.Country)
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
			cost, err := h.EstimateCost(c, carrier, sp)
			if err == nil {
				price = cost + price/100*cost
			}
			h.Logger.Info("LABEL CODE: ", price, cost, err)
		}

		if service.Code == constant.ServiceCNCode {
			sp.CustomCNBarcode = form.CustomCNBarcode
		}

		if form.Country == "AU" || service.Code == constant.ServiceUS48Code || service.Code == constant.ServiceINUSCode {
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

		if err == calculate.ErrorMaxWeight || err == calculate.ErrorMaxVolume && service.Code != constant.ServiceFBACode {
			sp.IsPackageExceed = true

			carrier := providers.NewCarrier(service.DomesticCarrier.Code, userId)
			if carrier == nil {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    "Bad request",
					Messages: []string{"The service code is invalid"},
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
				Error: constant.APIResponseMessageServerInternalError,
			})

			return
		}

		extraFees, err := h.BillManager.GetAllExtraFeeTypes()
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("get extra fees: %v", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.APIResponseMessageServerInternalError,
			})

			return
		}
		mapExtraFeeText := make(map[int64]string)
		for _, v := range extraFees {
			mapExtraFeeText[v.ID] = v.Name
		}

		sp.ShippingFee = price
		sp.UserID = userId
		if priceOutSize > 0 {
			sp.ExtraFee = []entity.ExtraFee{{Amount: priceOutSize, ExtraFeeTypeID: constant.ExtraFeeTypeOutSize}}
		}
		if !isErrorEsPrice {
			fees, err := h.CalculatePrice.PromotionExtras(sp, sp.ExtraFee, price)
			if err != nil {
				c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
					Error: constant.APIResponseMessageServerInternalError,
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
				Error: constant.APIResponseMessageServerInternalError,
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

		packageIDsCreated, err := h.PackageManager.CreatePackages([]*entity.Package{sp}, userId)
		if err != nil {
			errDetail := strings.Split(cast.ToString(err), ":")
			if errDetail[1] == " Incorrect string value" {
				c.JSON(http.StatusInternalServerError, "Invalid character")
				return
			}

			h.Logger.Errorf("create shipping package: %v", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{Error: constant.APIResponseMessageServerInternalError})

			return
		}

		if len(packageIDsCreated) <= 0 {
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{Error: constant.APIResponseMessageServerInternalError})
			return
		}

		form.ID = sp.ID
		form.CreatedAt = sp.CreatedAt
		form.UpdatedAt = sp.UpdatedAt
		form.Code = ""
		form.ShippingFee = price
		form.TotalCost = price
		form.ExtraFees = []order.ExtraFee{}
		form.Base64Label = ""

		for _, v := range sp.ExtraFee {
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

		form.TotalCost = utils.ToFixed(form.TotalCost, 2)
		form.Status = constant.MapTextStatusCustomerPackage[constant.PackageStatusCreated]
		form.Service = ""
		form.Recipient = ""
		form.FullName = sp.Recipient

		c.JSON(http.StatusOK, CreateResponse{Package: form})
	}
}
