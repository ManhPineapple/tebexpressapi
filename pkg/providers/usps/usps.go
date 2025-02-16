package usps

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/spf13/viper"
)

type Options struct {
	Domain      string
	TrackingURL string
	APIKey      string

	Name                        string
	Phone                       string
	AddressLine1                string
	CityLocality                string
	StateProvince               string
	PostalCode                  string
	CountryCode                 string
	AddressResidentialIndicator string

	WeightUnit    string
	DimensionUnit string
	ServiceCode   string
	CarrierID     string
}

type USPS struct {
	Domain      string
	TrackingURL string
	ShipFrom    *Address
	APIKey      string

	ServiceCode   string
	WeightUnit    string
	DimensionUnit string
	CarrierID     string
}

// New new a USPS service
func New(opts *Options) *USPS {
	if opts == nil {
		opts = &Options{
			Domain:                      viper.GetString("provider.usps.domain"),
			APIKey:                      viper.GetString("provider.usps.api_key"),
			TrackingURL:                 viper.GetString("provider.usps.tracking_url"),
			Name:                        viper.GetString("provider.usps.name"),
			Phone:                       viper.GetString("provider.usps.phone"),
			AddressLine1:                viper.GetString("provider.usps.address_line1"),
			CityLocality:                viper.GetString("provider.usps.city_locality"),
			StateProvince:               viper.GetString("provider.usps.state_province"),
			PostalCode:                  viper.GetString("provider.usps.postal_code"),
			CountryCode:                 viper.GetString("provider.usps.country_code"),
			AddressResidentialIndicator: viper.GetString("provider.usps.address_residential_indicator"),

			WeightUnit:    viper.GetString("provider.usps.weight_unit"),
			DimensionUnit: viper.GetString("provider.usps.dimension_unit"),
			ServiceCode:   viper.GetString("provider.usps.service_code"),
			CarrierID:     viper.GetString("provider.usps.carrier_id"),
		}
	}

	return &USPS{
		Domain:      opts.Domain,
		APIKey:      opts.APIKey,
		TrackingURL: opts.TrackingURL,
		ShipFrom: &Address{
			Name:                        opts.Name,
			Phone:                       opts.Phone,
			AddressLine1:                opts.AddressLine1,
			CityLocality:                opts.CityLocality,
			StateProvince:               opts.StateProvince,
			PostalCode:                  opts.PostalCode,
			CountryCode:                 opts.CountryCode,
			AddressResidentialIndicator: opts.AddressResidentialIndicator,
		},

		ServiceCode:   opts.ServiceCode,
		WeightUnit:    opts.WeightUnit,
		DimensionUnit: opts.DimensionUnit,
		CarrierID:     opts.CarrierID,
	}
}

