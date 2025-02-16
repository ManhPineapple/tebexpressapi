package yun

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/spf13/viper"
)

const (
	AUShippingMethod = "AUSP"
)

type Yun struct {
	Client           *Client
	Sender           *Sender
	ShippingMethod   string
	AUShippingMethod string
}

func NewYun(opts *Options) *Yun {
	if opts == nil {
		opts = &Options{
			BaseURL:  viper.GetString("provider.yun.base_url"),
			UserName: viper.GetString("provider.yun.username"),
			Password: viper.GetString("provider.yun.password"),

			Sender: &Sender{
				Company:     viper.GetString("provider.yun.company_name"),
				City:        viper.GetString("provider.yun.city_locality"),
				Street:      viper.GetString("provider.yun.address_line1"),
				State:       viper.GetString("provider.yun.state_province"),
				Zip:         viper.GetString("provider.yun.postal_code"),
				CountryCode: viper.GetString("provider.yun.country_code"),
			},
		}
	}

	return &Yun{
		Client:         NewClient(opts.BaseURL, opts.UserName, opts.Password),
		Sender:         opts.Sender,
		ShippingMethod: opts.ShippingMethod,
	}
}

func (m *Yun) USPSCreateLabel(in LabelRequest) (*USPSResponse, string, error) {

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

	req := USPSRequest{
		CustomerOrderNumber: in.Code,
		ShippingMethodCode:  AUShippingMethod,
		PackageCount:        1,
		Weight:              in.Weight,
		Receiver: &Receiver{
			CountryCode: in.Country,
			FirstName:   firstname,
			LastName:    lastname,
		},
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

func (m *Yun) DeleteLabel(trackingNumber string) (string, error) {
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

func (m *Yun) TrackingInfo(trackingNumber string) ([]*TrackingResponse, error) {
	var response interface{}
	err := m.Client.Get(fmt.Sprintf("/v1/track/%s", trackingNumber), &response, nil)
	if err != nil {
		return nil, err
	}

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

func (m *Yun) CreateManifest(in ManifestRequest) (*CreateManifestResponse, string, error) {

	return nil, "", nil
}

func (m *Yun) ValidateAddress(req AddressRequest) (*CleanseAddressResponse, string, error) {
	return nil, "", nil

}

func (m *Yun) EstimateCost(req EstimateCostRequest) (*EstimateCostResponse, string, error) {
	return nil, "", nil
}

func (m *Yun) USPSCreateLabel2(in LabelRequest) (*USPSResponse, string, error) {
	return nil, "", nil
}

func (m *Yun) USPSCreateLabel3(in LabelRequest) (*USPSResponse, string, error) {
	return nil, "", nil
}

func (m *Yun) USPSUpdateLabel(in LabelRequest) (*USPSResponse, string, error) {
	return nil, "", nil
}
