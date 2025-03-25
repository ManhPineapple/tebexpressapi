package order

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/providers/auspost"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils/string_util"

	"strings"
	"tebexpressapi/pkg/utils"
	"time"

	"github.com/spf13/cast"
	"gorm.io/gorm"
)

var maxWeight = constant.PackageMaxWeight / calculate.GramToPound
var maxWeightFBA = 35000
var maxLengthAndGirth = calculate.InchToCM * constant.PackageMaxLengthAndGirth

const DefaultLang = "VI"

type (
	PackageResource struct {
		ID             int64      `json:"id"`
		OrderNumber    string     `json:"order_number"`
		Code           string     `json:"code"`
		FullName       string     `json:"name"`
		Recipient      string     `json:"recipient,omitempty"`
		Company        string     `json:"company"`
		Phone          string     `json:"phone"`
		Address1       string     `json:"address_1"`
		Address2       string     `json:"address_2"`
		City           string     `json:"city"`
		State          string     `json:"state_code"`
		Zipcode        string     `json:"zipcode"`
		Country        string     `json:"country_code"`
		Detail         string     `json:"detail"`
		Weight         float64    `json:"weight"`
		Width          float64    `json:"width"`
		Length         float64    `json:"length"`
		Height         float64    `json:"height"`
		Status         string     `json:"status"`
		IncludeBattery bool       `json:"include_battery"`
		Service        string     `json:"service,omitempty"`
		ServiceCode    string     `json:"service_code"`
		Base64Label    string     `json:"-,omitempty"`
		CreatedAt      time.Time  `json:"created_at"`
		UpdatedAt      time.Time  `json:"updated_at"`
		TotalCost      float64    `json:"total_cost,omitempty"`
		ShippingFee    float64    `json:"shipping_fee,omitempty"`
		ExtraFees      []ExtraFee `json:"extra_fees,omitempty"`

		PackageProducts []*PackageProduct `json:"package_products"`
		// For tiktok label
		CustomTiktokBarcode string `json:"custom_tiktok_barcode"`
		IsEarlyScan         bool   `json:"is_early_scan,omitempty"`
		// For customs (Hải quan) check
		PackageName       string  `json:"package_name"`
		PackageQuantity   int64   `json:"package_quantity"`
		TotalProductPrice float64 `json:"product_price"`
		// For CN Service
		CNIsPurchased   bool    `json:"is_purchased,omitempty"`
		CNProductLink   string  `json:"cn_product_link,omitempty"`
		CNProductPrice  float64 `json:"cn_product_price,omitempty"`
		CNInvoiceImage  string  `json:"cn_invoice_image"`
		CNShippingFee   float64 `json:"cn_shipping_fee,omitempty"`
		CustomCNBarcode *string `json:"custom_cn_barcode,omitempty"`
	}

	PackageProduct struct {
		ID        int64  `json:"id,omitempty"`
		ProductID int64  `json:"product_id,omitempty"`
		SKU       string `json:"sku"`
		Name      string `json:"name"`
		Quantity  int64  `json:"quantity"`
	}

	ExtraFee struct {
		ExtraFeeType string  `json:"extra_fee_type"`
		Amount       float64 `json:"amount"`
		Description  string  `json:"description"`
	}
)

type OrderValidator struct {
	lang         string
	stateManager *sqlmanager.StateManager
	errors       []string
	valueErrors  []string
	err          error
	states       map[string]*entity.State
}

func MakeValidator(stateManager *sqlmanager.StateManager) *OrderValidator {
	return &OrderValidator{
		lang:         DefaultLang,
		stateManager: stateManager,
		errors:       []string{},
		err:          nil,
	}
}

func (v *OrderValidator) Reset() {
	v.lang = DefaultLang
	v.errors = []string{}
	v.valueErrors = []string{}
	v.err = nil
}

func (v *OrderValidator) SetLang(lang string) *OrderValidator {
	v.lang = lang
	return v
}

func (v *OrderValidator) SetStates(states map[string]*entity.State) *OrderValidator {
	v.states = states
	return v
}

func (v *OrderValidator) Errors() []string {
	return v.errors
}

func (v *OrderValidator) ValueErrors() []string {
	return v.valueErrors
}

func (v *OrderValidator) Error() error {
	return v.err
}

func (v *OrderValidator) decode(r *http.Request) *PackageResource {
	form := &PackageResource{}
	if err := json.NewDecoder(r.Body).Decode(form); err != nil {
		v.err = err
		return nil
	}

	return form
}

