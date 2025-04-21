package kiloship

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"tebexpressapi/pkg/calculate"

	"github.com/spf13/viper"
)

type Kiloship struct {
	Client *Client
}

func NewKiloship(opts *KiloshipOptions) *Kiloship {
	if opts == nil {
		opts = &KiloshipOptions{
			BaseURL: viper.GetString("provider.kiloship.base_url"),
			ApiKey:  viper.GetString("provider.kiloship.api_key"),
		}
	}

	return &Kiloship{
		Client: NewClient(opts.BaseURL, opts.ApiKey),
	}
}

func (m *Kiloship) CreateDomesticLabel(in KiloshipCreateLabelObject) (*KiloshipCreateLabelResponse, error) {
	if in.PackageID <= 0 {
		return nil, errors.New("Package ID cannot empty")
	}

	serviceLevelToken := USPS_GROUND_ADVENTAGE
	if CheckChangeClass(in) {
		serviceLevelToken = USPS_PRIORITY
	}

	req := KiloshipCreateLabelRequest{
		Shipment: KiloshipShipment{
			Async: false,
			Parcels: []KiloshipParcel{
				{
					Weight:       fmt.Sprintf("%.2f", in.Weight*gramToOz),
					Width:        fmt.Sprintf("%.2f", in.Width),
					Length:       fmt.Sprintf("%.2f", in.Length),
					Height:       fmt.Sprintf("%.2f", in.Height),
					MassUnit:     "oz",
					DistanceUnit: "cm",
				},
			},
			AddressTo: KiloshipAddress{
				Name:      in.ToName,
				Address_1: in.ToStreet1,
				City:      in.ToCity,
				State:     in.ToState,
				Zipcode:   in.ToZip,
				Country:   "US",
			},
		},
		ServiceLevelToken: serviceLevelToken,
		Metadata:          in.Metadata,
	}

	if in.WarehouseAddress1 == "" {
		req.Shipment.AddressFrom = KiloshipAddress{
			Name:      in.WarehouseCompany,
			Address_1: in.WarehouseAddress1,
			City:      in.WarehouseCity,
			State:     in.WarehouseState,
			Zipcode:   in.WarehouseZipcode,
			Country:   in.WarehouseCountry,
		}
	} else {
		req.Shipment.AddressFrom = ANANBAY_ADDRESS
	}

	logRequest, _ := json.Marshal(req)
	log.Printf("Kiloship request payload: %v", string(logRequest))

	var response interface{}
	// Production API endpoint (/api)
	err := m.Client.Post("/shipping-labels/domestic", req, &response)
	// Sandbox API endpoint (/api/test)
	// err := m.Client.Post("/shipping-labels/domestics", req, &response)

	if err != nil {
		return nil, err
	}

	result := &KiloshipCreateLabelResponse{}
	b, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, result)
	if err != nil {
		return nil, err
	}

	if result.LabelImageURL != "" {
		return result, nil
	}

	responseError := &KiloshipErrorResponse{}
	err = json.Unmarshal(b, responseError)
	if err != nil {
		return nil, err
	}

	if responseError != nil && len(responseError.Error.Errors) > 0 {
		return nil, errors.New(responseError.Error.Errors[0].Detail)
	}

	return nil, errors.New("Unknown error")
}

func CheckChangeClass(in KiloshipCreateLabelObject) bool {
	length, height, width := calculate.ParseVolumes(in.Length, in.Width, in.Height)
	lag := calculate.LengthAndGirth(length, height, width)

	if length > 22 || width > 18 || height > 15 || lag > 108 {
		return true
	}

	return false
}

func (m *Kiloship) TrackingLabel(trackingNumber string) (*KiloshipTrackingInfoResponse, error) {
	var response interface{}
	err := m.Client.Get(fmt.Sprintf("/shipping-labels/%s", trackingNumber), &response, nil)
	if err != nil {
		return nil, err
	}

	result := &KiloshipTrackingInfoResponse{}

	b, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (m *Kiloship) GetCityAndStateFromZipcode() {
	// Not needed yet — to be implemented later
}

func (m *Kiloship) GetZipcodeFromAddress() {
	// Not needed yet — to be implemented later
}
