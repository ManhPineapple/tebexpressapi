package kiloship

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"tebexpressapi/pkg/calculate"
	"time"

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

	serviceLevelToken := USPS_GROUND_ADVANTAGE

	// Kiloship GROUND_ADVENTAGE only
	// if CheckChangeClass(in) {
	// 	serviceLevelToken = USPS_PRIORITY
	// }

	req := KiloshipCreateLabelRequest{
		Shipment: KiloshipShipment{
			Async: false,
			Parcels: []KiloshipParcel{
				{
					Weight:       fmt.Sprintf("%.2f", in.Weight*gramToOz),
					Width:        fmt.Sprintf("%.2f", in.Width*cmToInch),
					Length:       fmt.Sprintf("%.2f", in.Length*cmToInch),
					Height:       fmt.Sprintf("%.2f", in.Height*cmToInch),
					MassUnit:     "oz",
					DistanceUnit: "in",
				},
			},
			AddressTo: KiloshipAddress{
				Name:      in.ToName,
				Address_1: in.ToStreet1,
				Address_2: in.ToStreet2,
				City:      in.ToCity,
				State:     in.ToState,
				Zipcode:   in.ToZip,
				Country:   "US",
			},
		},
		ServiceLevelToken: serviceLevelToken,
		Metadata:          in.Metadata,
	}

	if in.WarehouseAddress1 != "" {
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
	err := m.Client.Post("/api/shipping-labels/domestic", req, &response)

	if err != nil {
		log.Printf("Kiloship POST request failed: %v", err)
		return nil, err
	}

	result := &KiloshipCreateLabelResponse{}
	b, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling Kiloship response: %v", err)
		return nil, err
	}

	err = json.Unmarshal(b, result)
	if err != nil {
		log.Printf("Error unmarshaling into KiloshipCreateLabelResponse: %v", err)
		return nil, err
	}

	if result.LabelImageURL != "" {
		log.Println("Kiloship label created successfully")
		return result, nil
	}

	responseError := &KiloshipErrorResponse{}
	err = json.Unmarshal(b, responseError)
	if err != nil {
		log.Printf("Error unmarshaling into KiloshipErrorResponse: %v", err)
		return nil, err
	}

	if responseError != nil && len(responseError.Error.Errors) > 0 {
		log.Printf("Kiloship error detail: %s", responseError.Error.Errors[0].Detail)
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
	err := m.Client.Get(fmt.Sprintf("/api/tracking/%s?responseType=DETAIL", trackingNumber), &response, nil)
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

func (m *Kiloship) CancelLabel(trackingNumber string) (bool, error) {
	var rawResp interface{}
	endpoint := fmt.Sprintf("/api/shipping-labels/domestic/%s", trackingNumber)

	if err := m.Client.Delete(endpoint, nil, &rawResp); err != nil {
		return false, err
	}
	if rawResp == nil {
		return false, fmt.Errorf("no response from Kiloship")
	}

	var resp KiloshipCancelLabelResponse
	data, err := json.Marshal(rawResp)
	if err != nil {
		return false, fmt.Errorf("failed to marshal raw response: %w", err)
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return false, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !resp.Success {
		return false, fmt.Errorf("kiloship error: %s", resp.Message)
	}

	return true, nil
}

func (m *Kiloship) EstimateCost(fromZipCode string, toZipCode string) (zone int, cost float64, err error) {
	var rawZoneResp interface{}
	now := time.Now().Format("2006-01-02")
	endpoint := fmt.Sprintf("/api/addresses/zone/number?originZIPCode=%s&destinationZIPCode=%s&mailingDate=%s", fromZipCode, toZipCode, now)

	if err = m.Client.Get(endpoint, &rawZoneResp, nil); err != nil {
		return 0, 0, err
	}
	if rawZoneResp == nil {
		return 0, 0, fmt.Errorf("no response from Kiloship")
	}

	zoneResp := struct {
		Zone int `json:"zone"`
	}{}

	data, err := json.Marshal(rawZoneResp)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to marshal raw response: %w", err)
	}
	if err := json.Unmarshal(data, &zoneResp); err != nil {
		return 0, 0, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return zoneResp.Zone, float64(zoneResp.Zone), nil
}

func (m *Kiloship) GetCityAndStateFromZipcode() {
	// Not needed yet — to be implemented later
}

func (m *Kiloship) GetZipcodeFromAddress() {
	// Not needed yet — to be implemented later
}
