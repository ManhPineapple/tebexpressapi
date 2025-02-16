package ibblue

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/array"
	"time"

	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

type IBBlue struct {
	Client          *Client
	Weight          float64    `json:"weight"`
	WeightUnit      string     `json:"weight_unit"`
	ImageFormat     string     `json:"image_format"`
	ImageResolution int        `json:"image_resolution"`
	Dimensions      Dimensions `json:"dimensions"`
	DimensionsUnit  string     `json:"dimensions_unit"`
	Address         *USPSAddress
	Usps            *USPS
}

func NewIBBlue(opts *Options) *IBBlue {
	if opts == nil {
		opts = &Options{
			BaseURL:         viper.GetString("provider.ibblue.base_url"),
			APIKey:          viper.GetString("provider.ibblue.api_key"),
			WeightUnit:      viper.GetString("provider.ibblue.weight_unit"),
			ImageFormat:     viper.GetString("provider.ibblue.image_format"),
			ImageResolution: viper.GetInt("provider.ibblue.image_resolution"),
			DimensionsUnit:  viper.GetString("provider.ibblue.dimensions_unit"),

			Address: &USPSAddress{
				Company:    viper.GetString("provider.ibblue.company_name"),
				City:       viper.GetString("provider.ibblue.city_locality"),
				Line1:      viper.GetString("provider.ibblue.address_line1"),
				State:      viper.GetString("provider.ibblue.state_province"),
				PostalCode: viper.GetString("provider.ibblue.postal_code"),
				Country:    viper.GetString("provider.ibblue.country_code"),
				Phone:      viper.GetString("provider.ibblue.phone"),
				Email:      viper.GetString("provider.ibblue.email"),
			},
			Usps: &USPS{
				Shape:                viper.GetString("provider.ibblue.shape"),
				MailClass:            viper.GetString("provider.ibblue.mail_class"),
				MailClassPriority:    viper.GetString("provider.ibblue.mail_class_priority"),
				MailClassParcel:      viper.GetString("provider.ibblue.mail_class_parcel"),
				ImageSize:            viper.GetString("provider.ibblue.image_size"),
				GdeOriginCountryCode: viper.GetString("provider.ibblue.gde_origin_country_code"),
			},
		}
	}

	return &IBBlue{
		Client:          NewClient(opts.BaseURL, opts.APIKey),
		WeightUnit:      opts.WeightUnit,
		DimensionsUnit:  opts.DimensionsUnit,
		ImageFormat:     opts.ImageFormat,
		ImageResolution: opts.ImageResolution,
		Address:         opts.Address,
		Usps:            opts.Usps,
	}
}

