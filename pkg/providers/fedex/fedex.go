package fedex

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/utils"

	"github.com/spf13/viper"
)

type FedEx struct {
	BaseURL            string
	ClientID           string
	ClientSecret       string
	AuthGrantType      string
	AccountNumber      string
	ServiceType        string
	PickUpType         string
	PackagingType      string
	PaymentType        string
	DescriptionProduct string
	Shipper            AddressInfo
}

const (
	DemensionUnit       = "CM"
	CurrencyUnit        = "USD"
	WeightUnit          = "KG"
	ImageLabelType      = "PDF"
	LabelStockType      = "PAPER_85X11_TOP_HALF_LABEL"
	LabelResponseOption = "LABEL"
	QuantityUnit        = "PCS"
)

func InitFedEx() *FedEx {
	return &FedEx{
		BaseURL:            viper.GetString("fedex.base_url"),
		ClientID:           viper.GetString("fedex.client_id"),
		ClientSecret:       viper.GetString("fedex.client_secret"),
		AuthGrantType:      viper.GetString("fedex.auth_grant_type"),
		AccountNumber:      viper.GetString("fedex.account_number"),
		ServiceType:        viper.GetString("fedex.service_type"),
		PickUpType:         viper.GetString("fedex.pickup_type"),
		PackagingType:      viper.GetString("fedex.packaging_type"),
		PaymentType:        viper.GetString("fedex.payment_type"),
		DescriptionProduct: viper.GetString("fedex.description"),
		Shipper: AddressInfo{
			Contact: Contact{
				PhoneNumber: viper.GetString("fedex.contact.phone_number"),
				PersonName:  viper.GetString("fedex.contact.name"),
				CompanyName: viper.GetString("fedex.contact.company_name"),
			},
		},
	}
}

func InitFedEx2() *FedEx {
	return &FedEx{
		BaseURL:            viper.GetString("fedex_2.base_url"),
		ClientID:           viper.GetString("fedex_2.client_id"),
		ClientSecret:       viper.GetString("fedex_2.client_secret"),
		AuthGrantType:      viper.GetString("fedex_2.auth_grant_type"),
		AccountNumber:      viper.GetString("fedex_2.account_number"),
		ServiceType:        viper.GetString("fedex_2.service_type"),
		PickUpType:         viper.GetString("fedex_2.pickup_type"),
		PackagingType:      viper.GetString("fedex_2.packaging_type"),
		PaymentType:        viper.GetString("fedex_2.payment_type"),
		DescriptionProduct: viper.GetString("fedex_2.description"),
		Shipper: AddressInfo{
			Contact: Contact{
				PhoneNumber: viper.GetString("fedex_2.contact.phone_number"),
				PersonName:  viper.GetString("fedex_2.contact.name"),
				CompanyName: viper.GetString("fedex_2.contact.company_name"),
			},
		},
	}
}

