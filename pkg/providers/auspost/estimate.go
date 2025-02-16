package auspost

import "encoding/json"

type ShipmentPriceForm struct {
	Form  Address `json:"from"`
	To    Address `json:"to"`
	Items []Item  `json:"items"`
}

type EstimateRequest struct {
	Shipments []ShipmentPriceForm `json:"shipments"`
}

type EstimateResponseItem struct {
	Weight        float64 `json:"weight"`
	Height        float64 `json:"height"`
	Length        float64 `json:"length"`
	Width         float64 `json:"width"`
	ItemReference string  `json:"item_reference"`

	Prices []struct {
		CalculatedGst        float64 `json:"calculated_gst"`
		CalculatedPrice      float64 `json:"calculated_price"`
		CalculatedPriceExGst float64 `json:"calculated_price_ex_gst"`
		ProductID            string  `json:"product_id"`
		ProductType          string  `json:"product_type"`
		BundledPrice         float64 `json:"bundled_price"`
		BundledPriceExGst    float64 `json:"bundled_price_ex_gst"`
		BundledGst           float64 `json:"bundled_gst"`

		Options struct {
			SignatureOnDeliveryOption bool `json:"signature_on_delivery_option"`
			AuthorityToLeaveOption    bool `json:"authority_to_leave_option"`
			DangerousGoodsAllowed     bool `json:"dangerous_goods_allowed"`
		} `json:"options"`
	} `json:"prices"`

	Warnings []ResponseError `json:"warnings"`
	Errors   []ResponseError `json:"errors"`
}

type EstimateResponse struct {
	Shipments []struct {
		Form            Address     `json:"from"`
		To              Address     `json:"to"`
		Items           []Item      `json:"items"`
		Options         interface{} `json:"options"`
		Features        interface{} `json:"features"`
		ShipmentSummary struct {
			TotalCost      float64 `json:"total_cost"`
			TotalCostExGst float64 `json:"total_cost_ex_gst"`
			ShippingCost   float64 `json:"shipping_cost"`
			FuelSurcharge  float64 `json:"fuel_surcharge"`
			TotalGst       float64 `json:"total_gst"`
			Status         string  `json:"status"`
			NumberOfItems  int     `json:"number_of_items"`
		} `json:"shipment_summary"`
	} `json:"shipments"`
}

func (au *Auspost) Estimate(in LabelRequest) (*EstimateResponse, *ResponseErrors, error) {
	shipment := ShipmentPriceForm{
		Form: Address{
			Name:         in.WarehouseCompany,
			Lines:        []string{in.WarehouseAddress1},
			Suburb:       in.WarehouseCity,
			State:        in.WarehouseState,
			Postcode:     in.WarehouseZipcode,
			Phone:        in.WarehousePhone,
			Country:      in.WarehouseCountry,
			BusinessName: in.WarehouseCompany,
		},
		To: Address{
			Name:         in.FullName,
			Lines:        []string{in.Address1, in.Address2},
			Suburb:       in.City,
			State:        in.State,
			Postcode:     in.Zipcode,
			Phone:        in.Phone,
			Email:        in.Email,
			Country:      in.Country,
			BusinessName: in.Company,
		},
		Items: []Item{
			{
				ProductID:     au.ProductParcelPostID,
				ItemReference: in.OrderNumber,
				Length:        in.Length,
				Height:        in.Height,
				Width:         in.Width,
				Weight:        in.Weight,
			},
		},
	}

	if shipment.Form.Postcode == "" {
		shipment.Form = au.Address
	}

	payload := EstimateRequest{
		Shipments: []ShipmentPriceForm{shipment},
	}

	var res interface{}
	if err := au.Client.Post("/shipping/v1/prices/shipments", payload, &res); err != nil {
		return nil, nil, err
	}

	result := &EstimateResponse{}
	b, err := json.Marshal(res)
	if err != nil {
		return nil, nil, err
	}

	err = json.Unmarshal(b, result)
	if err != nil {
		return nil, nil, err
	}

	if len(result.Shipments) > 0 {
		return result, nil, nil
	}

	errors := &ResponseErrors{}
	err = json.Unmarshal(b, errors)
	if err != nil {
		return nil, nil, err
	}

	return nil, errors, nil
}