func (m *IBBlue) USPSCreateLabel(in LabelRequest) (*USPSResponse, string, error) {

	if in.PackageID <= 0 {
		return nil, "", errors.New("Package ID cannot empty")
	}

	firstname := in.FullName
	lastname := " "

	if in.Address1 == "" && in.Address2 != "" {
		in.Address1 = in.Address2
	}

	// Convert cm -> in
	if in.DistanceUnit == "in" {
		in.Width = in.Width * CentimeterToInches
		in.Length = in.Length * CentimeterToInches
		in.Height = in.Height * CentimeterToInches
	}

	postmarkDate := time.Now().Add(144 * time.Hour)

	if in.PostmarkDate > 0 {
		postmarkDate = time.Now().Add(time.Duration(in.PostmarkDate) * 24 * time.Hour)
	}

	req := USPSRequest{
		RequestID:   fmt.Sprintf("%v-%v", in.PackageID, time.Now().Unix()),
		OrderNumber: fmt.Sprintf("#%v", in.PackageID),
		Usps: USPS{
			Shape:                m.Usps.Shape,
			MailClass:            m.Usps.MailClass,
			ImageSize:            m.Usps.ImageSize,
			GdeOriginCountryCode: m.Usps.GdeOriginCountryCode,
		},
		Metadata: Metadata{
			Rubberstamp1: fmt.Sprintf("%s-%s", in.Code, in.WarehouseState),
		},
		ToAddress: USPSAddress{
			Company:    in.Company,
			FirstName:  firstname,
			LastName:   lastname,
			City:       in.City,
			Line1:      in.Address1,
			Line2:      in.Address2,
			State:      in.State,
			PostalCode: in.Zipcode,
			Phone:      in.Phone,
			Country:    in.Country,
		},
		WeightUnit:      m.WeightUnit,
		ImageFormat:     m.ImageFormat,
		ImageResolution: m.ImageResolution,
		DimensionsUnit:  m.DimensionsUnit,
		Weight:          in.Weight,
		PostmarkDate:    fmt.Sprintf("%sT%s", postmarkDate.Format("2006-01-02"), postmarkDate.Format("15:04:05")),
		Dimensions: Dimensions{
			Length: in.Length,
			Width:  in.Width,
			Height: in.Height,
		},
	}

	if in.OrderNumber != "" {
		req.Metadata.Rubberstamp2 = fmt.Sprintf("%v", in.OrderNumber)
	}

	// Nếu không truyền thông tin ware house -> lấy thông tin ware house mặc định
	if in.WarehouseAddress1 == "" {
		req.FromAddress = USPSAddress{
			Company:    m.Address.Company,
			City:       m.Address.City,
			Line1:      m.Address.Line1,
			State:      m.Address.State,
			PostalCode: m.Address.PostalCode,
			Country:    m.Address.Country,
			Phone:      m.Address.Phone,
			Email:      m.Address.Email,
		}
	} else {
		req.FromAddress = USPSAddress{
			Company:    in.WarehouseCompany,
			City:       in.WarehouseCity,
			Line1:      in.WarehouseAddress1,
			State:      in.WarehouseState,
			PostalCode: in.WarehouseZipcode,
			Country:    in.WarehouseCountry,
			Phone:      in.WarehousePhone,
			Email:      m.Address.Email,
		}
	}

	isChangeClass := m.CheckChangeClass(in)
	// Choose Package shipping
	if isChangeClass {
		req.Usps.MailClass = m.Usps.MailClassPriority
	}

	logRequest, _ := json.Marshal(req)
	log.Println(string(logRequest))

	var response interface{}
	err := m.Client.Post("/v1/labels", req, &response)
	if err != nil {
		return nil, "", err
	}

	result := &USPSResponse{}
	b, err := json.Marshal(response)
	if err != nil {
		return nil, "", err
	}

	err = json.Unmarshal(b, result)
	if err != nil {
		return nil, "", err
	}

	responseError := &ErrorResponse{}
	if result == nil || result.RequestID == "" {
		b, err := json.Marshal(response)
		if err != nil {
			return nil, "", err
		}

		err = json.Unmarshal(b, responseError)
		if err != nil {
			return nil, "", err
		}

		return nil, responseError.Error(), nil
	}

	return result, "", nil
}

func (m *IBBlue) DeleteLabel(trackingNumber string) (string, error) {
	var response interface{}
	err := m.Client.Delete(fmt.Sprintf("/v1/labels/%s", trackingNumber), nil, &response)
	if err != nil {
		return "", err
	}

	if response == nil {
		return "", nil
	}

	rerr := &ErrorResponse{}
	b, err := json.Marshal(response)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(b, rerr)
	if err != nil {
		return "", err
	}

	if rerr == nil {
		return "", nil
	}

	return rerr.Error(), nil
}