func (v *OrderValidator) Validate(form *PackageResource) {
	form.Country = strings.ToUpper(string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Country))
	if form.Country == "" {
		v.valueErrors = append(v.valueErrors, form.Country)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The country code is required")
		} else {
			v.errors = append(v.errors, "Mã quốc gia không để trống")
		}
	} else {
		if form.Country == "AU" || form.Country == "AUSTRALIA" {
			form.Country = "AU"
		} else if utils.ContainsString(constant.EUCountries, form.Country) {
			// do nothing
		} else {
			if form.Country != "US" && form.Country != "UNITED STATES" {
				v.valueErrors = append(v.valueErrors, form.Country)

				if v.lang == "EN" {
					v.errors = append(v.errors, "The country code must be US(United States) or AU(Australia)")
				} else {
					v.errors = append(v.errors, "Mã quốc gia chỉ chấp nhận US(United States) hoặc AU(Australia)")
				}
			}

			form.Country = "US"
		}
	}

	form.Recipient = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Recipient)
	if form.Recipient == "" {
		form.Recipient = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.FullName)
	}

	if form.Recipient == "" {
		v.valueErrors = append(v.valueErrors, form.Recipient)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The full name is required")
		} else {
			v.errors = append(v.errors, "Tên người nhận không để trống")
		}
	}

	if form.Country == "AU" && len(form.Recipient) > auspost.MaxLengthFullName {
		v.valueErrors = append(v.valueErrors, form.Recipient)

		if v.lang == "EN" {
			v.errors = append(v.errors, fmt.Sprintf("The full name length should not exceed %d characters", auspost.MaxLengthFullName))
		} else {
			v.errors = append(v.errors, fmt.Sprintf("Tên người nhận không được vượt quá %d ký tự", auspost.MaxLengthFullName))
		}
	}

	form.OrderNumber = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.OrderNumber)
	if form.OrderNumber == "" {
		v.valueErrors = append(v.valueErrors, form.OrderNumber)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The order number is required")
		} else {
			v.errors = append(v.errors, "Mã đơn hàng không để trống")
		}
	}

	if len(form.OrderNumber) > 200 {
		v.valueErrors = append(v.valueErrors, form.OrderNumber)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The order number length should not exceed 200 characters")
		} else {
			v.errors = append(v.errors, "Mã đơn hàng không được vượt quá 200 ký tự")
		}
	}

	form.Phone = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Phone)
	if form.Phone != "" {
		var regexPhone = regexp.MustCompile("^[0-9 +()-]*$")
		if !regexPhone.MatchString(form.Phone) {
			v.valueErrors = append(v.valueErrors, form.Phone)

			if v.lang == "EN" {
				v.errors = append(v.errors, "The phone number is invalid")
			} else {
				v.errors = append(v.errors, "Số điện thoại người nhận không đúng format")
			}
		}
	}

	form.Address1 = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Address1)
	if form.Address1 == "" {
		v.valueErrors = append(v.valueErrors, form.Address1)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The address1 is required")
		} else {
			v.errors = append(v.errors, "Địa chỉ người nhận không để trống")
		}
	}

	maxLengthAddress := 200
	if form.Country == "AU" {
		maxLengthAddress = auspost.MaxLengthAddress
	}

	if form.ServiceCode != constant.ServiceFBACode {
		if len(form.Address1) > maxLengthAddress {
			v.valueErrors = append(v.valueErrors, form.Address1)

			if v.lang == "EN" {
				v.errors = append(v.errors, fmt.Sprintf("The address1 length should not exceed %d characters", maxLengthAddress))
			} else {
				v.errors = append(v.errors, fmt.Sprintf("Địa chỉ người nhận không được vượt quá %d ký tự", maxLengthAddress))
			}
		}
	}

	form.Address2 = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Address2)
	if len(form.Address2) > maxLengthAddress {
		v.valueErrors = append(v.valueErrors, form.Address2)

		if v.lang == "EN" {
			v.errors = append(v.errors, fmt.Sprintf("The address2 length should not exceed %d characters", maxLengthAddress))
		} else {
			v.errors = append(v.errors, fmt.Sprintf("Địa chỉ người nhận phụ không được vượt quá %d ký tự", maxLengthAddress))
		}
	}

	form.City = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.City)
	if form.City == "" {
		v.valueErrors = append(v.valueErrors, form.City)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The city is required")
		} else {
			v.errors = append(v.errors, "Thành phố không để trống")
		}
	}

	maxLengthCity := 50
	if form.Country == "AU" {
		maxLengthCity = auspost.MaxLengthCity
	}
	if len(form.City) > maxLengthCity {
		v.valueErrors = append(v.valueErrors, form.City)

		if v.lang == "EN" {
			v.errors = append(v.errors, fmt.Sprintf("The city length should not exceed %d characters", maxLengthCity))
		} else {
			v.errors = append(v.errors, fmt.Sprintf("Thành phố không được vượt quá %d ký tự", maxLengthCity))
		}
	}

	form.State = strings.ToUpper(string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.State))
	if form.State == "" {
		v.valueErrors = append(v.valueErrors, form.State)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The state code is required")
		} else {
			v.errors = append(v.errors, "Mã vùng không để trống")
		}
	} else {
		stateCode, msg, err := v.ValidState(form.Country, form.State, form.ServiceCode)
		if err != nil {
			v.err = err
		}

		if msg != "" {
			v.valueErrors = append(v.valueErrors, form.State)
			v.errors = append(v.errors, msg)
		}

		form.State = stateCode
	}

	form.Zipcode = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Zipcode)
	if form.Zipcode == "" {
		v.valueErrors = append(v.valueErrors, form.Zipcode)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The zipcode is required")
		} else {
			v.errors = append(v.errors, "Mã bưu điện không để trống")
		}
	}

	var zipcodeRegex = regexp.MustCompile("^[0-9-]*$")
	if !zipcodeRegex.MatchString(form.Zipcode) {
		v.valueErrors = append(v.valueErrors, form.Zipcode)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The zipcode is invalid")
		} else {
			v.errors = append(v.errors, "Mã bưu điện không đúng format")
		}
	}

	maxLengthPostcode := 15
	if form.Country == "AU" && len(form.Zipcode) != auspost.MaxLengthPostcode {
		v.valueErrors = append(v.valueErrors, form.Zipcode)

		if v.lang == "EN" {
			v.errors = append(v.errors, fmt.Sprintf("The postcode should be %d characters in length.", auspost.MaxLengthPostcode))
		} else {
			v.errors = append(v.errors, fmt.Sprintf("Mã bưu điện phải có độ dài %d ký tự.", auspost.MaxLengthPostcode))
		}
	} else if len(form.Zipcode) > maxLengthPostcode {
		v.valueErrors = append(v.valueErrors, form.Zipcode)

		if v.lang == "EN" {
			v.errors = append(v.errors, fmt.Sprintf("The zipcode length should not exceed %d characters", maxLengthPostcode))
		} else {
			v.errors = append(v.errors, fmt.Sprintf("Mã bưu điện không được vượt quá %d ký tự", maxLengthPostcode))
		}
	}

	form.Detail = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Detail)
	if form.Detail == "" {
		v.valueErrors = append(v.valueErrors, form.Detail)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The detail is required")
		} else {
			v.errors = append(v.errors, "Chi tiết sản phẩm không để trống")
		}
	}

	if len(form.Detail) > 1000 {
		v.valueErrors = append(v.valueErrors, form.Detail)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The detail length should not exceed 1000 characters")
		} else {
			v.errors = append(v.errors, "Chi tiết sản phẩm không được vượt quá 1000 ký tự")
		}
	}

	form.Weight = utils.Ceil(form.Weight, 2)
	if form.Weight == 0 {
		v.valueErrors = append(v.valueErrors, cast.ToString(form.Weight))

		if v.lang == "EN" {
			v.errors = append(v.errors, "The weight is required")
		} else {
			v.errors = append(v.errors, "Trọng lượng không để trống")
		}
	}
	if form.Weight < 0 {
		v.valueErrors = append(v.valueErrors, cast.ToString(form.Weight))

		if v.lang == "EN" {
			v.errors = append(v.errors, "The weight is invalid")
		} else {
			v.errors = append(v.errors, "Trọng lượng không hợp lệ")
		}
	}

	form.Width = utils.Ceil(form.Width, 2)
	if form.Width == 0 {
		v.valueErrors = append(v.valueErrors, cast.ToString(form.Width))

		if v.lang == "EN" {
			v.errors = append(v.errors, "The width is required")
		} else {
			v.errors = append(v.errors, "Chiều rộng không để trống")
		}
	}
	if form.Width < 0 {
		v.valueErrors = append(v.valueErrors, cast.ToString(form.Width))

		if v.lang == "EN" {
			v.errors = append(v.errors, "The width is invalid")
		} else {
			v.errors = append(v.errors, "Chiều rộng không hợp lệ")
		}
	}

	form.Length = utils.Ceil(form.Length, 2)
	if form.Length == 0 {
		v.valueErrors = append(v.valueErrors, cast.ToString(form.Length))

		if v.lang == "EN" {
			v.errors = append(v.errors, "The length is required")
		} else {
			v.errors = append(v.errors, "Chiều dài không để trống")
		}
	}

	if form.Length < 0 {
		v.valueErrors = append(v.valueErrors, cast.ToString(form.Length))

		if v.lang == "EN" {
			v.errors = append(v.errors, "The length is invalid")
		} else {
			v.errors = append(v.errors, "Chiều dài không hợp lệ")
		}
	}

	form.Height = utils.Ceil(form.Height, 2)
	if form.Height == 0 {
		v.valueErrors = append(v.valueErrors, cast.ToString(form.Height))

		if v.lang == "EN" {
			v.errors = append(v.errors, "The height is invalid")
		} else {
			v.errors = append(v.errors, "Chiều cao không để trống")
		}
	}
	if form.Height < 0 {
		v.valueErrors = append(v.valueErrors, cast.ToString(form.Height))

		if v.lang == "EN" {
			v.errors = append(v.errors, "The height is invalid")
		} else {
			v.errors = append(v.errors, "Chiều cao không hợp lệ")
		}
	}

	form.Service = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Service)
	if form.Service == "" {
		form.Service = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.ServiceCode)
	}

	if form.Service == "" {
		if v.lang == "EN" {
			v.errors = append(v.errors, "The service code is required")
		} else {
			v.errors = append(v.errors, "Dịch vụ không được trống")
		}
	}
}

