package ups

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/utils"

	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

type UPS struct {
	ShipFrom      ShipInfo
	ShipTo        ShipInfo
	Shipper       ShipInfo
	AccountNumber string
	Username      string
	Password      string
	// AccessKey     string
	BaseURL     string
	BeeUrl      string
	Beetoken    string
	Description string
}

func NewUPS() *UPS {

	return &UPS{
		ShipFrom: ShipInfo{
			Name:          viper.GetString("ship_from.name"),
			AttentionName: viper.GetString("ship_from.attention_name"),
			Phone: Phone{
				Number: viper.GetString("ship_from.phone_number"),
			},
			ShipperNumber: viper.GetString("ups.ship_number"),
			Address: Address{
				AddressLine:       []string{viper.GetString("ship_from.address.address_line1"), viper.GetString("ship_from.address.address_line2")},
				City:              viper.GetString("ship_from.address.city"),
				StateProvinceCode: viper.GetString("ship_from.address.state_code"),
				PostalCode:        viper.GetString("ship_from.address.postal_code"),
				CountryCode:       viper.GetString("ship_from.address.country_code"),
			},
		},
		Shipper: ShipInfo{
			Name:                           viper.GetString("ship_from.name"),
			AttentionName:                  viper.GetString("ship_from.attention_name"),
			ShipperTaxIdentificationNumber: viper.GetString("ship_from.tax_identification_number"),
			Phone: Phone{
				Number: viper.GetString("ship_from.phone_number"),
			},
			ShipperNumber: viper.GetString("ups.ship_number"),
			Address: Address{
				AddressLine:       []string{viper.GetString("ship_from.address.address_line1"), viper.GetString("ship_from.address.address_line2")},
				City:              viper.GetString("ship_from.address.city"),
				StateProvinceCode: viper.GetString("ship_from.address.state_code"),
				PostalCode:        viper.GetString("ship_from.address.postal_code"),
				CountryCode:       viper.GetString("ship_from.address.country_code"),
			},
		},
		AccountNumber: viper.GetString("ups.ship_number"),
		Username:      viper.GetString("ups.username"),
		Password:      viper.GetString("ups.password"),
		// AccessKey:     viper.GetString("ups.access_key"),
		BaseURL:     viper.GetString("ups.domain"),
		Description: viper.GetString("ups.description"),

		BeeUrl:   viper.GetString("bee.domain"),
		Beetoken: viper.GetString("bee.token"),
	}
}

func NewUPS2() *UPS {
	return &UPS{
		ShipFrom: ShipInfo{
			Name:          viper.GetString("ship_from_2.name"),
			AttentionName: viper.GetString("ship_from_2.attention_name"),
			Phone: Phone{
				Number: viper.GetString("ship_from_2.phone_number"),
			},
			ShipperNumber: viper.GetString("ups_2.ship_number"),
			Address: Address{
				AddressLine:       []string{viper.GetString("ship_from_2.address.address_line1"), viper.GetString("ship_from_2.address.address_line2")},
				City:              viper.GetString("ship_from_2.address.city"),
				StateProvinceCode: viper.GetString("ship_from_2.address.state_code"),
				PostalCode:        viper.GetString("ship_from_2.address.postal_code"),
				CountryCode:       viper.GetString("ship_from_2.address.country_code"),
			},
		},
		Shipper: ShipInfo{
			Name:          viper.GetString("ship_from_2.name"),
			AttentionName: viper.GetString("ship_from_2.attention_name"),
			Phone: Phone{
				Number: viper.GetString("ship_from_2.phone_number"),
			},
			ShipperNumber: viper.GetString("ups_2.ship_number"),
			Address: Address{
				AddressLine:       []string{viper.GetString("ship_from_2.address.address_line1"), viper.GetString("ship_from_2.address.address_line2")},
				City:              viper.GetString("ship_from_2.address.city"),
				StateProvinceCode: viper.GetString("ship_from_2.address.state_code"),
				PostalCode:        viper.GetString("ship_from_2.address.postal_code"),
				CountryCode:       viper.GetString("ship_from_2.address.country_code"),
			},
		},
		AccountNumber: viper.GetString("ups_2.ship_number"),
		Username:      viper.GetString("ups_2.username"),
		Password:      viper.GetString("ups_2.password"),
		// AccessKey:     viper.GetString("ups_2.access_key"),
		BaseURL:     viper.GetString("ups_2.domain"),
		BeeUrl:      viper.GetString("bee.domain"),
		Beetoken:    viper.GetString("bee.token"),
		Description: viper.GetString("ups_2.description"),
	}
}

