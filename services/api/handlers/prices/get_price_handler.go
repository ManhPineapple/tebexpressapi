package prices

import (
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
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/string_util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type PriceHandler struct {
	Logger *zap.SugaredLogger

	CalculatePrice *calculate.CalculatePrice
	PartnerMaps    map[string]int64

	StateManager     *sqlmanager.StateManager
	ServiceManager   *sqlmanager.ServiceManager
	WareHouseManager *sqlmanager.WareHouseManager
}

type GetPriceRequest struct {
	Weight  float64 `json:"weight"`
	Width   float64 `json:"width"`
	Length  float64 `json:"length"`
	Height  float64 `json:"height"`
	Service string  `json:"service"`
	Address string  `json:"address,omitempty"`
	City    string  `json:"city,omitempty"`
	State   string  `json:"state,omitempty"`
	Zipcode string  `json:"zipcode,omitempty"`
	Country string  `json:"country,omitempty"`

	IncludeBattery      bool    `json:"include_battery"`
	IsTradeMark         bool    `json:"is_trade_mark"`
	CustomTiktokBarcode *string `json:"custom_tiktok_label"`
	IsEarlyScan         bool    `json:"is_early_scan"`
	CNProductPrice      float64 `json:"cn_product_price"`
	CNShippingFee       float64 `json:"cn_shipping_fee"`
}

type GetPriceResponse struct {
	Price         float64 `json:"price"`
	TotalExtrafee float64 `json:"total_extra_fee"`
}

func (h *PriceHandler) GetPackagePrice() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		form := &GetPriceRequest{}
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.MessageValidateInput,
				Messages: []string{"Input is invalid"},
			})
			return
		}

		if form.Weight <= 0 {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.MessageValidateInput,
				Messages: []string{"The weight is invalid"},
			})

			return
		}

		if form.Width == 0 {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.MessageValidateInput,
				Messages: []string{"The width is required"},
			})

			return
		}

		if form.Width < 0 {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.MessageValidateInput,
				Messages: []string{"The width is invalid"},
			})

			return
		}

		if form.Length == 0 {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.MessageValidateInput,
				Messages: []string{"The length is required"},
			})

			return
		}

		if form.Length < 0 {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.MessageValidateInput,
				Messages: []string{"The length is invalid"},
			})

			return
		}

		if form.Height == 0 {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.MessageValidateInput,
				Messages: []string{"The height is required"},
			})

			return
		}

		if form.Height < 0 {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.MessageValidateInput,
				Messages: []string{"The height is invalid"},
			})

			return
		}

		if form.Service = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Service); form.Service == "" {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.MessageValidateInput,
				Messages: []string{"Service is required"},
			})

			return
		}

		service, err := h.ServiceManager.GetServiceByCode(form.Service)
		if err != nil {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.MessageValidateInput,
				Messages: []string{"Service is invalid"},
			})

			return
		}

		if form.Country != "" {
			form.Country = strings.ToUpper(form.Country)
			if constant.CountryTransform[form.Country] == "" && !utils.ContainsString(constant.EUCountries, form.Country) {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.MessageValidateInput,
					Messages: []string{"The country code must be US(United States) or AU(Australia)"},
				})

				return
			}

			if constant.CountryTransform[form.Country] != "" {
				form.Country = constant.CountryTransform[form.Country]
			}
			if form.Country != service.Country {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.MessageValidateInput,
					Messages: []string{fmt.Sprintf("The service %s not support country %s", service.Name, form.Country)},
				})

				return
			}
		}

		if form.State != "" {
			form.State = strings.ToUpper(form.State)

			states, err := h.StateManager.GetState(sqlmanager.StateOption{
				Code:    form.State,
				Country: strings.ToUpper(service.Country),
			})

			if err != nil {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.MessageValidateInput,
					Messages: []string{"State is invalid"},
				})

				return
			}

			if states.Status != constant.StatusActive {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.MessageValidateInput,
					Messages: []string{"State is not supported"},
				})

				return
			}
		}

		valid := order.MakeValidator(nil).SetLang("EN")
		valid.ValidateVolumes(&order.PackageResource{
			Country: service.Country,
			Weight:  form.Weight,
			Length:  form.Length,
			Height:  form.Height,
			Width:   form.Width,
		})

		if messages := valid.Errors(); len(messages) > 0 {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.MessageValidateInput,
				Messages: messages,
			})

			return
		}

		form.Weight = math.Ceil(form.Weight*100) / 100
		form.Length = math.Ceil(form.Length*100) / 100
		form.Width = math.Ceil(form.Width*100) / 100
		form.Height = math.Ceil(form.Height*100) / 100

		var serviceIDToCalculatePrice int64
		if form.CustomTiktokBarcode != nil && *form.CustomTiktokBarcode != "" {
			if constant.IsPriorityService(service.Code) {
				serviceIDToCalculatePrice = 28 // tiktok priority price
			} else {
				serviceIDToCalculatePrice = 25 // tiktok price
			}
		} else {
			serviceIDToCalculatePrice = service.ID
		}

		exampleCustomerId := int64(2742)
		price, priceOutSize, err := h.CalculatePrice.Price3(c, exampleCustomerId, serviceIDToCalculatePrice, 1, form.Weight, form.Length, form.Height, form.Width, form.Country)
		if err == calculate.ErrorNotService {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.MessageValidateInput,
				Messages: []string{"Service is invalid"},
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
			cost, err := h.EstimateCostPkg(form, carrier)
			if err == nil {
				price = cost + price/100*cost
			}
			h.Logger.Info("LABEL CODE: ", price, cost, err)
		}

		var isErrorEsPrice bool
		// exceed package
		if err == calculate.ErrorMaxVolume || err == calculate.ErrorMaxWeight {
			// validate
			if form.Address == "" || form.City == "" || form.Country == "" || form.State == "" || form.Zipcode == "" {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.MessageValidateInput,
					Messages: []string{"Address is required"},
				})

				return
			}

			form.Country = strings.ToUpper(form.Country)

			if constant.CountryTransform[form.Country] == "" && !utils.ContainsString(constant.EUCountries, form.Country) {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.MessageValidateInput,
					Messages: []string{"The country code must be US(United States) or AU(Australia)"},
				})

				return
			}

			if constant.CountryTransform[form.Country] != "" {
				form.Country = constant.CountryTransform[form.Country]
			}

			if form.Country == "AU" {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.MessageValidateInput,
					Messages: []string{"The country AU(Australia) not support over size"},
				})

				return
			}

			if form.Country != service.Country {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.MessageValidateInput,
					Messages: []string{fmt.Sprintf("The service %s not support country %s", service.Name, form.Country)},
				})

				return
			}

			form.State = strings.ToUpper(form.State)
			states, err := h.StateManager.GetState(sqlmanager.StateOption{
				Code:    form.State,
				Country: form.Country,
			})

			if err != nil {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.MessageValidateInput,
					Messages: []string{"State is invalid"},
				})

				return
			}

			if states.Status != constant.StatusActive {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.MessageValidateInput,
					Messages: []string{"State is not supported"},
				})

				return
			}

			form.State = states.Code

			carrier := providers.NewCarrier(service.DomesticCarrier.Code, userId)
			if carrier == nil {
				isErrorEsPrice = true
			}

			cost, err := h.EstimateCostPkg(form, carrier)
			if err != nil {
				h.Logger.Errorf("estimate cost: %v", err)
				if strings.Contains(err.Error(), "invalid city") {
					c.JSON(http.StatusOK, httputil.ErrorResponse{
						Error:    constant.MessageValidateInput,
						Messages: []string{"invalid city"},
					})

					return
				}

				isErrorEsPrice = true
			}

			if cost == 0 {
				isErrorEsPrice = true
			}

			if !isErrorEsPrice {
				pw, _ := calculate.CalcPriceWeight(form.Weight, form.Length, form.Height, form.Width, service.ID)
				price, err = h.CalculatePrice.CalculateExceedPackagePrice(pw, cost)

				if err != nil {
					h.Logger.Errorf("calculate: %v", err)
					isErrorEsPrice = true
				}
			}
		}

		if isErrorEsPrice {
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.MessageServerInternalError,
			})
			return
		}

		if err != nil && err != calculate.ErrorMaxVolume && err != calculate.ErrorMaxWeight {
			h.Logger.Errorf("parse body: %v", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.MessageServerInternalError,
			})

			return
		}

		totalExtrafee := 0.0
		if form.IsTradeMark {
			fee := 1.0
			totalExtrafee += fee
		}

		if form.IncludeBattery {
			fee := h.CalculatePrice.GetExtraFeeBaterry()
			totalExtrafee += fee
		}

		extraFeeService := h.CalculatePrice.GetServiceExtraFeee(form.Width, form.Height, form.Length, *service)
		if extraFeeService > 0 {
			totalExtrafee += extraFeeService
		}

		if service.Code == constant.ServiceCNCode {
			if form.CNProductPrice != 0 {
				var cnPricePercentage float64
				if form.CNProductPrice > 200 {
					cnPricePercentage = viper.GetFloat64("extra_fees.cn_high_price_percentage")
				} else {
					cnPricePercentage = viper.GetFloat64("extra_fees.cn_low_price_percentage")
				}
				minProxyPrice := viper.GetFloat64("extra_fees.cn_min_proxy_buying_fee")
				proxyFee := math.Max(form.CNProductPrice*cnPricePercentage, minProxyPrice)
				totalExtrafee += proxyFee
				totalExtrafee += form.CNProductPrice
			} else if form.CNShippingFee != 0 {
				totalExtrafee += form.CNShippingFee
			}

			defaultCNShippingFeeToVN := viper.GetFloat64("extra_fees.default_cn_ship_to_vn_fee")
			totalExtrafee += defaultCNShippingFeeToVN

			defaultCNHandlingFee := viper.GetFloat64("extra_fees.default_cn_handling_fee")
			totalExtrafee += defaultCNHandlingFee
		}

		if service.Code == constant.ServiceWarehouseCode {
			defaultWsHandlingFee := viper.GetFloat64("extra_fees.default_ws_handling_fee")

			if form.CustomTiktokBarcode != nil && *form.CustomTiktokBarcode != "" {
				totalExtrafee += defaultWsHandlingFee
			}
		}

		tiktokEarlyScanFee := viper.GetFloat64("extra_fees.default_tiktok_early_scan_fee")
		if form.IsEarlyScan {
			totalExtrafee += tiktokEarlyScanFee
		}

		totalPrice := price + priceOutSize
		c.JSON(http.StatusOK, GetPriceResponse{Price: totalPrice, TotalExtrafee: totalExtrafee})
	}
}