func (v *OrderValidator) IsValid() bool {
	return v.err == nil && len(v.errors) < 1
}

func (v *OrderValidator) DecodeAndValidate(r *http.Request, edit bool) *PackageResource {
	form := v.decode(r)
	if v.err != nil {
		return nil
	}

	v.Validate(form)
	return form
}

func (v *OrderValidator) Decode(r *http.Request, edit bool) (*PackageResource, error) {
	form := v.decode(r)
	if v.err != nil {
		return nil, v.err
	}

	form.Country = strings.ToUpper(string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Country))
	if form.Country == "AU" || form.Country == "AUSTRALIA" {
		form.Country = "AU"
	} else if form.Country == "US" || form.Country == "UNITED STATES" {
		form.Country = "US"
	}
	return form, nil
}

func (v *OrderValidator) ValidState(country, stateCode, serviceCode string) (code string, message string, err error) {
	if utils.ContainsString(constant.EUCountries, strings.ToUpper(country)) {
		code = strings.ToUpper(stateCode)
		return
	}

	if len(v.states) > 0 {
		key := fmt.Sprintf("%s-%s", country, stateCode)
		if v.states[key] == nil {
			err = nil
			message = "Mã vùng không hợp lệ"
			if v.lang == "EN" {
				message = "The state code is invalid"
			}

			return
		}

		if v.states[key].Status != constant.StatusActive && serviceCode != constant.ServiceACTUSCode {
			message = "Mã vùng không hỗ trợ ship"
			if v.lang == "EN" {
				message = "The state code no support shipping"
			}
			return
		}

		code = v.states[key].Code
		return
	}

	state, err := v.stateManager.GetState(sqlmanager.StateOption{Country: country, Code: stateCode})
	if err == gorm.ErrRecordNotFound {
		err = nil
		message = "Mã vùng không hợp lệ"
		if v.lang == "EN" {
			message = "The state code is invalid"
		}
		return
	}

	if err != nil {
		fmt.Errorf("get state: %v", err)
		return
	}

	code = state.Code
	if state.Status != constant.StatusActive && serviceCode != constant.ServiceACTUSCode {
		message = "Mã vùng không hỗ trợ ship"
		if v.lang == "EN" {
			message = "The state code no support shipping"
		}
		return
	}

	return
}