func (u *UPS) getAccessToken() (string, error) {
	client := utils.NewClient(u.BaseURL, "")

	body := map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     u.Username,
		"client_secret": u.Password,
	}

	var resp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}

	req, err := client.NewRequest("POST", "security/v1/oauth/token", body, nil, nil)
	if err != nil {
		return "", err
	}

	err = client.Do(req, &resp)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s %s", resp.TokenType, resp.AccessToken), nil
}

func (u *UPS) CreateLabel(containers []entity.Container, warehouse *entity.Warehouse) (*UPSResponse, error) {
	data := u.makeBodyRequest(containers, warehouse)

	token, err := u.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get UPS access token: %w", err)
	}

	headers := map[string]string{
		"Authorization": token,
		"Content-Type":  "application/json",
	}

	path := "/api/shipments/v2409/ship"
	client := utils.NewClient(u.BaseURL, "")
	response := new(UPSResponse)

	req, err := client.NewRequest("POST", path, data, nil, headers)
	if err != nil {
		return nil, err
	}

	err = client.Do(req, &response)
	if err != nil {
		return nil, err
	}
	return response, nil

}

func (u *UPS) CreateLabelOnePackage(containers []entity.Container, warehouse *entity.Warehouse) (*UPSResponseOnePackage, error) {
	data := u.makeBodyRequest(containers, warehouse)

	token, err := u.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get UPS access token: %w", err)
	}

	headers := map[string]string{
		"Authorization": token,
		"Content-Type":  "application/json",
	}

	path := "/api/shipments/v2409/ship"
	client := utils.NewClient(u.BaseURL, "")
	response := new(UPSResponseOnePackage)

	req, err := client.NewRequest("POST", path, data, nil, headers)
	if err != nil {
		return nil, err
	}

	err = client.Do(req, &response)
	if err != nil {
		if client.ErrorReponse != nil {
			return nil, client.ErrorReponse
		}
		return nil, err
	}
	return response, nil

}

func (u *UPS) makeBodyRequest(containers []entity.Container, warehouse *entity.Warehouse) *UPSRequest {
	upsServiceCode := constant.ServiceCodeWorldWideSaver
	if containers[0].FbaType == constant.FbaTypeStandard {
		upsServiceCode = constant.ServiceCodeExpedited
	}

	packageUPSRequest := make([]Package, 0)
	for _, container := range containers {
		tmp := Package{
			Packaging: Code{
				Code: constant.PackagingCodeCustomerSuppliedPackage,
			},
			Dimensions: Dimesions{
				UnitOfMeasurement: Code{Code: constant.UnitOfMeasurementCodeCentimeters},
				Length:            cast.ToString(container.Length),
				Width:             cast.ToString(container.Width),
				Height:            cast.ToString(container.Height),
			},
			PackageWeight: PackageWeight{
				UnitOfMeasurement: Code{
					Code: constant.UnitOfMeasurementCodeKilograms,
				},
				Weight: cast.ToString(container.Weight),
			},
		}
		packageUPSRequest = append(packageUPSRequest, tmp)
	}

	data := &UPSRequest{
		ShipmentRequest: ShipmentRequest{
			Shipment: Shipment{
				Description: u.Description,
				Shipper:     u.Shipper,
				ShipTo: ShipInfo{
					Name:          warehouse.Name,
					AttentionName: warehouse.Company,
					Phone: Phone{
						Number: warehouse.Phone,
					},
					ShipperNumber: viper.GetString("ups.ship_number"),
					Address: Address{
						AddressLine:       []string{warehouse.Address},
						City:              warehouse.City,
						StateProvinceCode: warehouse.State,
						PostalCode:        warehouse.Zipcode,
						CountryCode:       warehouse.Country,
					},
				},
				ShipFrom: u.ShipFrom,
				PaymentInformation: PaymentInformation{
					ShipmentCharge: ShipmentCharge{
						Type:        constant.ShipmentChargeTypeTransportation,
						BillShipper: BillShipper{AccountNumber: u.AccountNumber},
					},
				},
				Service: Code{
					Code: upsServiceCode,
				},
				Package: packageUPSRequest,
			},
		},
	}
	data.ShipmentRequest.Request.RequestOption = "nonvalidate"

	return data
}

