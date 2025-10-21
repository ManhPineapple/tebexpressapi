package admin

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/createlabel"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/order"
	"tebexpressapi/pkg/providers"
	"tebexpressapi/pkg/providers/auspost"
	"tebexpressapi/pkg/providers/ibblue"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/string_util"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func (h *PackageHandler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		packageID := cast.ToInt64(c.Param("package_id"))

		if role == constant.UserRoleSupport || role == constant.UserRoleSale {
			ok, err := h.PackageManager.CheckPermissionUserPackages(userID, []int64{packageID})
			if err != nil {
				h.Logger.Errorf("check permission %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			if !ok {
				c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
				return
			}
		}

		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		if packageID < 1 {
			c.JSON(http.StatusBadRequest, "Invalid order id")
			return
		}

		rKey := fmt.Sprintf("%s_%d", constant.RedisKeyPackageCheckExists, packageID)

		val, err := h.Redis.SetNX(c, rKey, "value", constant.RedisKeyPackageCheckExistsExp).Result()

		if err != nil {
			h.Logger.Errorf("SetNX Redis : %s", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
		}

		if !val {
			h.Logger.Errorf("Update package error : %s", err)
			c.JSON(http.StatusBadRequest, "Vui lòng thử lại sau")
			return
		}

		defer h.Redis.Del(c, rKey)

		currentPackage, err := h.PackageManager.GetPackageByPackageID(packageID)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, constant.MessageNotFound)
			return
		}

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Detail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		isPackageReturn := false
		if currentPackage.Alert == constant.PackageAlertTypeHubReturn {
			isPackageReturn = true
		}

		if currentPackage.Status == constant.PackageStatusCancelled || (currentPackage.Status == constant.PackageStatusDelivered && !isPackageReturn) || currentPackage.Status == constant.PackageStatusExpired || currentPackage.Status == constant.PackageStatusWareHouseInContainer || currentPackage.Status == constant.PackageStatusWareHouseInShipment || currentPackage.Status == constant.PackageStatusWareHouseExport {
			c.JSON(http.StatusBadRequest, "Bạn không được sửa đơn ở trạng thái này")
			return
		}

		if currentPackage.Service.Code == constant.ServiceFBACode || currentPackage.Service.Code == constant.ServiceFastFBACode {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Service %s không được hỗ trợ", currentPackage.Service.Name))
			return
		}

		customer, err := h.UserManager.GetUserByID(currentPackage.UserID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		UpdateForm := &UpdateForm{}
		if err := c.ShouldBindJSON(UpdateForm); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		UpdateForm.OrderNumber = currentPackage.OrderNumber

		if isPackageReturn {
			UpdateForm = h.getFormPackageReturn(UpdateForm, currentPackage)
		}

		service, err := h.ServiceManager.GetServiceByKeyword(UpdateForm.Service)
		if err != nil {
			h.Logger.Errorf("get service: %v", err)
			c.JSON(http.StatusBadRequest, "Dịch vụ không hợp lệ")
			return
		}

		// validate updateform
		if currentPackage.CustomTiktokBarcode == nil {
			if UpdateForm.Recipient = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(UpdateForm.Recipient); UpdateForm.Recipient == "" {
				c.JSON(http.StatusBadRequest, "Tên người nhận không để trống")
				return
			}

			UpdateForm.PhoneNumber = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(UpdateForm.PhoneNumber)
			var regexPhone = regexp.MustCompile("^[0-9 +()-]*$")
			if UpdateForm.PhoneNumber != "" && !regexPhone.MatchString(UpdateForm.PhoneNumber) {
				c.JSON(http.StatusBadRequest, "Số điện thoại người nhận không đúng format")
				return
			}

			if UpdateForm.Service == "" {
				c.JSON(http.StatusBadRequest, "Dịch vụ không được trống ")
				return
			}

			if service.Code == constant.ServiceFBACode || service.Code == constant.ServiceFastFBACode {
				c.JSON(http.StatusBadRequest, fmt.Sprintf("Dịch vụ %s không được hỗ trợ", service.Name))
				return
			}

			if UpdateForm.Address1 = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(UpdateForm.Address1); UpdateForm.Address1 == "" {
				c.JSON(http.StatusBadRequest, "Địa chỉ người nhận không để trống")
				return
			} else if service.Code != constant.ServiceFBACode && service.Code != constant.ServiceFastFBACode {
				if len(UpdateForm.Address1) > 200 {
					c.JSON(http.StatusBadRequest, "Địa chỉ người nhận không được vượt quá 200 ký tự")
					return
				}
			}

			UpdateForm.Address2 = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(UpdateForm.Address2)
			if UpdateForm.Address2 != "" && len(UpdateForm.Address2) > 200 {
				c.JSON(http.StatusBadRequest, "Địa chỉ người nhận phụ không được vượt quá 200 ký tự")
				return
			}

			UpdateForm.CountryCode = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(UpdateForm.CountryCode)
			if UpdateForm.CountryCode == "" {
				c.JSON(http.StatusBadRequest, "Mã quốc gia không để trống")
				return
			}

			UpdateForm.CountryCode = strings.ToUpper(UpdateForm.CountryCode)
			if UpdateForm.CountryCode == "AU" || UpdateForm.CountryCode == "AUSTRALIA" {
				UpdateForm.CountryCode = "AU"
			} else {
				if UpdateForm.CountryCode == "UNITED STATES" {
					UpdateForm.CountryCode = "US"
				}

				if UpdateForm.CountryCode != "US" {
					c.JSON(http.StatusBadRequest, "Mã quốc gia chỉ chấp nhận US(United States) hoặc AU(Australia)")
					return
				}
			}

			if service.Country != UpdateForm.CountryCode {
				c.JSON(http.StatusBadRequest, fmt.Sprintf("Dịch vụ không hỗ trợ %v", UpdateForm.CountryCode))
				return
			}

			UpdateForm.City = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(UpdateForm.City)
			if UpdateForm.City == "" {
				c.JSON(http.StatusBadRequest, "Thành phố không để trống")
				return
			} else {
				if len(UpdateForm.City) > 50 {
					c.JSON(http.StatusBadRequest, "Thành phố không được vượt quá 50 ký tự")
					return
				}
			}

			var stateCode string

			UpdateForm.StateCode = strings.ToUpper(UpdateForm.StateCode)

			UpdateForm.StateCode = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(UpdateForm.StateCode)
			if UpdateForm.StateCode == "" {
				c.JSON(http.StatusBadRequest, "Mã vùng không để trống")
				return
			} else {
				state, err := h.StateManager.GetState(sqlmanager.StateOption{Country: UpdateForm.CountryCode, Code: UpdateForm.StateCode})
				if err == gorm.ErrRecordNotFound {
					c.JSON(http.StatusBadRequest, "Mã vùng không hợp lệ")
					return
				}
				if err != nil {
					h.Logger.Errorf("Error get list state US: %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}
				if state.Status != constant.StatusActive {
					c.JSON(http.StatusBadRequest, "Mã vùng không hỗ trợ ship")
					return
				}

				stateCode = state.Code
			}

			if stateCode != "" {
				UpdateForm.StateCode = stateCode
			}

			UpdateForm.Zipcode = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(UpdateForm.Zipcode)
			if UpdateForm.Zipcode == "" {
				c.JSON(http.StatusBadRequest, "Mã bưu điện không để trống")
				return
			} else {
				if len(UpdateForm.Zipcode) > 15 {
					c.JSON(http.StatusBadRequest, "Mã bưu điện không được vượt quá 15 ký tự")
					return
				}
			}

			var zipcodeRegex = regexp.MustCompile("^[0-9-]*$")
			if !zipcodeRegex.MatchString(UpdateForm.Zipcode) {
				c.JSON(http.StatusBadRequest, "Mã bưu điện không đúng format")
				return
			}

			if UpdateForm.Detail = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(UpdateForm.Detail); UpdateForm.Detail == "" {
				c.JSON(http.StatusBadRequest, "Chi tiết hàng hóa không để trống")
				return
			}

			UpdateForm.Sku = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(UpdateForm.Sku)
			if UpdateForm.Sku == "" {
				c.JSON(http.StatusBadRequest, "Mã đơn hàng không để trống")
				return
			} else {
				if len(UpdateForm.Sku) > 200 {
					c.JSON(http.StatusBadRequest, "Mã đơn hàng không được vượt quá 200 ký tự")
					return
				}
			}

			UpdateForm.Detail = strings.TrimSpace(UpdateForm.Detail)
			if utils.InvalidTag(UpdateForm.Detail) || len(UpdateForm.Detail) > 1000 {
				c.JSON(http.StatusBadRequest, "Chi tiết hàng hóa không được vượt quá 1000 ký tự")
				return
			}

			UpdateForm.Note = strings.TrimSpace(UpdateForm.Note)
			if utils.InvalidTag(UpdateForm.Note) || len(UpdateForm.Note) > 1000 {
				c.JSON(http.StatusBadRequest, "Yêu cầu không được vượt quá 1000 ký tự")
				return
			}

			if UpdateForm.CountryCode == "AU" {
				if len(UpdateForm.Recipient) > auspost.MaxLengthFullName {
					c.JSON(http.StatusBadRequest, fmt.Sprintf("Mã đơn hàng không được vượt quá %d ký tự", auspost.MaxLengthFullName))
					return
				}

				if len(UpdateForm.Address1) > auspost.MaxLengthAddress {
					c.JSON(http.StatusBadRequest, fmt.Sprintf("Địa chỉ người nhận không được vượt quá %d ký tự", auspost.MaxLengthAddress))
					return
				}

				if len(UpdateForm.Address2) > auspost.MaxLengthAddress {
					c.JSON(http.StatusBadRequest, fmt.Sprintf("Địa chỉ người nhận phụ không được vượt quá %d ký tự", auspost.MaxLengthAddress))
					return
				}

				if len(UpdateForm.City) > auspost.MaxLengthCity {
					c.JSON(http.StatusBadRequest, fmt.Sprintf("Thành phố không được vượt quá %d ký tự", auspost.MaxLengthCity))
					return
				}

				if len(UpdateForm.Zipcode) != auspost.MaxLengthPostcode {
					c.JSON(http.StatusBadRequest, fmt.Sprintf("Mã bưu điện phải có độ dài %d ký tự.", auspost.MaxLengthPostcode))
					return
				}
			}

			if service.Code == constant.ServiceFBACode || service.Code == constant.ServiceFastFBACode {
				if UpdateForm.PhoneNumber == "" {
					c.JSON(http.StatusBadRequest, "Số điện thoại đơn FBA là bắt buộc")
					return
				}

				if len(UpdateForm.PhoneNumber) < 10 || len(UpdateForm.PhoneNumber) > 15 {
					c.JSON(http.StatusBadRequest, "Số điện thoại nhập từ 10 đến 15 ký tự")
					return
				}

				if len(UpdateForm.Address1) > 35 {
					c.JSON(http.StatusBadRequest, "Địa chỉ bắt buộc phải nhập và có độ dài không quá 35 ký tự")
					return
				}

			}
		}

		//update package audit log
		var logs []entity.PackageAuditLog
		var cnPriceUpdate []entity.PackageAuditLog
		mapchange := make(map[string]interface{})
		var hasupdateprice bool
		var hasupdatelabel bool
		var hasupdateadd bool
		var hasupdateservice bool

		if UpdateForm.Recipient != currentPackage.Recipient {
			hasupdatelabel = true
			mapchange["recipient"] = UpdateForm.Recipient
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.Recipient,
				Value:    UpdateForm.Recipient,
				Type:     constant.PackageUpdateTypeRecipient,
			})
		}

		if UpdateForm.PhoneNumber != currentPackage.PhoneNumber {
			hasupdatelabel = true
			mapchange["phone_number"] = UpdateForm.PhoneNumber
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.PhoneNumber,
				Value:    UpdateForm.PhoneNumber,
				Type:     constant.PackageUpdateTypePhoneNumber,
			})
		}

		if UpdateForm.Address1 != currentPackage.Address1 {
			hasupdatelabel = true
			hasupdateadd = true
			mapchange["address_1"] = UpdateForm.Address1
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.Address1,
				Value:    UpdateForm.Address1,
				Type:     constant.PackageUpdateTypeAddress,
			})
		}

		if UpdateForm.Address2 != currentPackage.Address2 {
			hasupdateadd = true
			mapchange["address_2"] = UpdateForm.Address2
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.Address2,
				Value:    UpdateForm.Address2,
				Type:     constant.PackageUpdateTypeAddress2,
			})
		}

		if UpdateForm.City != currentPackage.City {
			hasupdateadd = true
			hasupdatelabel = true
			mapchange["city"] = UpdateForm.City
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.City,
				Value:    UpdateForm.City,
				Type:     constant.PackageUpdateTypeCity,
			})
		}

		if UpdateForm.StateCode != currentPackage.StateCode {
			hasupdateadd = true
			hasupdatelabel = true
			mapchange["state_code"] = UpdateForm.StateCode
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.StateCode,
				Value:    UpdateForm.StateCode,
				Type:     constant.PackageUpdateTypeStateCode,
			})
		}

		if UpdateForm.Zipcode != currentPackage.Zipcode {
			hasupdateadd = true
			hasupdatelabel = true
			mapchange["zipcode"] = UpdateForm.Zipcode
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.Zipcode,
				Value:    UpdateForm.Zipcode,
				Type:     constant.PackageUpdateTypeZipcode,
			})
		}

		if UpdateForm.CountryCode != currentPackage.CountryCode {
			hasupdatelabel = true
			hasupdateadd = true
			hasupdateprice = true
			mapchange["country_code"] = UpdateForm.CountryCode
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.CountryCode,
				Value:    UpdateForm.CountryCode,
				Type:     constant.PackageUpdateTypeCountryCode,
			})
		}

		if service.ID != currentPackage.ServiceID {
			hasupdateprice = true
			hasupdatelabel = true
			hasupdateservice = true
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
			if UpdateForm.Status != 0 && UpdateForm.Status != currentPackage.Status {
				mapchange["status"] = UpdateForm.Status
			}
			if UpdateForm.CNProductLink != currentPackage.CNProductLink {
				mapchange["cn_product_link"] = UpdateForm.CNProductLink
				logs = append(logs, entity.PackageAuditLog{
					OldValue: currentPackage.CNProductLink,
					Value:    UpdateForm.CNProductLink,
					Type:     constant.PackageUpdateTypeCNLabel,
				})
			}

			if UpdateForm.CNProductPrice != currentPackage.CNProductPrice {
				hasupdateprice = true
				mapchange["cn_product_price"] = UpdateForm.CNProductPrice
				logs = append(logs, entity.PackageAuditLog{
					OldValue: fmt.Sprintf("%v", currentPackage.CNProductPrice),
					Value:    fmt.Sprintf("%v", UpdateForm.CNProductPrice),
					Type:     constant.PackageUpdateExtraFeeCNProduct,
				})
				cnPriceUpdate = append(cnPriceUpdate, entity.PackageAuditLog{
					OldValue: fmt.Sprintf("%v", currentPackage.CNProductPrice),
					Value:    fmt.Sprintf("%v", UpdateForm.CNProductPrice),
					Type:     constant.PackageUpdateExtraFeeCNProduct,
				})
			}

			if UpdateForm.CNShippingFee != currentPackage.CNShippingFee {
				hasupdateprice = true
				mapchange["cn_shipping_fee"] = UpdateForm.CNShippingFee
				logs = append(logs, entity.PackageAuditLog{
					OldValue: fmt.Sprintf("%v", currentPackage.CNShippingFee),
					Value:    fmt.Sprintf("%v", UpdateForm.CNShippingFee),
					Type:     constant.PackageUpdateExtraFeeCNShipping,
				})
				cnPriceUpdate = append(cnPriceUpdate, entity.PackageAuditLog{
					OldValue: fmt.Sprintf("%v", currentPackage.CNShippingFee),
					Value:    fmt.Sprintf("%v", UpdateForm.CNShippingFee),
					Type:     constant.PackageUpdateExtraFeeCNShipping,
				})
			}

			if UpdateForm.CustomCNBarcode != nil {
				if currentPackage.CustomCNBarcode == nil || *UpdateForm.CustomCNBarcode != *currentPackage.CustomCNBarcode {
					oldValue := ""
					if currentPackage.CustomCNBarcode != nil {
						oldValue = *currentPackage.CustomCNBarcode
					}

					newValue := *UpdateForm.CustomCNBarcode
					mapchange["custom_cn_barcode"] = newValue

					fmt.Printf("customcnbarcode: %v-%v", oldValue, newValue)

					logs = append(logs, entity.PackageAuditLog{
						OldValue: oldValue,
						Value:    newValue,
						Type:     constant.PackageUpdateTypeCNLabel,
					})
				}
			}

			if UpdateForm.CNShippingToVNFee != nil && *UpdateForm.CNShippingToVNFee != 0 {
				hasupdateprice = true
				cnPriceUpdate = append(cnPriceUpdate, entity.PackageAuditLog{
					Value: fmt.Sprintf("%v", *UpdateForm.CNShippingToVNFee),
					Type:  constant.PackageUpdateExtraFeeCNShippingToVN,
				})
			}

			if UpdateForm.CNLabelExtraFee != nil && *UpdateForm.CNLabelExtraFee != 0 {
				hasupdateprice = true
				cnPriceUpdate = append(cnPriceUpdate, entity.PackageAuditLog{
					Value: fmt.Sprintf("%v", *UpdateForm.CNLabelExtraFee),
					Type:  constant.PackageUpdateExtraFeeCNLabel,
				})
			}
		}

		if UpdateForm.Sku != currentPackage.OrderNumber {
			hasupdatelabel = true
			mapchange["order_number"] = UpdateForm.Sku
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.OrderNumber,
				Value:    UpdateForm.Sku,
				Type:     constant.PackageUpdateTypeOrderNumber,
			})
		}

		if UpdateForm.Detail != currentPackage.Detail {
			hasupdatelabel = true
			mapchange["detail"] = UpdateForm.Detail
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.Detail,
				Value:    UpdateForm.Detail,
				Type:     constant.PackageUpdateTypeDetail,
			})
		}

		if currentPackage.Status == constant.PackageStatusCreated || currentPackage.Status == constant.PackageStatusCNPurchased {
			if UpdateForm.Weight != currentPackage.Weight {
				hasupdateprice = true
				mapchange["weight"] = UpdateForm.Weight
			}

			if UpdateForm.Width != currentPackage.Width {
				hasupdateprice = true
				mapchange["width"] = UpdateForm.Width
			}

			if UpdateForm.Length != currentPackage.Length {
				hasupdateprice = true
				mapchange["length"] = UpdateForm.Length
			}

			if UpdateForm.Height != currentPackage.Height {
				hasupdateprice = true
				mapchange["height"] = UpdateForm.Height
			}
		}

		if UpdateForm.PackageName != currentPackage.PackageName {
			mapchange["package_name"] = UpdateForm.PackageName
		}

		if UpdateForm.PackageQuantity != currentPackage.PackageQuantity {
			mapchange["package_quantity"] = UpdateForm.PackageQuantity
		}

		if UpdateForm.TotalProductPrice != currentPackage.TotalProductPrice {
			mapchange["total_product_price"] = UpdateForm.TotalProductPrice
		}

		if UpdateForm.CustomCNBarcode != currentPackage.CustomCNBarcode {
			mapchange["custom_cn_barcode"] = UpdateForm.CustomCNBarcode
		}
		var pkgHasProd []*entity.PackageProducts
		UpdateForm.Weight = utils.Ceil(UpdateForm.Weight, 2)
		UpdateForm.Length = utils.Ceil(UpdateForm.Length, 2)
		UpdateForm.Width = utils.Ceil(UpdateForm.Width, 2)
		UpdateForm.Height = utils.Ceil(UpdateForm.Height, 2)

		if UpdateForm.CountryCode == "AU" {
			_, _, ww := calculate.ParseVolumes(UpdateForm.Length, UpdateForm.Height, UpdateForm.Width)
			if ww < 5 {
				c.JSON(http.StatusBadRequest, "Tối thiểu 2 kích thước phải là 5 cm.")
				return
			}
		}

		var price float64 = 0
		var priceOutSize float64 = 0
		var priceByWeight bool = true

		if UpdateForm.CountryCode == "AU" {
			currentPackage.IsPackageExceed = false
		}

		var isErrorEsPrice bool
		if hasupdateprice || currentPackage.IsPackageExceed {

			_, priceByWeight = calculate.CalcPriceWeight(UpdateForm.Weight, UpdateForm.Length, UpdateForm.Height, UpdateForm.Width, service.ID)

			serviceIDToCalculatePrice := utils.GetServiceIDToCalculatePrice(currentPackage.CustomTiktokBarcode, service)
			price, priceOutSize, err = h.CalculatePrice.Price3(c, currentPackage.UserID, serviceIDToCalculatePrice, customer.Class, UpdateForm.Weight, UpdateForm.Length, UpdateForm.Height, UpdateForm.Width, currentPackage.CountryCode)
			if err == calculate.ErrorNotService {
				if service.Code != constant.ServiceCNCode {
					c.JSON(http.StatusBadRequest, "Dịch vụ không hợp lệ")
					return
				} else {
					err = nil
				}
			}

			if service.Code == constant.ServiceCNCode {
				err := h.PackageManager.UpdateExtraFee(currentPackage.ID, 0, userID, cnPriceUpdate)
				if err != nil {
					h.Logger.Errorf("Update cn extra fee err: %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}
			}

			if service.Code == constant.ServiceLABELCode {
				h.Logger.Info("LABEL CODE: ", price)
				cost, _, err := h.EstimateCost(c, currentPackage)
				if err == nil {
					price = cost + price/100*cost
				}
				h.Logger.Info("LABEL CODE: ", price, cost, err)
			}

			if err != nil && err != calculate.ErrorMaxWeight && err != calculate.ErrorMaxVolume {
				h.Logger.Errorf("parse body: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			// if UpdateForm.CountryCode == "AU" || service.Code == constant.ServiceFBACode || service.Code != constant.ServiceFastFBACode {
			if err == calculate.ErrorMaxWeight {
				msg := "Trọng lượng cho phép vượt quá giới hạn"
				if price > 0 {
					msg = fmt.Sprintf("Trọng lượng không được vượt quá %v grams", math.Ceil(price)-1)
				}

				c.JSON(http.StatusBadRequest, msg)
				return
			}

			if err == calculate.ErrorMaxVolume {
				msg := "Kích thước vượt quá giới hạn cho phép"
				if price > 0 {
					msg = fmt.Sprintf("Kích thước không hợp lệ (LxHxW/5 <= %v)", math.Ceil(price)-1)
				}

				c.JSON(http.StatusBadRequest, msg)
				return
			}

			if (service.Code == constant.ServiceFBACode || service.Code == constant.ServiceFastFBACode) && currentPackage.IsPackageExceed {
				mapchange["is_package_exceed"] = false
			}
		}

		if UpdateForm.IsReship {
			price = currentPackage.ShippingFee
		}

		if isErrorEsPrice && currentPackage.Status > constant.PackageStatusCreated {
			c.JSON(http.StatusInternalServerError, "Đơn quá cỡ tính giá thất bại")
			return
		}

		//isPackageExceed = false,price = 0 skip
		if (price > 0 && price != currentPackage.ShippingFee) || isErrorEsPrice {
			mapchange["shipping_fee"] = price
		}

		if currentPackage.PackageCode == nil {
			hasupdatelabel = false
		}

		carrier := providers.NewCarrier(service.DomesticCarrier.Code, currentPackage.UserID)
		if carrier == nil {
			h.Logger.Errorf("New carrier service %s not found", currentPackage.Service.DomesticCarrier.Code)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		extraFee, err := h.PackageManager.GetExtraFeeByPkgID(packageID)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Deliver Logs %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		oldTotalAMount := currentPackage.ShippingFee
		for _, v := range extraFee {
			oldTotalAMount += v.Amount
		}

		labelType := createlabel.LabelTypeUpdate
		rKeyLabel := "package_call_label"

		if UpdateForm.IsReship {
			goto checkReship
		}

		if currentPackage.LabelPromotion && currentPackage.Status > constant.PackageStatusCreated {
			if hasupdateservice && currentPackage.CountryCode == UpdateForm.CountryCode {
				c.JSON(http.StatusInternalServerError, "Đơn hàng không được phép thay đổi dịch vụ")
				return
			}

			if hasupdateadd {
				h.Logger.Errorf("Create new label disabled")
				c.JSON(http.StatusBadRequest, "Create new label disabled")
				return
			}

			if (hasupdateadd || hasupdateservice) && currentPackage.CustomTiktokBarcode == nil {
				hasupdatelabel = false
				isCallLabel, err := h.Redis.SIsMember(c, rKeyLabel, currentPackage.ID).Result()
				if err != nil {
					h.Logger.Errorf("Get redis error %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				if isCallLabel {
					c.JSON(http.StatusBadRequest, "Đơn hàng đã đang được tạo label, thử lại sau !")
					return
				}

				err = h.Redis.SAdd(c, rKeyLabel, currentPackage.ID).Err()
				if err != nil {
					h.Logger.Errorf("Save redis error: %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				defer h.Redis.SRem(c, rKeyLabel, currentPackage.ID)

				labelType = createlabel.LabelTypeNew

				currentPackage.Service = service

				input := currentPackage
				input.Service = service
				input.OrderNumber = UpdateForm.Sku
				input.Recipient = UpdateForm.Recipient
				input.PhoneNumber = UpdateForm.PhoneNumber
				input.Address1 = UpdateForm.Address1
				input.Address2 = UpdateForm.Address2
				input.City = UpdateForm.City
				input.StateCode = UpdateForm.StateCode
				input.Zipcode = UpdateForm.Zipcode
				input.CountryCode = UpdateForm.CountryCode
				input.Weight = UpdateForm.Weight

				if !currentPackage.IsPackageExceed {
					var carrierCode string
					var carrier providers.Carrier = nil
					dbcarrier := &currentPackage.Service.DomesticCarrier

					if input.CountryCode == "AU" {
						carrier = providers.NewCarrier(dbcarrier.Code, input.UserID)
					} else {
						carrierCode, err = h.CreateLabel.GetCarrierCode(c, *input, input.UserID, "")
						if err != nil {
							h.Logger.Errorf("parse carrier code: %v", err)
							c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
							return
						}

						if carrierCode != "" {
							carrier = providers.NewCarrier(carrierCode, input.UserID)
						} else {
							carrier = providers.NewCarrier(dbcarrier.Code, input.UserID)
						}
					}

					if carrier == nil {
						c.JSON(http.StatusBadRequest, "Invalid carrier")
						return
					}

					if carrierCode != dbcarrier.Code && carrierCode != "" {
						dbcarrier, err = h.ServiceManager.GetCarrierByCode(carrierCode)
						if err != nil {
							c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
							return
						}
					}

					_, err = h.EstimateCostPkg(c, input, UpdateForm, carrier, dbcarrier)
					if err != nil {
						h.Redis.SRem(c, rKeyLabel, currentPackage.ID).Err()
						h.Logger.Errorf("Estimate Cost Error %v", err)
						c.JSON(http.StatusInternalServerError, err.Error())
						return
					}
				}

				warehouses, err := h.WareHouseManager.GetEstimateCostByWareHouse(currentPackage.ID)

				if err != nil && err != gorm.ErrRecordNotFound {
					h.Redis.SRem(c, rKeyLabel, currentPackage.ID).Err()
					h.Logger.Errorf("Get warehouse error:, %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				minOb := warehouses[0]

				for _, wh := range warehouses {
					if wh.Cost < minOb.Cost {
						minOb = wh
					}
				}

				if currentPackage.Tracking != nil && currentPackage.Tracking.TrackingNumber != "" {
					if err := h.cancelLabel(currentPackage); err != nil {
						msg := err.Error()
						if !strings.Contains(msg, "Message already in refund process or refunded") && !strings.Contains(msg, "Refund with this Transaction already exists") {
							h.Logger.Errorf("cancel label %v", err)
							c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
							return
						}
					}

				}
				//check is white list user
				isWlUser := false
				wL := strings.Split(viper.GetString("white_list.label"), ",")
				if len(wL) > 0 {
					for _, email := range wL {
						if email == customer.Email {
							isWlUser = true
						}
					}
				}

				tracking, message, err := h.label(c, input, carrier, minOb.Warehouse, labelType, isWlUser, service.DomesticCarrier.Code, minOb.Zone)

				_ = h.Redis.SRem(c, rKeyLabel, currentPackage.ID).Err()
				if message != "" {
					c.JSON(http.StatusBadRequest, message)
					return
				}

				if err != nil {
					h.Logger.Errorf("create label %v", err)
					if strings.Contains(err.Error(), "Insufficient funds") || strings.Contains(err.Error(), "You are required to have a valid payment method on file to purchase labels") {
						c.JSON(http.StatusBadRequest, "Đã hết tiền trong ví dịch vụ không thể tạo lại label !")
						return
					}
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				ext := "png"
				if carrier.GetCode() == providers.CarrierTypeDarius {
					ext = "pdf"
				}

				path, err := order.StoreLabelS3(h.StorageS3, tracking.LabelURL, ext, tracking.TrackingNumber)
				if err != nil {
					h.Logger.Errorf("store label: %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				tracking.LabelURL = path
				tracking.UserID = userID
				tracking.Version = viper.GetString("tracking_version")

				if tracking.CarrierID < 1 {
					tracking.CarrierID = service.DomesticCarrierID
				}

				billID, err := h.BillManager.GetOrCreateNowBillID(currentPackage.UserID)
				if err != nil {
					h.Logger.Errorf("Get now bill %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}
				err = h.TrackingManager.UpdateTrackingAdmin(billID, tracking, currentPackage, userID, oldTotalAMount)
				if err != nil {
					h.Logger.Errorf("update tracking %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}
				// if order.CheckIsManifestNow() {
				// 	_ = order.SendQueueManifest(h.Producer, []int64{currentPackage.ID}, true)
				// }
			}
		}

		// labelpath := ""
		// if hasupdatelabel && currentPackage.CustomTiktokBarcode == nil {
		// 	input := currentPackage
		// 	input.Service = service
		// 	input.OrderNumber = UpdateForm.Sku
		// 	input.Recipient = UpdateForm.Recipient
		// 	input.Address1 = UpdateForm.Address1
		// 	input.Address2 = UpdateForm.Address2
		// 	input.City = UpdateForm.City
		// 	input.StateCode = UpdateForm.StateCode
		// 	input.CountryCode = UpdateForm.CountryCode
		// 	input.Zipcode = UpdateForm.Zipcode
		// 	input.Weight = UpdateForm.Weight
		// 	_, labelpath, err = label.CreateLabel(input, h.SettingManager, h.StorageS3)
		// 	if err != nil {
		// 		h.Logger.Error("Create label error", err)
		// 	}
		// }

		// if labelpath != "" && labelpath != currentPackage.Label {
		// 	mapchange["label"] = labelpath
		// }

		if UpdateForm.CountryCode == "AU" {
			mapchange["validate_address"] = constant.PackageValidAddress
			mapchange["is_package_exceed"] = false
		}

		if currentPackage.Status != constant.PackageStatusCreated && currentPackage.Status != constant.PackageStatusCNPurchased {
			billID, err := h.BillManager.GetOrCreateNowBillID(currentPackage.UserID)
			if err != nil {
				h.Logger.Errorf("get now bill id: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			err = h.PackageManager.SaveUpdatePackageAdmin(currentPackage.ID, userID, mapchange, pkgHasProd, logs, priceOutSize, priceByWeight, utils.Int64(billID))
			if err != nil {
				errDetail := strings.Split(cast.ToString(err), ":")
				if errDetail[1] == " Incorrect string value" {
					c.JSON(http.StatusInternalServerError, "Kí tự không hợp lệ")
					return
				}
				h.Logger.Errorf("update change package: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		} else {
			mapchange["include_battery"] = UpdateForm.IncludeBattery
			var exFee []entity.ExtraFee
			var cPrice float64 = price
			if !isErrorEsPrice {
				if priceOutSize > 0 {
					exFee = []entity.ExtraFee{{Amount: priceOutSize, ExtraFeeTypeID: constant.ExtraFeeTypeOutSize}}
				}
				if !currentPackage.IsPackageExceed && !hasupdateprice {
					cPrice = currentPackage.ShippingFee
				}
				fees, err := h.CalculatePrice.PromotionExtras(currentPackage, exFee, cPrice)
				if err != nil {
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}
				if len(fees) > 0 {
					exFee = append(exFee, fees...)
				}
			}

			if UpdateForm.IncludeBattery {
				fee := h.CalculatePrice.GetExtraFeeBaterry()
				exFee = append(exFee, entity.ExtraFee{
					PackageID:      &currentPackage.ID,
					Amount:         fee,
					ExtraFeeTypeID: constant.ExtraFeeTypeBattery,
					Status:         constant.ExtraFeeStatusEnable,
				})
			}

			extraFeeService := h.CalculatePrice.GetServiceExtraFeee(UpdateForm.Width, UpdateForm.Height, UpdateForm.Length, *service)
			if extraFeeService > 0 {
				exFee = append(exFee, entity.ExtraFee{
					Amount:         extraFeeService,
					PackageID:      utils.Int64(currentPackage.ID),
					ExtraFeeTypeID: constant.ExtraFeeTypeService,
					Status:         constant.ExtraFeeStatusEnable,
				})
			}

			err = h.PackageManager.SaveUpdatePackage2(currentPackage.ID, userID, mapchange, logs, priceOutSize, exFee, constant.PackageStatusCreated)

			if err != nil {
				errDetail := strings.Split(cast.ToString(err), ":")
				if errDetail[1] == " Incorrect string value" {
					c.JSON(http.StatusInternalServerError, "Kí tự không hợp lệ")
					return
				}
				h.Logger.Errorf("update change package: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		}

		if hasupdatelabel {
			err = h.ShipmentEstimateCost.Handle(c, []int64{currentPackage.ID}, 0)
			if err != nil {
				log.Printf("Error publish message queue shipment estimate cost: %v", err)
			}
		}

	checkReship:
		options := sqlmanager.PackageQueryOption{
			ID: packageID,
		}

		pkg, err := h.PackageManager.GetPackageDetail(options)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, constant.MessageNotFound)
			return
		}

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Detail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		deliverLogs, err := h.PackageManager.GetDeliverLogsByPkgID(packageID)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Deliver Logs %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		//relabel

		if UpdateForm.IsReship {
			if UpdateForm.CountryCode != pkg.CountryCode {
				c.JSON(http.StatusBadRequest, "Mã quốc gia mới không được khác mã quốc gia cũ")
				return
			}

			user, err := h.UserManager.GetUserByID(currentPackage.UserID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			oldTracking := currentPackage.Tracking
			if oldTracking == nil {
				c.JSON(http.StatusBadRequest, "Đơn chưa có tracking")
				return
			}

			carrierDB, err := h.TrackingManager.GetCarrierByID(oldTracking.CarrierID)
			if err != nil {
				h.Logger.Errorf("Get carrier from db error %v", err)
				c.JSON(http.StatusBadRequest, "Loại vận chuyển không hợp lệ")
				return
			}

			warehouse, err := h.WareHouseManager.GetWareHouse(sqlmanager.OptionWareHouse{ID: utils.Int64Value(oldTracking.HubID), IsHasDisable: true})
			if err != nil {
				h.Logger.Errorf("get package detail: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			carrier := providers.NewCarrier(carrierDB.Code, pkg.UserID)
			if carrier == nil {
				c.JSON(http.StatusBadRequest, "Loại vận chuyển không hợp lệ")
				return
			}

			rKey = "package_call_label"
			isCallLabel, err := h.Redis.SIsMember(c, rKey, pkg.ID).Result()
			if err != nil {
				h.Logger.Errorf("Get redis error %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			if isCallLabel {
				c.JSON(http.StatusBadRequest, "Đơn hàng đã đang được tạo label, thử lại sau !")
				return
			}

			err = h.Redis.SAdd(c, rKey, pkg.ID).Err()
			if err != nil {
				h.Logger.Errorf("Save redis error: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			defer h.Redis.SRem(c, rKey, pkg.ID)

			//check is white list user
			isWlUser := false
			wL := strings.Split(viper.GetString("white_list.label"), ",")
			if len(wL) > 0 {
				for _, email := range wL {
					if email == user.Email {
						isWlUser = true
					}
				}
			}

			tracking, msg, err := h.labelReship(c, UpdateForm, pkg, carrier, warehouse, 0, isWlUser, carrierDB.Code, oldTracking.Zone)
			if msg != "" {
				c.JSON(http.StatusBadRequest, msg)
				return
			}

			if err != nil {
				h.Logger.Errorf("create label %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			// Kiểm tra số dư
			amount := utils.Ceil(tracking.ShipmentCost+viper.GetFloat64("extra_fees.reship_fee"), 2)
			if user.Balance+0.01 < amount && (user.UserInfo == nil || user.UserInfo.DebtMaxAmount <= 0) {
				c.JSON(http.StatusBadRequest, "Số dư ví của khách không đủ.")
				return
			}

			if user.Balance+0.01-amount < 0 && user.UserInfo != nil && user.UserInfo.DebtMaxAmount > 0 {
				if err != gorm.ErrRecordNotFound {
					c.JSON(http.StatusBadRequest, constant.MessageServerInternalError)
					return
				}

				if user.Balance+0.01 < 0 && user.UserInfo.DebtTime != nil && user.UserInfo.DebtTime.AddDate(0, 0, user.UserInfo.DebtMaxDay).Before(time.Now()) {
					c.JSON(http.StatusBadRequest, "Tài khoản của khách đã nợ quá thời hạn cho phép.")
					return
				}

				if math.Abs(user.Balance+0.01-amount) > user.UserInfo.DebtMaxAmount {
					c.JSON(http.StatusBadRequest, "Tài khoản của khách đã nợ quá giới hạn cho phép.")
					return
				}
			}

			ext := "png"
			if carrier.GetCode() == providers.CarrierTypeDarius {
				ext = "pdf"
			}

			path, err := order.StoreLabelS3(h.StorageS3, tracking.LabelURL, ext, tracking.TrackingNumber)
			if err != nil {
				h.Logger.Errorf("store label: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			tracking.LabelURL = path
			tracking.UserID = userID
			tracking.Version = viper.GetString("tracking_version")

			if tracking.CarrierID < 1 {
				tracking.CarrierID = carrierDB.ID
			}

			billID, err := h.BillManager.GetOrCreateNowBillID(pkg.UserID)
			if err != nil {
				h.Logger.Errorf("Get now bill %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			if UpdateForm.Description == "" {
				UpdateForm.Description = fmt.Sprintf("Phí reship cho đơn: %s", currentPackage.PackageCode.Code)
			}

			err = h.PackageManager.Reship(pkg.ID, mapchange, tracking, userID, pkg.UserID, billID, amount, UpdateForm.Description, logs)
			if err != nil {
				h.Logger.Errorf("update package %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			// send queue make manifest
			// order.SendQueueManifest(h.Producer, []int64{pkg.ID}, false)
		}

		_, auditLog, _ := utils.CreateOrUpdateVat(pkg, currentPackage.BillID, userID)
		if auditLog == nil {
			h.Logger.Errorf("Update Vat err: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		if auditLog.Value != "0" {
			err = h.PackageManager.UpdateExtraFee(currentPackage.ID, 0, userID, []entity.PackageAuditLog{*auditLog})
		}
		if err != nil {
			h.Logger.Errorf("Update cn extra fee err: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, UpdatePackageResponse{Package: pkg, DeliverLogs: deliverLogs, ExtraFree: extraFee})
	}
}

func (h *PackageHandler) getFormPackageReturn(UpdateForm *UpdateForm, pkg *entity.Package) *UpdateForm {
	// UpdateForm.Recipient = pkg.Recipient
	// UpdateForm.PhoneNumber = pkg.PhoneNumber
	UpdateForm.Sku = pkg.OrderNumber
	UpdateForm.Detail = pkg.Detail
	UpdateForm.Service = pkg.Service.Name
	return UpdateForm
}

func (h *PackageHandler) labelReship(c context.Context, form *UpdateForm, sp *entity.Package, carrier providers.Carrier, warehouse *entity.Warehouse, labelType int, isWlUser bool, oldCarrierCode string, zone int) (*entity.Tracking, string, error) {
	body := providers.RequestCreateLabel{
		ID:           sp.ID,
		OrderNumber:  sp.OrderNumber,
		Code:         sp.PackageCode.Code,
		Company:      sp.Company,
		FirstName:    form.Recipient,
		LastName:     form.Recipient,
		FullName:     form.Recipient,
		City:         form.City,
		Address1:     form.Address1,
		Address2:     form.Address2,
		State:        form.StateCode,
		Zipcode:      form.Zipcode,
		Phone:        form.PhoneNumber,
		Country:      form.CountryCode,
		Weight:       sp.ActualWeight,
		Height:       sp.ActualHeight,
		Length:       sp.ActualLength,
		Width:        sp.ActualWidth,
		DistanceUnit: "in",

		ServiceCode:            sp.Service.Code[0:1],
		DomesticCarrierService: sp.Service.DomesticCarrierService,
		HubStateCode:           warehouse.State,
		DisplayWeight:          sp.Weight,
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
	if labelType == createlabel.LabelTypeNew {
		tracking.HandlingFee = warehouse.HandlingFee

		if sp.LabelPromotion {
			body.LabelTemplate = ibblue.TemplateTebexpress
			if isWlUser {
				body.LabelTemplate = ibblue.TemplateWLTebexpress
			}
		}
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

func (h *PackageHandler) label(c context.Context, sp *entity.Package, carrier providers.Carrier, warehouse *entity.Warehouse, labelType int, isWlUser bool, oldCarrierCode string, zone int) (*entity.Tracking, string, error) {
	body := providers.RequestCreateLabel{
		ID:           sp.ID,
		OrderNumber:  sp.OrderNumber,
		Code:         sp.PackageCode.Code,
		Company:      sp.Company,
		FirstName:    sp.Recipient,
		LastName:     sp.Recipient,
		FullName:     sp.Recipient,
		City:         sp.City,
		Address1:     sp.Address1,
		Address2:     sp.Address2,
		State:        sp.StateCode,
		Zipcode:      sp.Zipcode,
		Phone:        sp.PhoneNumber,
		Country:      sp.CountryCode,
		Weight:       sp.ActualWeight,
		Height:       sp.ActualHeight,
		Length:       sp.ActualLength,
		Width:        sp.ActualWidth,
		DistanceUnit: "in",

		ServiceCode:            sp.Service.Code[0:1],
		DomesticCarrierService: sp.Service.DomesticCarrierService,
		HubStateCode:           warehouse.State,
		DisplayWeight:          sp.Weight,
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
	if labelType == createlabel.LabelTypeNew {
		tracking.HandlingFee = warehouse.HandlingFee

		if sp.LabelPromotion {
			body.LabelTemplate = ibblue.TemplateTebexpress
			if isWlUser {
				body.LabelTemplate = ibblue.TemplateWLTebexpress
			}
		}
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

func (h *PackageHandler) EstimateCostPkg(c context.Context, pkg *entity.Package, UpdateForm *UpdateForm, carrier providers.Carrier, dbcarrier *entity.Carrier) ([]float64, error) {
	var costs []float64
	var msg []string

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
		return nil, err
	}

	if err := h.WareHouseManager.DeactivateOldCost(*pkg); err != nil {
		h.Logger.Errorf("Estimate cost err: %v", err)
		return nil, err
	}

	clone := &entity.Package{}
	if err := utils.DeepCopy(pkg, clone); err != nil {
		h.Logger.Errorf("Estimate cost err: %v", err)
		return nil, err
	}

	clone.Address1 = UpdateForm.Address1
	clone.Address2 = UpdateForm.Address2
	clone.City = UpdateForm.City
	clone.CountryCode = UpdateForm.CountryCode
	clone.StateCode = UpdateForm.StateCode
	clone.Zipcode = UpdateForm.Zipcode

	var wg sync.WaitGroup
	var m sync.Mutex

	for _, wareHouse := range wareHouses {
		wg.Add(1)

		go func(wareHouse entity.Warehouse) {
			defer wg.Done()

			if wareHouse.Country != clone.CountryCode {
				return
			}

			cost := &entity.PackageWarehouseCost{
				PackageID: clone.ID,
				HubID:     wareHouse.ID,
				Warehouse: &wareHouse,
				Cost:      0,
				OrgCost:   0,
			}
			var ApiCost float64

			resultOrg, s, err := order.EstimateCost(carrier, clone, wareHouse)
			if s != "" {
				h.Logger.Errorf("Estimate cost org err: %v", s)
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
				ApiCost = resultOrg.TotalCost
			}

			if clone.CountryCode == "US" {
				weight, length, height, width, err := h.CreateLabel.Fake(c, clone.ActualWeight, clone.ActualLength, clone.ActualHeight, clone.ActualWidth)
				if err != nil {
					h.Logger.Errorf("fake volume: %v", err)
					return
				}

				if clone.IsPackageExceed {
					weight, length, height, width = clone.Weight, clone.Length, clone.Height, clone.Width
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
				ApiCost = result.TotalCost
				if clone.IsPackageExceed {
					if result.TotalCost > 0 {
						cost.Cost = wareHouse.HandlingFee + result.TotalCost
						cost.OrgCost = cost.Cost
					}
				} else {
					cost.Cost = wareHouse.HandlingFee + result.TotalCost
				}
				cost.Zone = result.Zone
			}

			if dbcarrier != nil {
				cost.CarrierID = dbcarrier.ID
			}

			if err := h.WareHouseManager.CreateEstimateCost(cost); err != nil {
				h.Logger.Errorf("Update estimate cost err: %v", err)
				return
			}

			if wareHouse.Status == constant.WareHouseStatusActive {
				m.Lock()
				costs = append(costs, ApiCost)
				m.Unlock()
			}
		}(wareHouse)

	}

	wg.Wait()

	if len(costs) > 0 {
		return costs, nil
	}

	if len(msg) > 0 {
		return nil, errors.New(msg[0])
	}

	return nil, errors.New("can't find lowest cost warehouse")
}

func (h *PackageHandler) cancelLabel(sp *entity.Package) error {
	if sp.Tracking == nil || sp.Tracking.TrackingNumber == "" {
		return nil
	}

	carrierDB, err := h.TrackingManager.GetCarrierByID(sp.Tracking.CarrierID)
	if err != nil {
		return err
	}

	carrier := providers.NewCarrier(carrierDB.Code, sp.UserID)
	if carrier == nil {
		return errors.New("carrier is not found")
	}

	_, err = order.CancelLabel(carrier, sp)
	return err
}