func MergeDuplicateProducts(sample []*PackageProduct) []*PackageProduct {
	unique := make(map[int64]*PackageProduct)

	for _, v := range sample {
		if v.ID < 1 || v.Quantity < 1 {
			continue
		}

		if unique[v.ID] == nil {
			unique[v.ID] = v
		} else {
			unique[v.ID].Quantity += v.Quantity
		}
	}

	products := []*PackageProduct{}
	for _, v := range unique {
		products = append(products, v)
	}

	return products
}

func (v *OrderValidator) ValidateVolumes(in *PackageResource) {
	if in.Country == "AU" {
		v.AuValidateVolumes(in)
		return
	}

	v.UsValidateVolumes(in)
}

func UsLengthAndGirth(length, height, width float64) float64 {
	l, h, w := calculate.ParseVolumes(length, height, width)
	return 2*(h+w) + l
}

func UsValidateWeight(in *PackageResource, isEN bool) string {
	max := maxWeight
	if in.ServiceCode == constant.ServiceFBACode {
		max = float64(maxWeightFBA)
	}

	if in.Weight <= max {
		return ""
	}

	max = utils.Floor(max, 2)
	if isEN {
		return fmt.Sprintf("The weight must be less than or equal to %v gram", max)
	}

	return fmt.Sprintf("Trọng lượng không vượt quá %v gram", max)
}

func UsValidateLengthAndGirth(in *PackageResource, isEN bool) string {
	l, h, w := calculate.ParseVolumes(in.Length, in.Height, in.Width)
	if lag := UsLengthAndGirth(l, h, w); lag <= maxLengthAndGirth {
		return ""
	}

	max := utils.Floor(maxLengthAndGirth, 2)
	if isEN {
		return fmt.Sprintf("“2*(weight + width) + length” must be less than or equal to %v cm", max)
	}

	return fmt.Sprintf("“Dài + 2*(cao + rộng)” không vượt quá %v cm", max)
}

