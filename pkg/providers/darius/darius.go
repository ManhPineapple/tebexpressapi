package darius

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/storage"

	"tebexpressapi/pkg/utils"

	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

var (
	DevYellowProductId  = "4021"
	DevRedProductId     = "4061"
	ProdYellowProductId = "2401"
	ProdRedProductId    = "2421"
	AUProductId         = "4561"
	AUVipproductId      = "4661"
	EUProductId         = "4541"
)

type Darius struct {
	Client *Client

	S3              storage.S3
	CustomerId      string
	CustomerUserId  string
	YellowProductId string
	RedProductId    string
	AUProductId     string
	AUVipproductId  string
	EUProductId     string

	EUShipperName     string
	EUShipperAddress  string
	EUShipperPostcode string
	EUShipperCity     string
	EUShipperCountry  string
	EUShipperPhone    string
}

func NewDarius(opts *Options) *Darius {
	if opts == nil {
		opts = &Options{
			BaseURL:  viper.GetString("provider.darius.base_url"),
			LabelURL: viper.GetString("provider.darius.label_url"),
			Username: viper.GetString("provider.darius.username"),
			Password: viper.GetString("provider.darius.password"),
		}
	}

	client := NewClient(opts)
	auth, err := client.Auth()
	if err != nil {
		panic(err)
	}

	darius := &Darius{
		Client:          client,
		CustomerId:      auth.CustomerId,
		CustomerUserId:  auth.CustomerUserId,
		YellowProductId: DevYellowProductId,
		RedProductId:    DevRedProductId,
		AUProductId:     AUProductId,
		AUVipproductId:  AUVipproductId,
		EUProductId:     EUProductId,

		EUShipperName:     viper.GetString("provider.darius.eu_shipper_name"),
		EUShipperAddress:  viper.GetString("provider.darius.eu_shipper_address"),
		EUShipperPostcode: viper.GetString("provider.darius.eu_shipper_postcode"),
		EUShipperCity:     viper.GetString("provider.darius.eu_shipper_city"),
		EUShipperCountry:  viper.GetString("provider.darius.eu_shipper_country"),
		EUShipperPhone:    viper.GetString("provider.darius.eu_shipper_phone"),
	}

	if viper.GetString("env") != "development" {
		darius.YellowProductId = ProdYellowProductId
		darius.RedProductId = ProdRedProductId
	}

	return darius
}

func (m *Darius) USPSCreateLabel(in LabelRequest) (*CreateLabelResponse, string, error) {
	req := &USPSRequest{
		BuyerId:           in.BuyerId,
		OrderPiece:        "1",
		ConsigneeMobile:   in.ConsigneeMobile,
		OrderReturnSign:   "N",
		TradeType:         "ZYXT",
		DutyType:          "DDP",
		ConsigneeName:     in.ConsigneeName,
		ConsigneeAddress:  in.ConsigneeAddress,
		Country:           in.Country,
		ConsigneeState:    in.ConsigneeState,
		ConsigneeCity:     in.ConsigneeCity,
		ConsigneePostcode: in.ConsigneePostcode,
		CustomerId:        m.CustomerId,
		CustomerUserId:    m.CustomerUserId,
		Weight:            in.Weight,
		CargoType:         "P",
		OrderVolumeParam: []*OrderVolumeParam{
			&OrderVolumeParam{
				VolumeHeight: in.VolumeHeight,
				VolumeLength: in.VolumeLength,
				VolumeWidth:  in.VolumeWidth,
				VolumeWeight: cast.ToString(cast.ToFloat64(in.VolumeWeight) / 1000),
			},
		},
		OrderInvoiceParam: []*OrderInvoiceParam{
			&OrderInvoiceParam{
				InvoiceAmount: "1",
				InvoicePcs:    "1",
				InvoiceTitle:  "Package",
				InvoiceWeight: cast.ToString(cast.ToFloat64(in.VolumeWeight) / 1000),
				SKU:           fmt.Sprintf("%v - %v", in.OrderNumber, in.Code),
			},
		},
	}

	if in.ServiceCode == constant.ServiceAUCode {
		if postCodes, ok := constant.AUZipCode[strings.ToUpper(in.ConsigneeState)]; ok && utils.ContainsString(postCodes, in.ConsigneePostcode) {
			req.ProductId = m.AUProductId
		} else {
			req.ProductId = m.AUProductId
		}
	}

	if in.ServiceCode == constant.ServiceEUCode {
		req.ProductId = m.EUProductId
		req.ShipperName = m.EUShipperName
		req.ShipperAddress1 = m.EUShipperAddress
		req.ShipperCity = m.EUShipperCity
		req.ShipperCountry = m.EUShipperCountry
		req.ShipperPostcode = m.EUShipperPostcode
		req.ShipperTelephone = m.EUShipperPhone
	}

	var response interface{}
	err := m.Client.Post("/createOrderApi.htm", req, &response)
	if err != nil {
		return nil, "", err
	}

	result := &CreateLabelResponse{}
	b, err := json.Marshal(response)
	if err != nil {
		return nil, "", err
	}

	err = json.Unmarshal(b, result)
	if err != nil {
		return nil, "", err
	}

	responseError := &ErrorResponse{}
	if result == nil || !cast.ToBool(result.Ack) {
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

func (m *Darius) GetLabel(orderId string) (string, error) {
	options := url.Values{}
	options.Set("PrintType", "lab10_10")
	options.Set("order_id", orderId)

	url := fmt.Sprintf("%v/order/FastRpt/PDF_NEW.aspx?%s", m.Client.LabelURL, options.Encode())
	log.Println(url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Create an HTTP client
	client := &http.Client{}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	base64PDF := base64.StdEncoding.EncodeToString(body)
	return base64PDF, nil
}

func (m *Darius) DeleteLabel(trackingNumber string) (string, error) {
	return "", nil
}

func (m *Darius) TrackingInfo(trackingNumber string) ([]*TrackingResponse, error) {
	return nil, nil
}
