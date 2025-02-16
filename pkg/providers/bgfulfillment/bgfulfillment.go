package bgfulfillment

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type BGFulfillment struct {
	Client          *Client
	APIKey          string
	Weight          float64    `json:"weight"`
	WeightUnit      string     `json:"weight_unit"`
	ImageFormat     string     `json:"image_format"`
	ImageResolution int        `json:"image_resolution"`
	Dimensions      Dimensions `json:"dimensions"`
	DimensionsUnit  string     `json:"dimensions_unit"`
	Address         *USPSAddress
	Usps            *USPS
}

func NewBG(opts *Options) *BGFulfillment {
	if opts == nil {
		opts = &Options{
			APIKey:          viper.GetString("provider.bgfulfillment.api_key"),
			WeightUnit:      viper.GetString("provider.bgfulfillment.weight_unit"),
			ImageFormat:     viper.GetString("provider.bgfulfillment.image_format"),
			ImageResolution: viper.GetInt("provider.bgfulfillment.image_resolution"),
			DimensionsUnit:  viper.GetString("provider.bgfulfillment.dimensions_unit"),

			Address: &USPSAddress{
				Company:    viper.GetString("provider.bgfulfillment.company_name"),
				City:       viper.GetString("provider.bgfulfillment.city_locality"),
				Line1:      viper.GetString("provider.bgfulfillment.address_line1"),
				State:      viper.GetString("provider.bgfulfillment.state_province"),
				PostalCode: viper.GetString("provider.bgfulfillment.postal_code"),
				Country:    viper.GetString("provider.bgfulfillment.country_code"),
				Phone:      viper.GetString("provider.bgfulfillment.phone"),
			},
			Usps: &USPS{
				Shape:             viper.GetString("provider.bgfulfillment.shape"),
				MailClass:         viper.GetString("provider.bgfulfillment.mail_class"),
				MailClassPriority: viper.GetString("provider.bgfulfillment.mail_class_priority"),
				ImageSize:         viper.GetString("provider.bgfulfillment.image_size"),
			},
		}
	}

	return &BGFulfillment{
		Client:          NewClient(BaseURL),
		APIKey:          opts.APIKey,
		WeightUnit:      opts.WeightUnit,
		DimensionsUnit:  opts.DimensionsUnit,
		ImageFormat:     opts.ImageFormat,
		ImageResolution: opts.ImageResolution,
		Address:         opts.Address,
		Usps:            opts.Usps,
	}
}

func (m *BGFulfillment) USPSCreateLabel(in LabelRequest) (*USPSResponse, string, error) {
	firstname := in.FirstName
	lastname := in.LastName
	if firstname == "" {
		firstname = in.FullName
	}

	if lastname == "" {
		lastname = in.FullName
	}

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
		APIKey:    m.APIKey,
		RequestID: in.OrderName,
		Usps: USPS{
			Shape:     m.Usps.Shape,
			MailClass: m.Usps.MailClass,
			ImageSize: m.Usps.ImageSize,
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
		Dimensions: Dimensions{
			Length: in.Length,
			Width:  in.Width,
			Height: in.Height,
		},
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
			PostalCode: in.WarehouseState,
			Country:    in.WarehouseCountry,
			Phone:      in.WarehousePhone,
		}
	}

	// Choose Package shipping
	if req.Weight*GramToOz > FirstClassLimit {
		req.Usps.MailClass = m.Usps.MailClassPriority
	}

	fmt.Println("==================", req.Dimensions, "=================", req.DimensionsUnit)

	var response interface{}
	err := m.Client.Post("/api/labels", req, &response)
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

	rerr := &ErrorResponse{}
	if result == nil || result.RequestID == "" {
		b, err := json.Marshal(response)
		if err != nil {
			return nil, "", err
		}

		err = json.Unmarshal(b, rerr)
		if err != nil {
			return nil, "", err
		}

		return nil, rerr.Error(), nil
	}

	return result, "", nil
}

func (m *BGFulfillment) DHLCreateLabel(in LabelRequest) (*DHLResponse, error) {
	if m.DimensionsUnit == "in" {
		in.Length = in.Length * CentimeterToInches
		in.Width = in.Width * CentimeterToInches
		in.Height = in.Height * CentimeterToInches
	}

	if in.Address1 == "" && in.Address2 != "" {
		in.Address1 = in.Address2
	}

	req := DhlRequest{
		APIKey:      m.APIKey,
		OrderNumber: in.OrderName,
		Shipment: DHLShipment{
			AddressFrom: DHLAddress{
				Name:    m.Address.Company,
				Street1: m.Address.Line1,
				City:    m.Address.City,
				State:   m.Address.State,
				Zip:     m.Address.PostalCode,
				Country: m.Address.Country,
			},
			AddressTo: DHLAddress{
				Name:    in.FullName,
				Street1: in.Address1,
				Street2: in.Address2,
				City:    in.City,
				State:   in.State,
				Zip:     in.Zipcode,
				Country: in.Country,
			},
			Parcels: []DHLParcels{
				{
					MassUnit:     m.WeightUnit,
					Weight:       fmt.Sprintf("%.0f", in.Weight),
					DistanceUnit: m.DimensionsUnit,
					Length:       fmt.Sprintf("%.4f", in.Length),
					Width:        fmt.Sprintf("%.4f", in.Width),
					Height:       fmt.Sprintf("%.4f", in.Height),
				},
			},
		},
	}

	// Nếu không truyền thông tin ware house -> lấy thông tin ware house mặc định
	if in.WarehouseAddress1 == "" {
		req.Shipment.AddressFrom = DHLAddress{
			Name:    m.Address.Company,
			City:    m.Address.City,
			Street1: m.Address.Line1,
			State:   m.Address.State,
			Zip:     m.Address.PostalCode,
			Country: m.Address.Country,
		}
	} else {
		req.Shipment.AddressFrom = DHLAddress{
			Name:    in.WarehouseCompany,
			City:    in.WarehouseCity,
			Street1: in.WarehouseAddress1,
			State:   in.WarehouseState,
			Zip:     in.WarehouseZipcode,
			Country: in.WarehouseCountry,
		}
	}

	var response interface{}
	err := m.Client.Post("/api/dhl/labels", req, &response)
	if err != nil {
		return nil, err
	}

	var res *DHLResponse
	b, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &res)
	if err != nil {
		return nil, err
	}

	if res.Status != "SUCCESS" {
		return res, errors.Errorf("%v", response)
	}

	return res, nil
}

func (m *BGFulfillment) DeleteLabel(tracking_number string) (string, error) {
	req := USPSRequest{APIKey: m.APIKey}
	var response interface{}
	err := m.Client.Delete(fmt.Sprintf("/api/labels/%s", tracking_number), req, &response)
	if err != nil {
		return "", err
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