func (m *IBBlue) TrackingInfo(trackingNumber string) ([]*TrackingResponse, error) {
	var response interface{}
	err := m.Client.Get(fmt.Sprintf("/v1/track/%s", trackingNumber), &response, nil)
	if err != nil {
		return nil, err
	}
	// text := []byte(`[
	// {
	// 	"status": "Delivered, In/At Mailbox",
	// 	"location": {
	// 	  "city": "PASADENA",
	// 	  "state": "CA",
	// 	  "zip": "91107",
	// 	  "country": "USA"
	// 	},
	// 	"timestamp": "2023-02-03T18:46:00.000"
	// },
	// {
	// 	"status": "Out for Delivery",
	// 	"location": {
	// 	  "city": "PASADENA",
	// 	  "state": "CA",
	// 	  "zip": "91107",
	// 	  "country": "USA"
	// 	},
	// 	"timestamp": "2023-02-03T07:10:00.000"
	// },
	// {
	// 	"status": "Arrived at Unit",
	// 	"location": {
	// 	  "city": "PASADENA",
	// 	  "state": "CA",
	// 	  "zip": "91109",
	// 	  "country": "USA"
	// 	},
	// 	"timestamp": "2023-02-03T04:03:00.000"
	// },
	// {
	// 	"status": "Departed USPS Regional Facility",
	// 	"location": {
	// 	  "city": "SANTA CLARITA CA DISTRIBUTION CENTER",
	// 	  "state": null,
	// 	  "zip": null,
	// 	  "country": "USA"
	// 	},
	// 	"timestamp": "2023-02-02T16:44:00.000"
	// },
	// {
	// 	"status": "In Transit to Next Facility",
	// 	"location": {
	// 	  "city": null,
	// 	  "state": null,
	// 	  "zip": null,
	// 	  "country": "USA"
	// 	},
	// 	"timestamp": null
	// },
	// {
	// 	"status": "Arrived at USPS Regional Facility",
	// 	"location": {
	// 	  "city": "SANTA CLARITA CA DISTRIBUTION CENTER",
	// 	  "state": null,
	// 	  "zip": null,
	// 	  "country": "USA"
	// 	},
	// 	"timestamp": "2023-02-01T11:03:00.000"
	// },
	// {
	// 	"status": "Departed USPS Regional Facility",
	// 	"location": {
	// 	  "city": "SAN FRANCISCO CA DISTRIBUTION CENTER",
	// 	  "state": null,
	// 	  "zip": null,
	// 	  "country": "USA"
	// 	},
	// 	"timestamp": "2023-02-01T02:58:00.000"
	// },
	// {
	// 	"status": "Arrived at USPS Regional Origin Facility",
	// 	"location": {
	// 	  "city": "SAN FRANCISCO CA DISTRIBUTION CENTER",
	// 	  "state": null,
	// 	  "zip": null,
	// 	  "country": "USA"
	// 	},
	// 	"timestamp": "2023-01-31T20:42:00.000"
	// },
	// {
	// 	"status": "Accepted at USPS Origin Facility",
	// 	"location": {
	// 	  "city": "PALO ALTO",
	// 	  "state": "CA",
	// 	  "zip": "94306",
	// 	  "country": "USA"
	// 	},
	// 	"timestamp": "2023-01-31T19:27:00.000"
	// },
	// {
	// 	"status": "Shipping Label Created, USPS Awaiting Item",
	// 	"location": {
	// 	  "city": "PALO ALTO",
	// 	  "state": "CA",
	// 	  "zip": "94306",
	// 	  "country": "USA"
	// 	},
	// 	"timestamp": "2023-01-30T06:20:00.000"
	// },
	// {
	// 	"status": "Pre-Shipment Info Sent to USPS, USPS Awaiting Item",
	// 	"location": {
	// 	  "city": null,
	// 	  "state": null,
	// 	  "zip": null,
	// 	  "country": "USA"
	// 	},
	// 	"timestamp": null
	// }
	// ]`)

	result := []*TrackingResponse{}
	// err = json.Unmarshal(text, &result)
	// if err != nil {
	// 	return nil, err
	// }
	b, err := json.Marshal(response)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &result)
	if err != nil {
		return nil, err
	}
	// logger.Log.Infof("response: %v", result)
	return result, nil
}

func (m *IBBlue) CreateManifest(in ManifestRequest) (*CreateManifestResponse, string, error) {

	req := CreateManifestRequest{
		RequestID:       fmt.Sprintf("%v-%v", in.ShipmentID, time.Now().Unix()),
		Usps:            UspsRequest{},
		ImageFormat:     m.ImageFormat,
		ImageResolution: m.ImageResolution,
	}

	for _, trackingNumber := range in.TrackingNumbers {
		req.Usps.TrackingNumbers = append(req.Usps.TrackingNumbers, trackingNumber)
	}

	logRequest, _ := json.Marshal(req)
	log.Println("closeShipment Req Create Manifest:", string(logRequest))

	var response interface{}
	err := m.Client.Post("/v1/manifests.json", req, &response)
	log.Println("closeShipment Req Create Manifest:", err)
	if err != nil {
		return nil, "", err
	}

	result := &CreateManifestResponse{}
	b, err := json.Marshal(response)
	if err != nil {
		return nil, "", err
	}

	err = json.Unmarshal(b, result)
	if err != nil {
		return nil, "", err
	}

	responseError := &ErrorResponse{}
	if result == nil || result.RequestID == "" {
		b, err := json.Marshal(response)
		if err != nil {
			return nil, "", err
		}

		err = json.Unmarshal(b, responseError)
		if err != nil {
			return nil, "", err
		}

		return nil, responseError.Error(), nil
	}

	return result, "", nil
}