func (v *OrderValidator) UsValidateVolumes(in *PackageResource) {
	if in.ServiceCode == constant.ServiceFBACode {
		return
	}
	pw, _ := calculate.CalcPriceWeight(in.Weight, in.Length, in.Height, in.Width, 0)

	var max float64 = constant.PackageMaxLength
	if pw >= calculate.MaxWeightPriceAllow {
		max = constant.PackageMaxLengthOversize
	} else {
		l, _, _ := calculate.ParseVolumes(in.Length, in.Height, in.Width)
		if l > constant.PackageMaxLength {
			max = constant.PackageMaxLengthOversize
		}
	}

	if in.Length > max {
		v.valueErrors = append(v.valueErrors, cast.ToString(in.Length))

		if v.lang == "EN" {
			v.errors = append(v.errors, fmt.Sprintf("The length must be less than or equal to %v cm", max))
		} else {
			v.errors = append(v.errors, fmt.Sprintf("Chiều dài không được vượt quá %v cm", max))
		}
	}

	if in.Height > max {
		v.valueErrors = append(v.valueErrors, cast.ToString(in.Height))

		if v.lang == "EN" {
			v.errors = append(v.errors, fmt.Sprintf("The height must be less than or equal to %v cm", max))
		} else {
			v.errors = append(v.errors, fmt.Sprintf("Chiều cao không được vượt quá %v cm", max))
		}
	}

	if in.Width > max {
		v.valueErrors = append(v.valueErrors, cast.ToString(in.Width))

		if v.lang == "EN" {
			v.errors = append(v.errors, fmt.Sprintf("The width must be less than or equal to %v cm", max))
		} else {
			v.errors = append(v.errors, fmt.Sprintf("Chiều rộng không được vượt quá %v cm", max))
		}
	}

	if msg := UsValidateWeight(in, v.lang == "EN"); msg != "" {
		v.valueErrors = append(v.valueErrors, cast.ToString(in.Weight))
		v.errors = append(v.errors, msg)
	}

	if msg := UsValidateLengthAndGirth(in, v.lang == "EN"); msg != "" {
		v.valueErrors = append(v.valueErrors, fmt.Sprintf("%.2fx%.2fx%.2f", in.Length, in.Height, in.Width))
		v.errors = append(v.errors, msg)
	}
}

func (v *OrderValidator) AuValidateVolumes(in *PackageResource) {
	if in.Weight > auspost.MaxWeight*1000 {
		v.valueErrors = append(v.valueErrors, cast.ToString(in.Weight))

		if v.lang == "EN" {
			v.errors = append(v.errors, fmt.Sprintf("The weight must be less than or equal to %v kg", auspost.MaxWeight))
		} else {
			v.errors = append(v.errors, fmt.Sprintf("Trọng lượng không được vượt quá %v kg", auspost.MaxWeight))
		}
	}

	if in.Width > 105 {
		v.valueErrors = append(v.valueErrors, cast.ToString(in.Width))

		if v.lang == "EN" {
			v.errors = append(v.errors, "The width must be less than or equal to 105 cm")
		} else {
			v.errors = append(v.errors, "Chiều rộng không được vượt quá 105 cm")
		}
	}

	if in.Length > 105 {
		v.valueErrors = append(v.valueErrors, cast.ToString(in.Length))

		if v.lang == "EN" {
			v.errors = append(v.errors, "The length must be less than or equal to 105 cm")
		} else {
			v.errors = append(v.errors, "Chiều dài không được vượt quá 105 cm")
		}
	}

	if in.Height > 105 {
		v.valueErrors = append(v.valueErrors, cast.ToString(in.Height))

		if v.lang == "EN" {
			v.errors = append(v.errors, "The height must be less than or equal to 105 cm")
		} else {
			v.errors = append(v.errors, "Chiều cao không được vượt quá 105 cm")
		}
	}

	_, _, w := calculate.ParseVolumes(in.Length, in.Height, in.Width)
	if w < 5 {
		v.valueErrors = append(v.valueErrors, fmt.Sprintf("%.2fx%.2fx%.2f", in.Length, in.Height, in.Width))

		if v.lang == "EN" {
			v.errors = append(v.errors, "At least 2 dimensions must be 5 cm")
		} else {
			v.errors = append(v.errors, "Tối thiểu 2 kích thước phải là 5 cm")
		}
	}

	if in.Length*in.Height*in.Width > auspost.MaxVolume {
		v.valueErrors = append(v.valueErrors, fmt.Sprintf("%.2fx%.2fx%.2f", in.Length, in.Height, in.Width))

		if v.lang == "EN" {
			v.errors = append(v.errors, "Maximum volume must not exceed 0.25 m3")
		} else {
			v.errors = append(v.errors, "Thể tích không được vượt quá 0.25 m3")
		}
	}
}