// GenerateLabel create a label
func (p *USPS) GenerateLabel(pkg PackageResource) (*ResultResource, error) {
	service := p.ServiceCode
	if pkg.ServiceCode != "" {
		service = pkg.ServiceCode
	}

	data := RequestBody{
		Shipment: Shipment{
			ServiceCode: service,
			ShipFrom:    *p.ShipFrom,

			ShipTo: Address{
				CompanyName:                 pkg.CompanyName,
				Name:                        pkg.Name,
				Phone:                       pkg.Phone,
				AddressLine1:                pkg.AddressLine1,
				AddressLine2:                pkg.AddressLine2,
				CityLocality:                pkg.CityLocality,
				StateProvince:               pkg.StateProvince,
				PostalCode:                  pkg.PostalCode,
				CountryCode:                 pkg.CountryCode,
				AddressResidentialIndicator: pkg.AddressResidentialIndicator,
			},
			PackagesRequest: []PackagesRequest{
				{
					Weight: Weight{
						Value: pkg.Weight,
						Unit:  p.WeightUnit,
					},
					Dimensions: Dimensions{
						Length: pkg.Length,
						Width:  pkg.Width,
						Height: pkg.Height,
						Unit:   p.DimensionUnit,
					},
				},
			},
		},
	}

	out, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	payload := strings.NewReader(string(out))
	client := &http.Client{}
	url := fmt.Sprintf("%s/v1/labels", p.Domain)
	req, err := http.NewRequest(http.MethodPost, url, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Add("API-Key", p.APIKey)
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var dataResponse ResponseBody
	err = json.Unmarshal(body, &dataResponse)
	if err != nil {
		return nil, err
	}

	if dataResponse.Status == "completed" {
		result := &ResultResource{
			TrackingNumber: dataResponse.TrackingNumber,
			TrackingStatus: dataResponse.TrackingStatus,
			LabelURL:       dataResponse.LabelDownload.Png,
			ShipmentCost:   dataResponse.ShipmentCost.Amount,
			InsuranceCost:  dataResponse.InsuranceCost.Amount,
		}

		return result, nil
	}

	var errorResponse ErrorResponse
	err = json.Unmarshal(body, &errorResponse)
	if err != nil {
		return nil, err
	}

	if len(errorResponse.Errors) > 0 {
		result := &ResultResource{ErrorMessage: errorResponse.Errors[0].Message}
		return result, nil
	}

	result := &ResultResource{ErrorMessage: "Cant find error"}
	return result, nil
}

// TrackingLabel get information tracking
func (p *USPS) TrackingLabel(trackingNumber string) (*TrackingResult, error) {
	client := &http.Client{}
	url := fmt.Sprintf("%s/v1/tracking", p.Domain)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("API-Key", p.APIKey)

	q := req.URL.Query()
	q.Add("carrier_code", "usps")
	q.Add("tracking_number", trackingNumber)
	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	var errorResponse ErrorResponse
	err = json.Unmarshal(body, &errorResponse)
	if err != nil {
		return nil, err
	}

	if len(errorResponse.Errors) > 0 {
		return nil, errors.New(errorResponse.Errors[0].Message)
	}

	var dataResponse *TrackingResult
	err = json.Unmarshal(body, &dataResponse)

	if err != nil {
		return nil, err
	}
	//dataResponse.Events = append(dataResponse.Events, EventShipping{
	//	OccurredAt:        cast.ToTime("2019-09-13T12:32:00Z"),
	//	CarrierOccurredAt: cast.ToTime("2019-09-13T05:32:00"),
	//	Description:       "Arrived at USPS Facility",
	//	CityLocality:      "OCEANSIDE",
	//	StateProvince:     "CA",
	//	PostalCode:        "92056",
	//	CountryCode:       "",
	//	CompanyName:       "",
	//	Signer:            "",
	//	EventCode:         "U1",
	//})
	return dataResponse, nil
}

// GetServices list carrier services
func (p *USPS) GetServices() ([]CarrierService, error) {
	url := fmt.Sprintf("%s/v1/carriers/%s/services", p.Domain, p.CarrierID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("API-Key", p.APIKey)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	var dataResponse ServicesResponse
	err = json.Unmarshal(body, &dataResponse)
	if err != nil {
		return nil, err
	}

	return dataResponse.Services, nil
}

// GetCarrier view a Single Carrier
func (p *USPS) GetCarrier() (*Carrier, error) {
	url := fmt.Sprintf("%s/v1/carriers/%s", p.Domain, p.CarrierID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("API-Key", p.APIKey)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	carrier := &Carrier{}
	err = json.Unmarshal(body, carrier)
	if err != nil {
		return nil, err
	}

	return carrier, nil
}

// Rate Calculate Shipping Costs
func (p *USPS) Rate(form RateForm) (*RateResult, error, string) {
	url := fmt.Sprintf("%s/v1/rates", p.Domain)

	if form.AddressResidentialIndicator == "" {
		form.AddressResidentialIndicator = "no"
	}

	data := RateRequest{
		RateOptions{
			CarrierIds:   []string{p.CarrierID},
			ServiceCodes: form.ServiceCodes,
			PackageTypes: form.PackageTypes,
		},
		Shipment{
			ValidateAddress: form.ValidateAddress,
			ShipFrom:        *p.ShipFrom,
			ShipTo: Address{
				CompanyName:                 form.CompanyName,
				Name:                        form.Name,
				Phone:                       form.Phone,
				AddressLine1:                form.AddressLine1,
				AddressLine2:                form.AddressLine2,
				CityLocality:                form.CityLocality,
				StateProvince:               form.StateProvince,
				PostalCode:                  form.PostalCode,
				CountryCode:                 form.CountryCode,
				AddressResidentialIndicator: form.AddressResidentialIndicator,
			},
			PackagesRequest: []PackagesRequest{
				{
					Weight: Weight{
						Value: form.Weight,
						Unit:  p.WeightUnit,
					},
					Dimensions: Dimensions{
						Length: form.Length,
						Width:  form.Width,
						Height: form.Height,
						Unit:   p.DimensionUnit,
					},
				},
			},
		},
	}

	buf, err := json.Marshal(data)
	if err != nil {
		return nil, err, ""
	}

	payload := strings.NewReader(string(buf))
	req, err := http.NewRequest(http.MethodPost, url, payload)
	if err != nil {
		return nil, err, ""
	}

	req.Header.Add("API-Key", p.APIKey)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err, ""
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err, ""
	}

	var errorResponse ErrorResponse
	err = json.Unmarshal(body, &errorResponse)
	if err != nil {
		return nil, err, ""
	}

	if len(errorResponse.Errors) > 0 {
		return nil, nil, errorResponse.Errors[0].Message
	}

	result := &RateResult{}
	err = json.Unmarshal(body, result)
	if err != nil {
		return nil, err, ""
	}

	return result, nil, ""
}

func (p *USPS) CreateLabelRate(rateID string) (*ResultResource, error) {
	client := &http.Client{}
	url := fmt.Sprintf("%s/v1/labels/rates/%s", p.Domain, rateID)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("API-Key", p.APIKey)
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var dataResponse ResponseBody
	err = json.Unmarshal(body, &dataResponse)
	if err != nil {
		return nil, err
	}

	if dataResponse.Status == "completed" {
		result := &ResultResource{
			TrackingNumber: dataResponse.TrackingNumber,
			TrackingStatus: dataResponse.TrackingStatus,
			LabelURL:       dataResponse.LabelDownload.Png,
			ShipmentCost:   dataResponse.ShipmentCost.Amount,
			InsuranceCost:  dataResponse.InsuranceCost.Amount,
		}

		return result, nil
	}

	var errorResponse ErrorResponse
	err = json.Unmarshal(body, &errorResponse)
	if err != nil {
		return nil, err
	}

	if len(errorResponse.Errors) > 0 {
		result := &ResultResource{ErrorMessage: errorResponse.Errors[0].Message}
		return result, nil
	}

	result := &ResultResource{ErrorMessage: "Cant find error"}
	return result, nil
}