func (m *IBBlue) ValidateAddress(req AddressRequest) (*CleanseAddressResponse, string, error) {
	var response interface{}
	err := m.Client.Post("/v1/address/validate", req, &response)
	if err != nil {
		return nil, "", err
	}

	result := &CleanseAddressResponse{}
	b, err := json.Marshal(response)
	if err != nil {
		return nil, "", err
	}

	err = json.Unmarshal(b, result)
	if err != nil {
		return nil, "", err
	}

	responseError := &ErrorResponse{}
	if result == nil {
		b, err := json.Marshal(response)
		if err != nil {
			return nil, "", err
		}

		err = json.Unmarshal(b, responseError)
		if err != nil {
			return nil, "", err
		}

		return nil, responseError.Error(), nil
	}

	return result, "", nil

}

func (m *IBBlue) EstimateCost(req EstimateCostRequest) (*EstimateCostResponse, string, error) {
	req.WeightUnit = m.WeightUnit
	req.ImageFormat = m.ImageFormat
	req.ImageResolution = m.ImageResolution
	req.DimensionsUnit = m.DimensionsUnit

	if req.PostmarkDate == "" {
		postmarkDate := time.Now().Add(144 * time.Hour)
		req.PostmarkDate = fmt.Sprintf("%sT%s", postmarkDate.Format("2006-01-02"), postmarkDate.Format("15:04:05"))
	}

	req.Usps = USPS{
		Shape:                m.Usps.Shape,
		MailClass:            m.Usps.MailClass,
		ImageSize:            m.Usps.ImageSize,
		GdeOriginCountryCode: m.Usps.GdeOriginCountryCode,
	}

	if req.DimensionsUnit == "in" {
		req.Dimensions.Width = req.Dimensions.Width * CentimeterToInches
		req.Dimensions.Length = req.Dimensions.Length * CentimeterToInches
		req.Dimensions.Height = req.Dimensions.Height * CentimeterToInches
	}

	length, height, width := calculate.ParseVolumes(req.Dimensions.Length, req.Dimensions.Width, req.Dimensions.Height)
	lag := calculate.LengthAndGirth(length, height, width)

	isChangeClass := false
	if length > 22 || width > 18 || height > 15 || lag > 108 {
		isChangeClass = true
	}

	// Choose Package shipping
	log.Println("isChangeClass: ", isChangeClass, req.IsExceedPackage)
	log.Println("isChangeClass: ", m.Usps.MailClassPriority, m.Usps.MailClassParcel)
	if isChangeClass {
		req.Usps.MailClass = m.Usps.MailClassPriority
	}

	if req.IsExceedPackage {
		req.Usps.MailClass = m.Usps.MailClassParcel
	}

	if req.FromAddress.Line1 == "" {
		req.FromAddress = USPSAddress{
			Company:    m.Address.Company,
			City:       m.Address.City,
			Line1:      m.Address.Line1,
			State:      m.Address.State,
			PostalCode: m.Address.PostalCode,
			Country:    m.Address.Country,
			Phone:      m.Address.Phone,
		}
	}

	var response interface{}
	a, _ := json.Marshal(req)
	log.Println("hahah:", string(a))
	err := m.Client.Post("/v1/price.json", req, &response)
	if err != nil {
		return nil, "", err
	}

	resErr := &ErrorResponse{}
	b, err := json.Marshal(response)
	if err != nil {
		return nil, "", err
	}

	err = json.Unmarshal(b, resErr)
	if err != nil {
		return nil, "", err
	}

	if resErr != nil && resErr.Code != "" && resErr.Message != "" {
		return nil, resErr.Error(), nil
	}

	result := &EstimateCostResponse{}
	b, err = json.Marshal(response)
	if err != nil {
		return nil, "", err
	}

	err = json.Unmarshal(b, result)
	if err != nil {
		return nil, "", err
	}

	return result, "", nil
}