func (v *OrderValidator) ValidateFbaServicePackage(in *PackageResource) {

	if in.ServiceCode != constant.ServiceFBACode {
		return
	}

	if in.Phone == "" {
		v.valueErrors = append(v.valueErrors, cast.ToString(in.Phone))
		if v.lang == "EN" {
			v.errors = append(v.errors, "Phone field is required !")
		} else {
			v.errors = append(v.errors, "Số điện thoại là bắt buộc")
		}
	}

	if len(in.Phone) < 10 || len(in.Phone) > 15 {
		v.valueErrors = append(v.valueErrors, cast.ToString(in.Phone))
		if v.lang == "EN" {
			v.errors = append(v.errors, "Phone field must be between 10 and 15 characters !")
		} else {
			v.errors = append(v.errors, "Số điện thoại nhập từ 10 đến 15 ký tự")
		}
	}

	if len(in.Address1) > 35 {
		v.valueErrors = append(v.valueErrors, cast.ToString(in.Address1))
		if v.lang == "EN" {
			v.errors = append(v.errors, "The address is required and address length must be less than or equal to 35 characters !")
		} else {
			v.errors = append(v.errors, "Địa chỉ bắt buộc phải nhập và có độ dài không quá 35 ký tự")
		}
	}

	if in.Weight > constant.PackageFBAMaxWeight*calculate.KgToGram {
		v.valueErrors = append(v.valueErrors, cast.ToString(in.Weight))
		if v.lang == "EN" {
			v.errors = append(v.errors, fmt.Sprintf("The weight must less more than or equal to %v kg", constant.PackageFBAMaxWeight))
		} else {
			v.errors = append(v.errors, fmt.Sprintf("Trọng lượng không lớn hơn %v kg", constant.PackageFBAMaxWeight))
		}
	}

	lag := UsLengthAndGirth(in.Length, in.Height, in.Width)
	if lag > constant.PackageFBAMaxLengthAndGirth {
		v.valueErrors = append(v.valueErrors, fmt.Sprintf("%.2fx%.2fx%.2f", in.Length, in.Height, in.Width))

		if v.lang == "EN" {
			v.errors = append(v.errors, fmt.Sprintf("“2*(weight + width) + length” must be less than or equal to %v cm", constant.PackageFBAMaxLengthAndGirth))
		} else {
			v.errors = append(v.errors, fmt.Sprintf("“Dài + 2*(cao + rộng)” không vượt quá %v cm", constant.PackageFBAMaxLengthAndGirth))
		}
	}

	if in.Width > constant.PackageFBAMaxDimension {
		v.valueErrors = append(v.valueErrors, cast.ToString(in.Width))
		if v.lang == "EN" {
			v.errors = append(v.errors, fmt.Sprintf("The width must be less than or equal to %v cm", constant.PackageFBAMaxDimension))
		} else {
			v.errors = append(v.errors, fmt.Sprintf("Chiều rộng không vượt quá %v cm", constant.PackageFBAMaxDimension))
		}
	}

	if in.Length > constant.PackageFBAMaxDimension {
		v.valueErrors = append(v.valueErrors, cast.ToString(in.Length))
		if v.lang == "EN" {
			v.errors = append(v.errors, fmt.Sprintf("The length must be less than or equal to %v cm", constant.PackageFBAMaxDimension))
		} else {
			v.errors = append(v.errors, fmt.Sprintf("Chiều dài không vượt quá %v cm", constant.PackageFBAMaxDimension))
		}
	}

	if in.Height > constant.PackageFBAMaxDimension {
		v.valueErrors = append(v.valueErrors, cast.ToString(in.Height))
		if v.lang == "EN" {
			v.errors = append(v.errors, fmt.Sprintf("The height must be less than or equal to %v cm", constant.PackageFBAMaxDimension))
		} else {
			v.errors = append(v.errors, fmt.Sprintf("Chiều cao không vượt quá %v cm", constant.PackageFBAMaxDimension))
		}
	}
}