func (f *FedEx) Authorization() (*FedExAuthReponse, error) {
	client := &http.Client{}
	data := url.Values{}
	data.Set("client_id", f.ClientID)
	data.Set("client_secret", f.ClientSecret)
	data.Set("grant_type", f.AuthGrantType)
	encodedData := data.Encode()
	path := fmt.Sprintf("%s%s", f.BaseURL, "/oauth/token")
	log.Println("fedex path: ", path)
	req, err := http.NewRequest("POST", path, strings.NewReader(encodedData))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	result := &FedExAuthReponse{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (f *FedEx) CreateLabel(containers []entity.Container, warehouse, sWarehouse *entity.Warehouse, shipmentValue float64) (*CreateFedExShipmentResponse, error) {
	body := f.MakeBodyRequest(containers, warehouse, sWarehouse, shipmentValue)
	result, err := f.Authorization()
	if err != nil {
		return nil, err
	}
	headers := make(map[string]string)
	headers["Authorization"] = fmt.Sprintf("Bearer %s", result.AccessToken)
	path := "ship/v1/shipments"
	client := utils.NewClient(f.BaseURL, "")
	response := new(CreateFedExShipmentResponse)

	logRequest, _ := json.Marshal(body)
	log.Println("Fedex request creat label:", string(logRequest))

	req, err := client.NewRequest("POST", path, body, nil, headers)
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

func (f *FedEx) TrackInfo(trackingNumber string) (*FedExTrackingInfoResponse, error) {
	data := make([]Tracking, 0)
	data = append(data, Tracking{TrackingNumberInfo: TrackingData{
		TrackingNumber: trackingNumber,
	}})
	body := TrackingFedExShipmentRequest{
		TrackingInfo:         data,
		IncludeDetailedScans: true,
	}
	headers := make(map[string]string)
	result, err := f.Authorization()
	if err != nil {
		return nil, err
	}
	headers["Authorization"] = fmt.Sprintf("Bearer %s", result.AccessToken)
	path := "track/v1/trackingnumbers"
	client := utils.NewClient(f.BaseURL, "")
	response := new(FedExTrackingInfoResponse)

	req, err := client.NewRequest("POST", path, body, nil, headers)
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

func (f *FedEx) MakeBodyRequest(containers []entity.Container, warehouse, sWarehouse *entity.Warehouse, shipmentValue float64) CreateFexExShipmentRequest {
	items := make([]RequestedPackageLineItem, 0)
	totalW := float64(0)
	for _, container := range containers {
		tmp := RequestedPackageLineItem{
			Weight: Weight{
				Units: WeightUnit,
				Value: container.Weight,
			},
			Dimensions: Dimension{
				Length: container.Length,
				Width:  container.Width,
				Height: container.Height,
				Units:  DemensionUnit,
			},
		}
		totalW += container.Weight
		items = append(items, tmp)
	}

	Recipients := make([]AddressInfo, 0)
	Commodities := make([]Commodity, 0)
	additionalMeasures := make([]AdditionalMeasure, 0)
	unitPrice := DeclaredValue{
		Amount:   math.Floor(shipmentValue/(float64)(len(containers))*100) / 100, //upload amount
		Currency: CurrencyUnit,
	}
	totalPrice := DeclaredValue{
		Amount:   shipmentValue, //upload amount
		Currency: CurrencyUnit,
	}
	additionalMeasures = append(additionalMeasures, AdditionalMeasure{
		Quantity: len(containers),
		Units:    QuantityUnit,
	})
	stateCode := warehouse.State
	if warehouse.Country != "US" {
		stateCode = ""
	}
	Recipients = append(Recipients, AddressInfo{
		Address: Address{StreetLines: []string{warehouse.Address},
			StateOrProvinceCode: stateCode,
			PostalCode:          warehouse.Zipcode,
			City:                warehouse.City,
			CountryCode:         warehouse.Country,
		},
		Contact: Contact{
			PhoneNumber: warehouse.Phone,
			CompanyName: warehouse.Company,
			PersonName:  warehouse.Name,
		}})
	Commodities = append(Commodities, Commodity{
		Description:          f.DescriptionProduct,
		CountryOfManufacture: sWarehouse.Country,
		Weight: Weight{
			Units: WeightUnit,
			Value: totalW,
		},
		UnitPrice:          unitPrice,
		AdditionalMeasures: additionalMeasures,
		Quantity:           len(containers),
		QuantityUnits:      QuantityUnit,
		NumberOfPieces:     len(containers),
	})
	data := CreateFexExShipmentRequest{
		RequestedShipment: RequestShipment{
			CustomsClearanceDetail: CustomsClearance{
				DutiesPayment: Payment{
					PaymentType: f.PaymentType,
				},
				TotalCustomsValue: totalPrice,
				Commodities:       Commodities,
			},
			Shipper: AddressInfo{
				Address: Address{StreetLines: []string{sWarehouse.Address},
					StateOrProvinceCode: sWarehouse.Country,
					PostalCode:          sWarehouse.Zipcode,
					City:                sWarehouse.City,
					CountryCode:         sWarehouse.Country,
				},
				Contact: Contact{
					PhoneNumber: sWarehouse.Phone,
					CompanyName: f.Shipper.Contact.CompanyName,
					PersonName:  f.Shipper.Contact.PersonName,
				}},
			Recipients:                Recipients,
			PickupType:                f.PickUpType,
			ServiceType:               f.ServiceType,
			PackagingType:             f.PackagingType,
			RequestedPackageLineItems: items,
			LabelSpecification: LabelSpecification{
				LabelStockType: LabelStockType,
				ImageType:      ImageLabelType,
			},
			ShippingChargesPayment: Payment{
				PaymentType: f.PaymentType,
			},
		},
		LabelResponseOptions: LabelResponseOption,
		AccountNumber:        AccountNumber{Value: f.AccountNumber},
	}
	return data
}