func (m *IBBlue) USPSCreateLabel2(in LabelRequest) (*USPSResponse, string, error) {

	if in.PackageID <= 0 {
		return nil, "", errors.New("Package ID cannot empty")
	}

	firstname := in.FullName
	lastname := " "

	if in.Address1 == "" && in.Address2 != "" {
		in.Address1 = in.Address2
	}

	// Convert cm -> in
	if in.DistanceUnit == "in" {
		in.Width = in.Width * CentimeterToInches
		in.Length = in.Length * CentimeterToInches
		in.Height = in.Height * CentimeterToInches
	}

	postmarkDate := time.Now().Add(144 * time.Hour)

	if in.PostmarkDate > 0 {
		postmarkDate = time.Now().Add(time.Duration(in.PostmarkDate) * 24 * time.Hour)
	}

	if in.LabelTemplate != TemplateTebexpress && in.LabelTemplate != TemplateTebexpress {
		in.LabelTemplate = TemplateTebexpress
	}

	req := USPSRequest{
		RequestID:    fmt.Sprintf("%v-%v", in.PackageID, time.Now().Unix()),
		TemplateName: "",
		OrderNumber:  fmt.Sprintf("#%v", in.PackageID),
		Usps: USPS{
			Shape:                m.Usps.Shape,
			MailClass:            m.Usps.MailClass,
			ImageSize:            m.Usps.ImageSize,
			GdeOriginCountryCode: m.Usps.GdeOriginCountryCode,
		},
		Metadata: Metadata{
			Rubberstamp1: in.Code,
			Rubberstamp3: fmt.Sprintf("%v", in.DisplayWeight),
		},
		ToAddress: USPSAddress{
			Company:    in.Company,
			FirstName:  firstname,
			LastName:   lastname,
			City:       in.City,
			Line1:      in.Address1,
			Line2:      in.Address2,
			State:      in.State,
			PostalCode: in.Zipcode,
			Phone:      in.Phone,
			Country:    in.Country,
		},
		WeightUnit:      m.WeightUnit,
		ImageFormat:     m.ImageFormat,
		ImageResolution: m.ImageResolution,
		DimensionsUnit:  m.DimensionsUnit,
		Weight:          in.Weight,
		PostmarkDate:    fmt.Sprintf("%sT%s", postmarkDate.Format("2006-01-02"), postmarkDate.Format("15:04:05")),
		Dimensions: Dimensions{
			Length: in.Length,
			Width:  in.Width,
			Height: in.Height,
		},
	}

	if in.OrderNumber != "" {
		req.Metadata.Rubberstamp2 = fmt.Sprintf("%v", in.OrderNumber)
	}

	// Nếu không truyền thông tin ware house -> lấy thông tin ware house mặc định
	if in.WarehouseAddress1 == "" {
		req.FromAddress = USPSAddress{
			Company:    m.Address.Company,
			City:       m.Address.City,
			Line1:      m.Address.Line1,
			State:      m.Address.State,
			PostalCode: m.Address.PostalCode,
			Country:    m.Address.Country,
			Phone:      m.Address.Phone,
		}
	} else {
		req.FromAddress = USPSAddress{
			Company:    in.WarehouseCompany,
			City:       in.WarehouseCity,
			Line1:      in.WarehouseAddress1,
			State:      in.WarehouseState,
			PostalCode: in.WarehouseZipcode,
			Country:    in.WarehouseCountry,
			Phone:      in.WarehousePhone,
		}
	}

	isChangeClass := m.CheckChangeClass(in)
	// Choose Package shipping
	if isChangeClass {
		req.Usps.MailClass = m.Usps.MailClassPriority
	}

	logRequest, _ := json.Marshal(req)
	log.Printf("ibblue request payload: %v", string(logRequest))

	var response interface{}
	err := m.Client.Post("/v1/labels", req, &response)
	if err != nil {
		return nil, "", err
	}

	result := &USPSResponse{}
	b, err := json.Marshal(response)
	if err != nil {
		return nil, "", err
	}

	err = json.Unmarshal(b, result)
	if err != nil {
		return nil, "", err
	}

	responseError := &ErrorResponse{}
	if result == nil || result.RequestID == "" {
		b, err := json.Marshal(response)
		if err != nil {
			return nil, "", err
		}

		err = json.Unmarshal(b, responseError)
		if err != nil {
			return nil, "", err
		}

		return nil, responseError.Error(), nil
	}

	return result, "", nil
}

