package auspost

import (
	"encoding/json"
	"fmt"
	"tebexpressapi/pkg/utils"
	"time"
)

type FormUpdateShipment ShipmentRequest

type UpdateShipmentResponse struct {
	ShipmentID           string    `json:"shipment_id"`
	ShipmentReference    string    `json:"shipment_reference"`
	ShipmentCreationDate time.Time `json:"shipment_creation_date"`
	ShipmentModifiedDate time.Time `json:"shipment_modified_date"`
	EmailTrackingEnabled bool      `json:"email_tracking_enabled"`
	MovementType         string    `json:"movement_type"`
	ChargeToAccount      string    `json:"charge_to_account"`

	Items []struct {
		Weight                 float64     `json:"weight"`
		ContainsDangerousGoods bool        `json:"contains_dangerous_goods"`
		AuthorityToLeave       bool        `json:"authority_to_leave"`
		SafeDropEnabled        bool        `json:"safe_drop_enabled"`
		AllowPartialDelivery   bool        `json:"allow_partial_delivery"`
		ItemID                 string      `json:"item_id"`
		ItemReference          string      `json:"item_reference"`
		ProductID              string      `json:"product_id"`
		ItemContents           interface{} `json:"item_contents"`

		TrackingDetails struct {
			ArticleID     string `json:"article_id"`
			ConsignmentID string `json:"consignment_id"`
		} `json:"tracking_details"`

		ItemSummary struct {
			TotalCost      float64 `json:"total_cost"`
			TotalCostExGst float64 `json:"total_cost_ex_gst"`
			TotalGst       float64 `json:"total_gst"`
			Status         string  `json:"status"`
		} `json:"item_summary"`
	} `json:"items"`

	ShipmentSummary struct {
		TotalCost       float64        `json:"total_cost"`
		TotalCostExGst  float64        `json:"total_cost_ex_gst"`
		ShippingCost    float64        `json:"shipping_cost"`
		FuelSurcharge   float64        `json:"fuel_surcharge"`
		TotalGst        float64        `json:"total_gst"`
		Status          string         `json:"status"`
		NumberOfItems   int            `json:"number_of_items"`
		TrackingSummary map[string]int `json:"tracking_summary"`
	} `json:"shipment_summary"`

	Options interface{} `json:"options"`
}

func (au *Auspost) UpdateShipment(shipmentID string, in LabelRequest) (*UpdateShipmentResponse, *ResponseErrors, error) {
	to := Address{
		Name:         in.FullName,
		Lines:        []string{in.Address1},
		Suburb:       in.City,
		State:        in.State,
		Postcode:     in.Zipcode,
		Phone:        in.Phone,
		Email:        in.Email,
		Country:      in.Country,
		BusinessName: in.Company,
	}

	if in.Address2 != "" {
		to.Lines = append(to.Lines, in.Address2)
	}

	from := au.Address
	if in.WarehouseAddress1 != "" {
		from = Address{
			Name:     in.WarehouseCompany,
			Lines:    []string{in.WarehouseAddress1},
			Suburb:   in.WarehouseCity,
			State:    in.WarehouseState,
			Postcode: in.WarehouseZipcode,
			Phone:    in.WarehousePhone,
			Country:  in.WarehouseCountry,
			// BusinessName: in.WarehouseCompany,
		}
	}

	var emailTrackingEnabled bool
	if in.Email != "" {
		emailTrackingEnabled = true
	}

	references := []string{}
	if in.ServiceCode != "" {
		references = append(references, in.ServiceCode)
	}

	if in.Code != "" {
		references = append(references, in.Code)
	}

	payload := FormUpdateShipment{
		ShipmentReference:    fmt.Sprintf("%v-%v", in.PackageID, time.Now().Unix()),
		SenderReferences:     references,
		EmailTrackingEnabled: emailTrackingEnabled,
		From:                 from,
		To:                   to,
		Items: []Item{
			{
				ItemReference:        in.OrderNumber,
				ProductID:            au.ProductParcelPostID,
				Length:               in.Length,
				Height:               in.Height,
				Width:                in.Width,
				Weight:               in.Weight,
				AuthorityToLeave:     utils.Bool(true),
				AllowPartialDelivery: utils.Bool(true),
			},
		},
	}

	var response interface{}
	if err := au.Client.Put("/shipping/v1/shipments/"+shipmentID, payload, &response); err != nil {
		return nil, nil, err
	}

	b, err := json.Marshal(response)
	if err != nil {
		return nil, nil, err
	}

	result := &UpdateShipmentResponse{}
	err = json.Unmarshal(b, result)
	if err != nil {
		return nil, nil, err
	}

	if result.ShipmentID != "" {
		return result, nil, nil
	}

	errors := &ResponseErrors{}
	err = json.Unmarshal(b, errors)
	if err != nil {
		return nil, nil, err
	}

	return nil, errors, nil
}