func (h *PriceHandler) EstimateCostPkg(pkg *GetPriceRequest, carrier providers.Carrier) (float64, error) {
	wareHouses, err := h.WareHouseManager.GetWareHouses(sqlmanager.OptionWareHouse{
		Type:   constant.WareHouseTypeInternational,
		Status: constant.WareHouseStatusActive,
	})

	if err != nil {
		h.Logger.Errorf("Get estimate warehouses error, %v", err)
		return 0, err
	}

	defer func() {
		if r := recover(); r != nil {
			h.Logger.Errorf("Get estimate warehouses error, %v", r)
		}
	}()

	var pkgWhs []entity.PackageWarehouseCost
	var wg sync.WaitGroup
	var Errs []string
	var m sync.Mutex
	apiCost := make(map[int64]float64, 0)
	for _, wareHouse := range wareHouses {

		wg.Add(1)

		go func(wareHouse entity.Warehouse) {
			defer wg.Done()

			if wareHouse.Country != pkg.Country {
				return
			}

			cost := entity.PackageWarehouseCost{
				HubID:     wareHouse.ID,
				Warehouse: &wareHouse,
				OrgCost:   0,
			}

			sp := &entity.Package{
				Weight:          pkg.Weight,
				Width:           pkg.Width,
				Length:          pkg.Length,
				Height:          pkg.Height,
				Address1:        pkg.Address,
				City:            pkg.City,
				StateCode:       pkg.State,
				Zipcode:         pkg.Zipcode,
				CountryCode:     pkg.Country,
				Recipient:       "Ananbay",
				IsPackageExceed: true,
				Service: &entity.Service{
					DomesticCarrierService: "",
				},
			}

			resultOrg, s, err := order.EstimateCost(carrier, sp, wareHouse)
			if s != "" {
				Errs = append(Errs, s)
				h.Logger.Errorf("Estimate cost org err: %v", s)
				return
			}

			if err != nil {
				Errs = append(Errs, err.Error())
				h.Logger.Errorf("Estimate cost org err: %v", err)
				return
			}

			cost.OrgCost = resultOrg.TotalCost + wareHouse.HandlingFee
			cost.Zone = resultOrg.Zone
			if wareHouse.Status == constant.WareHouseStatusActive {
				m.Lock()
				pkgWhs = append(pkgWhs, cost)
				apiCost[wareHouse.ID] = resultOrg.TotalCost
				m.Unlock()
			}
		}(wareHouse)
	}

	wg.Wait()

	if len(Errs) > 0 {
		return 0, errors.New(Errs[0])
	}

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