func (m *IBBlue) CheckChangeClass(in LabelRequest) bool {
	length, height, width := calculate.ParseVolumes(in.Length, in.Width, in.Height)
	lag := calculate.LengthAndGirth(length, height, width)

	if length > 22 || width > 18 || height > 15 || lag > 108 {
		return true
	}

	return false
}

func (m *IBBlue) USPSCreateLabel3(in LabelRequest) (*USPSResponse, string, error) {

	if in.PackageID <= 0 {
		return nil, "", errors.New("Package ID cannot empty")
	}

	firstname := in.FullName
	lastname := " "

	if in.Address1 == "" && in.Address2 != "" {
		in.Address1 = in.Address2
	}

	// Convert cm -> in
	if in.DistanceUnit == "in" {
		in.Width = in.Width * CentimeterToInches
		in.Length = in.Length * CentimeterToInches
		in.Height = in.Height * CentimeterToInches
	}

	postmarkDate := time.Now().Add(144 * time.Hour)

	if in.PostmarkDate > 0 {
		postmarkDate = time.Now().Add(time.Duration(in.PostmarkDate) * 24 * time.Hour)
	}

	if in.LabelTemplate != TemplateTebexpress {
		in.LabelTemplate = TemplateTebexpress
	}

	zipcodeArr := array.SliceStringToSliceInt(strings.Split(viper.GetString("zipcode.check_ibblue"), ","))
	if utils.ContainsNumber(zipcodeArr, cast.ToInt64(in.WarehouseZipcode)) {
		in.HubStateCode += "2"
	}

	req := USPSRequest{
		RequestID:    fmt.Sprintf("%v-%v", in.PackageID, time.Now().Unix()),
		TemplateName: in.LabelTemplate,
		OrderNumber:  fmt.Sprintf("#%v", in.PackageID),
		Usps: USPS{
			Shape:                m.Usps.Shape,
			MailClass:            m.Usps.MailClass,
			ImageSize:            m.Usps.ImageSize,
			GdeOriginCountryCode: m.Usps.GdeOriginCountryCode,
		},
		Metadata: Metadata{
			Rubberstamp1: in.Code,
			Rubberstamp3: fmt.Sprintf("%v", in.DisplayWeight),
			Rubberstamp4: fmt.Sprintf("%v%v", in.ServiceCode, in.HubStateCode),
		},
		ToAddress: USPSAddress{
			Company:    in.Company,
			FirstName:  firstname,
			LastName:   lastname,
			City:       in.City,
			Line1:      in.Address1,
			Line2:      in.Address2,
			State:      in.State,
			PostalCode: in.Zipcode,
			Phone:      in.Phone,
			Country:    in.Country,
		},
		WeightUnit:      m.WeightUnit,
		ImageFormat:     m.ImageFormat,
		ImageResolution: m.ImageResolution,
		DimensionsUnit:  m.DimensionsUnit,
		Weight:          in.Weight,
		PostmarkDate:    fmt.Sprintf("%sT%s", postmarkDate.Format("2006-01-02"), postmarkDate.Format("15:04:05")),
		Dimensions: Dimensions{
			Length: in.Length,
			Width:  in.Width,
			Height: in.Height,
		},
	}

	if in.DomesticCarrierService == constant.DomesticCarrierServiceGround {
		req.Usps.MailClass = m.Usps.MailClass
	}

	if in.DomesticCarrierService == constant.DomesticCarrierServicePriority {
		req.Usps.MailClass = m.Usps.MailClassPriority
	}

	if in.OrderNumber != "" {
		req.Metadata.Rubberstamp2 = fmt.Sprintf("%v", in.OrderNumber)
	}

	// Nếu không truyền thông tin ware house -> lấy thông tin ware house mặc định
	if in.WarehouseAddress1 == "" {
		req.FromAddress = USPSAddress{
			Company:    m.Address.Company,
			City:       m.Address.City,
			Line1:      m.Address.Line1,
			State:      m.Address.State,
			PostalCode: m.Address.PostalCode,
			Country:    m.Address.Country,
			Phone:      m.Address.Phone,
		}
	} else {
		req.FromAddress = USPSAddress{
			Company:    in.WarehouseCompany,
			City:       in.WarehouseCity,
			Line1:      in.WarehouseAddress1,
			State:      in.WarehouseState,
			PostalCode: in.WarehouseZipcode,
			Country:    in.WarehouseCountry,
			Phone:      in.WarehousePhone,
		}
	}

	isChangeClass := m.CheckChangeClass(in)
	// Choose Package shipping
	if isChangeClass {
		req.Usps.MailClass = m.Usps.MailClassPriority
	}

	if in.IsExceedPkg {
		req.Usps.MailClass = m.Usps.MailClassParcel
	}

	logRequest, _ := json.Marshal(req)
	log.Printf("ibblue request payload: %v", string(logRequest))

	var response interface{}
	err := m.Client.Post("/v1/labels", req, &response)
	if err != nil {
		return nil, "", err
	}

	result := &USPSResponse{}
	b, err := json.Marshal(response)
	if err != nil {
		return nil, "", err
	}

	err = json.Unmarshal(b, result)
	if err != nil {
		return nil, "", err
	}

	responseError := &ErrorResponse{}
	if result == nil || result.RequestID == "" {
		b, err := json.Marshal(response)
		if err != nil {
			return nil, "", err
		}

		err = json.Unmarshal(b, responseError)
		if err != nil {
			return nil, "", err
		}

		return nil, responseError.Error(), nil
	}

	return result, "", nil
}