func (v *OrderValidator) ValidateChinaPackage(form *PackageResource) {
	form.Country = strings.ToUpper(string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Country))
	if form.Country == "" {
		v.valueErrors = append(v.valueErrors, form.Country)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The country code is required")
		} else {
			v.errors = append(v.errors, "Mã quốc gia không để trống")
		}
	} else {
		if form.Country == "AU" || form.Country == "AUSTRALIA" {
			form.Country = "AU"
		} else if utils.ContainsString(constant.EUCountries, form.Country) {
			// do nothing
		} else {
			if form.Country != "US" && form.Country != "UNITED STATES" {
				v.valueErrors = append(v.valueErrors, form.Country)

				if v.lang == "EN" {
					v.errors = append(v.errors, "The country code must be US(United States) or AU(Australia)")
				} else {
					v.errors = append(v.errors, "Mã quốc gia chỉ chấp nhận US(United States) hoặc AU(Australia)")
				}
			}

			form.Country = "US"
		}
	}

	form.Recipient = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Recipient)
	if form.Recipient == "" {
		form.Recipient = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.FullName)
	}

	if form.Recipient == "" {
		v.valueErrors = append(v.valueErrors, form.Recipient)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The full name is required")
		} else {
			v.errors = append(v.errors, "Tên người nhận không để trống")
		}
	}

	if form.Country == "AU" && len(form.Recipient) > auspost.MaxLengthFullName {
		v.valueErrors = append(v.valueErrors, form.Recipient)

		if v.lang == "EN" {
			v.errors = append(v.errors, fmt.Sprintf("The full name length should not exceed %d characters", auspost.MaxLengthFullName))
		} else {
			v.errors = append(v.errors, fmt.Sprintf("Tên người nhận không được vượt quá %d ký tự", auspost.MaxLengthFullName))
		}
	}

	form.OrderNumber = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.OrderNumber)
	if form.OrderNumber == "" {
		v.valueErrors = append(v.valueErrors, form.OrderNumber)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The order number is required")
		} else {
			v.errors = append(v.errors, "Mã đơn hàng không để trống")
		}
	}

	if len(form.OrderNumber) > 200 {
		v.valueErrors = append(v.valueErrors, form.OrderNumber)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The order number length should not exceed 200 characters")
		} else {
			v.errors = append(v.errors, "Mã đơn hàng không được vượt quá 200 ký tự")
		}
	}

	form.Phone = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Phone)
	if form.Phone != "" {
		var regexPhone = regexp.MustCompile("^[0-9 +()-]*$")
		if !regexPhone.MatchString(form.Phone) {
			v.valueErrors = append(v.valueErrors, form.Phone)

			if v.lang == "EN" {
				v.errors = append(v.errors, "The phone number is invalid")
			} else {
				v.errors = append(v.errors, "Số điện thoại người nhận không đúng format")
			}
		}
	}

	form.Address1 = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Address1)
	if form.Address1 == "" {
		v.valueErrors = append(v.valueErrors, form.Address1)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The address1 is required")
		} else {
			v.errors = append(v.errors, "Địa chỉ người nhận không để trống")
		}
	}

	maxLengthAddress := 200
	if form.Country == "AU" {
		maxLengthAddress = auspost.MaxLengthAddress
	}

	if form.ServiceCode != constant.ServiceFBACode {
		if len(form.Address1) > maxLengthAddress {
			v.valueErrors = append(v.valueErrors, form.Address1)

			if v.lang == "EN" {
				v.errors = append(v.errors, fmt.Sprintf("The address1 length should not exceed %d characters", maxLengthAddress))
			} else {
				v.errors = append(v.errors, fmt.Sprintf("Địa chỉ người nhận không được vượt quá %d ký tự", maxLengthAddress))
			}
		}
	}

	form.Address2 = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Address2)
	if len(form.Address2) > maxLengthAddress {
		v.valueErrors = append(v.valueErrors, form.Address2)

		if v.lang == "EN" {
			v.errors = append(v.errors, fmt.Sprintf("The address2 length should not exceed %d characters", maxLengthAddress))
		} else {
			v.errors = append(v.errors, fmt.Sprintf("Địa chỉ người nhận phụ không được vượt quá %d ký tự", maxLengthAddress))
		}
	}

	form.City = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.City)
	if form.City == "" {
		v.valueErrors = append(v.valueErrors, form.City)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The city is required")
		} else {
			v.errors = append(v.errors, "Thành phố không để trống")
		}
	}

	maxLengthCity := 50
	if form.Country == "AU" {
		maxLengthCity = auspost.MaxLengthCity
	}
	if len(form.City) > maxLengthCity {
		v.valueErrors = append(v.valueErrors, form.City)

		if v.lang == "EN" {
			v.errors = append(v.errors, fmt.Sprintf("The city length should not exceed %d characters", maxLengthCity))
		} else {
			v.errors = append(v.errors, fmt.Sprintf("Thành phố không được vượt quá %d ký tự", maxLengthCity))
		}
	}

	form.State = strings.ToUpper(string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.State))
	if form.State == "" {
		v.valueErrors = append(v.valueErrors, form.State)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The state code is required")
		} else {
			v.errors = append(v.errors, "Mã vùng không để trống")
		}
	} else {
		stateCode, msg, err := v.ValidState(form.Country, form.State, form.ServiceCode)
		if err != nil {
			v.err = err
		}

		if msg != "" {
			v.valueErrors = append(v.valueErrors, form.State)
			v.errors = append(v.errors, msg)
		}

		form.State = stateCode
	}

	form.Zipcode = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Zipcode)
	if form.Zipcode == "" {
		v.valueErrors = append(v.valueErrors, form.Zipcode)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The zipcode is required")
		} else {
			v.errors = append(v.errors, "Mã bưu điện không để trống")
		}
	}

	var zipcodeRegex = regexp.MustCompile("^[0-9-]*$")
	if !zipcodeRegex.MatchString(form.Zipcode) {
		v.valueErrors = append(v.valueErrors, form.Zipcode)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The zipcode is invalid")
		} else {
			v.errors = append(v.errors, "Mã bưu điện không đúng format")
		}
	}

	maxLengthPostcode := 15
	if form.Country == "AU" && len(form.Zipcode) != auspost.MaxLengthPostcode {
		v.valueErrors = append(v.valueErrors, form.Zipcode)

		if v.lang == "EN" {
			v.errors = append(v.errors, fmt.Sprintf("The postcode should be %d characters in length.", auspost.MaxLengthPostcode))
		} else {
			v.errors = append(v.errors, fmt.Sprintf("Mã bưu điện phải có độ dài %d ký tự.", auspost.MaxLengthPostcode))
		}
	} else if len(form.Zipcode) > maxLengthPostcode {
		v.valueErrors = append(v.valueErrors, form.Zipcode)

		if v.lang == "EN" {
			v.errors = append(v.errors, fmt.Sprintf("The zipcode length should not exceed %d characters", maxLengthPostcode))
		} else {
			v.errors = append(v.errors, fmt.Sprintf("Mã bưu điện không được vượt quá %d ký tự", maxLengthPostcode))
		}
	}

	form.Detail = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Detail)
	if form.Detail == "" {
		v.valueErrors = append(v.valueErrors, form.Detail)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The detail is required")
		} else {
			v.errors = append(v.errors, "Chi tiết sản phẩm không để trống")
		}
	}

	if len(form.Detail) > 1000 {
		v.valueErrors = append(v.valueErrors, form.Detail)

		if v.lang == "EN" {
			v.errors = append(v.errors, "The detail length should not exceed 1000 characters")
		} else {
			v.errors = append(v.errors, "Chi tiết sản phẩm không được vượt quá 1000 ký tự")
		}
	}

	form.Weight = utils.Ceil(form.Weight, 2)
	if form.Weight < 0 {
		v.valueErrors = append(v.valueErrors, cast.ToString(form.Weight))

		if v.lang == "EN" {
			v.errors = append(v.errors, "The weight is invalid")
		} else {
			v.errors = append(v.errors, "Trọng lượng không hợp lệ")
		}
	}

	form.Width = utils.Ceil(form.Width, 2)
	if form.Width < 0 {
		v.valueErrors = append(v.valueErrors, cast.ToString(form.Width))

		if v.lang == "EN" {
			v.errors = append(v.errors, "The width is invalid")
		} else {
			v.errors = append(v.errors, "Chiều rộng không hợp lệ")
		}
	}

	form.Length = utils.Ceil(form.Length, 2)
	if form.Length < 0 {
		v.valueErrors = append(v.valueErrors, cast.ToString(form.Length))

		if v.lang == "EN" {
			v.errors = append(v.errors, "The length is invalid")
		} else {
			v.errors = append(v.errors, "Chiều dài không hợp lệ")
		}
	}

	form.Height = utils.Ceil(form.Height, 2)
	if form.Height < 0 {
		v.valueErrors = append(v.valueErrors, cast.ToString(form.Height))

		if v.lang == "EN" {
			v.errors = append(v.errors, "The height is invalid")
		} else {
			v.errors = append(v.errors, "Chiều cao không hợp lệ")
		}
	}

	form.Service = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Service)
	if form.Service == "" {
		form.Service = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.ServiceCode)
	}

	if form.Service == "" {
		if v.lang == "EN" {
			v.errors = append(v.errors, "The service code is required")
		} else {
			v.errors = append(v.errors, "Dịch vụ không được trống")
		}
	}

	if form.CNProductLink != "" {
		if !isValidURL(form.CNProductLink) {
			if v.lang == "EN" {
				v.errors = append(v.errors, "Invalid CN Product Link. Please enter a valid URL.")
			} else {
				v.errors = append(v.errors, "Link sản phẩm CN không hợp lệ. Vui lòng nhập URL hợp lệ.")
			}
		}
	}
}

func isValidURL(link string) bool {
	parsedURL, err := url.ParseRequestURI(link)
	return err == nil && (parsedURL.Scheme == "http" || parsedURL.Scheme == "https")
}