func (u *UPS) TrackDetail(track string) (*TrackDetailResponse, error) {
	token, err := u.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get UPS access token: %w", err)
	}

	headers := map[string]string{
		"Authorization": token,
		"Content-Type":  "application/json",
	}

	path := fmt.Sprintf("track/v1/details/%s?locale=en_US", track)
	client := utils.NewClient(u.BaseURL, "")

	response := new(TrackDetailResponse)
	req, err := client.NewRequest("GET", path, nil, nil, headers)
	if err != nil {
		return nil, err
	}

	err = client.Do(req, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (u *UPS) TrackBeeApi(tracking string) (*BeeTrackDetailResponse, error) {
	data := &BeeRequest{
		Hawbno: tracking,
		Token:  u.Beetoken,
	}
	path := "ofone/tracing/airshipment"
	client := utils.NewClient(u.BeeUrl, "")
	var response interface{}
	req, err := client.NewRequest("POST", path, data, nil, nil)
	if err != nil {
		return nil, err
	}

	err = client.Do(req, &response)
	if err != nil {
		return nil, err
	}
	result := new(BeeTrackDetailResponse)
	b, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, result)
	if err != nil {
		return nil, err
	}
	responseError := &BeeTrackDetailResponseErr{}
	if result.TrackInfo[0].MasterBillNo == "" {
		b, err := json.Marshal(response)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(b, responseError)
		if err != nil {
			return nil, err
		}

		return nil, errors.New(responseError.TrackInfo[0].MsgResult)
	}

	return result, nil
}

func (u *UPS) EstimateCost(input UPSInputRateRequest) (*UPSRateResponse, error) {
	packages := []RatePackage{}
	for _, item := range input.Items {
		weight := item.Weight / constant.KgToGram

		tmp := RatePackage{
			Packaging: Code{
				Code: constant.PackagingCodeCustomerSuppliedPackage,
			},
			Dimensions: Dimesions{
				UnitOfMeasurement: Code{Code: constant.UnitOfMeasurementCodeCentimeters},
				Length:            cast.ToString(item.Length),
				Width:             cast.ToString(item.Width),
				Height:            cast.ToString(item.Height),
			},
			PackageWeight: PackageWeight{
				UnitOfMeasurement: Code{
					Code: constant.UnitOfMeasurementCodeKilograms,
				},
				Weight: cast.ToString(weight),
			},
		}

		packages = append(packages, tmp)
	}

	data := &UPSRateRequest{
		RateRequest: RateRequest{
			RateShipment: RateShipment{
				Shipper: u.Shipper,
				ShipTo: ShipInfo{
					Name:          input.ShipAddress.Name,
					AttentionName: input.ShipAddress.Company,
					Phone: Phone{
						Number: input.ShipAddress.Phone,
					},
					ShipperNumber: viper.GetString("ups.ship_number"),
					Address: Address{
						AddressLine:       []string{input.ShipAddress.Address},
						City:              input.ShipAddress.City,
						StateProvinceCode: input.ShipAddress.State,
						PostalCode:        input.ShipAddress.Zipcode,
						CountryCode:       input.ShipAddress.Country,
					},
				},
				ShipFrom: u.ShipFrom,
				PaymentInformation: PaymentInformation{
					ShipmentCharge: ShipmentCharge{
						Type:        constant.ShipmentChargeTypeTransportation,
						BillShipper: BillShipper{AccountNumber: u.AccountNumber},
					},
				},
				Service: Code{
					Code: constant.ServiceCodeWorldWideSaver,
				},
				Package: packages,
			},
		},
	}

	b, _ := json.Marshal(data)
	log.Println(string(b))

	token, err := u.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get UPS access token: %w", err)
	}

	headers := map[string]string{
		"Authorization": token,
		"Content-Type":  "application/json",
	}

	path := "ship/v1807/rating/Rate"
	client := utils.NewClient(u.BaseURL, "")
	response := new(UPSRateResponse)

	req, err := client.NewRequest("POST", path, data, nil, headers)
	if err != nil {
		return nil, err
	}

	err = client.Do(req, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}