func (m *IBBlue) USPSUpdateLabel(in LabelRequest) (*USPSResponse, string, error) {

	if len(in.TrackingNumber) < 1 {
		return nil, "", errors.New("Tracking number cannot empty")
	}

	// Convert cm -> in
	if in.DistanceUnit == "in" {
		in.Width = in.Width * CentimeterToInches
		in.Length = in.Length * CentimeterToInches
		in.Height = in.Height * CentimeterToInches
	}

	postmarkDate := time.Now().Add(144 * time.Hour)

	if in.PostmarkDate > 0 {
		postmarkDate = time.Now().Add(time.Duration(in.PostmarkDate) * 24 * time.Hour)
	}

	req := UpdateRequest{
		WeightUnit:     m.WeightUnit,
		DimensionsUnit: m.DimensionsUnit,
		Weight:         in.Weight,
		Dimensions: Dimensions{
			Length: in.Length,
			Width:  in.Width,
			Height: in.Height,
		},
		PostmarkDate: fmt.Sprintf("%sT%s", postmarkDate.Format("2006-01-02"), postmarkDate.Format("15:04:05")),
	}

	logRequest, _ := json.Marshal(req)
	log.Printf("ibblue update request payload: %v", string(logRequest))

	var response interface{}
	err := m.Client.Put(fmt.Sprintf("/v1/labels/%s", in.TrackingNumber), req, &response)
	if err != nil {
		return nil, "", err
	}

	result := &USPSResponse{}
	b, err := json.Marshal(response)
	if err != nil {
		return nil, "", err
	}

	err = json.Unmarshal(b, result)
	if err != nil {
		return nil, "", err
	}

	responseError := &ErrorResponse{}
	if result == nil || result.RequestID == "" {
		b, err := json.Marshal(response)
		if err != nil {
			return nil, "", err
		}

		err = json.Unmarshal(b, responseError)
		if err != nil {
			return nil, "", err
		}

		return nil, responseError.Error(), nil
	}

	return result, "", nil
}
